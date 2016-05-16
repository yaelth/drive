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

// Package v1 implements the first version of encryption for drive
// It uses AES-256 in CTR mode for encryption and uses authenticates
// with an HMAC using SHA-512.
//
// This package should always be able to decrypt files that were encrypted
// using this package. If there is a change that needs to be made that would
// prevent decryption of old files, it should be done in a new version.

package v1

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"hash"
	"io"
	"os"

	"github.com/odeke-em/go-utils/tmpfile"
	"golang.org/x/crypto/scrypt"
)

const (
	// The size of the HMAC sum.
	hmacSize = sha512.Size

	// The size of the HMAC key.
	hmacKeySize = 32 // 256 bits

	// The size of the random salt.
	saltSize = 32 // 256 bits

	// The size of the AES key.
	aesKeySize = 32 // 256 bits

	// The size of the AES block.
	blockSize = aes.BlockSize

	// The number of iterations to use in for key generation
	// See N value in https://godoc.org/golang.org/x/crypto/scrypt#Key
	// Must be a power of 2.
	scryptIterations = 262144 // 2^18
)

const _16KB = 16 * 1024

var (
	// The underlying hash function to use for HMAC.
	hashFunc = sha512.New

	// The amount of key material we need.
	keySize = hmacKeySize + aesKeySize
)

var DecryptErr = errors.New("message corrupt or incorrect password")

// randBytes returns random bytes in a byte slice of size.
func randBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}

// keys derives AES and HMAC keys from a password and salt.
func keys(pass, salt []byte, iterations int) (aesKey, hmacKey []byte, err error) {
	key, err := scrypt.Key(pass, salt, iterations, 8, 1, keySize)
	if err != nil {
		return nil, nil, err
	}
	aesKey = append(aesKey, key[:aesKeySize]...)
	hmacKey = append(hmacKey, key[aesKeySize:keySize]...)
	return aesKey, hmacKey, nil
}

type hashReadWriter struct {
	hash hash.Hash
	done bool
	sum  io.Reader
}

func (h *hashReadWriter) Write(p []byte) (int, error) {
	if h.done {
		return 0, errors.New("writing to hashReadWriter after read is not allowed")
	}
	return h.hash.Write(p)
}

func (h *hashReadWriter) Read(p []byte) (int, error) {
	if !h.done {
		h.done = true
		h.sum = bytes.NewBuffer(h.hash.Sum(nil))
	}
	return h.sum.Read(p)
}

// NewEncryptReader returns an io.Reader wrapping the provided io.Reader.
// It uses a user provided password and a random salt to derive keys.
// If the key is provided interactively, it should be verified since there
// is no recovery.
func NewEncryptReader(r io.Reader, pass []byte) (io.Reader, error) {
	salt, err := randBytes(saltSize)
	if err != nil {
		return nil, err
	}
	return newEncryptReader(r, pass, salt, scryptIterations)
}

// newEncryptReader returns a encryptReader wrapping an io.Reader.
// It uses a user provided password and the provided salt iterated the
// provided number of times to derive keys.
func newEncryptReader(r io.Reader, pass, salt []byte, iterations int) (io.Reader, error) {
	aesKey, hmacKey, err := keys(pass, salt, iterations)
	if err != nil {
		return nil, err
	}
	b, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	h := hmac.New(hashFunc, hmacKey)
	iv, err := randBytes(blockSize)
	if err != nil {
		return nil, err
	}
	hr := &hashReadWriter{hash: h}
	sr := &cipher.StreamReader{R: r, S: cipher.NewCTR(b, iv)}
	var header []byte
	header = append(header, salt...)
	header = append(header, iv...)
	return io.MultiReader(io.TeeReader(io.MultiReader(bytes.NewBuffer(header), sr), hr), hr), nil
}

// decryptReader wraps a io.Reader decrypting its content.
type decryptReader struct {
	tmpFile *tmpfile.TmpFile
	sReader *cipher.StreamReader
}

// NewDecryptReader creates an io.ReadCloser wrapping an io.Reader.
// It has to read the entire io.Reader to disk using a temp file so that it can
// hash the contents to verify that it is safe to decrypt.
// If the file is athenticated, the DecryptReader will be returned and
// the resulting bytes will be the plaintext.
func NewDecryptReader(r io.Reader, pass []byte) (io.ReadCloser, error) {
	return newDecryptReader(r, pass, scryptIterations)
}

func newDecryptReader(r io.Reader, pass []byte, iterations int) (d *decryptReader, err error) {
	salt := make([]byte, saltSize)
	iv := make([]byte, blockSize)
	mac := make([]byte, hmacSize)
	_, err = io.ReadFull(r, salt)
	if err != nil {
		return nil, err
	}
	_, err = io.ReadFull(r, iv)
	if err != nil {
		return nil, err
	}
	aesKey, hmacKey, err := keys(pass, salt, iterations)
	if err != nil {
		return nil, err
	}
	// Start Verifying the HMAC of the message.
	h := hmac.New(hashFunc, hmacKey)
	h.Write(salt)
	h.Write(iv)
	dst, err := tmpfile.New(&tmpfile.Context{
		Dir:    os.TempDir(),
		Suffix: "drive-encrypted-",
	})
	if err != nil {
		return nil, err
	}
	// If there is an error, try to delete the temp file.
	defer func() {
		if err != nil {
			dst.Done()
		}
	}()
	b, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	d = &decryptReader{
		tmpFile: dst,
		sReader: &cipher.StreamReader{R: dst, S: cipher.NewCTR(b, iv)},
	}
	w := io.MultiWriter(h, dst)
	buf := bufio.NewReaderSize(r, _16KB)
	for {
		b, err := buf.Peek(_16KB)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if err == io.EOF {
			left := buf.Buffered()
			copy(mac, b[left-hmacSize:left])
			_, err = io.CopyN(w, buf, int64(left-hmacSize))
			if err != nil {
				return nil, err
			}
			break
		}
		_, err = io.CopyN(w, buf, _16KB-hmacSize)
		if err != nil {
			return nil, err
		}
	}
	if !hmac.Equal(mac, h.Sum(nil)) {
		return nil, DecryptErr
	}
	dst.Seek(0, 0)
	return d, nil
}

// Read implements io.Reader.
func (d *decryptReader) Read(dst []byte) (int, error) {
	return d.sReader.Read(dst)
}

// Close implements io.Closer.
func (d *decryptReader) Close() error {
	return d.tmpFile.Done()
}
