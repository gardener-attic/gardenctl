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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// unregisterCmd represents the unregister command
var unregisterCmd = &cobra.Command{
	Use:   "unregister",
	Short: "Unregister as cluster admin at the end of the operator shift\n",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 1 {
			fmt.Println("Command must be in the format: unregister (e-mail)")
			os.Exit(2)
		}
		email := ""
		if len(args) == 1 {
			email = args[0]
		}
		if len(args) < 1 {
			githubURL := getGithubURL()
			if email == "" {
				if githubURL == "" {
					fmt.Println("No email specified and no github url configured in garden config")
					os.Exit(2)
				}
				email = getEmail(githubURL)
				if email == "null" {
					fmt.Println("Could not read github email address")
					os.Exit(2)
				}
			}
		}
		err = checkmail.ValidateFormat(email)
		checkError(err)
		fmt.Println("Format Validated")
		if !unregisterAll {
			config, err := clientcmd.BuildConfigFromFlags("", getKubeConfigOfClusterType("garden"))
			checkError(err)
			clientset, err := k8s.NewForConfig(config)
			checkError(err)
			clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get("garden-administrators", metav1.GetOptions{})
			checkError(err)
			for k, subject := range clusterRoleBinding.Subjects {
				if subject.Kind == "User" && subject.Name == email {
					clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects[:k], clusterRoleBinding.Subjects[k+1:]...)
					_, err = clientset.RbacV1().ClusterRoleBindings().Update(clusterRoleBinding)
					checkError(err)
					fmt.Printf("User %s unregistered \n", email)
					break
				}
			}
		} else {
			currentKubeConfig := getGardenKubeConfig()
			var gardenConfig GardenConfig
			GetGardenConfig(pathGardenConfig, &gardenConfig)
			for _, cluster := range gardenConfig.GardenClusters {
				config, err := clientcmd.BuildConfigFromFlags("", cluster.KubeConfig)
				checkError(err)
				clientset, err := k8s.NewForConfig(config)
				checkError(err)
				clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get("garden-administrators", metav1.GetOptions{})
				checkError(err)
				for k, subject := range clusterRoleBinding.Subjects {
					if subject.Kind == "User" && subject.Name == email {
						clusterRoleBinding.Subjects = append(clusterRoleBinding.Subjects[:k], clusterRoleBinding.Subjects[k+1:]...)
						_, err = clientset.RbacV1().ClusterRoleBindings().Update(clusterRoleBinding)
						checkError(err)
						fmt.Printf("User %s unregistered on %s \n", email, cluster.Name)
						break
					}
				}
			}
			kubeconfig = &currentKubeConfig
		}
	},
}

//unregisterAll flag unregisters for all clusters as operator if it is set
var unregisterAll bool

func init() {
	unregisterCmd.PersistentFlags().BoolVarP(&unregisterAll, "all", "a", false, "unregisters as operator for all clusters")
}
