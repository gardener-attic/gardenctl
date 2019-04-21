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
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
)

// dropCmd represents the drop command
var dropCmd = &cobra.Command{
	Use:   "drop [(project|seed)]",
	Short: "Drop scope for next operations (default: last target)",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Command must be in the format: gardenctl drop [(project|seed)]")
			os.Exit(2)
		}
		if len(args) == 0 {
			var target Target
			ReadTarget(pathTarget, &target)
			if len(target.Target) == 1 {
				fmt.Printf("Dropped %s %s\n", target.Target[0].Kind, target.Target[0].Name)
			} else if len(target.Target) == 2 {
				fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
			} else if len(target.Target) == 3 {
				fmt.Printf("Dropped %s %s\n", target.Target[2].Kind, target.Target[2].Name)
			}
			drop()
		} else if len(args) == 1 {
			var target Target
			ReadTarget(pathTarget, &target)
			switch args[0] {
			case "project":
				if len(target.Target) == 2 && target.Target[1].Kind == "project" {
					drop()
					fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
				} else if len(target.Target) == 3 && target.Target[1].Kind == "project" {
					drop()
					drop()
					fmt.Printf("Dropped %s %s\n", target.Target[2].Kind, target.Target[2].Name)
					fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
				} else {
					fmt.Println("A seed is targeted")
				}
			case "seed":
				if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
					drop()
					fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
				} else if len(target.Target) == 3 && target.Target[1].Kind == "seed" {
					drop()
					drop()
					fmt.Printf("Dropped %s %s\n", target.Target[2].Kind, target.Target[2].Name)
					fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
				} else {
					fmt.Println("A project is targeted")
				}
			default:
				fmt.Println("Command must be in the format: gardenctl drop <project|seed>")
			}
		}
	},
	ValidArgs: []string{"project", "seed"},
}

// drop drops target until stack is empty
func drop() {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) > 0 {
		target.Target = target.Target[:len(target.Target)-1]
	} else {
		fmt.Println("Target stack is empty")
	}
	file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	checkError(err)
	content, err := yaml.Marshal(target)
	checkError(err)
	file.Write(content)
	file.Close()
	KUBECONFIG = getKubeConfigOfCurrentTarget()
}
