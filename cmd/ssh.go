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
	"io/ioutil"
	"os"
	"strings"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "ssh to a node\n",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		path := downloadTerraformFiles("infra")
		if path != "" {
			path += "/terraform.tfstate"
		}
		pathToKey := downloadSSHKeypair()
		fmt.Printf(pathToKey)
		var target Target
		ReadTarget(pathTarget, &target)
		Client, err = clientToTarget("garden")
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
		checkError(err)
		shootList, err := gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		var ind int
		for index, shoot := range shootList.Items {
			if shoot.Name == target.Target[2].Name && (shoot.Namespace == target.Target[1].Name || *shoot.Spec.Cloud.Seed == target.Target[1].Name) {
				ind = index
				break
			}
		}
		infraType := shootList.Items[ind].Spec.Cloud.Profile
		switch infraType {
		case "aws":
			sshToAWSNode(path)
		case "gcp":
		case "az":
		case "openstack":
		default:
			fmt.Printf("Infrastructure type %s not found\n", infraType)
		}
	},
}

func init() {
	RootCmd.AddCommand(sshCmd)
}

func sshToAWSNode(path string) {
	// create bastion host
	clustername := getShootClusterName()
	name := clustername + "-bastions"
	keyName := clustername + "-ssh-publickey"
	subnetID, err := ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r .modules[].outputs.subnet_public_utility_z0.value")
	checkError(err)
	securityGroupID, err := ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.modules[].resources[\"aws_security_group_rule.bastion_ssh_bastion\"].primary[\"attributes\"].security_group_id'")
	checkError(err)
	arguments := "aws " + fmt.Sprintf("ec2 run-instances -- --iam-instance-profile Name=%s --image-id ami-d0dcef3b --count 1 --instance-type t2.nano --key-name %s --security-group-ids %s --subnet-id %s --associate-public-ip-address", name, keyName, securityGroupID, subnetID)
	// pipe output to file
	captured := capture()
	operate("aws", arguments)
	capturedOutput, err := captured()
	checkError(err)
	words := strings.Fields(capturedOutput)
	instanceID := ""
	for _, value := range words {
		if strings.HasPrefix(value, "i-") {
			instanceID = value
		}
	}
	arguments = "aws " + fmt.Sprintf("ec2 describe-instances -- --instance-ids %s", instanceID)
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	words = strings.Fields(capturedOutput)
	ip := ""
	for _, value := range words {
		if is_ip(value) && !strings.HasPrefix(value, "10.") {
			ip = value
			break
		}
	}
	fmt.Printf("Run: ssh -i key core@%s\n", ip)
	fmt.Printf("     gardenctl aws ec2 terminate-instances -- --instance-ids %s\n", instanceID)
}

func downloadSSHKeypair() string {
	var target Target
	ReadTarget(pathTarget, &target)
	shootName := target.Target[2].Name
	shootNamespace := getSeedNamespaceNameForShoot(shootName)
	Client, err = clientToTarget("seed")
	checkError(err)
	secret, err := Client.CoreV1().Secrets(shootNamespace).Get("ssh-keypair", metav1.GetOptions{})
	checkError(err)
	pathSSKeypair, err := os.Getwd()
	checkError(err)
	err = ioutil.WriteFile(pathSSKeypair+"/key", []byte(secret.Data["id_rsa"]), 0600)
	checkError(err)
	return "Downloaded id_rsa key\n"
}
