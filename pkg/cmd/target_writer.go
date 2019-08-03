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
	"os"

	"gopkg.in/yaml.v2"
)

// WriteTarget writes <target> to <targetPath>.
func (w *GardenctlTargetWriter) WriteTarget(targetPath string, target TargetInterface) (err error) {
	var content []byte
	if content, err = yaml.Marshal(target); err != nil {
		return err
	}

	var file *os.File
	if file, err = os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644); err != nil {
		return err
	}
	defer file.Close()

	file.Write(content)
	file.Sync()

	return
}
