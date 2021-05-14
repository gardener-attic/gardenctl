// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	flagoutput string
)

// NewShowCmd returns a new show command.
func NewShowCmd(targetReader TargetReader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show (infra|operator|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|tf (infra|dns|ingress)|cluster-autoscaler)",
		Short: `Show details about endpoint/service and open in default browser if applicable`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return errors.New("Command must be in the format: show (infra|operator|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|tf (infra|dns|ingress)|cluster-autoscaler)")
			}
			t := targetReader.ReadTarget(pathTarget)
			if (len(t.Stack()) < 3 || (len(t.Stack()) == 3 && t.Stack()[2].Kind == "namespace")) && (args[0] != "operator") && (args[0] != "tf") && (args[0] != "kubernetes-dashboard") && (args[0] != "etcd-operator") {
				fmt.Println("No shoot targeted")
				os.Exit(2)
			} else if (len(t.Stack()) < 2 && (args[0] == "tf")) || len(t.Stack()) < 3 && (args[0] == "tf") && (t.Stack()[1].Kind != "seed") {
				fmt.Println("No seed or shoot targeted")
				os.Exit(2)
			} else if len(t.Stack()) == 0 {
				fmt.Println("Target stack is empty")
				os.Exit(2)
			}

			// Set up global map variable targetInfo and key validation check

			switch args[0] {
			case "infra":
				if flagoutput == "" {
					flagoutput = "json"
				}
				showCloudInfra(targetReader, flagoutput)
			case "operator":
				showOperator()
			case "gardener-dashboard":
				showGardenerDashboard()
			case "api":
				showAPIServer(targetReader)
			case "scheduler":
				showScheduler(targetReader)
			case "controller-manager":
				showControllerManager(targetReader)
			case "etcd-operator":
				showEtcdOperator()
			case "etcd-main":
				showEtcdMain(targetReader)
			case "etcd-events":
				showEtcdEvents(targetReader)
			case "addon-manager":
				showAddonManager(targetReader)
			case "vpn-seed":
				showVpnSeed(targetReader)
			case "vpn-shoot":
				showVpnShoot(targetReader)
			case "machine-controller-manager":
				showMachineControllerManager(targetReader)
			case "kubernetes-dashboard":
				showKubernetesDashboard(targetReader)
			case "prometheus":
				showPrometheus(targetReader)
			case "grafana":
				showGrafana(targetReader)
			case "tf":
				if len(args) == 1 {
					showTf()
					break
				}
				switch args[1] {
				case "infra":
					showInfra()
				case "dns":
					showDNS()
				case "ingress":
					showIngress()
				default:
					fmt.Println("Command must be in the format: show (infra|operator|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|tf (infra|dns|ingress)|cluster-autoscaler)")
				}
			case "cluster-autoscaler":
				showClusterAutoscaler(targetReader)
			default:
				fmt.Println("Command must be in the format: show (infra|operator|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|tf (infra|dns|ingress)|cluster-autoscaler)")
			}
			return nil
		},
		ValidArgs: []string{"operator", "gardener-dashboard", "api", "scheduler", "controller-manager", "etcd-operator", "etcd-main", "etcd-events", "addon-manager", "vpn-seed", "vpn-shoot", "machine-controller-manager", "kubernetes-dashboard", "prometheus", "grafana", "tf", "cluster-autoscaler"},
	}

	cmd.PersistentFlags().StringVarP(&flagoutput, "format", "f", "", "output format (default: json)")
	return cmd
}

// showPodGarden
func showPodGarden(podName string, namespace string) {
	var err error
	Client, err = clientToTarget("garden")
	checkError(err)
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, podName) {
			err := ExecCmd(nil, "kubectl get pods "+pod.Name+" -o wide -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

// showOperator shows the garden operator pod in the garden cluster
func showOperator() {
	showPodGarden("gardener-apiserver", "garden")
	showPodGarden("gardener-controller-manager", "garden")
}

// showUI opens the gardener landing page
func showGardenerDashboard() {
	showPodGarden("gardener-dashboard", "garden")
	output, err := ExecCmdReturnOutput("kubectl", "--kubeconfig="+KUBECONFIG, "get", "ingress", "gardener-dashboard-ingress", "-n", "garden")
	if err != nil {
		fmt.Println("Cmd was unsuccessful")
		os.Exit(2)
	}
	list := strings.Split(output, " ")
	url := "-"
	for _, val := range list {
		if strings.HasPrefix(val, "dashboard.") {
			url = val
		}
	}
	urls := strings.Split(url, ",")
	var filteredUrls []string
	match := false
	opened := false
	for index, url := range urls {
		for _, u := range filteredUrls {
			if url == u {
				match = true
			}
		}
		if !match {
			filteredUrls = append(filteredUrls, url)
			fmt.Println("URL-" + strconv.Itoa(index+1) + ": " + "https://" + url)
			if !opened {
				err := browser.OpenURL("https://" + url)
				checkError(err)
				opened = true
			}
		}
	}
}

// showPod is an abstraction to show pods in seed cluster controlplane or kube-system namespace of shoot
func showPod(toMatch string, toTarget TargetKind, targetReader TargetReader) {
	target := targetReader.ReadTarget(pathTarget)

	var namespace string
	if len(target.Stack()) == 2 {
		namespace = "garden"
	} else if len(target.Stack()) == 3 {
		namespace = getSeedNamespaceNameForShoot(target.Stack()[2].Name)
	}

	var err error
	Client, err = clientToTarget("seed")
	checkError(err)
	if toTarget == TargetKindShoot {
		namespace = "kube-system"
		Client, err = clientToTarget(TargetKindShoot)
		checkError(err)
	}
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) {
			err := ExecCmd(nil, "kubectl get pods "+pod.Name+" -o wide -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

// showCloudInfra shows the infra resources for the targeted shoot cluster
func showCloudInfra(targetReader TargetReader, output string) {
	target := targetReader.ReadTarget(pathTarget)
	shoot, err := FetchShootFromTarget(target)
	checkError(err)
	infraType := shoot.Spec.Provider.Type

	switch infraType {
	case "aws":
		showCloudInfraTypeAWS(targetReader, output)
	case "azure":
		showCloudInfraTypeAzure(targetReader, output)
	case "gcp":
		showCloudInfraTypeGCP(targetReader, output)
	case "openstack":
		showCloudInfraTypeOpenstack(targetReader, output)
	case "alicloud":
		showCloudInfraTypeAlicloud(targetReader)
	default:
		fmt.Println("infra type not found")
	}
}

// showCloudInfraTypeAWS shows the AWS infra resources for the targeted shoot cluster
func showCloudInfraTypeAWS(targetReader TargetReader, output string) {

	shoottag := GetFromTargetInfo(targetReader, "shootTechnicalID")

	capturedOutput := execInfraOperator("aws", "ec2 describe-instances --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-volumes --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-vpcs --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-subnets --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-route-tables --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-security-groups --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-internet-gateways --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-nat-gateways --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aws", "ec2 describe-addresses --filter Name=tag:kubernetes.io/cluster/"+shoottag+",Values=1 --output "+output)
	fmt.Println(capturedOutput)
}

// showCloudInfraTypeAzure shows the Azure infra resources for the targeted shoot cluster
func showCloudInfraTypeAzure(targetReader TargetReader, output string) {

	shoottag := GetFromTargetInfo(targetReader, "shootTechnicalID")

	capturedOutput := execInfraOperator("az", "vm list -d -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("az", "disk list -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("az", "network vnet list -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)

	vnets := make([]string, 0)
	vnets = findInfraResourcesMatch(`\"id\".*(virtualNetworks\/[a-z0-9-]*)\"`, capturedOutput, vnets)
	if len(vnets) > 0 {
		for _, vnet := range vnets {
			s := strings.Split(vnet, "/")
			vnetName := s[1]
			capturedOutput = execInfraOperator("az", "network vnet subnet list -g "+shoottag+" --vnet-name "+vnetName+" --output "+output)
			fmt.Println(capturedOutput)
		}
	}

	capturedOutput = execInfraOperator("az", "network route-table list -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("az", "network nsg list -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("az", "network lb list -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("az", "network nic list -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("az", "network public-ip list -g "+shoottag+" --output "+output)
	fmt.Println(capturedOutput)
}

// showCloudInfraTypeGCP shows the GCP infra resources for the targeted shoot cluster
func showCloudInfraTypeGCP(targetReader TargetReader, output string) {

	shoottag := GetFromTargetInfo(targetReader, "shootTechnicalID")

	capturedOutput := execInfraOperator("gcp", "compute instances list --filter=name~"+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("gcp", "compute disks list --filter=name~"+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("gcp", "compute networks list --filter=name="+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("gcp", "compute networks subnets list --filter=name~"+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("gcp", "compute routers list --filter=name~"+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("gcp", "compute routes list --filter=network="+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("gcp", "compute firewall-rules list --filter=network="+shoottag+" --format "+output)
	fmt.Println(capturedOutput)
}

// showCloudInfraTypeOpenstack shows the Openstack infra resources for the targeted shoot cluster
func showCloudInfraTypeOpenstack(targetReader TargetReader, output string) {

	shoottag := GetFromTargetInfo(targetReader, "shootTechnicalID")

	capturedOutput := execInfraOperator("openstack", "server list --name "+shoottag+".* --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("openstack", "volume list --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("openstack", "network list --name "+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("openstack", "subnet list --name "+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("openstack", "router list --name "+shoottag+" --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("openstack", "floating ip list --format "+output)
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("openstack", "security group list --format "+output)
	fmt.Println(capturedOutput)
}

// showCloudInfraTypeAlicloud shows the Alicloud infra resources for the targeted shoot cluster
func showCloudInfraTypeAlicloud(targetReader TargetReader) {

	shoottag := GetFromTargetInfo(targetReader, "shootTechnicalID")

	capturedOutput := execInfraOperator("aliyun", "ecs DescribeInstances --InstanceName "+shoottag+"*")
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aliyun", "ecs DescribeDisks")
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aliyun", "vpc DescribeVpcs --VpcName "+shoottag+"-vpc")
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aliyun", "ecs DescribeVSwitches")
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aliyun", "ecs DescribeVRouters")
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aliyun", "ecs DescribeRouteTables")
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aliyun", "ecs DescribeEipAddresses")
	fmt.Println(capturedOutput)

	capturedOutput = execInfraOperator("aliyun", "ecs DescribeSecurityGroups --SecurityGroupName "+shoottag+"-sg")
	fmt.Println(capturedOutput)
}

// showAPIServer shows the pod for the api-server running in the targeted seed cluster
func showAPIServer(targetReader TargetReader) {
	showPod("kube-apiserver", "seed", targetReader)
}

// showScheduler shows the pod for the running scheduler in the targeted seed cluster
func showScheduler(targetReader TargetReader) {
	showPod("kube-scheduler", "seed", targetReader)
}

// showControllerManager shows the pod for the running controller-manager in the targeted seed cluster
func showControllerManager(targetReader TargetReader) {
	showPod("kube-controller-manager", "seed", targetReader)
}

// showEtcdOperator shows the pod for the running etcd-operator in the targeted garden cluster
func showEtcdOperator() {
	showPodGarden("etcd-operator", "kube-system")
}

// showEtcdMain shows the pod for the running etcd-main in the targeted seed cluster
func showEtcdMain(targetReader TargetReader) {
	showPod("etcd-main", "seed", targetReader)
}

// showEtcdEvents shows the pod for the running etcd-events in the targeted seed cluster
func showEtcdEvents(targetReader TargetReader) {
	showPod("etcd-events", "seed", targetReader)
}

// showAddonManager shows the pod for the running addon-manager in the targeted seed cluster
func showAddonManager(targetReader TargetReader) {
	showPod("kube-addon-manager", "seed", targetReader)
}

// showVpnSeed shows the pod for the running vpn-seed in the targeted seed cluster
func showVpnSeed(targetReader TargetReader) {
	showPod("kube-apiserver", "seed", targetReader)
	showPod("prometheus-0", "seed", targetReader)
}

// showVpnShoot shows the pod for the running vpn-shoot in the targeted shoot cluster
func showVpnShoot(targetReader TargetReader) {
	showPod("vpn-shoot", "shoot", targetReader)
}

// showPrometheus shows the prometheus pod in the targeted seed cluster
func showPrometheus(targetReader TargetReader) {
	username, password = getMonitoringCredentials()
	showPod("prometheus", "seed", targetReader)
	KUBECONFIG := getKubeConfigOfClusterType("seed")
	url, err := ExecCmdReturnOutput("kubectl", "--kubeconfig="+KUBECONFIG, "get", "ingress", "prometheus", "-n", GetFromTargetInfo(targetReader, "shootTechnicalID"), "--no-headers", "-o", "custom-columns=:spec.rules[].host")
	if err != nil {
		log.Fatalf("Cmd was unsuccessful")
	}
	url = "https://" + username + ":" + password + "@" + url
	fmt.Println("URL: " + url)
	err = browser.OpenURL(url)
	checkError(err)
}

// showMachineControllerManager shows the prometheus pods in the targeted seed cluster
func showMachineControllerManager(targetReader TargetReader) {
	showPod("machine-controller-manager", "seed", targetReader)
}

// showKubernetesDashboard shows the kubernetes dashboard for the targeted cluster
func showKubernetesDashboard(targetReader TargetReader) {
	target := targetReader.ReadTarget(pathTarget)
	gardenName := target.Stack()[0].Name
	if len(target.Stack()) == 1 {
		var err error
		Client, err = clientToTarget("garden")
		checkError(err)
		pods, err := Client.CoreV1().Pods("kube-system").List(metav1.ListOptions{})
		checkError(err)
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "kubernetes-dashboard") {
				err := ExecCmd(nil, "kubectl get pods "+pod.Name+" -o wide -n kube-system", false, "KUBECONFIG="+KUBECONFIG)
				checkError(err)
			}
		}
	} else if len(target.Stack()) == 2 {
		namespace := "kube-system"
		if len(target.Stack()) == 2 && target.Stack()[1].Kind == "seed" {
			KUBECONFIG = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Stack()[1].Name, "kubeconfig.yaml")
		} else if len(target.Stack()) == 2 && target.Stack()[1].Kind == "project" {
			fmt.Println("Project targeted")
			os.Exit(2)
		}
		config, err := clientcmd.BuildConfigFromFlags("", KUBECONFIG)
		checkError(err)
		Client, err := kubernetes.NewForConfig(config)
		checkError(err)
		pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		checkError(err)
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "kubernetes-dashboard") {
				err := ExecCmd(nil, "kubectl get pods "+pod.Name+" -o wide -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
				checkError(err)
			}
		}
	} else if len(target.Stack()) == 3 {
		showPod("kubernetes-dashboard", "shoot", targetReader)
	} else if len(target.Stack()) == 0 {
		fmt.Println("No target")
		os.Exit(2)
	}
	url := "http://127.0.0.1:8002/api/v1/namespaces/kube-system/services/https:kubernetes-dashboard:/proxy/"
	err := browser.OpenURL(url)
	checkError(err)
	err = ExecCmd(nil, "kubectl proxy -p 8002", false, "KUBECONFIG="+KUBECONFIG)
	checkError(err)
}

// showGrafana shows the grafana dashboard for the targeted cluster
func showGrafana(targetReader TargetReader) {
	username, password = getMonitoringCredentials()
	showPod("grafana", "seed", targetReader)
	output, err := ExecCmdReturnOutput("kubectl", "--kubeconfig="+KUBECONFIG, "get", "ingress", "grafana-operators", "-n", GetFromTargetInfo(targetReader, "shootTechnicalID"))
	if err != nil {
		log.Fatalf("Cmd was unsuccessful")
	}
	list := strings.Split(output, " ")
	url := "-"
	for _, val := range list {
		if strings.HasPrefix(val, "go.") {
			formattedURL := strings.Split(val, ",")
			url = formattedURL[0]
		}
	}
	url = "https://" + username + ":" + password + "@" + url
	fmt.Println("URL: " + url)
	err = browser.OpenURL(url)
	checkError(err)
}

// showTerraform pods for specified name
func showTerraform(name string) {
	var err error
	Client, err = clientToTarget("seed")
	checkError(err)
	pods, err := Client.CoreV1().Pods("").List(metav1.ListOptions{})
	checkError(err)
	output := ""
	count := 0
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, name) && pod.Status.Phase == "Running" {
			output, err = ExecCmdReturnOutput("kubectl", "--kubeconfig="+KUBECONFIG, "get", "pods", pod.Name, "-o", "wide", "-n", pod.Namespace)
			if err != nil {
				fmt.Println("Cmd was unsuccessful")
				os.Exit(2)
			}
			if count != 0 {
				fmt.Printf("%s\n", strings.Split(output, "\n")[1])
			} else {
				fmt.Printf("%s", output)
				count++
			}
		}
	}
}

// showTf shows the currently running infra tf-pods
func showTf() {
	showTerraform(".tf-job")
}

// showInfra shows the currently running infra tf-pods
func showInfra() {
	showTerraform(".infra.tf-job")
}

// showDNS shows the currently running dns tf-pods
func showDNS() {
	showTerraform(".dns.tf-job")
}

// showIngress shows the currently running ingress tf-pods
func showIngress() {
	showTerraform(".ingress.tf-job")
}

// showClusterAutoscaler shows the pod for the running cluster-autoscaler in the targeted seed cluster
func showClusterAutoscaler(targetReader TargetReader) {
	showPod("cluster-autoscaler", "seed", targetReader)
}
