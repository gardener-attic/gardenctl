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
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	mcmv1alpha1 "github.com/gardener/machine-controller-manager/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// GCPInstanceAttribute stores all the critical information for creating an instance on GCP.
type GCPInstanceAttribute struct {
	ShootName                   string
	BastionHostName             string
	BastionHostFirewallRuleName string
	BastionIP                   string
	FirewallRuleName            string
	VpcName                     string
	Subnetwork                  string
	Zone                        string
	UserData                    []byte
	SSHPublicKey                []byte
}

// sshToGCPNode provides cmds to ssh to gcp via a public ip and clean it up afterwards
func sshToGCPNode(nodeName, path, user string, sshPublicKey []byte) {
	g := &GCPInstanceAttribute{}
	g.SSHPublicKey = sshPublicKey
	fmt.Println("")

	fmt.Println("(1/4) Fetching data from target shoot cluster")
	g.fetchGCPAttributes(nodeName, path)
	fmt.Println("Data fetched from target shoot cluster.")
	fmt.Println("")

	fmt.Println("(2/4) Setting up bastion host firewall rule")
	g.createBastionHostFirewallRule()
	fmt.Println("")

	fmt.Println("(3/4) Creating bastion host")
	g.createBastionHostInstance()

	bastionNode := user + "@" + g.BastionIP
	node := user + "@" + nodeName
	fmt.Println("Waiting 45 seconds until ports are open.")
	time.Sleep(45 * time.Second)

	sshCmd := fmt.Sprintf("ssh -i key -o \"ProxyCommand ssh -W %%h:%%p -i key -o StrictHostKeyChecking=no " + bastionNode + "\" " + node + " -o StrictHostKeyChecking=no")
	fmt.Println(sshCmd)
	cmd := exec.Command("bash", "-c", sshCmd)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	checkError(err)

	fmt.Println("(4/4) Cleanup")
	g.cleanupGcpBastionHost()
}

// fetchAttributes gets all the needed attributes for creating bastion host and its security group with given <nodeName>.
func (g *GCPInstanceAttribute) fetchGCPAttributes(nodeName, path string) {
	var err error
	g.ShootName = getShootClusterName()
	g.BastionHostName = g.ShootName + "-bastions"
	g.BastionHostFirewallRuleName = g.ShootName + "-fw"
	g.Subnetwork = g.ShootName + "-nodes"
	g.Zone, err = fetchZone(g.ShootName, nodeName)
	checkError(err)
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
		fmt.Println(path)
		g.FirewallRuleName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.resources[] | select(.name == \"rule-allow-external-access\").instances[0].attributes.id'")
		checkError(err)
		g.VpcName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.outputs.vpc_name.value'")
		checkError(err)
	} else {
		g.FirewallRuleName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.modules[].resources[\"google_compute_firewall.rule-allow-external-access\"].primary[\"id\"]''")
		checkError(err)
		g.VpcName, err = ExecCmdReturnOutput("bash", "-c", "cat "+path+" | jq -r '.modules[].outputs.vpc_name.value'")
		checkError(err)
	}
	g.UserData = getBastionUserData(g.SSHPublicKey)
}

// createBastionHostFirewallRule finds the or creates a security group for the bastion host.
func (g *GCPInstanceAttribute) createBastionHostFirewallRule() {
	var err error
	fmt.Println("Add ssh rule")
	arguments := "gcloud " + fmt.Sprintf("compute firewall-rules update %s --allow tcp:22,tcp:80,tcp:443", g.FirewallRuleName)
	captured := capture()
	operate("gcp", arguments)
	capturedOutput, err := captured()
	checkError(err)
	fmt.Println(capturedOutput)
}

// createBastionHostInstance finds or creates a bastion host instance.
func (g *GCPInstanceAttribute) createBastionHostInstance() {
	fmt.Println("Create bastion host")
	tmpfile, err := ioutil.TempFile(os.TempDir(), "gardener-user.sh")
	checkError(err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(g.UserData)
	checkError(err)
	arguments := fmt.Sprintf("gcloud compute instances create %s --network %s --subnet %s --zone %s --metadata-from-file startup-script=%s --labels component=gardenctl", g.BastionHostName, g.VpcName, g.Subnetwork, g.Zone, tmpfile.Name())
	captured := capture()
	operate("gcp", arguments)
	capturedOutput, err := captured()
	checkError(err)
	fmt.Println(capturedOutput)
	arguments = fmt.Sprintf("gcloud compute disks add-labels %s --labels component=gardenctl --zone=%s", g.BastionHostName, g.Zone)
	operate("gcp", arguments)

	// check if bastion host is up and running, timeout after 3 minutes
	attemptCnt := 0
	for attemptCnt < 60 {
		arguments = fmt.Sprintf("gcloud compute instances describe %s --zone %s --flatten=[status]", g.BastionHostName, g.Zone)
		captured = capture()
		operate("gcp", arguments)
		capturedOutput, err = captured()
		capturedOutput = strings.Trim(capturedOutput, "-\n ")
		checkError(err)
		fmt.Println("Instance State: " + capturedOutput)
		if strings.Trim(capturedOutput, "\n") == "RUNNING" {
			arguments := fmt.Sprintf("gcloud compute instances describe %s --zone %s --flatten=networkInterfaces[0].accessConfigs[0].natIP", g.BastionHostName, g.Zone)
			captured := capture()
			operate("gcp", arguments)
			capturedOutput, err := captured()
			capturedOutput = strings.Trim(capturedOutput, "-\n ")
			words := strings.Fields(capturedOutput)
			checkError(err)
			ip := ""
			for _, value := range words {
				if isIP(value) && !strings.HasPrefix(value, "10.") {
					ip = value
					break
				}
			}
			g.BastionIP = ip
			return
		}
		time.Sleep(time.Second * 2)
		attemptCnt++
	}
	if attemptCnt == 90 {
		fmt.Println("Bastion server instance timeout. Please try again.")
		os.Exit(2)
	}

}

// getGCPMachineClasses returns the machine classes for shoot
func getGCPMachineClasses() *v1alpha1.GCPMachineClassList {
	tempTarget := Target{}
	ReadTarget(pathTarget, &tempTarget)
	shootName := tempTarget.Target[2].Name
	shootNamespace := getSeedNamespaceNameForShoot(shootName)

	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfigOfClusterType("seed"))
	checkError(err)
	mcmClient, err := mcmv1alpha1.NewForConfig(config)
	checkError(err)

	machineClasses, err := mcmClient.MachineV1alpha1().GCPMachineClasses(shootNamespace).List(metav1.ListOptions{})
	checkError(err)

	return machineClasses
}

// fetchZone returns the zone for instance with the given <nodeName>.
func fetchZone(shootName, nodeName string) (string, error) {
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

	machineClasses := getGCPMachineClasses()
	for _, machineClass := range machineClasses.Items {
		if machineClass.Name == machineClassName {
			return machineClass.Spec.Zone, nil
		}
	}

	return "", fmt.Errorf("Cannot find zone for node %q", nodeName)
}

// cleanupGcpBastionHost cleans up the bastion host for the targeted cluster.
func (g *GCPInstanceAttribute) cleanupGcpBastionHost() {
	fmt.Println("Cleaning up bastion host configurations...")
	fmt.Println("")
	fmt.Println("Starting cleanup")
	fmt.Println("")

	// clean up bastion host instance
	fmt.Println("  (1/2) Cleaning up bastion host instance")
	arguments := fmt.Sprintf("gcloud --quiet compute instances delete %s --zone %s", g.BastionHostName, g.Zone)
	captured := capture()
	operate("gcp", arguments)
	capturedOutput, err := captured()
	checkError(err)
	fmt.Println(capturedOutput)

	// remove shh port from firewall rule
	fmt.Println("  (2/2) Close SSH Port on Node.")
	fmt.Println("Close SSH Port on Node.")
	arguments = "gcloud " + fmt.Sprintf("compute firewall-rules update %s --allow tcp:80,tcp:443", g.FirewallRuleName)
	captured = capture()
	operate("gcp", arguments)
	capturedOutput, err = captured()
	checkError(err)
	fmt.Println(capturedOutput)
	fmt.Println("Bastion host configurations successfully cleaned up.")
}
