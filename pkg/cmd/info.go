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
	"sort"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewInfoCmd returns a new info command.
func NewInfoCmd(targetReader TargetReader, ioStreams IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:          "info",
		Short:        "Get landscape informations",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := targetReader.ReadTarget(pathTarget)
			targetStack := target.Stack()
			if len(targetStack) < 1 {
				return errors.New("no garden cluster targeted")
			}

			// Show landscape
			gardenClientset, err := target.GardenerClient()
			if err != nil {
				return err
			}

			shootList, err := gardenClientset.CoreV1alpha1().Shoots("").List(metav1.ListOptions{})
			if err != nil {
				return err
			}

			var unscheduled = 0
			shootsCountPerSeed := make(map[string]int)
			for _, shoot := range shootList.Items {
				if shoot.Spec.SeedName == nil {
					unscheduled++
					continue
				}
				shootsCountPerSeed[*shoot.Spec.SeedName]++
			}

			var sortedSeeds []string
			for seed := range shootsCountPerSeed {
				sortedSeeds = append(sortedSeeds, seed)
			}
			sort.Strings(sortedSeeds)

			fmt.Fprintf(ioStreams.Out, "Garden: %s\n", targetStack[0].Name)
			fmt.Fprintf(ioStreams.Out, "Shoots:\n")
			fmt.Fprintf(ioStreams.Out, "\ttotal: %d\n", len(shootList.Items))
			fmt.Fprintf(ioStreams.Out, "\tunscheduled: %d\n", unscheduled)
			for _, seed := range sortedSeeds {
				fmt.Fprintf(ioStreams.Out, "\t%s: %d\n", seed, shootsCountPerSeed[seed])
			}

			return nil
		},
	}
}
