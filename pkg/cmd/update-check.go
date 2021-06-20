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
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/Masterminds/semver"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

const (
	green = "\033[1;32m%s\033[0m"
)

// NewUpdateCheckCmd checks whether a newer version of gardenctl is available
func NewUpdateCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-check",
		Short: "Check whether new gardenctl version is available, e.g. \"gardenctl update-check\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			isAvailable, latestVersion, err := newVersionAvailable(version)
			if err != nil {
				return err
			}
			if isAvailable {
				latestVersion := *latestVersion
				fmt.Printf("New version %s of Gardenctl is available at https://github.com/gardener/gardenctl/releases/latest \n", latestVersion)
				if askForConfirmation(green, "Do you want to install this version of Gardenctl (Y/N): ") {
					out, err := exec.LookPath("gardenctl")
					if err != nil {
						log.Fatal(err)
					}
					download(out, "https://github.com/gardener/gardenctl/releases/download/"+latestVersion+"/gardenctl-darwin-amd64")
				}
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

func download(file string, url string) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		log.Fatal(err)
	}
}
