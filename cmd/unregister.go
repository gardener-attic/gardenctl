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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/badoux/checkmail"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	unregisterExample = `
	# Unregister as cluster admin from Garden cluster named 'prod'.
	gardenctl target garden prod
	gardenctl unregister john.doe@example.com
	
	# Unregister can also fetch the e-mail from the githubURL property (if set) in the Garden config.
	gardenctl unregister`
)

// Unregisters for all clusters as operator if it is set
var unregisterAll bool

// NewUnregisterCmd returns a new unregister command.
func NewUnregisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "unregister",
		Short:        "Unregister as cluster admin at the end of the operator shift",
		Example:      unregisterExample,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("command must be in the format: unregister (e-mail)")
			}
			email := ""
			if len(args) == 1 {
				email = args[0]
			}
			if len(args) < 1 {
				githubURL := getGithubURL()
				if email == "" {
					if githubURL == "" {
						return errors.New("no email specified and no GitHub url configured in garden config")
					}
					email = getEmail(githubURL)
					if email == "null" {
						return errors.New("could not read GitHub email address")
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
				clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(AdminClusterRoleBindingName, metav1.GetOptions{})
				if err != nil && strings.Contains(err.Error(), AdminClusterRoleBindingName) {
					kubeSecret, err := clientset.CoreV1().Secrets("garden").Get("virtual-garden-kubeconfig-for-admin", metav1.GetOptions{})
					checkError(err)
					virtualPath := pathDefault + "/virtual"
					os.MkdirAll(virtualPath, os.ModePerm)
					virtualPathKubeConfig := virtualPath + "/virtualKubeConfig.yaml"
					err = ioutil.WriteFile(virtualPathKubeConfig, kubeSecret.Data["kubeconfig"], 0644)
					checkError(err)
					config, err := clientcmd.BuildConfigFromFlags("", virtualPathKubeConfig)
					checkError(err)
					clientset, err = k8s.NewForConfig(config)
					checkError(err)
					clusterRoleBinding, err = clientset.RbacV1().ClusterRoleBindings().Get(AdminClusterRoleBindingName, metav1.GetOptions{})
					checkError(err)
				} else {
					checkError(err)
				}
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
					gardenKubeConfig := cluster.KubeConfig
					if strings.Contains(gardenKubeConfig, "~") {
						gardenKubeConfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(gardenKubeConfig, "~", "", 1)))
					}
					config, err := clientcmd.BuildConfigFromFlags("", gardenKubeConfig)
					checkError(err)
					clientset, err := k8s.NewForConfig(config)
					checkError(err)
					clusterRoleBinding, err := clientset.RbacV1().ClusterRoleBindings().Get(AdminClusterRoleBindingName, metav1.GetOptions{})
					if err != nil && strings.Contains(err.Error(), AdminClusterRoleBindingName) {
						kubeSecret, err := clientset.CoreV1().Secrets("garden").Get("virtual-garden-kubeconfig-for-admin", metav1.GetOptions{})
						checkError(err)
						virtualPath := pathDefault + "/virtual"
						os.MkdirAll(virtualPath, os.ModePerm)
						virtualPathKubeConfig := virtualPath + "/virtualKubeConfig.yaml"
						err = ioutil.WriteFile(virtualPathKubeConfig, kubeSecret.Data["kubeconfig"], 0644)
						checkError(err)
						config, err = clientcmd.BuildConfigFromFlags("", virtualPathKubeConfig)
						checkError(err)
						clientset, err = k8s.NewForConfig(config)
						checkError(err)
						clusterRoleBinding, err = clientset.RbacV1().ClusterRoleBindings().Get(AdminClusterRoleBindingName, metav1.GetOptions{})
						checkError(err)
					} else {
						checkError(err)
					}
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

			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&unregisterAll, "all", "a", false, "unregisters as operator for all clusters")

	return cmd
}
