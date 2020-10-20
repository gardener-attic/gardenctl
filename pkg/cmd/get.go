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
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

// NewGetCmd returns a new get command.
func NewGetCmd(targetReader TargetReader, configReader ConfigReader,
	kubeconfigReader KubeconfigReader, kubeconfigWriter KubeconfigWriter, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "get [(garden|project|seed|shoot|target) <name>]",
		Short:        "Get single resource instance or target stack, e.g. CRD of a shoot (default: current target)\n",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) < 1 || len(args) > 2 {
				return errors.New("command must be in the format: get [(garden|project|seed|shoot|target) <name>]")
			}

			switch args[0] {

			case "project":
				if isTargeted("project") {
					printProjectKubeconfig(targetReader, ioStreams)
				} else {
					return errors.New("no project targeted")
				}
			case "garden":
				if isTargeted("garden") {
					printGardenKubeconfig(configReader, targetReader, kubeconfigReader, ioStreams)
				} else {
					return errors.New("no garden targeted")
				}

			case "seed":
				if isTargeted("seed") {
					printSeedKubeconfig(targetReader, ioStreams)
				} else {
					return errors.New("no seed targeted")
				}

			case "shoot":
				if isTargeted("shoot") {
					printShootKubeconfig(targetReader, kubeconfigWriter, ioStreams)
				} else {
					return errors.New("no shoot targeted")
				}

			case "target":
				getTarget(targetReader, ioStreams)

			default:
				fmt.Fprint(ioStreams.Out, "command must be in the format: get [project|garden|seed|shoot|target] + <NAME>")
			}

			return nil
		},
		ValidArgs: []string{"project", "garden", "seed", "shoot", "target"},
	}

	return cmd
}

func print(path string) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	text := string(content)
	fmt.Println(text)
}

// printProjectKubeconfig lists
func printProjectKubeconfig(targetReader TargetReader, ioStreams IOStreams) {
	project, err := getProjectObject()
	if err != nil {
		log.Fatal(err, "get Project Object failed")
	}
	var output []byte
	if outputFormat == "yaml" {
		if output, err = yaml.Marshal(project); err != nil {
			log.Fatal(err, "output yaml error")
		}
	} else if outputFormat == "json" {
		if output, err = json.MarshalIndent(project, "", "  "); err != nil {
			log.Fatal(err, "output json error")
		}
	}
	fmt.Fprint(ioStreams.Out, string(output))
}

// printGardenKubeconfig lists kubeconfig of garden cluster
func printGardenKubeconfig(configReader ConfigReader, targetReader TargetReader, kubeconfigReader KubeconfigReader, ioStreams IOStreams) {
	print(getKubeConfigOfClusterType("garden"))
}

// printSeedKubeconfig lists kubeconfig of seed cluster
func printSeedKubeconfig(targetReader TargetReader, ioStreams IOStreams) {
	if outputFormat == "yaml" {
		print(getKubeConfigOfClusterType("seed"))
	} else if outputFormat == "json" {
		yaml2json(ioStreams, "seed")
	}
}

// printShootKubeconfig lists kubeconfig of shoot
func printShootKubeconfig(targetReader TargetReader, kubeconfigWriter KubeconfigWriter, ioStreams IOStreams) {
	if outputFormat == "yaml" {
		print(getKubeConfigOfClusterType("shoot"))
	} else if outputFormat == "json" {
		yaml2json(ioStreams, "shoot")
	}
}

func yaml2json(ioStreams IOStreams, clusterType TargetKind) {
	buffer, err := ioutil.ReadFile(getKubeConfigOfClusterType(clusterType))
	checkError(err)
	y, err := yaml.YAMLToJSON(buffer)
	checkError(err)
	var output []byte
	if output, err = json.MarshalIndent(string(y), "", " "); err != nil {
		log.Fatal(err, "out json error")
	}
	fmt.Fprint(ioStreams.Out, string(output))
}

// getTarget prints the target stack.
func getTarget(targetReader TargetReader, ioStreams IOStreams) {
	target := targetReader.ReadTarget(pathTarget)
	var err error
	if len(target.Stack()) == 0 {
		log.Fatal("target stack is empty")
	}

	if outputFormat == "yaml" {
		var output []byte
		if output, err = yaml.Marshal(target); err != nil {
			log.Fatal(err, "out yaml error")
		}
		fmt.Fprint(ioStreams.Out, string(output))
	} else if outputFormat == "json" {
		var output []byte
		if output, err = json.MarshalIndent(target, "", "  "); err != nil {
			log.Fatal(err, "out json error")
		}
		fmt.Fprint(ioStreams.Out, string(output))
	}
}
