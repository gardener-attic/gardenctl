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
	"io/ioutil"
	"os"
	"sort"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewInfoCmd returns a new info command.
func NewInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Get landscape informations\n",
		Run: func(cmd *cobra.Command, args []string) {
			var t Target
			targetFile, err := ioutil.ReadFile(pathTarget)
			checkError(err)
			err = yaml.Unmarshal(targetFile, &t)
			checkError(err)
			if len(t.Target) < 1 {
				fmt.Println("No garden targeted")
				os.Exit(2)
			}
			// show landscape
			Client, err = clientToTarget("garden")
			gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
			checkError(err)
			shootList, err := gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
			checkError(err)
			fmt.Printf("Garden: %s\n", t.Target[0].Name)
			fmt.Printf("Shoots:\n")
			fmt.Printf("    total: %d \n", len(shootList.Items))

			shootsCountPerSeed := make(map[string]int)
			for _, shoot := range shootList.Items {
				shootsCountPerSeed[*shoot.Spec.Cloud.Seed]++
			}
			var sortedSeeds []string
			for seed := range shootsCountPerSeed {
				sortedSeeds = append(sortedSeeds, seed)
			}
			sort.Strings(sortedSeeds)

			for _, seed := range sortedSeeds {
				fmt.Printf("    %s: %d \n", seed, shootsCountPerSeed[seed])
			}
			// show number shoots
			// show node, cpus ...
		},
	}
}
