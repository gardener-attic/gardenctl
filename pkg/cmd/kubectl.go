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
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// NewKubectlCmd returns a new kubectl command.
func NewKubectlCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "kubectl <args>",
		Short:              "e.g. \"gardenctl kubectl get pods -n kube-system\"",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Aliases:            []string{"k"},
		Run: func(cmd *cobra.Command, args []string) {
			arguments := "kubectl " + strings.Join(os.Args[2:], " ")
			kube(arguments)
		},
	}
}

// NewKaCmd returns a new 'kubectl --all-namespaces' command.
func NewKaCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "ka",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Hidden:             true,
		Run: func(cmd *cobra.Command, args []string) {
			arguments := "kubectl " + strings.Join(os.Args[2:], " ") + " --all-namespaces=true"
			kube(arguments)
		},
	}
}

// NewKsCmd returns a new 'kubectl --namespace=kube-system' command.
func NewKsCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "ks",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Hidden:             true,
		Run: func(cmd *cobra.Command, args []string) {
			arguments := "kubectl " + strings.Join(os.Args[2:], " ") + " --namespace=kube-system"
			kube(arguments)
		},
	}
}

// NewKgCmd returns a new 'kubectl --namespace=garden' command.
func NewKgCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "kg",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Hidden:             true,
		Run: func(cmd *cobra.Command, args []string) {
			arguments := "kubectl " + strings.Join(os.Args[2:], " ") + " --namespace=garden"
			kube(arguments)
		},
	}
}

// NewKnCmd returns a new 'kubectl --namespace=<arg>' command.
func NewKnCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "kn",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Hidden:             true,
		Run: func(cmd *cobra.Command, args []string) {
			arguments := "kubectl --namespace=" + strings.Join(os.Args[2:], " ")
			kube(arguments)
		},
	}
}

// kube executes a kubectl command on targeted cluster
func kube(args string) {
	KUBECONFIG = getKubeConfigOfCurrentTarget()
	_, err := exec.LookPath("kubectl")
	if err != nil {
		fmt.Println("Kubectl is not installed on your system")
		os.Exit(2)
	}
	err = ExecCmd(nil, args, false, "KUBECONFIG="+KUBECONFIG)
	checkError(err)
}
