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
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
func sshToAWSNode(nodeName []string, path, user, pathSSKeypair string, sshPublicKey []byte, myPublicIP string) {
	a := &AwsInstanceAttribute{}
	a.SSHPublicKey = sshPublicKey
	a.MyPublicIP = myPublicIP

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
	node := ""
	if nodeName[0] == "providerid" && nodeName[1] != "" {
		node = user + "@localhost"
	} else {
		node = user + "@" + nodeName[0]
	}

	fmt.Print("SSH " + bastionNode + " => " + node + "\n")
	key := filepath.Join(pathSSKeypair, "key")

	proxyCommandArgs := []string{"-W%h:%p", "-i" + key, "-oIdentitiesOnly=yes", "-oConnectionAttempts=2", "-oStrictHostKeyChecking=no", bastionNode}
	if debugSwitch {
		proxyCommandArgs = append([]string{"-vvv"}, proxyCommandArgs...)
	}
	args := []string{"-i" + key, "-oConnectionAttempts=2", "-oProxyCommand=ssh " + strings.Join(proxyCommandArgs[:], " "), node, "-oIdentitiesOnly=yes", "-oStrictHostKeyChecking=no"}
	if debugSwitch {
		args = append([]string{"-vvv"}, args...)
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
	}
}

// fetchAwsAttributes gets all the needed attributes for creating bastion host and its security group with given <nodeName>.
func (a *AwsInstanceAttribute) fetchAwsAttributes(nodeName []string, path string) {
	a.ShootName = getFromTargetInfo("shootTechnicalID")
	publicUtility := a.ShootName + "-public-utility-z0"
	arguments := fmt.Sprintf("ec2 describe-subnets --filters Name=tag:Name,Values=" + publicUtility + " --query Subnets[*].SubnetId")
	a.SubnetID = strings.Trim(operate("aws", arguments), "\n")

	if nodeName[0] == "providerid" && nodeName[1] != "" {
		arguments = fmt.Sprintf("ec2 describe-instances --filters Name=instance-id,Values=" + nodeName[1] + " --query Reservations[*].Instances[*].{VpcId:VpcId}")
	} else {
		arguments = fmt.Sprintf("ec2 describe-instances --filters Name=network-interface.private-dns-name,Values=" + nodeName[0] + " --query Reservations[*].Instances[*].{VpcId:VpcId}")
	}
	a.VpcID = strings.Trim(operate("aws", arguments), "\n")

	a.SecurityGroupName = a.ShootName + "-nodes"
	a.getSecurityGroupID()
	a.BastionInstanceName = a.ShootName + "-bastions"
	a.BastionSecurityGroupName = a.ShootName + "-bsg"

	if nodeName[0] == "providerid" && nodeName[1] != "" {
		arguments = fmt.Sprintf("ec2 describe-instances --filters Name=instance-id,Values=" + nodeName[1] + " --query Reservations[*].Instances[*].{ImageId:ImageId}")
	} else {
		arguments = fmt.Sprintf("ec2 describe-instances --filters Name=network-interface.private-dns-name,Values=" + nodeName[0] + " --query Reservations[*].Instances[*].{ImageId:ImageId}")
	}
	a.ImageID = strings.Trim(operate("aws", arguments), "\n")

	a.KeyName = a.ShootName + "-ssh-publickey"
	a.UserData = getBastionUserData(a.SSHPublicKey)
}

// createBastionHostSecurityGroup finds the or creates a security group for the bastion host.
func (a *AwsInstanceAttribute) createBastionHostSecurityGroup() {
	// check if security group exists
	a.getBastionSecurityGroupID()
	if a.BastionSecurityGroupID != "" {
		fmt.Println("Security Group exists " + a.BastionSecurityGroupID + " skipping creation.")
		return
	}

	// create security group for bastion host
	arguments := fmt.Sprintf("ec2 create-security-group --group-name %s --description ssh-access --vpc-id %s", a.BastionSecurityGroupName, a.VpcID)
	a.BastionSecurityGroupID = operate("aws", arguments)

	arguments = fmt.Sprintf("ec2 create-tags --resources %s  --tags Key=component,Value=gardenctl", a.BastionSecurityGroupID)
	operate("aws", arguments)

	if net.ParseIP(a.MyPublicIP).To4() != nil {
		arguments = fmt.Sprintf("ec2 authorize-security-group-ingress --group-id %s --protocol tcp --port 22 --cidr %s/32", a.BastionSecurityGroupID, a.MyPublicIP)
	} else if net.ParseIP(a.MyPublicIP).To16() != nil {
		arguments = fmt.Sprintf("ec2 authorize-security-group-ingress --group-id %s --ip-permissions IpProtocol=tcp,FromPort=22,ToPort=22,Ipv6Ranges=[{CidrIpv6=%s/64}]", a.BastionSecurityGroupID, a.MyPublicIP)
	}
	operate("aws", arguments)
	fmt.Println("Bastion host security group set up.")
}

func (a *AwsInstanceAttribute) createNodeHostSecurityGroup() {
	// add shh rule to ec2 instance
	arguments := fmt.Sprintf("ec2 authorize-security-group-ingress --group-id %s --protocol tcp --port 22 --cidr %s/32", a.SecurityGroupID, a.BastionPrivIP)
	operate("aws", arguments)
	fmt.Println("Opened SSH Port on Node.")
}

// getSecurityGroupID extracts security group id of ec2 instance
func (a *AwsInstanceAttribute) getSecurityGroupID() {
	arguments := fmt.Sprintf("ec2 describe-security-groups --filters Name=vpc-id,Values=%s Name=group-name,Values=%s --query SecurityGroups[*].{ID:GroupId}", a.VpcID, a.SecurityGroupName)
	a.SecurityGroupID = operate("aws", arguments)
}

// getBastionSecurityGroupID extracts security group id for bastion security group
func (a *AwsInstanceAttribute) getBastionSecurityGroupID() {
	arguments := fmt.Sprintf("ec2 describe-security-groups --filters Name=vpc-id,Values=%s Name=group-name,Values=%s --query SecurityGroups[*].{ID:GroupId}", a.VpcID, a.BastionSecurityGroupName)
	a.BastionSecurityGroupID = operate("aws", arguments)
}

// getBastionHostInstance gets bastion host instance if it exists
func (a *AwsInstanceAttribute) getBastionHostInstance() {
	arguments := fmt.Sprintf("ec2 describe-instances --filter Name=vpc-id,Values=%s Name=tag:Name,Values=%s Name=instance-state-name,Values=running --query Reservations[*].Instances[].{Instance:InstanceId} --output text", a.VpcID, a.BastionInstanceName)
	a.BastionInstanceID = operate("aws", arguments)
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
	arguments := fmt.Sprintf("ec2 describe-instance-type-offerings --query %s", "InstanceTypeOfferings[].InstanceType")
	words := strings.Fields(operate("aws", arguments))
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
	arguments = fmt.Sprintf("ec2 run-instances --image-id %s --count 1 --instance-type %s --key-name %s --security-group-ids %s --subnet-id %s --associate-public-ip-address --user-data file://%s --tag-specifications ResourceType=instance,Tags=[{Key=Name,Value=%s},{Key=component,Value=gardenctl}] ResourceType=volume,Tags=[{Key=component,Value=gardenctl}]", a.ImageID, instanceType, a.KeyName, a.BastionSecurityGroupID, a.SubnetID, tmpfile.Name(), a.BastionInstanceName)
	words = strings.Fields(operate("aws", arguments))
	for _, value := range words {
		if strings.HasPrefix(value, "i-") {
			a.BastionInstanceID = value
		}
	}
	fmt.Println("Bastion host instance " + a.BastionInstanceID + " Initializing.")
	fmt.Println("")

	// waiting instance running
	arguments = "ec2 wait instance-running --instance-ids " + a.BastionInstanceID
	operate("aws", arguments)
	fmt.Println("Bastion host instance running.")

	// fetch BastionInstanceID
	arguments = "ec2 describe-instances --instance-id " + a.BastionInstanceID + " --query Reservations[*].Instances[*].PublicIpAddress"
	a.BastionIP = strings.Trim(operate("aws", arguments), "\n")

	// get bastion private IP
	arguments = "ec2 describe-instances --instance-id " + a.BastionInstanceID + " --query Reservations[*].Instances[*].PrivateIpAddress"
	a.BastionPrivIP = strings.Trim(operate("aws", arguments), "\n")
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
	arguments := fmt.Sprintf("ec2 terminate-instances --instance-ids %s", a.BastionInstanceID)
	fmt.Println(operate("aws", arguments))

	// remove shh rule from ec2 instance
	fmt.Println("  (2/3) Close SSH Port on Node.")
	arguments = fmt.Sprintf("ec2 revoke-security-group-ingress --group-id %s --protocol tcp --port 22 --cidr %s/32", a.SecurityGroupID, a.BastionPrivIP)
	fmt.Println("  Closed SSH Port on Node.")
	fmt.Println(operate("aws", arguments))

	// clean up bastion security group
	fmt.Println("  (3/3) Clean up bastion host security group")
	fmt.Println("")
	arguments = "ec2 wait instance-terminated --instance-ids " + a.BastionInstanceID
	operate("aws", arguments)
	arguments = fmt.Sprintf("ec2 delete-security-group --group-id %s", a.BastionSecurityGroupID)
	operate("aws", arguments)
	fmt.Println("")
	fmt.Println("Bastion host configurations successfully cleaned up.")
}
