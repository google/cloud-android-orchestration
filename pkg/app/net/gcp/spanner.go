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

package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"google.golang.org/grpc/codes"
)

const (
	credentialsTable  = "Credentials"
	usernameColumn    = "username"
	credentialsColumn = "credentials"
)

// A database service that works with a Cloud Spanner database with the following schema:
//
//	table Credentials {
//	  username string primary key
//	  credentials byte array # wide enough to store an encrypted JSON-serialized oauth2.Token object
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
