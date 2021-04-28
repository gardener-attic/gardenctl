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
	"encoding/json"
	"fmt"
	"os"

	"github.com/gardener/gardenctl/pkg/internal/history"
	"github.com/spf13/cobra"
)

const (
	targetInfoGarden    = "garden"
	targetInfoProject   = "project"
	targetInfoSeed      = "seed"
	targetInfoShoot     = "shoot"
	targetInfoNamespace = "namespace"
)

//NewHistoryCmd use for list/search targting history
func NewHistoryCmd(targetWriter TargetWriter, historyWriter HistoryWriter) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "history",
		Short:        "List/Search targeting history, e.g. \"gardenctl x\"",
		SilenceUsage: true,
		Aliases:      []string{"x"},
		RunE: func(cmd *cobra.Command, args []string) error {
			h := history.SetPath(pathHistory)

			if len(h.Load().Items) <= 0 {
				fmt.Println("No Target History results")
				os.Exit(0)
			}

			items := h.Load().Reverse().Select()

			m, err := toMap(items.PromptItem)
			if err != nil {
				return err
			}

			var target Target
			if val, ok := m[targetInfoGarden]; ok {
				appendTarget(&target, targetInfoGarden, val)
			}

			if val, ok := m[targetInfoProject]; ok {
				appendTarget(&target, targetInfoProject, val)
			}

			if val, ok := m[targetInfoSeed]; ok {
				appendTarget(&target, targetInfoSeed, val)
			}

			if val, ok := m[targetInfoShoot]; ok {
				appendTarget(&target, targetInfoShoot, val)
			}

			if val, ok := m[targetInfoNamespace]; ok {
				appendTarget(&target, targetInfoNamespace, val)

				err := namespaceWrapper(nil, targetWriter, val)
				if err != nil {
					return err
				}
			}

			err = targetWriter.WriteTarget(pathTarget, &target)
			if err != nil {
				return fmt.Errorf("error write target %s", err)
			}

			kubeconfigPathOutput(&target)

			err = historyWriter.WriteStringln(pathHistory, items.Item)
			if err != nil {
				return fmt.Errorf("error write history %s", err)
			}

			return nil
		},
	}

	return cmd
}

func toMap(item history.PromptItem) (map[string]string, error) {
	toMap := make(map[string]string)
	tmp, err := json.Marshal(item)
	if err != nil {
		return nil, fmt.Errorf("convert error")
	}

	err = json.Unmarshal(tmp, &toMap)
	if err != nil {
		return nil, fmt.Errorf("convert error")
	}
	return toMap, nil
}

func appendTarget(target *Target, targetKind TargetKind, name string) *Target {
	target.Target = append(target.Target, TargetMeta{targetKind, name})
	return target
}

func kubeconfigPathOutput(target *Target) {
	for _, k := range target.Target {
		if k.Kind != targetInfoProject && k.Kind != TargetKindNamespace {
			fmt.Println(k.Kind + ":")
			KUBECONFIG = getKubeConfigOfClusterType(k.Kind)
			fmt.Println("KUBECONFIG=" + KUBECONFIG)
		}
	}
}
