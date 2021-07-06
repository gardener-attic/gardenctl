// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package cmd

import (
	"encoding/json"
	"errors"
	"os"
)

var tmp string

// WriteStringln writes history to given path
func (w *GardenctlHistoryWriter) WriteStringln(historyPath string, i interface{}) error {
	f, err := os.OpenFile(historyPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}

	switch x := i.(type) {
	case map[string]string:
		j, err := json.Marshal(x)
		if err != nil {
			return err
		}
		tmp = string(j)

	case string:
		tmp = x
	default:
		return errors.New("Invalid type not supported")
	}

	_, err = f.WriteString(tmp + "\n")
	return err
}
