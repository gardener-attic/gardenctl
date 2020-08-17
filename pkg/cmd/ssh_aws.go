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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	mcmv1alpha1 "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// AwsInstanceAttribute stores all the critical information for creating an instance on AWS.
type AwsInstanceAttribute struct {
	ShootName                string
	InstanceID               string
	SecurityGroupName        string
	SecurityGroupID          string
	ImageID                  string
	VpcID                    string
	KeyName                  string
	SubnetID                 string
	BastionSecurityGroupID   string
	BastionInstanceName      string
	BastionIP                string
	BastionPrivIP            string
	BastionInstanceID        string
	BastionSecurityGroupName string
	UserData                 []byte
	SSHPublicKey             []byte
	MyPublicIP               string
}

// sshToAWSNode provides cmds to ssh to aws via a bastions host and clean it up afterwards
func sshToAWSNode(nodeName, path, user, pathSSKeypair string, sshPublicKey []byte, myPublicIP string) {
	a := &AwsInstanceAttribute{}
	a.SSHPublicKey = sshPublicKey
	a.MyPublicIP = myPublicIP + "/32"

	fmt.Println("")

	fmt.Println("(1/4) Fetching data from target shoot cluster")
	a.fetchAwsAttributes(nodeName, path)
	fmt.Println("Data fetched from target shoot cluster.")
	fmt.Println("")

	fmt.Println("(2/4) Setting up bastion host security group")
	a.createBastionHostSecurityGroup()
	fmt.Println("")

	defer a.cleanupAwsBastionHost()

	fmt.Println("(3/4) Creating bastion host and node host security group")
	a.createBastionHostInstance()

	a.createNodeHostSecurityGroup()

	a.sshPortCheck()

	bastionNode := user + "@" + a.BastionIP
	node := user + "@" + nodeName

	fmt.Print("SSH " + bastionNode + " => " + node)
	key := filepath.Join(pathSSKeypair, "key")

	sshCmd := fmt.Sprintf("ssh -i " + key + "  -o ConnectionAttempts=2 -o \"ProxyCommand ssh -W %%h:%%p -i " + key + " -o IdentitiesOnly=yes -o ConnectionAttempts=2 -o StrictHostKeyChecking=no " + bastionNode + "\" " + node + " -o IdentitiesOnly=yes -o StrictHostKeyChecking=no")
	cmd := exec.Command("bash", "-c", sshCmd)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
	}
}

// fetchAwsAttributes gets all the needed attributes for creating bastion host and its security group with given <nodeName>.
func (a *AwsInstanceAttribute) fetchAwsAttributes(nodeName, path string) {
	a.ShootName = getShootClusterName()

	yamlData, err := ioutil.ReadFile(path)
	checkError(err)
	var yamlOut map[string]interface{}
	err = yaml.Unmarshal([]byte(yamlData), &yamlOut)
	checkError(err)

	terraformVersion := yamlOut["terraform_version"].(string)
	c, err := semver.NewConstraint(">= 0.12.0")
	if err != nil {
		fmt.Println("Handle version not being parsable.")
		os.Exit(2)
	}
	v, err := semver.NewVersion(terraformVersion)
	if err != nil {
		fmt.Println("Handle version not being parsable.")
		os.Exit(2)
	}
	if c.Check(v) {
		a.SubnetID = yamlOut["outputs"].(map[interface{}]interface{})["subnet_public_utility_z0"].(map[interface{}]interface{})["value"].(string)
		a.VpcID = yamlOut["outputs"].(map[interface{}]interface{})["vpc_id"].(map[interface{}]interface{})["value"].(string)
	} else {
		a.SubnetID = yamlOut["modules"].(map[interface{}]interface{})["outputs"].(map[interface{}]interface{})["subnet_public_utility_z0"].(map[interface{}]interface{})["value"].(string)
		a.VpcID = yamlOut["modules"].(map[interface{}]interface{})["outputs"].(map[interface{}]interface{})["vpc_id"].(map[interface{}]interface{})["value"].(string)
	}
	a.SecurityGroupName = a.ShootName + "-nodes"
	a.getSecurityGroupID()
	a.BastionInstanceName = a.ShootName + "-bastions"
	a.BastionSecurityGroupName = a.ShootName + "-bsg"
	a.ImageID, err = fetchAWSImageIDByNodeName(a.ShootName, nodeName)
	checkError(err)
	a.KeyName = a.ShootName + "-ssh-publickey"
	a.UserData = getBastionUserData(a.SSHPublicKey)
}

// fetchAWSImageIDByNodeName returns the image ID (AMI) for instance with the given <nodeName>.
func fetchAWSImageIDByNodeName(shootName, nodeName string) (string, error) {
	machines, err := getMachineList(shootName)
	if err != nil {
		return "", err
	}

	machineClassName := ""
	for _, machine := range machines.Items {
		if machine.Status.Node == nodeName {
			machineClassName = machine.Spec.Class.Name
			break
		}
	}

	if machineClassName == "" {
		return "", fmt.Errorf("Cannot find MachineClass for node %q", nodeName)
	}

	machineClasses := getAWSMachineClasses()
	for _, machineClass := range machineClasses.Items {
		if machineClass.Name == machineClassName {
			return machineClass.Spec.AMI, nil
		}
	}

	return "", fmt.Errorf("Cannot find ImageID for node %q", nodeName)
}

// createBastionHostSecurityGroup finds the or creates a security group for the bastion host.
func (a *AwsInstanceAttribute) createBastionHostSecurityGroup() {
	var err error
	// check if security group exists
	a.getBastionSecurityGroupID()
	if a.BastionSecurityGroupID != "" {
		fmt.Println("Security Group exists " + a.BastionSecurityGroupID + " skipping creation.")
		return
	}

	// create security group and ssh rule
	arguments := fmt.Sprintf("aws ec2 create-security-group --group-name %s --description ssh-access --vpc-id %s", a.BastionSecurityGroupName, a.VpcID)
	captured := capture()
	operate("aws", arguments)
	capturedOutput, err := captured()
	checkError(err)
	a.BastionSecurityGroupID = strings.Trim((capturedOutput), "\n")
	arguments = fmt.Sprintf("aws ec2 create-tags --resources %s  --tags Key=component,Value=gardenctl", a.BastionSecurityGroupID)
	operate("aws", arguments)
	arguments = fmt.Sprintf("aws ec2 authorize-security-group-ingress --group-id %s --protocol tcp --port 22 --cidr %s", a.BastionSecurityGroupID, a.MyPublicIP)
	operate("aws", arguments)
	fmt.Println("Bastion host security group set up.")

}

func (a *AwsInstanceAttribute) createNodeHostSecurityGroup() {
	// add shh rule to ec2 instance
	arguments := fmt.Sprintf("aws ec2 authorize-security-group-ingress --group-id %s --protocol tcp --port 22 --cidr %s/32", a.SecurityGroupID, a.BastionPrivIP)
	captured := capture()
	operate("aws", arguments)
	_, err := captured()
	checkError(err)
	fmt.Println("Opened SSH Port on Node.")
}

// getSecurityGroupID extracts security group id of ec2 instance
func (a *AwsInstanceAttribute) getSecurityGroupID() {
	var err error
	arguments := fmt.Sprintf("aws ec2 describe-security-groups --filters Name=vpc-id,Values=%s Name=group-name,Values=%s --query SecurityGroups[*].{ID:GroupId}", a.VpcID, a.SecurityGroupName)
	captured := capture()
	operate("aws", arguments)
	a.SecurityGroupID, err = captured()
	checkError(err)
}

// getBastionSecurityGroupID extracts security group id for bastion security group
func (a *AwsInstanceAttribute) getBastionSecurityGroupID() {
	var err error
	arguments := fmt.Sprintf("aws ec2 describe-security-groups --filters Name=vpc-id,Values=%s Name=group-name,Values=%s --query SecurityGroups[*].{ID:GroupId}", a.VpcID, a.BastionSecurityGroupName)
	captured := capture()
	operate("aws", arguments)
	a.BastionSecurityGroupID, err = captured()
	checkError(err)
}

// getBastionHostInstance gets bastion host instance if it exists
func (a *AwsInstanceAttribute) getBastionHostInstance() {
	var err error
	arguments := fmt.Sprintf("aws ec2 describe-instances --filter Name=vpc-id,Values=%s Name=tag:Name,Values=%s Name=instance-state-name,Values=running --query Reservations[*].Instances[].{Instance:InstanceId} --output text", a.VpcID, a.BastionInstanceName)
	captured := capture()
	operate("aws", arguments)
	a.BastionInstanceID, err = captured()
	checkError(err)
}

// createBastionHostInstance find or creates a bastion host instance.
func (a *AwsInstanceAttribute) createBastionHostInstance() {

	// check if bastion host exists
	a.getBastionHostInstance()
	if a.BastionInstanceID != "" {
		fmt.Println("Bastion Host exists, skipping creation.")
		return
	}

	tmpfile, err := ioutil.TempFile(os.TempDir(), "gardener-user.sh")
	checkError(err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(a.UserData)
	checkError(err)

	instanceType := ""
	arguments := fmt.Sprintf("aws ec2 describe-instance-type-offerings --query %s", "InstanceTypeOfferings[].InstanceType")
	captured := capture()
	operate("aws", arguments)
	capturedOutput, err := captured()
	checkError(err)
	words := strings.Fields(capturedOutput)
	for _, value := range words {
		if value == "t2.nano" {
			instanceType = "t2.nano"
		}
	}
	if instanceType == "" {
		for _, value := range words {
			if strings.HasPrefix(value, "t") {
				instanceType = value
				break
			}
		}
	}

	// create bastion host
	arguments = "aws " + fmt.Sprintf("ec2 run-instances --image-id %s --count 1 --instance-type %s --key-name %s --security-group-ids %s --subnet-id %s --associate-public-ip-address --user-data file://%s --tag-specifications ResourceType=instance,Tags=[{Key=Name,Value=%s},{Key=component,Value=gardenctl}] ResourceType=volume,Tags=[{Key=component,Value=gardenctl}]", a.ImageID, instanceType, a.KeyName, a.BastionSecurityGroupID, a.SubnetID, tmpfile.Name(), a.BastionInstanceName)
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	words = strings.Fields(capturedOutput)
	for _, value := range words {
		if strings.HasPrefix(value, "i-") {
			a.BastionInstanceID = value
		}
	}
	fmt.Println("Bastion host instance " + a.BastionInstanceID + " Initializing.")
	fmt.Println("")

	// waiting instance running
	arguments = "aws ec2 wait instance-running --instance-ids " + a.BastionInstanceID
	operate("aws", arguments)
	fmt.Println("Bastion host instance running.")

	// fetch BastionInstanceID
	arguments = "aws ec2 describe-instances --instance-id " + a.BastionInstanceID + " --query Reservations[*].Instances[*].PublicIpAddress"
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	a.BastionIP = strings.Trim(capturedOutput, "\n")

	// get bastion private IP
	arguments = "aws ec2 describe-instances --instance-id " + a.BastionInstanceID + " --query Reservations[*].Instances[*].PrivateIpAddress"
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	a.BastionPrivIP = strings.Trim(capturedOutput, "\n")
}

// getAWSMachineClasses returns machine classes for the cluster nodes
func getAWSMachineClasses() *v1alpha1.AWSMachineClassList {
	tempTarget := Target{}
	ReadTarget(pathTarget, &tempTarget)
	shootName := tempTarget.Target[2].Name
	shootNamespace := getSeedNamespaceNameForShoot(shootName)

	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfigOfClusterType("seed"))
	checkError(err)
	mcmClient, err := mcmv1alpha1.NewForConfig(config)
	checkError(err)

	machineClasses, err := mcmClient.MachineV1alpha1().AWSMachineClasses(shootNamespace).List(metav1.ListOptions{})
	checkError(err)

	return machineClasses
}

// Bastion SSH port check
func (a *AwsInstanceAttribute) sshPortCheck() {
	// waiting 60 seconds for SSH port open
	fmt.Println("Waiting 60 seconds for Bastion SSH port open")
	attemptCnt := 0
	for attemptCnt < 6 {
		ncCmd := fmt.Sprintf("timeout 10 nc -vtnz %s 22", a.BastionIP)
		cmd := exec.Command("bash", "-c", ncCmd)
		output, _ := cmd.CombinedOutput()
		fmt.Println("=>", string(output))
		if strings.Contains(string(output), "succeeded") {
			fmt.Println("Opened SSH Port on Bastion")
			return
		}
		time.Sleep(time.Second * 10)
		attemptCnt++
	}
	fmt.Println("SSH Port Open on Bastion TimeOut")
	a.cleanupAwsBastionHost()
	os.Exit(0)
}

// cleanupAwsBastionHost cleans up the bastion host for the targeted cluster.
func (a *AwsInstanceAttribute) cleanupAwsBastionHost() {
	fmt.Println("(4/4) Cleanup")
	fmt.Println("Cleaning up bastion host configurations...")
	fmt.Println("")
	fmt.Println("Starting cleanup")
	fmt.Println("")

	// clean up bastion host instance
	fmt.Println("  (1/3) Cleaning up bastion host instance")
	arguments := fmt.Sprintf("aws ec2 terminate-instances --instance-ids %s", a.BastionInstanceID)
	captured := capture()
	operate("aws", arguments)
	capturedOutput, err := captured()
	checkError(err)
	fmt.Println(capturedOutput)

	// remove shh rule from ec2 instance
	fmt.Println("  (2/3) Close SSH Port on Node.")
	arguments = fmt.Sprintf("aws ec2 revoke-security-group-ingress --group-id %s --protocol tcp --port 22 --cidr %s/32", a.SecurityGroupID, a.BastionPrivIP)
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	fmt.Println("  Closed SSH Port on Node.")
	fmt.Println(capturedOutput)

	// clean up bastion security group
	fmt.Println("  (3/3) Clean up bastion host security group")
	fmt.Println("")
	arguments = "aws ec2 wait instance-terminated --instance-ids " + a.BastionInstanceID
	captured = capture()
	operate("aws", arguments)
	_, err = captured()
	checkError(err)
	arguments = fmt.Sprintf("aws ec2 delete-security-group --group-id %s", a.BastionSecurityGroupID)
	captured = capture()
	operate("aws", arguments)
	_, err = captured()
	checkError(err)
	fmt.Println("")
	fmt.Println("Bastion host configurations successfully cleaned up.")
}
