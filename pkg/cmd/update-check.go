// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"
)

// NewUpdateCheckCmd checks whether a newer version of gardenctl is available
func NewUpdateCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-check",
		Short: "Check whether new gardenctl version is available",
		RunE: func(cmd *cobra.Command, args []string) error {

			isAvailable, latestVersion, err := newVersionAvailable(version)
			if err != nil {
				return err
			}
			if isAvailable {
				fmt.Printf("New version %q of Gardenctl is available at https://github.com/gardener/gardenctl/releases/latest \n", *latestVersion)
				fmt.Println("Please get latest version from above URL and see https://github.com/gardener/gardenctl#installation for how to upgrade")
			} else {
				fmt.Println("You are using the latest available version")
			}

			return nil
		},
	}
}

// newVersionAvailable returns whether new version is available.
func newVersionAvailable(currentVersion string) (bool, *string, error) {
	gardenctlLatestURL := "https://api.github.com/repos/gardener/gardenctl/releases/latest"
	resp, err := http.Get(gardenctlLatestURL)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, nil, err
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return false, nil, err
	}
	var latestVersion string
	if data["tag_name"] != nil {
		latestVersion = data["tag_name"].(string)
	}

	c, err := semver.NewConstraint("> " + currentVersion)
	if err != nil {
		return false, nil, err
	}

	latest, err := semver.NewVersion(latestVersion)
	if err != nil {
		return false, nil, err
	}

	return c.Check(latest), &latestVersion, nil
}
