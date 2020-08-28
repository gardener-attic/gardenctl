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
	"os"
	"path/filepath"
	"strings"
	"regexp"
	"io/ioutil"

	"github.com/spf13/cobra"
)

// NewInfraCmd returns a new infra command
func NewInfraCmd(targetReader TargetReader) *cobra.Command {
	return &cobra.Command{
		Use:          "infra [(orphan)] [(list)]",
		Short:        "Manage shoot infra resources\n",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := targetReader.ReadTarget(pathTarget)
			if !CheckShootIsTargeted(target) {
				return errors.New("no shoot targeted")
			}

			if len(args) < 2 || len(args) > 2 {
				return errors.New("command must be in the format: infra [orphan] [list]")
			}

			switch args[0] {
			case "orphan":
				switch args[1] {
				case "list":
					getOrphanInfraResources()
				default:
					fmt.Println("command must be in the format: infra [orphan] [list]")
				}
			default:
				fmt.Println("command must be in the format: infra [orphan] [list]")
			}

			return nil
		},
		ValidArgs: []string{"orphan"},
	}
}

// getOrphanInfraResources list orphan infra resources on targeted cluster
func getOrphanInfraResources() {
	var target Target
	var rs []string
	var has_orphan bool
	ReadTarget(pathTarget, &target)
	gardenName := target.Stack()[0].Name
	pathTerraform := ""

	if target.Target[1].Kind == "project" {
		pathTerraform = filepath.Join(pathGardenHome, "cache", gardenName, "projects", target.Target[1].Name, target.Target[2].Name, "terraform")
	} else if target.Target[1].Kind == "seed" {
		pathTerraform = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, target.Target[2].Name, "terraform")
	}

	_, err := os.Stat(pathTerraform + "/terraform.tfstate")
	if os.IsNotExist(err) {
		downloadTerraformFiles("infra")
	}

	_, err = os.Stat(pathTerraform + "/terraform.tfstate")
	if os.IsNotExist(err) {
		fmt.Println("Could not find terraform.tfstate: " + pathTerraform + "/terraform.tfstate")
		os.Exit(2)
	}

	shoot, err := FetchShootFromTarget(&target)
	checkError(err)
	infraType := shoot.Spec.Provider.Type

<<<<<<< HEAD
	switch infraType {
=======
    switch infraType {
>>>>>>> be1ec437... list orphan infra resources
		case "aws":
				rs = getAWSInfraResources()
		case "gcp":
		case "azure":
		case "alicloud":
		case "openstack":
		default:
				fmt.Printf("infra type %s not found", infraType)
	}

	if (len(rs) < 1) {
		fmt.Printf("No %s infra resource found\n", infraType)
		os.Exit(2)
	}

	buf, err := ioutil.ReadFile(pathTerraform + "/terraform.tfstate")
	if err != nil || len(buf) < 64 {
		fmt.Println("Could not read terraform.tfstate: " + pathTerraform + "/terraform.tfstate")
		os.Exit(2)
	}

	fmt.Printf("(%d) %s infra resources found: %s\n", len(rs), infraType, rs)
	tfstate := strings.ToLower(string(buf))
	for _, rsid := range rs {
		if !strings.Contains(tfstate, rsid) {
<<<<<<< HEAD
			fmt.Printf("\nOrphan: resource id %s not found\n", rsid)
=======
			fmt.Printf("Orphan: resource id %s not found\n", rsid)
>>>>>>> be1ec437... list orphan infra resources
			has_orphan = true
		}
	}
	if (!has_orphan) {
		fmt.Printf("\nNo orphan %s resource found\n", infraType)
	} 
<<<<<<< HEAD
	fmt.Printf("\nsearched %s\n", pathTerraform + "/terraform.tfstate")
=======
	fmt.Printf("searched %s\n", pathTerraform + "/terraform.tfstate")
>>>>>>> be1ec437... list orphan infra resources
}

func getAWSInfraResources() []string {
	var target Target
	ReadTarget(pathTarget, &target)
	rs := make([]string, 0, 16)

	shoottag := "shoot--" + target.Target[1].Name + "--" + target.Target[2].Name

	// fetch shoot vpc resources
	arguments := "aws ec2 describe-vpcs --filter Name=tag:kubernetes.io/cluster/" + shoottag + ",Values=1"
	captured := capture()
	operate("aws", arguments)
	capturedOutput, err := captured()
	checkError(err)
	re, _ := regexp.Compile(`VPCS.*(vpc-[a-z0-9]*)`)
<<<<<<< HEAD
	values := re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
=======
    values := re.FindAllStringSubmatch(capturedOutput, -1)
    if len(values) > 0 {
>>>>>>> be1ec437... list orphan infra resources
    	for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][1])
    	}
    }
<<<<<<< HEAD
	re, _ = regexp.Compile(`VPCS.*(dopt-[a-z0-9]*)`)
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
=======
    re, _ = regexp.Compile(`VPCS.*(dopt-[a-z0-9]*)`)
    values = re.FindAllStringSubmatch(capturedOutput, -1)
    if len(values) > 0 {
>>>>>>> be1ec437... list orphan infra resources
    	for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][1])
    	}
    }    
	// fetch shoot subnet resources
	arguments = "aws ec2 describe-subnets --filter Name=tag:kubernetes.io/cluster/" + shoottag + ",Values=1"
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	re, _ = regexp.Compile(`:subnet\/(subnet-[a-z0-9]*)`)
<<<<<<< HEAD
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
    	for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][1])
    	}
	}   
=======
    values = re.FindAllStringSubmatch(capturedOutput, -1)
    if len(values) > 0 {
    	for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][1])
    	}
    }   
>>>>>>> be1ec437... list orphan infra resources
    // fetch shoot security group resources
	arguments = "aws ec2 describe-security-groups --filter Name=tag:kubernetes.io/cluster/" + shoottag + ",Values=1"
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	re, _ = regexp.Compile(`sg-[a-z0-9]*`)
<<<<<<< HEAD
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
		for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][0])
		}
	}
=======
    values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
    	for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][0])
    	}
    }
>>>>>>> be1ec437... list orphan infra resources
    // fetch shoot route table resources
	arguments = "aws ec2 describe-route-tables --filter Name=tag:kubernetes.io/cluster/" + shoottag + ",Values=1"
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	re, _ = regexp.Compile(`rtb-[a-z0-9]*`)
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
 		for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][0])
    	}
	}
	re, _ = regexp.Compile(`igw-[a-z0-9]*`)
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
 		for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][0])
    	}
	}
	re, _ = regexp.Compile(`nat-[a-z0-9]*`)
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
 		for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][0])
    	}
	}
	// fetch shoot instance resources
	arguments = "aws ec2 describe-instances --filter Name=tag:kubernetes.io/cluster/" + shoottag + ",Values=1"
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	re, _ = regexp.Compile(`:instance-profile\/(shoot--[a-z0-9-]*-nodes)`)
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
 		for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][1])
    	}
	}
	re, _ = regexp.Compile(`shoot--[a-z0-9-]*-ssh-publickey`)
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
 		for i:=0; i < len(values); i++ {
    		rs = append(rs, values[i][0])
    	}
	}
<<<<<<< HEAD
	// fetch shoot bastion instance resource
	arguments = "aws ec2 describe-instances --filter Name=tag:Name,Values=" + shoottag + "-bastions"
	captured = capture()
	operate("aws", arguments)
	capturedOutput, err = captured()
	checkError(err)
	re, _ = regexp.Compile(`shoot--[a-z0-9-]*-bastions`)
	values = re.FindAllStringSubmatch(capturedOutput, -1)
	if len(values) > 0 {
		for i:=0; i < len(values); i++ {
			rs = append(rs, values[i][0])
		}
	}
=======
>>>>>>> be1ec437... list orphan infra resources
	return unique(rs)
}

func unique(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{} 
	for _, entry := range strSlice {
    	if _, value := keys[entry]; !value {
   			keys[entry] = true
   			list = append(list, entry)
    	}
	}    
	return list
}