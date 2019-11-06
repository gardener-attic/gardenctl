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

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// warningColor prints user warnings in red
	warningColor = "\033[1;31m%s\033[0m"
	// user is operating system user for bastion host and instances
	user = "gardener"
)

// NewSSHCmd returns a new ssh command.
func NewSSHCmd(reader TargetReader, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ssh",
		Short:        "SSH to a node",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := reader.ReadTarget(pathTarget)
			if len(target.Stack()) < 3 {
				return errors.New("no shoot targeted")
			}

			gardenClientset, err := target.GardenerClient()
			checkError(err)
			var shoot *gardencorev1alpha1.Shoot
			if target.Stack()[1].Kind == "project" {
				project, err := gardenClientset.CoreV1alpha1().Projects().Get(target.Stack()[1].Name, metav1.GetOptions{})
				checkError(err)
				shoot, err = gardenClientset.CoreV1alpha1().Shoots(*project.Spec.Namespace).Get(target.Stack()[2].Name, metav1.GetOptions{})
				checkError(err)
			} else {
				shootList, err := gardenClientset.CoreV1alpha1().Shoots("").List(metav1.ListOptions{})
				checkError(err)
				for index, s := range shootList.Items {
					if s.Name == target.Stack()[2].Name && *s.Spec.SeedName == target.Stack()[1].Name {
						shoot = &shootList.Items[index]
						break
					}
				}
			}

			infraType := shoot.Spec.Provider.Type
			var kind string
			switch infraType {
			case "aws":
				kind = "internal"
			case "gcp":
				kind = "internal"
			case "azure":
				kind = "internal"
			case "alicloud":
				kind = "internal"
			case "openstack":
			default:
				return fmt.Errorf("infrastructure type %q not found", infraType)
			}

			if len(args) == 0 {
				fmt.Printf("Node ips:\n")
				printNodeIPs(kind)
				return nil
			} else if len(args) != 1 || !isIP(args[0]) {
				if args[0] != "cleanup" {
					fmt.Printf("Select a valid node ip or use 'gardenctl ssh cleanup' to clean up settings\n\n")
					fmt.Printf("Node ips:\n")
					printNodeIPs(kind)
					os.Exit(2)
				}
			}

			path := downloadTerraformFiles("infra")
			if path != "" {
				path = filepath.Join(path, "terraform.tfstate")
			}

			// warning: entering untrusted zone
			fmt.Printf(warningColor, "\nWarning:\nBe aware that you are entering an untrusted environment!\nDo not enter credentials or sensitive data within the ssh session that cluster owners should not have access to.\n")
			fmt.Println("")

			sshKeypairSecret := getSSHKeypair(shoot)
			pathSSKeypair, err := os.Getwd()
			checkError(err)
			err = ioutil.WriteFile(filepath.Join(pathSSKeypair, "key"), sshKeypairSecret.Data["id_rsa"], 0600)
			checkError(err)
			fmt.Println("Downloaded id_rsa key")

			sshPublicKey := sshKeypairSecret.Data["id_rsa.pub"]
			switch infraType {
			case "aws":
				sshToAWSNode(args[0], path, user, sshPublicKey)
			case "gcp":
				sshToGCPNode(args[0], path, user, sshPublicKey)
			case "azure":
				sshToAZNode(args[0], path, user, sshPublicKey)
			case "alicloud":
				sshToAlicloudNode(args[0], path, user, sshPublicKey)
			case "openstack":
			default:
				return fmt.Errorf("infrastructure type %q not found", infraType)
			}

			return nil
		},
	}

	return cmd
}

// getSSHKeypair downloads ssh keypair for a shoot cluster
func getSSHKeypair(shoot *gardencorev1alpha1.Shoot) *v1.Secret {
	Client, err := clientToTarget("garden")
	checkError(err)
	secret, err := Client.CoreV1().Secrets(shoot.Namespace).Get(shoot.Name+".ssh-keypair", metav1.GetOptions{})
	checkError(err)
	return secret
}

// printNodeIPs print all nodes in k8s cluster
func printNodeIPs(kindIP string) {
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)
	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, node := range nodes.Items {
		if kindIP == "internal" {
			for _, v := range node.Status.Addresses {
				if v.Type == "InternalIP" {
					fmt.Println("- " + v.Address)
				}
			}
		} else if kindIP == "external" {
			for _, v := range node.Status.Addresses {
				if v.Type == "ExternalIP" {
					fmt.Println("- " + v.Address)
				}
			}
		}
	}
}

// getNodeForIP extract node for ip address
func getNodeForIP(ip string) *v1.Node {
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)
	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, node := range nodes.Items {
		for _, v := range node.Status.Addresses {
			if ip == v.Address {
				return &node
			}
		}
	}
	return nil
}

func getBastionUserData(sshPublicKey []byte) []byte {
	template := `#!/bin/bash -eu

id gardener || useradd gardener -mU
mkdir -p /home/gardener/.ssh
echo %q > /home/gardener/.ssh/authorized_keys
chown gardener:gardener /home/gardener/.ssh/authorized_keys
echo "gardener ALL=(ALL) NOPASSWD:ALL" >/etc/sudoers.d/99-gardener-user
`
	userData := fmt.Sprintf(template, sshPublicKey)
	return []byte(userData)
}
