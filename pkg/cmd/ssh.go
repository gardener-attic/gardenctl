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

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	mcmv1alpha1 "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
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
			if !CheckShootIsTargeted(target) {
				return errors.New("no shoot targeted")
			}

			shoot, err := FetchShootFromTarget(target)
			checkError(err)

			if len(args) == 0 {
				return printNodeNames(shoot.Name)
			}

			path := downloadTerraformFiles("infra", reader)
			if path != "" {
				path = filepath.Join(path, "terraform.tfstate")
			}

			// warning: entering untrusted zone
			fmt.Printf(warningColor, "\nWarning:\nBe aware that you are entering an untrusted environment!\nDo not enter credentials or sensitive data within the ssh session that cluster owners should not have access to.\n")
			fmt.Println("")

			gardenName := target.Stack()[0].Name
			shootName := target.Stack()[2].Name
			var pathSSKeypair string
			if target.Stack()[1].Kind == TargetKindProject {
				projectName := target.Stack()[1].Name
				pathSSKeypair = filepath.Join(pathGardenHome, "cache", gardenName, "projects", projectName, shootName)
			} else {
				seedName := target.Stack()[1].Name
				pathSSKeypair = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", seedName, shootName)
			}

			sshKeypairSecret := getSSHKeypair(shoot)
			checkError(err)
			err = ioutil.WriteFile(filepath.Join(pathSSKeypair, "key"), sshKeypairSecret.Data["id_rsa"], 0600)
			checkError(err)
			fmt.Println("Downloaded id_rsa key")

			fmt.Println("Check Public IP")
			myPublicIP := getPublicIP()

			sshPublicKey := sshKeypairSecret.Data["id_rsa.pub"]
			infraType := shoot.Spec.Provider.Type
			switch infraType {
			case "aws":
				sshToAWSNode(args[0], path, user, pathSSKeypair, sshPublicKey, myPublicIP)
			case "gcp":
				sshToGCPNode(args[0], path, user, pathSSKeypair, sshPublicKey, myPublicIP)
			case "azure":
				sshToAZNode(args[0], path, user, pathSSKeypair, sshPublicKey, myPublicIP)
			case "alicloud":
				sshToAlicloudNode(args[0], path, user, pathSSKeypair, sshPublicKey, myPublicIP)
			case "openstack":
				sshToOpenstackNode(args[0], path, user, pathSSKeypair, sshPublicKey, myPublicIP)
			default:
				return fmt.Errorf("infrastructure type %q not found", infraType)
			}

			return nil
		},
	}

	return cmd
}

// getSSHKeypair downloads ssh keypair for a shoot cluster
func getSSHKeypair(shoot *gardencorev1beta1.Shoot) *v1.Secret {
	Client, err := clientToTarget("garden")
	checkError(err)
	secret, err := Client.CoreV1().Secrets(shoot.Namespace).Get(shoot.Name+".ssh-keypair", metav1.GetOptions{})
	checkError(err)
	return secret
}

// printNodeNames print all nodes in k8s cluster
func printNodeNames(shootName string) error {
	machineList, err := getMachineList(shootName)
	checkError(err)

	fmt.Println("Nodes:")
	for _, machine := range machineList.Items {
		fmt.Println(fmt.Sprintf("%s (%s)", machine.Status.Node, string(machine.Status.CurrentStatus.Phase)))
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

func getMachineList(shootName string) (*v1alpha1.MachineList, error) {
	if getRole() == "user" {
		var target Target
		ReadTarget(pathTarget, &target)
		clientset, err := target.K8SClientToKind("shoot")
		checkError(err)
		fmt.Printf("%s\n", "Nodes")
		list, _ := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
		for _, node := range list.Items {
			fmt.Printf("%s\n", node.Name)
		}
		os.Exit(0)
	}

	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfigOfClusterType("seed"))
	checkError(err)
	client, err := mcmv1alpha1.NewForConfig(config)
	checkError(err)

	shootNamespace := getSeedNamespaceNameForShoot(shootName)
	machines, err := client.MachineV1alpha1().Machines(shootNamespace).List(metav1.ListOptions{})
	checkError(err)

	return machines, nil
}
