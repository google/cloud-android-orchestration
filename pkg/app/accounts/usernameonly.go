// Copyright 2024 Google LLC
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

package accounts

import (
	"html/template"
	"net/http"
	"strings"
)

const (
	UsernameOnlyAMType AMType = "username-only"
	unameCookie        string = "accountUsername"
)

// Implements the AccountManager interfaces for closed deployed cloud
// orchestrators. This AccountManager first gets the account information from
// HTTP cookies, and leverages the RFC 2617 HTTP Basic Authentication if the
// cookie does not present. Note that only username is used for both cases.
type UsernameOnlyAccountManager struct{}

func NewUsernameOnlyAccountManager() *UsernameOnlyAccountManager {
	return &UsernameOnlyAccountManager{}
}

func (m *UsernameOnlyAccountManager) UserFromRequest(r *http.Request) (User, error) {
	// Accept putting the username in a cookie to support using a browser
	// to interact with CO.
	cookie, err := r.Cookie(unameCookie)
	if err == nil && cookie.Value != "" {
		return &UsernameOnlyUser{cookie.Value}, nil
	}
	username, _, ok := r.BasicAuth()
	if !ok {
		return nil, nil
	}
	return &UsernameOnlyUser{username}, nil
}

type UsernameOnlyUser struct {
	username string
}

func (u *UsernameOnlyUser) Username() string { return u.username }

func (u *UsernameOnlyUser) Email() string { return "" }

type LoggingData struct {
	Username string
	Error    string
}

var loggingTemplate = template.Must(template.New("logging").Parse(`
<!DOCTYPE html>
<html>
<head><title>AM Logging</title></head>
<body>
    {{if .Error}}
        <p style="color: red; font-size:2vw;">{{.Error}}</p>
    {{end}}
    {{if .Username}}
        <h2> UsernameOnly account manager</h2>
        <p style="font-size:3vw;">Welcome {{.Username}}!</p>
        <p style="font-size:2vw;">You can now visit other pages.</p>
    {{else}}
        <form method="POST">
            <h2> Logging username for UsernameOnly account manager</h2>
            <label for="uname">username:</label>
            <input type="text" id="uname" name="username" required><br><br>
            <input type="submit" value="Submit">
        </form>
    {{end}}
</body>
</html>
`))

func UsernameOnlyLoggingForm(w http.ResponseWriter, r *http.Request) error {
	return loggingTemplate.Execute(w, nil)
}

func HandleUsernameOnlyLogging(w http.ResponseWriter, r *http.Request, redirect string) error {
	username := r.FormValue("username")
	if strings.TrimSpace(username) == "" {
		return loggingTemplate.Execute(w, LoggingData{
			Error: "Please enter a valid username",
		})
	}
	http.SetCookie(w, &http.Cookie{
		Name:  unameCookie,
		Value: username,
		Path:  "/",
	})
	if redirect != "" {
		http.Redirect(w, r, redirect, http.StatusFound)
		return nil
	}
	return loggingTemplate.Execute(w, LoggingData{
		Username: username,
	})
}
