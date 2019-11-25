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
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmoiron/jsonq"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AliyunInstanceAttribute stores all the critical information for creating a instance on Alicloud.
type AliyunInstanceAttribute struct {
	InstanceID               string
	RegionID                 string
	ZoneID                   string
	VpcID                    string
	ImageID                  string
	VSwitchID                string
	ShootName                string
	BastionSecurityGroupName string
	BastionInstanceName      string
	BastionInstanceID        string
	BastionSecurityGroupID   string
	InstanceType             string
	InstanceChargeType       string
	InternetChargeType       string
	InternetMaxBandwidthIn   string
	InternetMaxBandwidthOut  string
	IoOptimized              string
	KeyPairName              string
	PrivateIP                string
	BastionIP                string
	BastionSSHUser           string
}

// AliyunInstanceTypeSpec stores all the critical information for choosing a instance type on Alicloud.
type AliyunInstanceTypeSpec struct {
	CPUCoreCount                int
	EniQuantity                 int
	MemorySize                  float64
	GPUAmount                   int
	EniPrivateIPAddressQuantity int
	LocalStorageCategory        string
	GPUSpec                     string
	InstanceTypeID              string
	InstanceFamilyLevel         string
	InstanceTypeFamily          string
}

// sshToAlicloudNode provides cmds to ssh to alicloud via a public ip and clean it up afterwards.
func sshToAlicloudNode(nodeName, path, user string, sshPublicKey []byte) {
	// Check if this is a cleanup command
	if nodeName == "cleanup" {
		cleanupAliyunBastionHost()
		return
	}

	fmt.Println("(1/5) Configuring aliyun cli")
	configureAliyunCLI()
	var target Target
	ReadTarget(pathTarget, &target)
	gardenName := target.Stack()[0].Name
	aliyunPathSSHKey := ""
	if target.Target[1].Kind == "project" {
		aliyunPathSSHKey = filepath.Join(pathGardenHome, "cache", gardenName, "projects", target.Target[1].Name, target.Target[2].Name, ".aliyun") + string(filepath.Separator)
	} else if target.Target[1].Kind == "seed" {
		aliyunPathSSHKey = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, target.Target[2].Name, ".aliyun") + string(filepath.Separator)
	}
	err := ExecCmd(nil, "mv key "+aliyunPathSSHKey, false)
	checkError(err)
	fmt.Println("Aliyun cli configured.")

	a := &AliyunInstanceAttribute{}

	fmt.Println("")
	fmt.Println("(2/5) Fetching data from target shoot cluster")
	a.fetchAttributes(nodeName)
	fmt.Println("Data fetched from target shoot cluster.")

	fmt.Println("")
	fmt.Println("(3/5) Setting up bastion host security group")
	a.createBastionHostSecurityGroup()
	fmt.Println("Bastion host security group set up.")

	defer checkIsDeletionWanted(a.BastionInstanceID)

	fmt.Println("")
	fmt.Println("(4/5) Setting up bastion host")
	a.createBastionHostInstance(sshPublicKey)
	fmt.Println("Bastion host set up.")

	fmt.Println("")
	fmt.Println("(5/5) Starting bastion host")
	a.startBastionHostInstance()
	fmt.Println("Bastion host started.")

	sshCmd := "ssh -i " + aliyunPathSSHKey + "key -o \"ProxyCommand ssh -i " + aliyunPathSSHKey + "key -o StrictHostKeyChecking=no -W " + a.PrivateIP + ":22 " + a.BastionSSHUser + "@" + a.BastionIP + "\" " + user + "@" + a.PrivateIP + " -o StrictHostKeyChecking=no"
	cmd := exec.Command("bash", "-c", sshCmd)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
	}
}

// fetchAttributes gets all the needed attributes for creating bastion host and its security group with given <nodeName>.
func (a *AliyunInstanceAttribute) fetchAttributes(nodeName string) {
	a.ShootName = getShootClusterName()
	var err error
	a.InstanceID, err = fetchAlicloudInstanceIDByNodeName(nodeName)
	checkError(err)

	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstanceAttribute --InstanceId="+a.InstanceID)
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)

	a.RegionID, err = decodedQuery.String("RegionId")
	checkError(err)
	a.ZoneID, err = decodedQuery.String("ZoneId")
	checkError(err)
	a.VpcID, err = decodedQuery.String("VpcAttributes", "VpcId")
	checkError(err)
	a.VSwitchID, err = decodedQuery.String("VpcAttributes", "VSwitchId")
	checkError(err)
	a.ImageID, err = decodedQuery.String("ImageId")
	checkError(err)
	ips, err := decodedQuery.ArrayOfStrings("VpcAttributes", "PrivateIpAddress", "IpAddress")
	checkError(err)
	a.PrivateIP = ips[0]
	a.BastionSecurityGroupName = a.ShootName + "-bsg"
	a.BastionInstanceName = a.ShootName + "-bastion"

	a.InstanceChargeType = "PostPaid"
	a.InternetChargeType = "PayByTraffic"
	a.InternetMaxBandwidthIn = "10"
	a.InternetMaxBandwidthOut = "100"
	a.IoOptimized = "optimized"
	a.KeyPairName = a.ShootName + "-ssh-publickey"
	a.InstanceType = a.getMinimumInstanceSpec()
}

// createBastionHostSecurityGroup finds the or creates a security group for the bastion host.
func (a *AliyunInstanceAttribute) createBastionHostSecurityGroup() {
	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeSecurityGroups --VpcId="+a.VpcID)
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)

	securityGroupNames, err := decodedQuery.Array("SecurityGroups", "SecurityGroup")
	checkError(err)
	securityGroupExists := false
	for _, iter := range securityGroupNames {
		securityGroup := jsonq.NewQuery(iter)
		name, err := securityGroup.String("SecurityGroupName")
		checkError(err)
		if name == a.BastionSecurityGroupName {
			securityGroupExists = true
			a.BastionSecurityGroupID, err = securityGroup.String("SecurityGroupId")
			checkError(err)
		}
	}

	if !securityGroupExists {
		res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs CreateSecurityGroup --RegionId="+a.RegionID+" --VpcId="+a.VpcID+" --SecurityGroupName="+a.BastionSecurityGroupName)
		checkError(err)
		decodedQuery = decodeAndQueryFromJSONString(res)
		a.BastionSecurityGroupID, err = decodedQuery.String("SecurityGroupId")
		checkError(err)
		attemptCnt := 0
		for attemptCnt < 60 {
			res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeSecurityGroups --SecurityGroupIds=\"['"+a.BastionSecurityGroupID+"']\"")
			checkError(err)
			decodedQuery = decodeAndQueryFromJSONString(res)
			totalCount, err := decodedQuery.Int("TotalCount")
			checkError(err)
			if totalCount == 1 {
				time.Sleep(time.Second * 30)
				fmt.Println("Bastion host security group created.")
				break
			}
			fmt.Println("Creating bastion host security group...")
			time.Sleep(time.Second * 2)
			attemptCnt++
		}
		if attemptCnt == 60 {
			fmt.Println("Bastion host security group creation time out. Please try again.")
			os.Exit(2)
		}
		fmt.Println("Configuring bastion host security group rules...")
		_, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs AuthorizeSecurityGroup --Policy Accept --NicType intranet --Priority 1 --SourceCidrIp 0.0.0.0/0 --PortRange 22/22 --IpProtocol udp --SecurityGroupId="+a.BastionSecurityGroupID)
		checkError(err)
		_, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs AuthorizeSecurityGroup --Policy Accept --NicType intranet --Priority 1 --SourceCidrIp 0.0.0.0/0 --PortRange 22/22 --IpProtocol tcp --SecurityGroupId="+a.BastionSecurityGroupID)
		checkError(err)
		time.Sleep(time.Second * 10)
		fmt.Println("Bastion host security group rules configured.")
	}
}

// createBastionHostInstance finds the or creates a bastion host instance.
func (a *AliyunInstanceAttribute) createBastionHostInstance(sshPublicKey []byte) {
	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstances --VpcId="+a.VpcID)
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)

	instances, err := decodedQuery.Array("Instances", "Instance")
	checkError(err)
	bastionServerExists := false
	for _, iter := range instances {
		instance := jsonq.NewQuery(iter)
		instanceName, err := instance.String("InstanceName")
		checkError(err)
		if instanceName == a.BastionInstanceName {
			bastionServerExists = true
			a.BastionInstanceID, err = instance.String("InstanceId")
			checkError(err)
			if checkIsThereGardenerUser(a.BastionInstanceID) {
				a.BastionSSHUser = "gardener"
			} else {
				// The bastion is created before `gardener-user` change
				a.BastionSSHUser = "root"
			}
			break
		}
	}

	if !bastionServerExists {
		userData := getBastionUserData(sshPublicKey)
		encodedUserData := base64.StdEncoding.EncodeToString(userData)

		arguments := "aliyun ecs CreateInstance --ImageId=" + a.ImageID + " --InstanceType=" + a.InstanceType + " --RegionId=" + a.RegionID + " --ZoneId=" + a.ZoneID + " --VSwitchId=" + a.VSwitchID + " --InstanceChargeType=" + a.InstanceChargeType + " --InternetChargeType=" + a.InternetChargeType + " --InternetMaxBandwidthIn=" + a.InternetMaxBandwidthIn + " --InternetMaxBandwidthOut=" + a.InternetMaxBandwidthOut + " --IoOptimized=" + a.IoOptimized + " --KeyPairName=" + a.KeyPairName + " --InstanceName=" + a.BastionInstanceName + " --SecurityGroupId=" + a.BastionSecurityGroupID + " --UserData=" + encodedUserData
		res, err = ExecCmdReturnOutput("bash", "-c", arguments)
		checkError(err)
		decodedQuery = decodeAndQueryFromJSONString(res)
		a.BastionInstanceID, err = decodedQuery.String("InstanceId")
		a.BastionSSHUser = "gardener"
		checkError(err)
		attemptCnt := 0
		for attemptCnt < 60 {
			res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstances --InstanceIds=\"['"+a.BastionInstanceID+"']\"")
			checkError(err)
			decodedQuery = decodeAndQueryFromJSONString(res)
			totalCount, err := decodedQuery.Int("TotalCount")
			checkError(err)
			if totalCount == 1 {
				time.Sleep(time.Second * 30)
				fmt.Println("Bastion host created.")
				break
			}
			fmt.Println("Creating bastion host...")
			time.Sleep(time.Second * 2)
			attemptCnt++
		}
		if attemptCnt == 60 {
			fmt.Println("Bastion host creation time out. Please try again.")
			os.Exit(2)
		}
	}
}

// startBastionHostInstances starts the bastion host and allocates a public ip for it.
func (a *AliyunInstanceAttribute) startBastionHostInstance() {
	attemptCnt := 0
	for attemptCnt < 60 {
		res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstanceAttribute --InstanceId="+a.BastionInstanceID)
		checkError(err)
		decodedQuery := decodeAndQueryFromJSONString(res)
		status, err := decodedQuery.String("Status")
		checkError(err)
		if status == "Running" {
			time.Sleep(time.Second * 30)
			fmt.Println("Bastion host started.")
			break
		} else if status == "Stopped" {
			fmt.Println("Starting bastion host...")
			_, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs StartInstance --InstanceId="+a.BastionInstanceID)
			checkError(err)
		} else if status == "Starting" {
			fmt.Println("Waiting for bastion host to start...")
		} else if status == "Stopping" {
			fmt.Println("Bastion host is currently stopping...")
		}
		time.Sleep(time.Second * 2)
		attemptCnt++
	}
	if attemptCnt == 60 {
		fmt.Println("Bastion host starting time out. Please try again.")
		os.Exit(2)
	}
	fmt.Println("Allocating bastion host IP address...")
	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs AllocatePublicIpAddress --InstanceId="+a.BastionInstanceID)
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)
	a.BastionIP, err = decodedQuery.String("IpAddress")
	checkError(err)
	time.Sleep(time.Second * 10)
	fmt.Println("Bastion host IP address allocated.")
}

// deleteBastionHostInstance stops the bastion host instance and deletes it.
func (a *AliyunInstanceAttribute) deleteBastionHostInstance() {
	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstances")
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)
	instances, err := decodedQuery.Array("Instances", "Instance")
	checkError(err)

	for _, iter := range instances {
		instance := jsonq.NewQuery(iter)
		instanceName, err := instance.String("InstanceName")
		checkError(err)
		if instanceName == a.BastionInstanceName {
			a.BastionInstanceID, err = instance.String("InstanceId")
			checkError(err)
			a.VpcID, err = instance.String("VpcAttributes", "VpcId")
			checkError(err)
			break
		}
	}
	if a.BastionInstanceID == "" {
		fmt.Println("No bastion server instance found.")
	} else {
		attemptCnt := 0
		for attemptCnt < 60 {
			res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstanceAttribute --InstanceId="+a.BastionInstanceID)
			checkError(err)
			decodedQuery = decodeAndQueryFromJSONString(res)
			status, err := decodedQuery.String("Status")
			checkError(err)
			if status == "Stopped" {
				time.Sleep(time.Second * 30)
				break
			} else if status == "Running" {
				fmt.Println("Stopping bastion server instance...")
				_, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs StopInstance --InstanceId="+a.BastionInstanceID)
				checkError(err)
			} else if status == "Starting" {
				fmt.Println("Bastion server instance is currently starting...")
			} else if status == "Stopping" {
				fmt.Println("Waiting for bastion server instance to stop...")
			}
			time.Sleep(time.Second * 2)
			attemptCnt++
		}
		if attemptCnt == 60 {
			fmt.Println("Bastion server instance stopping timeout. Please try again.")
			os.Exit(2)
		}
		fmt.Println("Bastion server instance stopped.")

		attemptCnt = 0
		for attemptCnt < 60 {
			res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstances --InstanceIds=\"['"+a.BastionInstanceID+"']\"")
			checkError(err)
			decodedQuery = decodeAndQueryFromJSONString(res)
			totalCount, err := decodedQuery.Int("TotalCount")
			checkError(err)
			if totalCount == 0 {
				time.Sleep(time.Second * 30)
				break
			} else if totalCount == 1 {
				fmt.Println("Deleting bastion server instance...")
				_, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DeleteInstance --Force true --InstanceId="+a.BastionInstanceID)
				checkError(err)
			}
			fmt.Println("Waiting for bastion server instance to be deleted...")
			time.Sleep(time.Second * 2)
			attemptCnt++
		}
		if attemptCnt == 60 {
			fmt.Println("Bastion server instance deletion timeout. Please try again.")
			os.Exit(2)
		}
		fmt.Println("Bastion server instance deleted.")
	}
}

// deleteBastionHostSecurityGroup deletes the security group of the bastion host.
func (a *AliyunInstanceAttribute) deleteBastionHostSecurityGroup() {
	var (
		res string
		err error
	)
	if a.VpcID != "" {
		res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeSecurityGroups --VpcId="+a.VpcID)
	} else {
		res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeSecurityGroups")
	}
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)
	securityGroups, err := decodedQuery.Array("SecurityGroups", "SecurityGroup")
	checkError(err)

	for _, iter := range securityGroups {
		securityGroup := jsonq.NewQuery(iter)
		sgName, err := securityGroup.String("SecurityGroupName")
		checkError(err)
		if sgName == a.BastionSecurityGroupName {
			a.BastionSecurityGroupID, err = securityGroup.String("SecurityGroupId")
			checkError(err)
			break
		}
	}
	if a.BastionSecurityGroupID == "" {
		fmt.Println("No bastion server security group found.")
	} else {
		attemptCnt := 0
		for attemptCnt < 60 {
			res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeSecurityGroups --SecurityGroupIds=\"['"+a.BastionSecurityGroupID+"']\"")
			checkError(err)
			decodedQuery = decodeAndQueryFromJSONString(res)
			totalCount, err := decodedQuery.Int("TotalCount")
			checkError(err)
			if totalCount == 0 {
				time.Sleep(time.Second * 2)
				break
			} else if totalCount == 1 {
				fmt.Println("Deleting bastion server security group...")
				_, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DeleteSecurityGroup --SecurityGroupId="+a.BastionSecurityGroupID)
				checkError(err)
			}
			fmt.Println("Waiting for bastion server security group to be deleted...")
			time.Sleep(time.Second * 2)
			attemptCnt++
		}
		if attemptCnt == 60 {
			fmt.Println("Bastion server security group deletion time out. Please try again.")
			os.Exit(2)
		}
		fmt.Println("SecurityGroup " + a.BastionSecurityGroupName + " deleted.")
	}
}

// configureAliyunCLI sets up user credential configurations for aliyuncli.
func configureAliyunCLI() {
	operate("aliyun", "echo Configuring aliyun cli...")
}

// decodeAndQueryFromJSONString returns the decoded JsonQuery with the given json string.
func decodeAndQueryFromJSONString(jsonString string) *jsonq.JsonQuery {
	data := map[string]interface{}{}
	decoder := json.NewDecoder(strings.NewReader(jsonString))
	err := decoder.Decode(&data)
	checkError(err)
	return jsonq.NewQuery(data)
}

// fetchAlicloudInstanceIDByNodeName returns the instance ID for node for given <nodeName>.
func fetchAlicloudInstanceIDByNodeName(nodeName string) (string, error) {
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)

	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, node := range nodes.Items {
		if nodeName == node.Name {
			return strings.Split(node.Spec.ProviderID, ".")[1], nil
		}
	}

	return "", fmt.Errorf("Cannot find InstanceID for node %q", nodeName)
}

// checkIsThereGardenerUser checks if the bastion contains gardener user
func checkIsThereGardenerUser(instanceID string) bool {
	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeUserData --InstanceId="+instanceID)
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)
	userData, err := decodedQuery.String("UserData")
	checkError(err)

	return userData != ""
}

// parseAliyunInstanceTypeSpec parses instance type spec with given interface <data>.
func (spec *AliyunInstanceTypeSpec) parseAliyunInstanceTypeSpec(data interface{}) {
	instanceType := jsonq.NewQuery(data)
	var err error
	spec.CPUCoreCount, err = instanceType.Int("CpuCoreCount")
	checkError(err)
	spec.InstanceTypeFamily, err = instanceType.String("InstanceTypeFamily")
	checkError(err)
	spec.EniQuantity, err = instanceType.Int("EniQuantity")
	checkError(err)
	spec.InstanceTypeID, err = instanceType.String("InstanceTypeId")
	checkError(err)
	spec.InstanceFamilyLevel, err = instanceType.String("InstanceFamilyLevel")
	checkError(err)
	spec.GPUSpec, err = instanceType.String("GPUSpec")
	checkError(err)
	spec.MemorySize, err = instanceType.Float("MemorySize")
	checkError(err)
	spec.GPUAmount, err = instanceType.Int("GPUAmount")
	checkError(err)
	spec.LocalStorageCategory, err = instanceType.String("LocalStorageCategory")
	checkError(err)
	spec.EniPrivateIPAddressQuantity, err = instanceType.Int("EniPrivateIpAddressQuantity")
	checkError(err)
}

// getInstanceTypeSpecScore calculates the score of an instance type, the smaller it is, the smaller the instance is.
func getInstanceTypeSpecScore(spec AliyunInstanceTypeSpec) float64 {
	score := (float64(spec.CPUCoreCount*20) + spec.MemorySize*20 + float64(spec.GPUAmount*40) + float64(spec.EniPrivateIPAddressQuantity*2) + float64(spec.EniQuantity*10))
	switch spec.InstanceFamilyLevel {
	case "CreditEntryLevel":
		score += 100
	case "EnterpriseLevel":
		score += 500
	default:
		score += 200
	}
	return score
}

// compareAndGetMinimumInstanceTypeSpec compares the original and the challenging instance type, and sets the original spec to the smaller one.
func (spec *AliyunInstanceTypeSpec) compareAndGetMinimumInstanceTypeSpec(rival AliyunInstanceTypeSpec) {
	specScore := getInstanceTypeSpecScore(*spec)
	rivalScore := getInstanceTypeSpecScore(rival)
	if rivalScore < specScore || spec.InstanceTypeID == "" {
		spec.CPUCoreCount = rival.CPUCoreCount
		spec.EniPrivateIPAddressQuantity = rival.EniPrivateIPAddressQuantity
		spec.EniQuantity = rival.EniQuantity
		spec.GPUAmount = rival.GPUAmount
		spec.GPUSpec = rival.GPUSpec
		spec.InstanceFamilyLevel = rival.InstanceFamilyLevel
		spec.InstanceTypeFamily = rival.InstanceTypeFamily
		spec.InstanceTypeID = rival.InstanceTypeID
		spec.LocalStorageCategory = rival.LocalStorageCategory
		spec.MemorySize = rival.MemorySize
	}
}

// getMinimumInstanceSpec returns the name of the instance type with minimum specifications (such as minimum cpu).
func (a *AliyunInstanceAttribute) getMinimumInstanceSpec() string {
	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstanceTypes")
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)

	specs := map[string]AliyunInstanceTypeSpec{}
	instanceTypes, err := decodedQuery.Array("InstanceTypes", "InstanceType")
	checkError(err)
	for _, iter := range instanceTypes {
		spec := &AliyunInstanceTypeSpec{}
		spec.parseAliyunInstanceTypeSpec(iter)
		specs[spec.InstanceTypeID] = *spec
	}

	res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeAvailableResource --ZoneId="+a.ZoneID+" --DestinationResource=InstanceType --IoOptimized optimized")
	checkError(err)
	decodedQuery = decodeAndQueryFromJSONString(res)

	zones, err := decodedQuery.Array("AvailableZones", "AvailableZone")
	checkError(err)
	decodedQuery = jsonq.NewQuery(zones[0])
	availableResources, err := decodedQuery.Array("AvailableResources", "AvailableResource")
	checkError(err)
	decodedQuery = jsonq.NewQuery(availableResources[0])
	supportedResources, err := decodedQuery.Array("SupportedResources", "SupportedResource")
	checkError(err)

	currentMinimumSpec := &AliyunInstanceTypeSpec{}
	for _, iter := range supportedResources {
		resource := jsonq.NewQuery(iter)
		name, err := resource.String("Value")
		checkError(err)
		currentMinimumSpec.compareAndGetMinimumInstanceTypeSpec(specs[name])
	}

	return currentMinimumSpec.InstanceTypeID
}

//checkIsDeletionWanted checks if the user wants to delete the created IAS resources
func checkIsDeletionWanted(bastionInstanceID string) {
	fmt.Println("Would you like to cleanup the created bastion? (y/n)")

	reader := bufio.NewReader(os.Stdin)
	char, _, err := reader.ReadRune()
	checkError(err)

	switch char {
	case 'y', 'Y':
		fmt.Println("Cleanup")
		cleanupAliyunBastionHost()
	case 'n', 'N':
		fmt.Println("- Run following command to hibernate bastion host:")
		fmt.Println("gardenctl aliyun ecs StopInstance -- --InstanceId=" + bastionInstanceID)
		fmt.Println("")
		fmt.Println("- Run following command before shoot deletion:")
		fmt.Println("gardenctl ssh cleanup")
	default:
		fmt.Println("Unknown option")
	}
}

// cleanupAlicloudBastionHost cleans up the bastion host for the targeted cluster.
func cleanupAliyunBastionHost() {
	fmt.Println("Cleaning up bastion host configurations...")

	fmt.Println("")
	fmt.Println("(1/4) Configuring aliyun cli")
	configureAliyunCLI()
	fmt.Println("Aliyun cli configured.")

	a := &AliyunInstanceAttribute{}

	fmt.Println("")
	fmt.Println("(2/4) Fetching data from target shoot cluster")
	a.ShootName = getShootClusterName()
	a.BastionInstanceName = a.ShootName + "-bastion"
	a.BastionSecurityGroupName = a.ShootName + "-bsg"
	fmt.Println("Data fetched from target shoot cluster.")

	fmt.Println("")
	fmt.Println("(3/4) Cleaning up bastion host instance")
	a.deleteBastionHostInstance()

	// Clean up bastion security group
	fmt.Println("")
	fmt.Println("(4/4) Clean up bastion server security group")
	a.deleteBastionHostSecurityGroup()

	fmt.Println("")
	fmt.Println("Bastion server settings cleaned up.")
}
