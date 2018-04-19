// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package utils

import (
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
)

// EncodeBase64 takes a byte slice and returns the Base64-encoded string.
func EncodeBase64(in []byte) string {
	encodedLength := base64.StdEncoding.EncodedLen(len(in))
	buffer := make([]byte, encodedLength)
	out := buffer[0:encodedLength]
	base64.StdEncoding.Encode(out, in)
	return string(out)
}

// DecodeBase64 takes a Base64-encoded string and returns the decoded byte slice.
func DecodeBase64(in string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(in)
}

// EncodePrivateKey takes a RSA private key object, encodes it to the PEM format, and returns it as
// a byte slice.
func EncodePrivateKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// DecodePrivateKey takes a byte slice, decodes it from the PEM format, converts it to an rsa.PrivateKey
// object, and returns it. In case an error occurs, it returns the error.
func DecodePrivateKey(bytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("could not decode the PEM-encoded RSA private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

// EncodeCertificate takes a certificate as a byte slice, encodes it to the PEM format, and returns
// it as byte slice.
func EncodeCertificate(certificate []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate,
	})
}

// DecodeCertificate takes a byte slice, decodes it from the PEM format, converts it to an x509.Certificate
// object, and returns it. In case an error occurs, it returns the error.
func DecodeCertificate(bytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("could not decode the PEM-encoded certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return certificate, nil
}

// EncodeSHA1 takes a byte slice and returns the sha1-hashed string
func EncodeSHA1(in []byte) string {
	s := sha1.New()
	s.Write(in)
	return EncodeBase64(s.Sum(in))
}

// CreateSHA1Secret takes a username and a password and returns a sha1-schemed credentials pair as string.
func CreateSHA1Secret(username, password []byte) string {
	credentials := append([]byte(username), ":{SHA}"...)
	credentials = append(credentials, EncodeSHA1(password)...)
	return EncodeBase64(credentials)
}

// ComputeSHA256Sum computes the SHA-256 checksum for a given byte slice and returns it.
func ComputeSHA256Sum(in []byte) string {
	h := sha256.Sum256(in)
	return hex.EncodeToString(h[:])
}
