// Copyright 2018 The Gardener Authors.
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
	"strings"

	yaml "gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

// numberOfLines of logfile to return
var numberOfLines = "400"

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs (operator|ui|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-main-backup|etcd-events|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)",
	Short: "Show and optionally follow logs of given component\n",
	Long:  `s`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			fmt.Println("Command must be in the format: logs (operator|ui|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)")
			os.Exit(2)
		}
		switch args[0] {
		case "operator":
			logsOperator()
		case "ui":
			logsUI()
		case "api":
			logsAPIServer()
		case "scheduler":
			logsScheduler()
		case "controller-manager":
			logsControllerManager()
		case "etcd-operator":
			logsEtcdOpertor()
		case "etcd-main":
			logsEtcdMain()
		case "etcd-main-backup":
			logsEtcdMainBackup()
		case "etcd-events":
			logsEtcdEvents()
		case "addon-manager":
			logsAddonManager()
		case "vpn-seed":
			logsVpnSeed()
		case "vpn-shoot":
			logsVpnShoot()
		case "auto-node-repair":
			logsAutoNodeRepair()
		case "dashboard":
			logsDashboard()
		case "prometheus":
			logsPrometheus()
		case "grafana":
			logsGrafana()
		case "alertmanager":
			logsAlertmanager()
		case "tf":
			if len(args) == 1 {
				logsTf()
				break
			}
			switch args[1] {
			case "infra":
				logsInfra()
			case "dns":
				logsDNS()
			case "ingress":
				logsIngress()
			default:
				fmt.Println("Command must be in the format: logs (operator|ui|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)")
			}
		default:
			fmt.Println("Command must be in the format: logs (operator|ui|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)")
		}
	},
	ValidArgs: []string{"operator", "ui", "api", "scheduler", "controller-manager", "etcd-operator", "etcd-main", "etcd-events", "addon-manager", "vpn-seed", "vpn-shoot", "auto-node-repair", "dashboard", "prometheus", "grafana", "alertmanager", "tf"},
}

func init() {
}

// showPod is an abstraction to show pods in seed cluster controlplane or kube-system namespace of shoot
func logPod(toMatch string, toTarget string, container string) {
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	if len(target.Target) < 3 {
		fmt.Println("No shoot targeted")
		os.Exit(2)
	}
	namespace := getSeedNamespaceNameForShoot(target.Target[2].Name)
	Client, err = clientToTarget("seed")
	checkError(err)
	if toTarget == "shoot" {
		namespace = "kube-system"
		Client, err = clientToTarget(toTarget)
		checkError(err)
	}
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) {
			err := execCmd("kubectl logs --tail="+numberOfLines+" "+pod.Name+container+" -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

// logPodGarden print logfiles for garden pods
func logPodGarden(toMatch string) {
	Client, err = clientToTarget("garden")
	checkError(err)
	pods, err := Client.CoreV1().Pods("garden").List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		fmt.Println(pod.Name)
		if strings.Contains(pod.Name, toMatch) {
			err := execCmd("kubectl logs --tail="+numberOfLines+" "+pod.Name+" -n garden", false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
			break
		}
	}
}

// logsOperator prints the logfile of the operator
func logsOperator() {
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	if len(target.Target) != 3 {
		logPodGarden("garden-operator")
	} else {
		logPodGardenImproved()
	}
}

// logPodGardenImproved print logfiles for garden pods
func logPodGardenImproved() {
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	Client, err := clientToTarget("garden")
	checkError(err)
	pods, err := Client.CoreV1().Pods("garden").List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "garden-operator") {
			output := execCmdReturnOutput("kubectl logs "+pod.Name+" -n garden", "KUBECONFIG="+KUBECONFIG)
			lines := strings.Split("time="+output, `time=`)
			for _, line := range lines {
				if strings.Contains(line, ("shoot=" + target.Target[2].Name)) {
					fmt.Printf(line)
				}
			}
		}
	}
}

// logsUI
func logsUI() {
	logPodGarden("gardener")
}

// logsAPIServer prints the logfile of the api-server
func logsAPIServer() {
	logPod("kube-apiserver", "seed", " -c kube-apiserver")
}

// logsScheduler prints the logfile of the scheduler
func logsScheduler() {
	logPod("kube-scheduler", "seed", "")
}

// logsAPIServer prints the logfile of the controller-manager
func logsControllerManager() {
	logPod("kube-controller-manager", "seed", "")
}

// logsVpnSeed prints the logfile of the vpn-seed container
func logsVpnSeed() {
	fmt.Println("-----------------------Kube-Apiserver")
	logPod("kube-apiserver", "seed", " -c vpn-seed")
	fmt.Println("-----------------------Prometheus")
	logPod("prometheus", "seed", " -c vpn-seed")
}

// logsEtcdOpertor prints the logfile of the etcd-operator
func logsEtcdOpertor() {
	logPod("etcd-operator", "seed", "")
}

// logsEtcdMain prints the logfile of etcd-main
func logsEtcdMain() {
	logPod("etcd-main", "seed", "")
}

// logsEtcdMainBackup prints logfiles of etcd-main-backup-sidecar pod
func logsEtcdMainBackup() {
	logPod("etcd-main-backup-sidecar", "seed", "")
}

// logsEtcdEvents prints the logfile of etcd-events
func logsEtcdEvents() {
	logPod("etcd-events-", "seed", "")
}

// logsAddonManager prints the logfile of addon-manager
func logsAddonManager() {
	logPod("addon-manager", "seed", "")
}

// logsVpnShoot prints the logfile of vpn-shoot
func logsVpnShoot() {
	logPod("vpn-shoot", "shoot", "")
}

// logsAutoNodeRepair prints the logfile of auto-node-repair pod
func logsAutoNodeRepair() {
	logPod("auto-node-repair", "seed", "")
}

// logsDashboard prints the logfile of the dashboard
func logsDashboard() {
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	namespace := "kube-system"
	if len(target.Target) == 3 {
		Client, err = clientToTarget("shoot")
		checkError(err)
	} else if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
		KUBECONFIG = pathGardenHome + "/cache/seeds" + "/" + target.Target[1].Name + "/" + "kubeconfig.yaml"
		config, err := clientcmd.BuildConfigFromFlags("", KUBECONFIG)
		checkError(err)
		Client, err = kubernetes.NewForConfig(config)
		checkError(err)
	} else if len(target.Target) == 2 && target.Target[1].Kind == "project" {
		fmt.Println("Project targeted")
		os.Exit(2)
	} else if len(target.Target) == 1 {
		Client, err = clientToTarget("garden")
		checkError(err)
	} else {
		fmt.Println("No target")
		os.Exit(2)
	}
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "kubernetes-dashboard") {
			err := execCmd("kubectl logs --tail="+numberOfLines+" "+pod.Name+" -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

// logsPrometheus prints the logfiles of prometheus pod
func logsPrometheus() {
	logPod("prometheus", "seed", " -c prometheus")
}

// logsGrafana prints the logfiles of grafana pod
func logsGrafana() {
	logPod("grafana", "seed", " -c grafana")
}

// logsAlertmanager prints the logfiles of alertmanager
func logsAlertmanager() {
	logPod("alertmanager", "seed", " -c alertmanager") // TODO: TWO PODS ARE RUNNING
}

// logsTerraform prints the logfiles of tf pod
func logsTerraform(toMatch string) {
	var latestTime int64
	var podName string
	var podNamespace string
	Client, err = clientToTarget("garden")
	checkError(err)
	pods, err := Client.CoreV1().Pods("").List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) && pod.Status.Phase == "Running" {
			if latestTime < pod.ObjectMeta.CreationTimestamp.Unix() {
				latestTime = pod.ObjectMeta.CreationTimestamp.Unix()
				podName = pod.Name
				podNamespace = pod.Namespace
			}
		}
	}
	if podName == "" || podNamespace == "" {
		fmt.Println("No running tf " + toMatch)
	} else {
		fmt.Println("gardenctl logs " + podName + " namespace=" + podNamespace)
		err = execCmd("kubectl logs "+podName+" -n "+podNamespace, false, "KUBECONFIG="+KUBECONFIG)
		checkError(err)
	}
}

// logsTf prints the logfiles of tf job
func logsTf() {
	logsTerraform("tf-job")
}

// logsInfra prints the logfiles of tf infra job
func logsInfra() {
	logsTerraform("infra.tf-job")
}

// logsDNS prints the logfiles of tf dns job
func logsDNS() {
	logsTerraform("dns.tf-job")
}

// logsIngress prints the logfiles of tf ingress job
func logsIngress() {
	logsTerraform("ingress.tf-job")
}
