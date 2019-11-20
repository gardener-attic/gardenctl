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
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/jmoiron/jsonq"
)

// AzureInstanceAttribute stores all the critical information for creating an instance on Azure.
type AzureInstanceAttribute struct {
	NamePublicIP       string
	ShootName          string
	NicName            string
	PublicIP           string
	RescourceGroupName string
	SecurityGroupName  string
	SkuType            string
}

// sshToAZNode provides cmds to ssh to az via a node name and clean it up afterwards
func sshToAZNode(nodeName, path, user string, sshPublicKey []byte) {
	a := &AzureInstanceAttribute{}

	fmt.Println("")
	fmt.Println("(1/4) Fetching data from target shoot cluster")
	a.fetchAzureAttributes(nodeName, path)
	fmt.Println("Data fetched from target shoot cluster.")
	fmt.Println("")

	fmt.Println("(2/4) Configuring Azure")

	// add nsg rule
	a.addNsgRule()
	fmt.Println("")

	// create public ip
	a.createPublicIP()
	fmt.Println("Waiting 5 s until public ip is available.")
	fmt.Println("")
	time.Sleep(5 * time.Second)

	// update nic ip-config
	a.configureNic()

	node := user + "@" + a.PublicIP
	fmt.Println("Waiting 30 seconds until ports are open.")
	time.Sleep(30 * time.Second)
	fmt.Println("(3/4) Establishing SSH connection")
	fmt.Println("")

	sshCmd := fmt.Sprintf("ssh -i key -o StrictHostKeyChecking=no " + node)
	cmd := exec.Command("bash", "-c", sshCmd)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	checkError(err)

	fmt.Println("")
	fmt.Println("(4/4) Cleanup")
	a.cleanupAzure()
}

// fetchAttributes gets all the needed attributes for creating bastion host and its security group with given <nodeName>.
func (a *AzureInstanceAttribute) fetchAzureAttributes(nodeName, path string) {
	a.NamePublicIP = "sshIP"
	var err error
	terraformVersion, err := ExecCmdReturnOutput("bash", "-c", "cat "+path+"  | jq -r .terraform_version")
	checkError(err)
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
		a.RescourceGroupName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.outputs.resourceGroupName.value'")
		checkError(err)
		a.SecurityGroupName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.outputs.securityGroupName.value'")
		checkError(err)
	} else {
		a.RescourceGroupName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.modules[].outputs.resourceGroupName.value'")
		checkError(err)
		a.SecurityGroupName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.modules[].outputs.securityGroupName.value'")
		checkError(err)
	}

	targetMachineName, err := fetchAzureMachineNameByNodeName(nodeName)
	checkError(err)

	// remove invisible controll character which are somehow encoded in the *v1.Node object
	re := regexp.MustCompile("[[:^ascii:]]")
	a.NicName = re.ReplaceAllLiteralString(targetMachineName+"-nic", "")

	// parse sku type (Basic,Standard,...) from lbs
	arguments := fmt.Sprintf("az network lb list --resource-group %s", a.RescourceGroupName)
	captured := capture()
	operate("az", arguments)
	skuType, err := captured()
	checkError(err)
	tmpfile, err := ioutil.TempFile(os.TempDir(), "lbs.json")
	checkError(err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write([]byte(skuType))
	checkError(err)
	skuType, err = ExecCmdReturnOutput("bash", "-c", "cat "+tmpfile.Name()+" | jq .[0].sku.name")
	a.SkuType = strings.Trim(skuType, "\"")
	fmt.Println(a.SkuType)
	checkError(err)
}

// addNsgRule creates a nsg rule to open the ssh port
func (a *AzureInstanceAttribute) addNsgRule() {
	fmt.Println("Opened SSH Port.")
	arguments := fmt.Sprintf("az network nsg rule create --resource-group %s  --nsg-name %s --name ssh --protocol Tcp --priority 1000 --destination-port-range 22", a.RescourceGroupName, a.SecurityGroupName)
	operate("az", arguments)
}

// createPublicIP creates the public ip for nic
func (a *AzureInstanceAttribute) createPublicIP() {
	var err error
	fmt.Println("Create public ip")
	arguments := fmt.Sprintf("az network public-ip create -g %s -n %s --sku %s --allocation-method static --tags component=gardenctl", a.RescourceGroupName, a.NamePublicIP, a.SkuType)
	captured := capture()
	operate("az", arguments)
	a.PublicIP, err = captured()
	checkError(err)
	fmt.Println(a.PublicIP)
	data := map[string]interface{}{}
	dec := json.NewDecoder(strings.NewReader(a.PublicIP))
	err = dec.Decode(&data)
	checkError(err)
	jq := jsonq.NewQuery(data)
	a.PublicIP, err = jq.String("publicIp", "ipAddress")
	if err != nil {
		os.Exit(2)
	}
}

// configureNic attaches a public ip to the nic
func (a *AzureInstanceAttribute) configureNic() {
	var err error
	fmt.Println("Add public ip to nic")
	fmt.Println("")
	arguments := fmt.Sprintf("az network nic ip-config update -g %s --nic-name %s --public-ip-address %s -n %s", a.RescourceGroupName, a.NicName, a.NamePublicIP, a.NicName)
	captured := capture()
	operate("az", arguments)
	_, err = captured()
	checkError(err)
}

// fetchAzureMachineNameByNodeName returns the name of machine with the given <nodeName>.
func fetchAzureMachineNameByNodeName(nodeName string) (string, error) {
	machines := getMachines()
	for _, machine := range machines.Items {
		if machine.Status.Node == nodeName {
			return machine.Name, nil
		}
	}

	return "", fmt.Errorf("Cannot find Machine for node %q", nodeName)
}

// cleanupAzure cleans up all created azure resources required for ssh connection
func (a *AzureInstanceAttribute) cleanupAzure() {
	var err error

	// remove ssh rule
	fmt.Println("")
	fmt.Println("  (1/3) Remove SSH rule")
	arguments := fmt.Sprintf("az network nsg rule delete --resource-group %s  --nsg-name %s --name ssh", a.RescourceGroupName, a.SecurityGroupName)
	captured := capture()
	operate("az", arguments)
	_, err = captured()
	checkError(err)

	// remove public ip address from nic
	fmt.Println("")
	fmt.Println("  (2/3) Remove public ip from nic")
	arguments = fmt.Sprintf("az network nic ip-config update -g %s --nic-name %s --public-ip-address %s -n %s --remove publicIPAddress", a.RescourceGroupName, a.NicName, a.NamePublicIP, a.NicName)
	captured = capture()
	operate("az", arguments)
	_, err = captured()
	checkError(err)

	// delete ip
	fmt.Println("")
	fmt.Println("  (3/3) Delete public ip")
	arguments = fmt.Sprintf("az network public-ip delete -g %s -n %s", a.RescourceGroupName, a.NamePublicIP)
	captured = capture()
	operate("az", arguments)
	_, err = captured()
	checkError(err)
	fmt.Println("")
	fmt.Println("Configuration successfully cleaned up.")
}
