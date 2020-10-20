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
//
// SSH to openstack is performed in following steps:
// 1) get network ID of public network
// 2) create floating IP from public network
// 3) associate server node with FIP created
// 4) perform ssh
// 5) perform cleanup (de-associate FIP from server / delete FIP)
// Note: no Bastion VM is needed, no SG rule is needed
//

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

//OpenstackInstanceAttribute stores all the critical information for creating an instance on Openstack.
type OpenstackInstanceAttribute struct {
	InstanceID string
	networkID  string
	FIP        string
}

//sshToOpenstackNode ssh to openstack node
func sshToOpenstackNode(nodeName, path, user, pathSSKeypair string, sshPublicKey []byte, myPublicIP string) {
	a := &OpenstackInstanceAttribute{}
	a.InstanceID = nodeName
	var err error

	fmt.Println("(1/5) Getting the external network for creating FIP")
	resNetwork := operate("openstack", "network list --external -f json")
	if len(resNetwork) < 2 {
		fmt.Println("External network not found!")
		os.Exit(2)
	}
	resNetwork = resNetwork[1 : len(resNetwork)-2] // network returns with [], trim them before next step json decode
	decodedQueryNetwork := decodeAndQueryFromJSONString(resNetwork)
	a.networkID, err = decodedQueryNetwork.String("ID")
	fmt.Println("The external network ID is " + a.networkID)
	checkError(err)

	fmt.Println("(2/5) Creating floating IP from external network")
	resFloatingIP := operate("openstack", "floating ip create "+a.networkID+"  -f json")
	decodedQueryFIP := decodeAndQueryFromJSONString(resFloatingIP)
	a.FIP, err = decodedQueryFIP.String("floating_ip_address")
	fmt.Println("The floating IP created is " + a.FIP)
	checkError(err)
	time.Sleep(5000)

	fmt.Println("(3/5) Add floating IP to openstack server node")
	operate("openstack", "server add floating ip "+a.InstanceID+" "+a.FIP)
	time.Sleep(5000)

	defer a.cleanUpOpenstack()

	node := user + "@" + a.FIP
	fmt.Println("(4/5) Establishing SSH connection")
	fmt.Println("")

	key := filepath.Join(pathSSKeypair, "key")
	args := []string{"-i" + key, "-oStrictHostKeyChecking=no", node}
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

//cleanUpOpenstack clean the resource added to ssh to openstack node
func (a *OpenstackInstanceAttribute) cleanUpOpenstack() {
	fmt.Println("")
	fmt.Println("(5/5) Cleanup")

	fmt.Println("De-associate server with floating ip")
	operate("openstack", "server remove floating ip "+a.InstanceID+" "+a.FIP)

	fmt.Println("Delete the floating IP")
	operate("openstack", "floating ip delete "+a.FIP)

}
