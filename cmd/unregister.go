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

	"github.com/spf13/cobra"
)

// unregisterCmd represents the unregister command
var unregisterCmd = &cobra.Command{
	Use:   "unregister",
	Short: "Unregister as cluster admin at the end of the operator shift\n",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Command must be in the format: unregister (e-mail)")
			os.Exit(2)
		}
		cmdToExec := "kubectl get clusterrolebinding garden-administrators -o json | jq \".subjects | map(.name == \\\"" + args[0] + "\\\" ) | index(true)\""
		index := execCmdReturnOutput("bash", "-c", cmdToExec, "KUBECONFIG="+getGardenKubeConfig())
		cmdToExec = "kubectl patch clusterrolebinding garden-administrators --type=json -p=\"[{\\\"op\\\":\\\"remove\\\",\\\"path\\\":\\\"/subjects/" + index + "\\\"}]\""
		_ = execCmdReturnOutput("bash", "-c", cmdToExec, "KUBECONFIG="+getGardenKubeConfig())
	},
}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// unregisterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// unregisterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
