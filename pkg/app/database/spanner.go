// Copyright 2023 Google LLC
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

package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/cloud-android-orchestration/pkg/app/session"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const SpannerDBType = "Spanner"

const (
	credentialsTable  = "Credentials"
	usernameColumn    = "username"
	credentialsColumn = "credentials"

	sessionsTable            = "Sessions"
	sessionKeyColumn         = "session_key"
	sessionOAuth2StateColumn = "oauth2_state"
	sessionAccessColumn      = "accessed_at"

	sessionStateValidityHours = 48
)

// A database service that works with a Cloud Spanner database with the following schema:
//
//	table Credentials {
//	  username string primary key
//	  credentials byte array # wide enough to store an encrypted JSON-serialized oauth2.Token object
//	}
//	table Sessions {
//	  session_key string primary key
//	  oauth2_state string
//	  accessed_at timestamp
//	}
type SpannerDBService struct {
	db string
}

func NewSpannerDBService(db string) *SpannerDBService {
	return &SpannerDBService{db}
}

func (dbs *SpannerDBService) FetchBuildAPICredentials(username string) ([]byte, error) {
	ctx := context.TODO()
	client, err := spanner.NewClient(ctx, dbs.db)
	if err != nil {
		return nil, fmt.Errorf("Failed to create db client: %w", err)
	}
	defer client.Close()

	row, err := client.Single().ReadRow(ctx, credentialsTable, spanner.Key{username}, []string{credentialsColumn})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			// Not found is not an error
			return nil, nil
		}
		return nil, fmt.Errorf("Error querying database: %w", err)
	}

	var credentials []byte
	err = row.Column(0, &credentials)
	return credentials, err
}

func (dbs *SpannerDBService) StoreBuildAPICredentials(username string, credentials []byte) error {
	ctx := context.TODO()
	client, err := spanner.NewClient(ctx, dbs.db)
	if err != nil {
		return err
	}
	defer client.Close()

	columns := []string{usernameColumn, credentialsColumn}
	mutations := []*spanner.Mutation{
		spanner.InsertOrUpdate(credentialsTable, columns, []interface{}{username, credentials}),
	}
	_, err = client.Apply(ctx, mutations)
	return err
}

func (dbs *SpannerDBService) DeleteBuildAPICredentials(username string) error {
	ctx := context.TODO()
	client, err := spanner.NewClient(ctx, dbs.db)
	if err != nil {
		return err
	}
	defer client.Close()
	mutation := spanner.Delete(credentialsTable, spanner.KeySetFromKeys(spanner.Key{username}))
	_, err = client.Apply(ctx, []*spanner.Mutation{mutation})
	if spanner.ErrCode(err) == codes.NotFound {
		// Not an error if not found
		return nil
	}
	return err
}

func (dbs *SpannerDBService) CreateOrUpdateSession(s session.Session) error {
	ctx := context.TODO()
	client, err := spanner.NewClient(ctx, dbs.db)
	if err != nil {
		return err
	}
	defer client.Close()
	columns := []string{sessionKeyColumn, sessionOAuth2StateColumn, sessionAccessColumn}
	mutation := spanner.InsertOrUpdate(sessionsTable, columns, []interface{}{s.Key, s.OAuth2State, time.Now()})
	_, err = client.Apply(ctx, []*spanner.Mutation{mutation})
	go dbs.deleteExpiredSessions()
	return err
}

func (dbs *SpannerDBService) FetchSession(key string) (*session.Session, error) {
	ctx := context.TODO()
	client, err := spanner.NewClient(ctx, dbs.db)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	row, err := client.Single().ReadRow(ctx, sessionsTable, spanner.Key{key}, []string{sessionKeyColumn, sessionOAuth2StateColumn})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			// Not found is not an error
			return nil, nil
		}
		return nil, fmt.Errorf("Failed to retrive session: %w", err)
	}
	session := &session.Session{
		Key:         key,
		OAuth2State: "",
	}
	var state spanner.NullString
	if err := row.ColumnByName(sessionOAuth2StateColumn, &state); err != nil {
		return nil, err
	}
	if state.Valid {
		session.OAuth2State = state.StringVal
	}
	return session, nil
}

func (dbs *SpannerDBService) DeleteSession(key string) error {
	ctx := context.TODO()
	client, err := spanner.NewClient(ctx, dbs.db)
	if err != nil {
		return err
	}
	defer client.Close()
	mutation := spanner.Delete(sessionsTable, spanner.KeySetFromKeys(spanner.Key{key}))
	_, err = client.Apply(ctx, []*spanner.Mutation{mutation})
	if spanner.ErrCode(err) == codes.NotFound {
		// Not an error if not found
		return nil
	}
	return err
}

// TODO(jemoreira): Remove once sessions are used for more than just storing oauth2 states.
func (dbs *SpannerDBService) deleteExpiredSessions() {
	ctx := context.TODO()
	client, err := spanner.NewClient(ctx, dbs.db)
	if err != nil {
		log.Println("Failed to create db client to delete expired sessions")
		return
	}
	defer client.Close()
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: fmt.Sprintf("delete from %s where %s < @threshold", sessionsTable, sessionAccessColumn),
			Params: map[string]interface{}{
				"threshold": time.Now().Add(-sessionStateValidityHours * time.Hour),
			},
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		log.Printf("%d expired session(s) deleted.\n", rowCount)
		return nil
	})
	if err != nil {
		log.Println("Failed to delete expired sessions: ", err)
	}
}
