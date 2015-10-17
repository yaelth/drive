// Copyright 2015 Google Inc. All Rights Reserved.
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

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"

	"github.com/odeke-em/extractor"
	"github.com/odeke-em/meddler"
	"github.com/odeke-em/rsc/qr"
)

var envKeyAlias = &extractor.EnvKey{
	PubKeyAlias:  "DRIVE_SERVER_PUB_KEY",
	PrivKeyAlias: "DRIVE_SERVER_PRIV_KEY",
}

var envKeySet = extractor.KeySetFromEnv(envKeyAlias)

func main() {
	if envKeySet.PublicKey == "" {
		errorPrint("publicKey not set. Please set %s in your env.\n", envKeyAlias.PubKeyAlias)
		return
	}

	if envKeySet.PrivateKey == "" {
		errorPrint("privateKey not set. Please set %s in your env.\n", envKeyAlias.PrivKeyAlias)
		return
	}

	m := martini.Classic()

	m.Get("/qr", binding.Bind(meddler.Payload{}), presentQRCode)
	m.Post("/qr", binding.Bind(meddler.Payload{}), presentQRCode)

	m.Run()
}

func presentQRCode(pl meddler.Payload, res http.ResponseWriter, req *http.Request) {
	if pl.PublicKey != envKeySet.PublicKey {
		http.Error(res, "invalid publickey", 400)
		return
	}

	rawTextForSigning := pl.RawTextForSigning()
	if !envKeySet.Match([]byte(rawTextForSigning), []byte(pl.Signature)) {
		http.Error(res, "invalid signature", 400)
		return
	}

	uri := pl.URI
	code, err := qr.Encode(uri, qr.Q)
	if err != nil {
		fmt.Fprintf(res, "%s %v\n", uri, err)
		return
	}

	pngImage := code.PNG()
	fmt.Fprintf(res, "%s", pngImage)
}

func errorPrint(fmt_ string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "\033[31m")
	fmt.Fprintf(os.Stderr, fmt_, args...)
	fmt.Fprintf(os.Stderr, "\033[00m")
}
