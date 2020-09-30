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
	"strings"

	"github.com/spf13/cobra"
)

// NewDropCmd returns a new drop command.
func NewDropCmd(targetReader TargetReader, targetWriter TargetWriter, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "drop [(project|seed)]",
		Short:        "Drop scope for next operations (default: last target)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("command must be in the format: gardenctl drop [(project|seed)]")
			}
			if len(args) == 0 {
				target := targetReader.ReadTarget(pathTarget)
				stackLength := len(target.Stack())
				if stackLength == 0 {
					return errors.New("target stack is empty")
				}

				fmt.Fprintf(ioStreams.Out, "Dropped %s %s\n", target.Stack()[stackLength-1].Kind, target.Stack()[stackLength-1].Name)

				if target.Stack()[stackLength-1].Kind == "namespace" {
					setNamespaceDefault()
				}

				target.SetStack(target.Stack()[:stackLength-1])
				if err := targetWriter.WriteTarget(pathTarget, target); err != nil {
					return err
				}
			} else if len(args) == 1 {
				var target Target
				ReadTarget(pathTarget, &target)
				switch args[0] {
				case "project":
					if len(target.Target) == 2 && target.Target[1].Kind == "project" {
						drop(targetWriter)
						fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
					} else if len(target.Target) == 3 && target.Target[1].Kind == "project" {
						drop(targetWriter)
						drop(targetWriter)
						fmt.Printf("Dropped %s %s\n", target.Target[2].Kind, target.Target[2].Name)
						fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
					} else if len(target.Target) == 4 && target.Target[1].Kind == "project" {
						drop(targetWriter)
						drop(targetWriter)
						drop(targetWriter)
						fmt.Printf("Dropped %s %s\n", target.Target[3].Kind, target.Target[3].Name)
						fmt.Printf("Dropped %s %s\n", target.Target[2].Kind, target.Target[2].Name)
						fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
					} else {
						fmt.Println("A seed is targeted")
					}
				case "seed":
					if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
						drop(targetWriter)
						fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
					} else if len(target.Target) == 3 && target.Target[1].Kind == "seed" {
						drop(targetWriter)
						drop(targetWriter)
						fmt.Printf("Dropped %s %s\n", target.Target[2].Kind, target.Target[2].Name)
						fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
					} else if len(target.Target) == 4 && target.Target[1].Kind == "seed" {
						drop(targetWriter)
						drop(targetWriter)
						drop(targetWriter)
						fmt.Printf("Dropped %s %s\n", target.Target[3].Kind, target.Target[3].Name)
						fmt.Printf("Dropped %s %s\n", target.Target[2].Kind, target.Target[2].Name)
						fmt.Printf("Dropped %s %s\n", target.Target[1].Kind, target.Target[1].Name)
					} else {
						fmt.Println("A project is targeted")
					}
				case "namespace":
					if len(target.Target) > 1 && len(target.Target) < 5 {
						if target.Target[len(target.Target)-1].Kind == "namespace" {
							setNamespaceDefault()
							drop(targetWriter)
							fmt.Printf("Dropped %s %s\n", target.Target[len(target.Target)-1].Kind, target.Target[len(target.Target)-1].Name)
						} else {
							fmt.Println("No namespace targeted")
						}
					} else if len(target.Target) == 1 {
						fmt.Println("Only Garden is targeted, no namespace info")
					} else {
						fmt.Println("Size of target stack is illegal")
					}
				default:
					fmt.Println("Command must be in the format: gardenctl drop <project|seed|namespace>")
				}
			}

			return nil
		},
		ValidArgs: []string{"project", "seed", "namespace"},
	}

	return cmd
}

// drop drops target until stack is empty
func drop(targetWriter TargetWriter) {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) > 0 {
		target.Target = target.Target[:len(target.Target)-1]
	} else {
		fmt.Println("Target stack is empty")
	}

	err := targetWriter.WriteTarget(pathTarget, &target)
	checkError(err)

	KUBECONFIG = getKubeConfigOfCurrentTarget()
}

//set current namespace to default
func setNamespaceDefault() {
	cfg := getKubeConfigOfCurrentTarget()
	out, err := ExecCmdReturnOutput("kubectl", "--kubeconfig="+cfg, "config", "current-context")
	if err != nil {
		fmt.Println(err)
	}
	currentConext := strings.TrimSuffix(string(out), "\n")
	_, err = ExecCmdReturnOutput("kubectl", "--kubeconfig="+cfg, "config", "set-context "+currentConext, " --namespace=default")
	if err != nil {
		fmt.Println(err)
	}
}
