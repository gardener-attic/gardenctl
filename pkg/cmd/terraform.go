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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// NewTerraformCmd returns a new terraform command.
func NewTerraformCmd(targetReader TargetReader) *cobra.Command {
	return &cobra.Command{
		Use:          "terraform <args>",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := targetReader.ReadTarget(pathTarget)
			if !CheckShootIsTargeted(target) {
				return errors.New("no shoot targeted")
			}

			arguments := "terraform " + strings.Join(args[:], " ")
			terraform(arguments)

			return nil
		},
	}
}

// terraform executes a terraform command on targeted cluster
func terraform(args string) {
	_, err := exec.LookPath("terraform")
	if err != nil {
		fmt.Println("Terraform is not installed on your system")
		os.Exit(2)
	}
	var target Target
	ReadTarget(pathTarget, &target)
	gardenName := target.Stack()[0].Name
	pathTerraform := ""

	if target.Target[1].Kind == "project" {
		pathTerraform = filepath.Join(pathGardenHome, "cache", gardenName, "projects", target.Target[1].Name, target.Target[2].Name, "terraform")
	} else if target.Target[1].Kind == "seed" {
		pathTerraform = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, target.Target[2].Name, "terraform")
	}

	if (strings.HasSuffix(args, "init")) {
		pathTerraform = downloadTerraformFiles("infra")
		fmt.Println("Downloaded terraform config to " + pathTerraform)
	}

	_, err = os.Stat(pathTerraform)
	if os.IsNotExist(err) {
		fmt.Println("Please run terraform init first to fetch terraform config")
		os.Exit(2)
	}

	err = os.Chdir(pathTerraform)
	if err != nil {
		fmt.Println("Could not move into the directory " + pathTerraform)
		os.Exit(2)
	}

	err = ExecCmd(nil, args, false)
	checkError(err)
}