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

package dcrypto_test

import (
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"testing"

	"github.com/odeke-em/drive/src/dcrypto"
)

// randBytes returns random bytes in a byte slice of size.
func randBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}

// TestRoundTrip tests several size sets of data going through the encrypt/decrypt
// to make sure they come out the same.
func TestRoundTrip(t *testing.T) {
	sizes := []int{0, 24, 1337, 66560}
	spasswords := []string{
		"",
		"guest",
	}
	for _, x := range []int{13, 400} {
		rp, err := randBytes(x)
		if err != nil {
			t.Fatalf("randBytes(%d) => err", x)
		}
		spasswords = append(spasswords, string(rp))
	}
	for _, spass := range spasswords {
		password := []byte(spass)
		for _, size := range sizes {
			t.Logf("Testing file of size: %db", size)
			b, err := randBytes(size)
			if err != nil {
				t.Errorf("randBytes(%d) => %q; want nil", size, err)
				continue
			}
			encReader, err := dcrypto.NewEncrypter(bytes.NewBuffer(b), password)
			if err != nil {
				t.Errorf("NewEncrypter() => %q; want nil", err)
				continue
			}
			cipher, err := ioutil.ReadAll(encReader)
			if err != nil {
				t.Errorf("ioutil.ReadAll(*Encrypter) => %q; want nil", err)
				continue
			}
			decReader, err := dcrypto.NewDecrypter(bytes.NewBuffer(cipher), password)
			if err != nil {
				t.Errorf("NewDecrypter() => %q; want nil", err)
				continue
			}
			plain, err := ioutil.ReadAll(decReader)
			decReader.Close()
			if err != nil {
				t.Errorf("ioutil.ReadAll(*Decrypter) => %q; want nil", err)
				continue
			}
			if !bytes.Equal(b, plain) {
				t.Errorf("Encrypt/Decrypt of file size %d, resulted in different values", size)
			}
		}
	}
}
