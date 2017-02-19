// Copyright 2013 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package drive

import (
	"io/ioutil"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
)

func (g *Commands) Init() error {
	g.context.ClientId = os.Getenv(GoogleApiClientIdEnvKey)
	g.context.ClientSecret = os.Getenv(GoogleApiClientSecretEnvKey)
	if g.context.ClientId == "" || g.context.ClientSecret == "" {
		g.context.ClientId = "354790962074-7rrlnuanmamgg1i4feed12dpuq871bvd.apps.googleusercontent.com"
		g.context.ClientSecret = "RHjKdah8RrHFwu6fcc0uEVCw"
	}

	ctx := context.Background()
	refreshToken, err := RetrieveRefreshToken(ctx, g.context)
	if err != nil {
		return err
	}

	g.context.RefreshToken = refreshToken
	return g.context.Write()
}

// We don't need to perform an OAuth2.0 exchange
func (g *Commands) InitWithServiceAccount(gsaFilepath string) error {
	if gsaFilepath == "" {
		// We should be cautious enough not to blindly read in
		// by default a Google Service account, because this
		// could be a security breach to phish for accounts.
		// The user has to explicitly pass in the path
		return &Error{
			code:   StatusSecurityException,
			status: "a path has to be explicitly set for service accounts",
		}
	}

	blob, err := ioutil.ReadFile(gsaFilepath)
	if err != nil {
		return err
	}

	jwtConfig, err := google.JWTConfigFromJSON(blob, DriveScope)
	if err != nil {
		return err
	}

	// Next we'll just transfer the attributes directly
	// by means of JSON marshaling the already vetted JWTConfig
	g.context.GSAJWTConfig = jwtConfig

	// Since it validates alright, let's now write it to disk
	return g.context.Write()
}

func (g *Commands) DeInit() error {
	prompt := func(args ...interface{}) bool {
		if !g.opts.canPrompt() {
			return true
		}

		status := promptForChanges(args...)
		return accepted(status)
	}

	return g.context.DeInitialize(prompt, true)
}
