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

package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"

	version_tool "github.com/mcuadros/go-version"
	"github.com/spf13/cobra"
)

// version information
var version string
var buildDate string

// NewVersionCmd returns a new version command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the gardenctl version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf(`gardenctl:
		version     : %s
		build date  : %s
		go version  : %s
		go compiler : %s
		platform    : %s/%s
`, version, buildDate, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
			if NewVersionAvailable() {
				fmt.Println("New version of Gardenctl is available at https://github.com/gardener/gardenctl/releases/latest")
				fmt.Println("Please get latest version from above URL and see https://github.com/gardener/gardenctl#installation for how to upgrade")
			}
		},
	}
}

//NewVersionAvailable returns whether new version is available
func NewVersionAvailable() bool {
	var latestVersion string
	gardenctlLatestURL := "https://api.github.com/repos/gardener/gardenctl/releases/latest"
	resp, err := http.Get(gardenctlLatestURL)
	if err != nil {
		print(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		print(err)
	}

	data := make(map[string]interface{})
	err1 := json.Unmarshal([]byte(body), &data)
	if err1 != nil {
		print(err1)
	}
	if data["tag_name"] != nil {
		latestVersion = data["tag_name"].(string)
	}
	if version_tool.CompareSimple(latestVersion, version) == 1 {
		return true
	}
	return false
}
