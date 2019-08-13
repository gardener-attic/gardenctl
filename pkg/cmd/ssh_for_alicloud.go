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
	"encoding/json"
	"fmt"
	"os"
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
	BastionIP                string
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
func sshToAlicloudNode(nodeIP, path string) {
	// Check if this is a cleanup command
	if nodeIP == "clean" {
		cleanupAlicloudBastionHost()
		return
	}

	fmt.Println("(1/5) Configuring aliyun cli")
	configureAliyunCLI()
	var target Target
	ReadTarget(pathTarget, &target)
	aliyunPathSSHKey := ""
	if target.Target[1].Kind == "project" {
		aliyunPathSSHKey = pathGardenHome + "/cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.aliyun/"
	} else if target.Target[1].Kind == "seed" {
		aliyunPathSSHKey = pathGardenHome + "/cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.aliyun/"
	}
	err = ExecCmd(nil, "mv key "+aliyunPathSSHKey, false)
	checkError(err)
	fmt.Println("Aliyun cli configured.")

	a := &AliyunInstanceAttribute{}

	fmt.Println("")
	fmt.Println("(2/5) Fetching data from target shoot cluster")
	a.fetchAttributes(nodeIP)
	fmt.Println("Data fetched from target shoot cluster.")

	fmt.Println("")
	fmt.Println("(3/5) Setting up bastion host security group")
	a.createBastionHostSecurityGroup()
	fmt.Println("Bastion host security group set up.")

	fmt.Println("")
	fmt.Println("(4/5) Setting up bastion host")
	a.createBastionHostInstance()
	fmt.Println("Bastion host set up.")

	fmt.Println("")
	fmt.Println("(5/5) Starting bastion host")
	a.startBastionHostInstance()
	fmt.Println("Bastion host started.")

	fmt.Println("")
	fmt.Println("- Fill in the placeholders and run the following command to ssh onto the target node. For more information about the user, you can check the documentation of the cloud provider:")
	fmt.Println("ssh -i " + aliyunPathSSHKey + "key -o \"ProxyCommand ssh -i " + aliyunPathSSHKey + "key -W " + nodeIP + ":22 <user>@" + a.BastionIP + "\" <user>@" + nodeIP)
	fmt.Println("")
	fmt.Println("- Run following command to hibernate bastion host:")
	fmt.Println("gardenctl aliyun ecs StopInstance -- --InstanceId=" + a.BastionInstanceID)
	fmt.Println("")
	fmt.Println("- Run following command before shoot deletion:")
	fmt.Println("gardenctl ssh clean")
}

// cleanupAlicloudBastionHost cleans up the bastion host for the targeted cluster.
func cleanupAlicloudBastionHost() {
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

// fetchAttributes gets all the needed attributes for creating bastion host and its security group with given <nodeIP>.
func (a *AliyunInstanceAttribute) fetchAttributes(nodeIP string) {
	a.InstanceID = getAlicloudInstanceIDForIP(nodeIP)
	if a.InstanceID == "" {
		fmt.Println("No instance found for ip")
		os.Exit(2)
	}

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
	a.ShootName = getShootClusterName()
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
	checkSecurityGroupExists := false
	for _, iter := range securityGroupNames {
		securityGroup := jsonq.NewQuery(iter)
		name, err := securityGroup.String("SecurityGroupName")
		checkError(err)
		if name == a.BastionSecurityGroupName {
			checkSecurityGroupExists = true
			a.BastionSecurityGroupID, err = securityGroup.String("SecurityGroupId")
			checkError(err)
		}
	}

	if checkSecurityGroupExists == false {
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
		res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs AuthorizeSecurityGroup --Policy Accept --NicType intranet --Priority 1 --SourceCidrIp 0.0.0.0/0 --PortRange 22/22 --IpProtocol udp --SecurityGroupId="+a.BastionSecurityGroupID)
		checkError(err)
		res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs AuthorizeSecurityGroup --Policy Accept --NicType intranet --Priority 1 --SourceCidrIp 0.0.0.0/0 --PortRange 22/22 --IpProtocol tcp --SecurityGroupId="+a.BastionSecurityGroupID)
		checkError(err)
		time.Sleep(time.Second * 10)
		fmt.Println("Bastion host security group rules configured.")
	}
}

// createBastionHostInstance finds the or creates a bastion host instance.
func (a *AliyunInstanceAttribute) createBastionHostInstance() {
	res, err := ExecCmdReturnOutput("bash", "-c", "aliyun ecs DescribeInstances --VpcId="+a.VpcID)
	checkError(err)
	decodedQuery := decodeAndQueryFromJSONString(res)

	instances, err := decodedQuery.Array("Instances", "Instance")
	checkError(err)
	checkBastionServerExists := false
	for _, iter := range instances {
		instance := jsonq.NewQuery(iter)
		instanceName, err := instance.String("InstanceName")
		checkError(err)
		if instanceName == a.BastionInstanceName {
			checkBastionServerExists = true
			a.BastionInstanceID, err = instance.String("InstanceId")
			checkError(err)
		}
	}

	if checkBastionServerExists == false {
		arguments := "aliyun ecs CreateInstance --ImageId=" + a.ImageID + " --InstanceType=" + a.InstanceType + " --RegionId=" + a.RegionID + " --ZoneId=" + a.ZoneID + " --VSwitchId=" + a.VSwitchID + " --InstanceChargeType=" + a.InstanceChargeType + " --InternetChargeType=" + a.InternetChargeType + " --InternetMaxBandwidthIn=" + a.InternetMaxBandwidthIn + " --InternetMaxBandwidthOut=" + a.InternetMaxBandwidthOut + " --IoOptimized=" + a.IoOptimized + " --KeyPairName=" + a.KeyPairName + " --InstanceName=" + a.BastionInstanceName + " --SecurityGroupId=" + a.BastionSecurityGroupID
		res, err = ExecCmdReturnOutput("bash", "-c", arguments)
		checkError(err)
		decodedQuery = decodeAndQueryFromJSONString(res)
		a.BastionInstanceID, err = decodedQuery.String("InstanceId")
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
			res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs StartInstance --InstanceId="+a.BastionInstanceID)
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
				res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs StopInstance --InstanceId="+a.BastionInstanceID)
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
				res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DeleteInstance --Force true --InstanceId="+a.BastionInstanceID)
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
	var res string
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
				res, err = ExecCmdReturnOutput("bash", "-c", "aliyun ecs DeleteSecurityGroup --SecurityGroupId="+a.BastionSecurityGroupID)
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
	decoder.Decode(&data)
	return jsonq.NewQuery(data)
}

// getAlicloudInstanceIDForIP returns the instance ID with the given <nodeIP>.
func getAlicloudInstanceIDForIP(ip string) string {
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)
	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, node := range nodes.Items {
		if ip == node.Status.Addresses[0].Address {
			return strings.Split(node.Spec.ProviderID, ".")[1]
		}
	}
	return ""
}

// parseAliyunInstanceTypeSpec parses instance type spec with given interface <data>.
func (spec *AliyunInstanceTypeSpec) parseAliyunInstanceTypeSpec(data interface{}) {
	instanceType := jsonq.NewQuery(data)
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
		break
	case "EnterpriseLevel":
		score += 500
		break
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
