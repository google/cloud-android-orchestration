// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	hoapi "github.com/google/android-cuttlefish/frontend/src/liboperator/api/v1"
	apiv1 "github.com/google/cloud-android-orchestration/api/v1"
	"github.com/google/cloud-android-orchestration/pkg/app/accounts"
	"github.com/google/cloud-android-orchestration/pkg/app/config"
	"github.com/google/cloud-android-orchestration/pkg/app/database"
	"github.com/google/cloud-android-orchestration/pkg/app/encryption"
	apperr "github.com/google/cloud-android-orchestration/pkg/app/errors"
	"github.com/google/cloud-android-orchestration/pkg/app/instances"
	appOAuth2 "github.com/google/cloud-android-orchestration/pkg/app/oauth2"
	"github.com/google/cloud-android-orchestration/pkg/app/session"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

const (
	sessionIdCookie = "sessionid"
)

// The controller implements the web API of the cloud orchestrator. It parses
// and validates requests from the client and passes the information to the
// relevant modules
type App struct {
	instanceManager          instances.Manager
	accountManager           accounts.Manager
	oauth2Helper             *appOAuth2.Helper
	encryptionService        encryption.Service
	databaseService          database.Service
	connectorStaticFilesPath string
	infraConfig              apiv1.InfraConfig
}

func NewApp(
	im instances.Manager,
	am accounts.Manager,
	oc *appOAuth2.Helper,
	es encryption.Service,
	dbs database.Service,
	webStaticFilesPath string,
	webRTCConfig config.WebRTCConfig) *App {
	return &App{im, am, oc, es, dbs, webStaticFilesPath, buildInfraCfg(webRTCConfig.STUNServers)}
}

func (c *App) Handler() http.Handler {
	router := mux.NewRouter()

	// Instance Manager Routes
	router.Handle("/v1/zones/{zone}/hosts", c.Authenticate(c.createHost)).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts", c.Authenticate(c.listHosts)).Methods("GET")
	// Waits for the specified operation to be DONE or for the request to approach the specified deadline,
	// `503 Service Unavailable` error will be returned if the deadline is reached and the operation is not done.
	// Be prepared to retry if the deadline was reached.
	// It returns the expected response of the operation in case of success. If the original method returns no
	// data on success, such as `Delete`, response will be empty. If the original method is standard
	// `Get`/`Create`/`Update`, the response should be the relevant resource.
	router.Handle("/v1/zones/{zone}/operations/{operation}/:wait", c.Authenticate(c.waitOperation)).Methods("POST")
	router.Handle("/v1/zones/{zone}/hosts/{host}", c.Authenticate(c.deleteHost)).Methods("DELETE")

	// Host Orchestrator Proxy Routes
	router.Handle("/v1/zones/{zone}/hosts/{host}/{hostPath:.*}", c.Authenticate(c.ForwardToHost))

	// Infra route
	router.HandleFunc("/v1/zones/{zone}/hosts/{host}/infra_config", func(w http.ResponseWriter, r *http.Request) {
		// TODO(b/220891296): Make this configurable
		replyJSON(w, c.InfraConfig(), http.StatusOK)
	}).Methods("GET")

	// Global routes
	router.Handle("/auth", HTTPHandler(c.AuthHandler)).Methods("GET")
	router.Handle("/oauth2callback", HTTPHandler(c.OAuth2Callback))
	router.Handle("/", c.Authenticate(indexHandler))

	return router
}

func (a *App) InfraConfig() apiv1.InfraConfig {
	return a.infraConfig
}

func (a *App) ForwardToHost(w http.ResponseWriter, r *http.Request, user accounts.User) error {
	hostPath := "/" + mux.Vars(r)["hostPath"]

	if interceptFile, found := a.findInterceptFile(hostPath); found {
		http.ServeFile(w, r, interceptFile)
		return nil
	}

	hostClient, err := a.instanceManager.GetHostClient(getZone(r), getHost(r))
	if err != nil {
		return err
	}

	// Credentials are only needed in created cvd requests
	if hostPath == "/cvds" && r.Method == http.MethodPost {
		if err := a.injectCredentialsIntoCreateCVDRequest(r, user); err != nil {
			return err
		}
	}

	r.URL.Path = hostPath
	hostClient.GetReverseProxy().ServeHTTP(w, r)
	return nil
}

func (a *App) injectCredentialsIntoCreateCVDRequest(r *http.Request, user accounts.User) error {
	// Read and parse the request body
	msg := hoapi.CreateCVDRequest{}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return err
	}
	// Credentials are only injected on non-user builds that don't already provide user credentials.
	if msg.CVD != nil && msg.CVD.BuildSource != nil &&
		msg.CVD.BuildSource.AndroidCIBuildSource != nil &&
		msg.CVD.BuildSource.AndroidCIBuildSource.Credentials == "" {
		// Get the credentials.
		tk, err := a.fetchUserCredentials(user)
		if err != nil {
			return err
		}
		if tk == nil {
			// There are no credentials in database associated with this user.
			// Even though the user is already authenticated by the time this code executes, 401 is
			// returned to cause it to go the /auth URL and follow the OAuth2 flow. If 403 were to
			// be returned from here it could be construed as "this user doesn't have access to the
			// resource" instead of "this user has not authorized the app to access the resource on
			// their behalf". Another way to look at it is that the app lacks authentication
			// (credentials) to access the build API.
			return apperr.NewUnauthenticatedError(
				"The user must authorize the system to access the Build API on their behalf", nil)
		}
		credentials := tk.AccessToken
		msg.CVD.BuildSource.AndroidCIBuildSource.Credentials = credentials
	}
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	// Replace the request body with a new reader for the reverse proxy to consume.
	r.ContentLength = int64(len(buf))
	r.Body = io.NopCloser(bytes.NewReader(buf))
	return nil
}

func (c *App) createHost(w http.ResponseWriter, r *http.Request, user accounts.User) error {
	var msg apiv1.CreateHostRequest
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		return apperr.NewBadRequestError("Malformed JSON in request", err)
	}
	op, err := c.instanceManager.CreateHost(getZone(r), &msg, user)
	if err != nil {
		return err
	}
	replyJSON(w, op, http.StatusOK)
	return nil
}

func (c *App) listHosts(w http.ResponseWriter, r *http.Request, user accounts.User) error {
	listReq, err := BuildListHostsRequest(r)
	if err != nil {
		return err
	}
	res, err := c.instanceManager.ListHosts(getZone(r), user, listReq)
	if err != nil {
		return err
	}
	replyJSON(w, res, http.StatusOK)
	return nil
}

func (c *App) deleteHost(w http.ResponseWriter, r *http.Request, user accounts.User) error {
	name := mux.Vars(r)["host"]
	res, err := c.instanceManager.DeleteHost(getZone(r), user, name)
	if err != nil {
		return err
	}
	replyJSON(w, res, http.StatusOK)
	return nil
}

func (c *App) waitOperation(w http.ResponseWriter, r *http.Request, user accounts.User) error {
	name := mux.Vars(r)["operation"]
	op, err := c.instanceManager.WaitOperation(getZone(r), user, name)
	if err != nil {
		return err
	}
	replyJSON(w, op, http.StatusOK)
	return nil
}

func (c *App) AuthHandler(w http.ResponseWriter, r *http.Request) error {
	sessionKey := randomHexString()
	state := randomHexString()
	if err := c.databaseService.CreateOrUpdateSession(session.Session{
		Key:         sessionKey,
		OAuth2State: state,
	}); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionIdCookie,
		Value:    sessionKey,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
	authURL := c.oauth2Helper.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusSeeOther)
	return nil
}

func (c *App) OAuth2Callback(w http.ResponseWriter, r *http.Request) error {
	authCode, err := c.parseAuthorizationResponse(r)
	if err != nil {
		return err
	}
	tk, err := c.oauth2Helper.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		return fmt.Errorf("Error exchanging token: %w", err)
	}
	idToken, err := extractIDToken(tk)
	if err != nil {
		return fmt.Errorf("Error extracting id token: %w", err)
	}
	tokenClaims, ok := idToken.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("Id token in unexpected format")
	}
	user, err := c.accountManager.OnOAuth2Exchange(w, r, appOAuth2.IDTokenClaims(tokenClaims))
	if err != nil {
		return err
	}
	if err := c.storeUserCredentials(user, tk); err != nil {
		return err
	}
	// Don't return a real page here since any resource (i.e JS module) will have access to the
	// server response
	_, err = fmt.Fprintf(w, "Authorization successful, you may close this window now")
	return err
}

// Extracts the authorization code and state from the authorization provider's response.
func (c *App) parseAuthorizationResponse(r *http.Request) (string, error) {
	query := r.URL.Query()

	// Discard an authorization error first.
	if errMsg, ok := query["error"]; ok {
		return "", fmt.Errorf("Authentication error: %v", errMsg)
	}

	// Validate oauth2 state.
	stateSlice, ok := query["state"]
	if !ok {
		return "", fmt.Errorf("No OAuth2 State present")
	}
	state := stateSlice[0]

	sessionCookie, err := r.Cookie(sessionIdCookie)
	if err != nil {
		return "", fmt.Errorf("Error reading cookie from request: %w", err)
	}
	sessionKey := sessionCookie.Value
	session, err := c.databaseService.FetchSession(sessionKey)
	if err != nil {
		return "", fmt.Errorf("Error fetching session from db: %w", err)
	}
	if state != session.OAuth2State {
		return "", apperr.NewBadRequestError("OAuth2 State doesn't match session", nil)
	}

	// The state should be used only once. Delete the entire session since it's only being used for
	// OAuth2 state.
	c.databaseService.DeleteSession(sessionKey)

	// Extract the authorization code.
	code, ok := query["code"]
	if !ok {
		return "", fmt.Errorf("Authorization response does not include an authorization code")
	}
	return code[0], nil
}

func (c *App) storeUserCredentials(user accounts.User, tk *oauth2.Token) error {
	creds, err := json.Marshal(tk)
	if err != nil {
		return fmt.Errorf("Failed to serialize credentials: %w", err)
	}
	encryptedCreds, err := c.encryptionService.Encrypt(creds)
	if err != nil {
		return fmt.Errorf("Failed to encrypt credentials: %w", err)
	}
	if err := c.databaseService.StoreBuildAPICredentials(user.Username(), encryptedCreds); err != nil {
		return fmt.Errorf("Failed to store credentials: %w", err)
	}
	return nil
}

func (c *App) fetchUserCredentials(user accounts.User) (*oauth2.Token, error) {
	encryptedCreds, err := c.databaseService.FetchBuildAPICredentials(user.Username())
	if err != nil {
		return nil, fmt.Errorf("Error getting user credentials: %w", err)
	}
	if encryptedCreds == nil {
		return nil, nil
	}
	creds, err := c.encryptionService.Decrypt(encryptedCreds)
	if err != nil {
		// It's unlikely to be able to recover from this error in the future, the best approach is
		// probably to delete the user credentials and ask for authorization again.
		if err := c.databaseService.DeleteBuildAPICredentials(user.Username()); err != nil {
			log.Println("Error deleting user credentials: ", err)
		}
		return nil, err
	}
	tk := &oauth2.Token{}
	if err := json.Unmarshal(creds, tk); err != nil {
		// This is also likely unrecoverable.
		if err := c.databaseService.DeleteBuildAPICredentials(user.Username()); err != nil {
			log.Println("Error deleting user credentials: ", err)
		}
		return nil, fmt.Errorf("Error deserializing token: %w", err)
	}
	if !tk.Valid() {
		// Refresh the token and store it in the db.
		tks := c.oauth2Helper.TokenSource(context.TODO(), tk)
		tk, err = tks.Token()
		if err != nil {
			return nil, fmt.Errorf("Error refreshing token: %w", err)
		}
		if err := c.storeUserCredentials(user, tk); err != nil {
			// This won't stop the current operation, but will force a refresh in future requests.
			log.Println("Error storing refreshed tokens: ", err)
		}
	}
	return tk, nil
}

// Returns the received http handler wrapped in another that extracts user
// information from the request and passes it to to the original handler as
// the last parameter.
// The wrapper will only pass the request to the inner handler if a user is
// authenticated, otherwise it may choose to return an error or respond with
// an HTTP redirect to the login page.
func (a *App) Authenticate(fn AuthHTTPHandler) HTTPHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		user, err := a.accountManager.UserFromRequest(r)
		if err != nil {
			return err
		}
		if user == nil {
			return apperr.NewUnauthenticatedError("Authentication required", nil)
		}
		return fn(w, r, user)
	}
}

type AuthHTTPHandler func(http.ResponseWriter, *http.Request, accounts.User) error
type HTTPHandler func(http.ResponseWriter, *http.Request) error

// Intercept errors returned by the HTTPHandler and transform them into HTTP
// error responses
func (h HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, " ", r.URL, " ", r.RemoteAddr)
	if err := h(w, r); err != nil {
		log.Println("Error: ", err)
		var e *apperr.AppError
		if errors.As(err, &e) {
			replyJSON(w, e.JSONResponse(), e.StatusCode)
		} else {
			replyJSON(w, apiv1.Error{ErrorMsg: "Internal Server Error"}, http.StatusInternalServerError)
		}
	}
}

func (a *App) findInterceptFile(hostPath string) (string, bool) {
	// Currently only server_connector.js is to be intercepted
	if ok, _ := regexp.MatchString("/devices/[^/]+/files/js/server_connector.js", hostPath); ok {
		return a.connectorStaticFilesPath + "/intercept/js/server_connector.js", true
	}
	return "", false
}

func HostOrchestratorPath(path, host string) (string, error) {
	split := strings.SplitN(path, "hosts/"+host, 2)
	if len(split) != 2 {
		return "", fmt.Errorf("URL path doesn't have host component: %s", path)
	}
	return split[1], nil
}

func randomHexString() string {
	// This produces a 64 char random string from the [0-9a-f] alphabet or 256 bits.
	// The alternative of generating 32 random bytes and base64 encoding would add complexity and
	// overhead while reducing the size of the generated string to 48 characters.
	return fmt.Sprintf("%.16x%.16x%.16x%.16x", rand.Uint64(), rand.Uint64(), rand.Uint64(), rand.Uint64())
}

const (
	queryParamMaxResults = "maxResults"
)

func BuildListHostsRequest(r *http.Request) (*instances.ListHostsRequest, error) {
	maxResultsRaw := r.URL.Query().Get(queryParamMaxResults)
	maxResults, err := uint32Value(maxResultsRaw)
	if err != nil {
		return nil, newInvalidQueryParamError(queryParamMaxResults, maxResultsRaw, err)
	}
	res := &instances.ListHostsRequest{
		MaxResults: maxResults,
		PageToken:  r.URL.Query().Get("pageToken"),
	}
	return res, nil
}

func newInvalidQueryParamError(param, value string, err error) error {
	return apperr.NewBadRequestError(fmt.Sprintf("Invalid query parameter %q value: %q", param, value), err)
}

func uint32Value(value string) (uint32, error) {
	if value == "" {
		return uint32(0), nil
	}
	uint64v, err := strconv.ParseUint(value, 10, 32)
	return uint32(uint64v), err
}

func indexHandler(w http.ResponseWriter, r *http.Request, user accounts.User) error {
	fmt.Fprintln(w, "Home page")
	return nil
}

func extractIDToken(tk *oauth2.Token) (*jwt.Token, error) {
	val := tk.Extra("id_token")
	if val == nil {
		return nil, fmt.Errorf("No id token in oauth2 server response")
	}
	tokenString, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("Unexpected id token in oauth2 response")
	}
	// No need to verify the JWT here since it came directly from the oauth2 provider.
	noVerify := func(tk *jwt.Token) (interface{}, error) { return nil, nil }
	idTk, err := jwt.Parse(tokenString, noVerify)
	if err != nil && errors.Is(err, jwt.ErrInvalidKeyType) {
		// The error should be invalid key type because of the nil return in the lambda.
		return nil, err
	}
	return idTk, nil
}

// Get an int form value from the request. A form value can be in the query
// string or in the request body in either URL or Multipart encoding. If there
// is no form parameter with the given name the default value is returned. If a
// parameter with that name exists but the value can't be converted to an int an
// error is returned.
func intFormValue(r *http.Request, name string, def int) (int, error) {
	str := r.FormValue(name)
	if str == "" {
		return def, nil
	}
	i, e := strconv.Atoi(str)
	if e != nil {
		return 0, fmt.Errorf("Invalid %s value: %s", name, str)
	}
	return i, nil
}

// Send a JSON http response to the client
func replyJSON(w http.ResponseWriter, obj any, statusCode int) error {
	if statusCode != http.StatusOK {
		w.WriteHeader(statusCode)
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	return encoder.Encode(obj)
}

func getZone(r *http.Request) string {
	return mux.Vars(r)["zone"]
}

func getHost(r *http.Request) string {
	return mux.Vars(r)["host"]
}

func notAllowedHttpHandler(w http.ResponseWriter, r *http.Request) error {
	return apperr.NewMethodNotAllowedError("Operation is disabled", nil)
}

func buildInfraCfg(servers []string) apiv1.InfraConfig {
	iceServers := []apiv1.IceServer{}
	for _, server := range servers {
		iceServers = append(iceServers, apiv1.IceServer{URLs: []string{server}})
	}
	return apiv1.InfraConfig{
		IceServers: iceServers,
	}
}
