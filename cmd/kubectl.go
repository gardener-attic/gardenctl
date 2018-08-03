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
	"strings"

	"github.com/spf13/cobra"
)

// kubectlCmd represents the kubectl command
var kubectlCmd = &cobra.Command{
	Use:     "kubectl <args>",
	Aliases: []string{"k"},
	Short:   "",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		arguments := "kubectl " + strings.Join(args[:], " ")
		kube(arguments)
	},
}

var kaCmd = &cobra.Command{
	Use:    "ka",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		arguments := "kubectl " + strings.Join(args[:], " ") + " --all-namespaces=true"
		kube(arguments)
	},
}

var ksCmd = &cobra.Command{
	Use:    "ks",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		arguments := "kubectl " + strings.Join(args[:], " ") + " --namespace=kube-system"
		kube(arguments)
	},
}

var kgCmd = &cobra.Command{
	Use:    "kg",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		arguments := "kubectl " + strings.Join(args[:], " ") + " --namespace=garden"
		kube(arguments)
	},
}

func init() {
}

// kube executes a kubectl command on targeted cluster
func kube(args string) {
	KUBECONFIG = getKubeConfigOfCurrentTarget()
	err := ExecCmd(nil, "/usr/local/bin/"+args, false, "KUBECONFIG="+KUBECONFIG)
	checkError(err)
}
