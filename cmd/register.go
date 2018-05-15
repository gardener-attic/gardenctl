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

	"github.com/badoux/checkmail"
	"github.com/spf13/cobra"
)

// registerCmd represents the register command
var registerCmd = &cobra.Command{
	Use:   "register (e-mail)",
	Short: "Register as cluster admin for the operator shift",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Command must be in the format: register (e-mail)")
			os.Exit(2)
		}

		err = checkmail.ValidateFormat(args[0])
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		fmt.Println("Format Validated")
		err := ExecCmd("kubectl set subject clusterrolebinding garden-administrators --user="+args[0], false, "KUBECONFIG="+getGardenKubeConfig())
		checkError(err)
	},
}

func init() {
}
