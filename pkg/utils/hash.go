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
	"crypto/sha1"
	"encoding/hex"
)

// ComputeSHA1Hex computes the hexadecimal representation of the SHA1 hash of the given input string
// <in>, converts it to a string and returns it (length of returned string is 40 characters).
func ComputeSHA1Hex(in string) string {
	hash := sha1.New()
	hash.Write([]byte(in))
	return hex.EncodeToString(hash.Sum(nil))
}
