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
		var target Target
		ReadTarget(pathTarget, &target)
		if len(target.Target) < 3 {
			fmt.Println("No shoot targeted")
			os.Exit(2)
		}
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
		var kind string
		switch infraType {
		case "aws":
			kind = "internal"
		case "gcp":
			kind = "external"
		case "az":
		case "openstack":
		default:
			fmt.Printf("Infrastructure type %s not found\n", infraType)
		}
		if len(args) == 0 {
			fmt.Printf("Node ips:\n")
			printNodeIPs(kind)
			os.Exit(2)
		} else if len(args) != 1 || !is_ip(args[0]) {
			fmt.Printf("Select a valid node ip\n\n")
			fmt.Printf("Node ips:\n")
			printNodeIPs(kind)
			os.Exit(2)
		}
		path := downloadTerraformFiles("infra")
		if path != "" {
			path += "/terraform.tfstate"
		}
		pathToKey := downloadSSHKeypair()
		fmt.Printf(pathToKey)
		switch infraType {
		case "aws":
			region := shootList.Items[ind].Spec.Cloud.Region
			cloudprofile, err := gardenClientset.GardenV1beta1().CloudProfiles().Get(infraType, metav1.GetOptions{})
			checkError(err)
			var imageID string
			for _, v := range cloudprofile.Spec.AWS.Constraints.MachineImages[0].Regions {
				if v.Name == region {
					imageID = v.AMI
				}
			}
			sshToAWSNode(imageID, args[0], path)
		case "gcp":
			sshToGCPNode(args[0], path)
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

func sshToGCPNode(nodeIP, path string) {
	securityGroupID, err := ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.modules[].resources[\"google_compute_firewall.rule-allow-external-access\"].primary[\"id\"]'")
	checkError(err)
	arguments := "gcloud " + fmt.Sprintf("compute firewall-rules update %s --allow tcp:22,tcp:80,tcp:443", securityGroupID)
	operate("gcp", arguments)
	node := "core@" + nodeIP
	fmt.Println("Run: ssh -i key " + node)
	fmt.Printf("     gardenctl gcloud compute firewall-rules update %s -- --allow tcp:80,tcp:443\n", securityGroupID)
}

func sshToAWSNode(imageID, nodeIP, path string) {
	// create bastion host
	clustername := getShootClusterName()
	name := clustername + "-bastions"
	keyName := clustername + "-ssh-publickey"
	subnetID, err := ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r .modules[].outputs.subnet_public_utility_z0.value")
	checkError(err)
	securityGroupID, err := ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.modules[].resources[\"aws_security_group_rule.bastion_ssh_bastion\"].primary[\"attributes\"].security_group_id'")
	checkError(err)
	arguments := "aws " + fmt.Sprintf("ec2 run-instances -- --iam-instance-profile Name=%s --image-id %s --count 1 --instance-type t2.nano --key-name %s --security-group-ids %s --subnet-id %s --associate-public-ip-address", name, imageID, keyName, securityGroupID, subnetID)
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
	bastionNode := "core@" + ip
	node := "core@" + nodeIP
	fmt.Println("Run: ssh -i key -o \"ProxyCommand ssh -W %h:%p -i key " + bastionNode + "\" " + node)
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

// printNodes print all nodes in k8s cluster
func printNodeIPs(kindIP string) {
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)
	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, node := range nodes.Items {
		if kindIP == "internal" {
			fmt.Println("- " + node.Status.Addresses[0].Address)
		} else if kindIP == "external" {
			fmt.Println("- " + node.Status.Addresses[1].Address)
		}
	}
}
