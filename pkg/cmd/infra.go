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
	"regexp"
	"strings"

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
					var rs []string
					pathTerraformState := filepath.Join(downloadTerraformFiles("infra", targetReader), "terraform.tfstate")
					buf, err := ioutil.ReadFile(pathTerraformState)
					if err != nil || len(buf) < 64 {
						fmt.Println("Could not read terraform.tfstate: " + pathTerraformState)
						os.Exit(2)
					}
					terraformstate := string(buf)

					shoot, err := FetchShootFromTarget(target)
					checkError(err)
					infraType := shoot.Spec.Provider.Type

					switch infraType {
					case "aws":
						rs = getAWSInfraResources()
					case "azure":
						rs = getAzureInfraResources()
					case "gcp":
						rs = getGCPInfraResources()
					case "openstack":
						rs = getOstackInfraResources()
					case "alicloud":
						return errors.New("infra type not supported")
					default:
						return errors.New("infra type not found")
					}

					err = GetOrphanInfraResources(rs, terraformstate)
					checkError(err)
					fmt.Printf("\n\nsearched %s\n", pathTerraformState)
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

// GetOrphanInfraResources list orphan infra resources on targeted cluster
func GetOrphanInfraResources(rs []string, terraformstate string) error {
	var hasOrphan bool

	if len(rs) < 1 {
		return errors.New("No infra resources found")
	}

	fmt.Printf("(%d) infra resources found: \n%s\n", len(rs), rs)
	for _, rsid := range rs {
		if !strings.Contains(terraformstate, rsid) {
			fmt.Printf("\nOrphan: resource id %s not found in terraform state", rsid)
			hasOrphan = true
		}
	}

	if !hasOrphan {
		fmt.Printf("\nNo orphan resource found")
	}

	return nil
}

func getAWSInfraResources() []string {
	rs := make([]string, 0)
	shoottag := getTechnicalID()

	// fetch shoot vpc resources
	capturedOutput := execInfraOperator("aws", "aws ec2 describe-vpcs --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`VPCS.*(vpc-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot subnet resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-subnets --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`SUBNETS.*:subnet\/(subnet-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot dhcp options resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-dhcp-options --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`DHCPOPTIONS.*(dopt-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot ip address resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-addresses --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`ADDRESSES.*(eipalloc-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot nat gateway resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-nat-gateways --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`NATGATEWAYS.*(nat-[a-z0-9]*)`, capturedOutput, rs)
	rs = findInfraResourcesMatch(`NATGATEWAYADDRESSES.*(eni-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot internet gateway resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-internet-gateways --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`INTERNETGATEWAYS.*(igw-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot security group resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-security-groups --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`SECURITYGROUPS.*(sg-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot route table resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-route-tables --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`ROUTETABLES.*(rtb-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot instance resources
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-instances --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1")
	rs = findInfraResourcesMatch(`IAMINSTANCEPROFILE.*:instance-profile\/(shoot--[a-z0-9-]*-nodes)`, capturedOutput, rs)

	// fetch shoot bastion instance resource
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-instances --filter Name=tag:Name,Values="+shoottag+"-bastions")
	rs = findInfraResourcesMatch(`INSTANCES.*(i-[a-z0-9]*)`, capturedOutput, rs)

	// fetch shoot bastion security group
	capturedOutput = execInfraOperator("aws", "aws ec2 describe-security-groups --filter Name=tag:component,Values=gardenctl")
	rs = findInfraResourcesMatch("SECURITYGROUPS.*(sg-[a-z0-9]*).*"+shoottag, capturedOutput, rs)

	return unique(rs)
}

func getAzureInfraResources() []string {
	rs := make([]string, 0)
	shoottag := getTechnicalID()

	// fetch shoot resource group
	capturedOutput := execInfraOperator("az", "az group show --name "+shoottag)
	rs = findInfraResourcesMatch(`\"id\".*(resourceGroups\/[a-z0-9-]*)\"`, capturedOutput, rs)

	// fetch shoot vnet resources
	capturedOutput = execInfraOperator("az", "az network vnet list -g "+shoottag)
	vnets := make([]string, 0)
	vnets = findInfraResourcesMatch(`\"id\".*(virtualNetworks\/[a-z0-9-]*)\"`, capturedOutput, vnets)
	rs = findInfraResourcesMatch(`\"id\".*(virtualNetworks\/[a-z0-9-]*)\"`, capturedOutput, rs)

	// fetch shoot subnet resources
	if len(vnets) > 0 {
		for _, vnet := range vnets {
			s := strings.Split(vnet, "/")
			vnetName := s[1]
			capturedOutput = execInfraOperator("az", "az network vnet subnet list -g "+shoottag+" --vnet-name "+vnetName)
			rs = findInfraResourcesMatch(`\"id\".*(subnets\/[a-z0-9-]*)\"`, capturedOutput, rs)
		}
	}

	// fetch shoot nic resources
	capturedOutput = execInfraOperator("az", "az network nic list -g "+shoottag)
	rs = findInfraResourcesMatch(`\"id\".*(networkInterfaces\/[a-z0-9-]*)\"`, capturedOutput, rs)

	// fetch shoot security group resources
	capturedOutput = execInfraOperator("az", "az network nsg list -g "+shoottag)
	rs = findInfraResourcesMatch(`\"id\".*(networkSecurityGroups\/[a-z0-9-]*)\"`, capturedOutput, rs)

	// fetch shoot route resources
	capturedOutput = execInfraOperator("az", "az network route-table list -g "+shoottag)
	rs = findInfraResourcesMatch(`\"id\".*routes\/([a-z0-9-]*)\"`, capturedOutput, rs)

	return unique(rs)
}

func getGCPInfraResources() []string {
	rs := make([]string, 0)
	shoottag := getTechnicalID()

	// fetch shoot subnet resource
	capturedOutput := execInfraOperator("gcp", "gcloud compute networks subnets list")
	if strings.Contains(capturedOutput, shoottag+"-nodes") {
		rsShootSubnet := make([]string, 0)
		rsShootSubnet = findInfraResourcesMatch(shoottag+"-nodes(.*)", capturedOutput, rsShootSubnet)
		if len(rsShootSubnet) > 0 {
			rsShootSubnet = strings.Fields(rsShootSubnet[0])
			shootVpc := rsShootSubnet[1]
			rs = append(rs, shoottag+"-nodes")

			// fetch shoot vpc resource
			capturedOutput = execInfraOperator("gcp", "gcloud compute networks list")
			if strings.Contains(capturedOutput, shootVpc) {
				rs = append(rs, shootVpc)
			}

			// fetch shoot cloud router resource
			capturedOutput = execInfraOperator("gcp", "gcloud compute routers list")
			if strings.Contains(capturedOutput, shootVpc) {
				rsShootRouter := make([]string, 0)
				rsShootRouter = findInfraResourcesMatch("(.*)"+shootVpc, capturedOutput, rsShootRouter)
				if len(rsShootRouter) > 0 {
					rsShootRouter = strings.Fields(rsShootRouter[0])
					shootRouter := rsShootRouter[0]
					shootRouterRegion := rsShootRouter[1]
					if strings.Contains(capturedOutput, shootRouter) {
						rs = append(rs, shootRouter)

						// fetch shoot cloud nat resource
						capturedOutput = execInfraOperator("gcp", "gcloud compute routers nats list --router="+shootRouter+" --router-region="+shootRouterRegion)
						if strings.Contains(capturedOutput, shoottag+"-cloud-nat") {
							rs = append(rs, shoottag+"-cloud-nat")
						}
					}
				}
			}
		}
	}

	// fetch shoot service account
	capturedOutput = execInfraOperator("gcp", "gcloud iam service-accounts list")
	if strings.Contains(capturedOutput, shoottag) {
		rsserviceAccount := make([]string, 0)
		rsserviceAccount = findInfraResourcesMatch(shoottag+"(.*)False", capturedOutput, rsserviceAccount)
		if len(rsserviceAccount) > 0 {
			serviceAccount := strings.TrimSpace(rsserviceAccount[0])
			if strings.Contains(capturedOutput, serviceAccount) {
				rs = append(rs, serviceAccount)
			}
		}
	}

	return unique(rs)
}

func getOstackInfraResources() []string {
	rs := make([]string, 0)
	shoottag := getTechnicalID()

	// fetch shoot network id
	capturedOutput := execInfraOperator("openstack", "openstack network list")
	if strings.Contains(capturedOutput, shoottag) {
		rsShootNetwork := make([]string, 0)
		rsShootNetwork = findInfraResourcesMatch("(.*)"+shoottag, capturedOutput, rsShootNetwork)
		if len(rsShootNetwork) > 0 {
			rsShootNetwork = strings.Fields(rsShootNetwork[0])
			rsNetworkID := rsShootNetwork[1]
			rs = append(rs, rsNetworkID)
		}
	}

	// fetch shoot subnet id
	capturedOutput = execInfraOperator("openstack", "openstack subnet list")
	if strings.Contains(capturedOutput, shoottag) {
		rsShootSubnet := make([]string, 0)
		rsShootSubnet = findInfraResourcesMatch("(.*)"+shoottag, capturedOutput, rsShootSubnet)
		if len(rsShootSubnet) > 0 {
			rsShootSubnet = strings.Fields(rsShootSubnet[0])
			rsSubnet := rsShootSubnet[1]
			rs = append(rs, rsSubnet)
		}
	}

	// fetch shoot router id
	capturedOutput = execInfraOperator("openstack", "openstack router list")
	if strings.Contains(capturedOutput, shoottag) {
		rsShootRouter := make([]string, 0)
		rsShootRouter = findInfraResourcesMatch("(.*)"+shoottag, capturedOutput, rsShootRouter)
		if len(rsShootRouter) > 0 {
			rsShootRouter = strings.Fields(rsShootRouter[0])
			rsRouter := rsShootRouter[1]
			rs = append(rs, rsRouter)

			// fetch shoot floating network id
			capturedOutput = execInfraOperator("openstack", "openstack floating ip list --router "+rsRouter+" -f value")
			rsShootFloatingNetwork := make([]string, 0)
			rsShootFloatingNetwork = findInfraResourcesMatch(`([a-z0-9-]{36})`, capturedOutput, rsShootFloatingNetwork)
			if len(rsShootFloatingNetwork) > 0 {
				rsFloatingNetwork := rsShootFloatingNetwork[2]
				rs = append(rs, rsFloatingNetwork)
			}
		}
	}

	// fetch shoot security group id
	capturedOutput = execInfraOperator("openstack", "openstack security group list")
	if strings.Contains(capturedOutput, shoottag) {
		rsShootSecurityGroup := make([]string, 0)
		rsShootSecurityGroup = findInfraResourcesMatch("(.*)"+shoottag, capturedOutput, rsShootSecurityGroup)
		if len(rsShootSecurityGroup) > 0 {
			rsShootSecurityGroup = strings.Fields(rsShootSecurityGroup[0])
			rsSecurityGroup := rsShootSecurityGroup[1]
			rs = append(rs, rsSecurityGroup)
		}
	}

	return unique(rs)
}

func execInfraOperator(provider string, arguments string) string {
	captured := capture()
	operate(provider, arguments)
	capturedOutput, err := captured()
	checkError(err)
	return capturedOutput
}

func findInfraResourcesMatch(pattern string, out string, rs []string) []string {
	re, _ := regexp.Compile(pattern)
	values := re.FindAllStringSubmatch(out, -1)
	if len(values) > 0 {
		for _, rsid := range values {
			rs = append(rs, rsid[1])
		}
	}
	return rs
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
