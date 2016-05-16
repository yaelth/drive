// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package dcrypto provides end to end encryption for drive.

package dcrypto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/odeke-em/drive/src/dcrypto/v1"
)

// Version is the version of the en/decryption library used.
type Version uint32

// Decrypter is a function that creates a decrypter.
type Decrypter func(io.Reader, []byte) (io.ReadCloser, error)

// Encrypter is a function that creates a encrypter.
type Encrypter func(io.Reader, []byte) (io.Reader, error)

// These are the different versions of the en/decryption library.
const (
	V1 Version = iota
)

// PreferedVersion is the prefered version of encryption.
const PreferedVersion = V1

var encrypters map[Version]Encrypter
var decrypters map[Version]Decrypter

func init() {
	decrypters = map[Version]Decrypter{
		V1: v1.NewDecryptReader,
	}

	encrypters = map[Version]Encrypter{
		V1: v1.NewEncryptReader,
	}
}

// NewEncrypter returns an Encrypter using the PreferedVersion.
func NewEncrypter(r io.Reader, password []byte) (io.Reader, error) {
	v, err := writeVersion(PreferedVersion)
	if err != nil {
		return nil, err
	}
	encrypterFn, ok := encrypters[PreferedVersion]
	if !ok {
		return nil, fmt.Errorf("%v version could not be found", PreferedVersion)
	}
	encReader, err := encrypterFn(r, password)
	if err != nil {
		return nil, err
	}
	return io.MultiReader(bytes.NewBuffer(v), encReader), nil
}

// NewDecrypter returns a Decrypter based on the version used to encrypt.
func NewDecrypter(r io.Reader, password []byte) (io.ReadCloser, error) {
	version, err := readVersion(r)
	if err != nil {
		return nil, err
	}
	decrypterFn, ok := decrypters[version]
	if !ok {
		return nil, fmt.Errorf("unknown decrypter for version(%d)", version)
	}
	return decrypterFn(r, password)
}

// writeVersion converts a Version to a []byte.
func writeVersion(i Version) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, i); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// readVersion reads and returns a Version from reader.
func readVersion(r io.Reader) (v Version, err error) {
	err = binary.Read(r, binary.LittleEndian, &v)
	return v, err
}
