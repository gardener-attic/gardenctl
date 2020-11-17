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
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	maxLokiLogs           = 100000
	fourteenDaysInSeconds = 60 * 60 * 24 * 14
	emptyString           = ""
)

//flags passed to the command
var flags *logFlags

// NewLogsCmd returns a new logs command.
func NewLogsCmd(targetReader TargetReader) *cobra.Command {
	flags = newLogsFlags()
	cmd := &cobra.Command{
		Use:          "logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-main-backup|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|gardenlet|tf (infra|dns|ingress)|cluster-autoscaler)",
		Short:        "Show and optionally follow logs of given component\n",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := validateArgs(targetReader, args)
			if err != nil {
				return err
			}
			validateFlags(flags)
			runCommand(targetReader, args)
			return nil
		},
		ValidArgs: []string{"gardener-apiserver", "gardener-controller-manager", "gardener-dashboard", "api", "scheduler", "controller-manager", "etcd-operator", "etcd-main", "etcd-events", "addon-manager", "vpn-seed", "vpn-shoot", "auto-node-repair", "kubernetes-dashboard", "prometheus", "grafana", "gardenlet", "tf"},
		Aliases:   []string{"log"},
	}
	cmd.Flags().Int64Var(&flags.tail, "tail", 200, "Lines of recent log file to display. Defaults to 200 with no selector, if a selector is provided takes the number of specified lines (max 100 000 for loki).")
	cmd.Flags().DurationVar(&flags.sinceSeconds, "since", flags.sinceSeconds, "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	cmd.Flags().StringVar(&flags.sinceTime, "since-time", flags.sinceTime, "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	cmd.Flags().BoolVar(&flags.loki, "loki", flags.loki, "If the flag is set the logs are retrieved and shown from Loki, otherwise from the kubelet.")

	return cmd
}

func validateArgs(targetReader TargetReader, args []string) error {
	if len(args) < 1 || len(args) > 3 {
		return errors.New("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|gardenlet|tf (infra|dns|ingress)|cluster-autoscaler flags(--loki|--tail|--since|--since-time|--timestamps)")
	}
	var t Target
	ReadTarget(pathTarget, &t)
	if !IsTargeted(targetReader, "shoot") && args[0] != "gardener-apiserver" && args[0] != "gardener-controller-manager" && args[0] != "tf" && args[0] != "kubernetes-dashboard" {
		return errors.New("No shoot targeted")
	} else if !IsTargeted(targetReader, "project") && args[0] == "tf" || !IsTargeted(targetReader, "shoot") && args[0] == "tf" && !IsTargeted(targetReader, "seed") {
		return errors.New("No seed or shoot targeted")
	} else if !IsTargeted(targetReader) {
		return errors.New("Target stack is empty")
	}
	return nil
}

func validateFlags(flags *logFlags) {
	if flags.sinceSeconds != 0 && flags.sinceTime != emptyString {
		fmt.Println("Logs command can not contains --since and --since-time in the same time")
		os.Exit(2)
	} else if flags.sinceTime != emptyString {
		value, err := time.Parse(time.RFC3339, flags.sinceTime)
		if err != nil {
			fmt.Println("Incorrect value for flag: --since-time")
			os.Exit(2)
		} else {
			flags.sinceSeconds = time.Since(value)
		}
	} else if flags.tail < 0 {
		fmt.Println("Incorrect value for flag: --tail, value must be greater 0")
		os.Exit(2)
	} else if flags.loki && flags.tail > maxLokiLogs {
		fmt.Printf("Maximum number of logs that can be fetched from loki is %d", maxLokiLogs)
		os.Exit(2)
	}
}

func runCommand(targetReader TargetReader, args []string) {
	switch args[0] {
	case "all":
		saveLogsAll(targetReader)
	case "gardener-apiserver":
		logsGardenerApiserver()
	case "gardener-controller-manager":
		logsGardenerControllerManager(targetReader)
	case "gardener-dashboard":
		logsGardenerDashboard()
	case "api":
		logsAPIServer(targetReader)
	case "scheduler":
		logsScheduler(targetReader)
	case "controller-manager":
		logsControllerManager(targetReader)
	case "etcd-operator":
		logsEtcdOpertor()
	case "etcd-main":
		if len(args) == 2 {
			logsEtcdMain(targetReader, args[1])
		} else {
			logsEtcdMain(targetReader, emptyString)
		}
	case "etcd-main-backup":
		logsEtcdMainBackup(targetReader)
	case "etcd-events":
		if len(args) == 2 {
			logsEtcdEvents(targetReader, args[1])
		} else {
			logsEtcdEvents(targetReader, emptyString)
		}
	case "addon-manager":
		logsAddonManager(targetReader)
	case "vpn-seed":
		if len(args) == 2 {
			logsVpnSeed(targetReader, args[1])
		} else {
			logsVpnSeed(targetReader, emptyString)
		}
	case "vpn-shoot":
		logsVpnShoot()
	case "machine-controller-manager":
		logsMachineControllerManager(targetReader)
	case "kubernetes-dashboard":
		logsKubernetesDashboard(targetReader)
	case "prometheus":
		logsPrometheus(targetReader)
	case "grafana":
		logsGrafana(targetReader)
	case "gardenlet":
		logsGardenlet()
	case "cluster-autoscaler":
		logsClusterAutoscaler(targetReader)
	case "tf":
		if len(args) == 1 || len(args) < 3 {
			logsTfHelp()
			break
		}

		var prefixName string = (args[02])
		switch args[1] {
		case "infra":
			str := prefixName + ".infra.tf"
			logsInfra(str)
		case "dns":
			str := prefixName + ".dns.tf"
			logsDNS(str)
		case "ingress":
			str := prefixName + ".ingress.tf"
			logsIngress(str)
		default:
			fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|kubernetes-dashboard|prometheus|grafana|tf (infra|dns|ingress)|cluster-autoscaler)")
		}
	default:
		fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|kubernetes-dashboard|prometheus|grafana|tf (infra|dns|ingress)|cluster-autoscaler)")
	}
}

func saveLogsAll(targetReader TargetReader) {
	//flags.sinceSeconds = 600 * time.Second
	if _, err := os.Stat("./logs/"); !os.IsNotExist(err) {
		os.RemoveAll("./logs/")
	}
	err := os.MkdirAll("./logs/", os.ModePerm)
	checkError(err)

	fmt.Println("APIServer/Scheduler/ControllerManager/etcd/AddonManager/VpnShoot/Dashboard/Prometheus/Gardenlet/Autoscaler logs will be downloaded")

	saveLogsAPIServer(targetReader)
	saveLogsScheduler(targetReader)
	saveLogsControllerManager(targetReader)
	saveLogsEtcdMain(targetReader, "etcd")
	saveLogsEtcdMain(targetReader, "backup-restore")
	saveLogsEtcdMainBackup(targetReader)
	saveLogsEtcdEvents(targetReader, "etcd")
	saveLogsEtcdEvents(targetReader, "backup-restore")
	saveLogsAddonManager(targetReader)
	saveLogsVpnShoot()
	saveLogsMachineControllerManager(targetReader)
	saveLogsKubernetesDashboard()
	saveLogsPrometheus(targetReader)
	saveLogsGardenlet()
	saveLogsClusterAutoscaler(targetReader)
	saveLogsGrafana(targetReader)

	var target Target
	ReadTarget(pathTarget, &target)
	if !(len(target.Target) < 3 || (len(target.Stack()) == 3 && target.Stack()[2].Kind == "namespace")) {
		shoot, err := GetTargetedShootObject(targetReader)
		checkError(err)
		saveLogsTerraform(shoot.Name + ".infra.tf")
		saveLogsTerraform(shoot.Name + ".dns.tf")
		saveLogsTerraform(shoot.Name + ".ingress.tf")

	}

	path, err := os.Getwd()
	checkError(err)

	fmt.Println("All logs have been saved in " + path + "/logs/ folder")
}

// showPod is an abstraction to show pods in seed cluster controlplane or kube-system namespace of shoot
func logPod(targetReader TargetReader, toMatch string, toTarget string, container string) {
	var target Target
	ReadTarget(pathTarget, &target)
	if !IsTargeted(targetReader, "shoot") {
		log.Fatal("No shoot targeted")
	}

	namespace := GetFromTargetInfo(targetReader, "shootTechnicalID")
	var err error
	shoot, err := GetTargetedShootObject(targetReader)
	checkError(err)

	gardenerVersion, err := semver.NewVersion(shoot.Status.Gardener.Version)
	checkError(err)
	greaterThanLokiRelease, err := semver.NewConstraint(">=1.8.0")
	checkError(err)

	Client, err = clientToTarget("seed")
	checkError(err)
	if toTarget == "shoot" {
		namespace = "kube-system"
		Client, err = clientToTarget(TargetKindShoot)
		checkError(err)
	}

	if flags.loki {
		if greaterThanLokiRelease.Check(gardenerVersion) {
			showLogsFromLoki(namespace, toMatch, container)
		} else {
			fmt.Println("--loki flag is available only for gardener version >= 1.8.0")
			fmt.Println("Current version: " + gardenerVersion.String())
			os.Exit(2)
		}

	} else {
		showLogsFromKubectl(namespace, toMatch, container)
	}
}

// showPod is an abstraction to show pods in seed cluster controlplane or kube-system namespace of shoot
func saveLogPod(targetReader TargetReader, toMatch string, toTarget string, container string) {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) < 3 || (len(target.Stack()) == 3 && target.Stack()[2].Kind == "namespace") {
		fmt.Println("No shoot targeted")
		os.Exit(2)
	}
	namespace := getSeedNamespaceNameForShoot(target.Target[2].Name)
	var err error
	shoot, err := GetTargetedShootObject(targetReader)
	checkError(err)

	gardenerVersion, err := semver.NewVersion(shoot.Status.Gardener.Version)
	checkError(err)
	greaterThanLokiRelease, err := semver.NewConstraint(">=1.8.0")
	checkError(err)

	Client, err = clientToTarget("seed")
	checkError(err)
	if toTarget == "shoot" {
		namespace = "kube-system"
		Client, err = clientToTarget(TargetKindShoot)
		checkError(err)
	}

	if flags.loki {
		if greaterThanLokiRelease.Check(gardenerVersion) {
			saveLogsFromLoki(namespace, toMatch, container)
		} else {
			fmt.Println("--loki flag is available only for gardener version >= 1.8.0")
			fmt.Println("Current version: " + gardenerVersion.String())
			os.Exit(2)
		}

	} else {
		saveLogsFromKubectl(namespace, toMatch, container)
	}
}

func saveLogsFromLoki(namespace, toMatch, container string) {
	args := BuildLokiCommandArgs(KUBECONFIG, namespace, toMatch, container, flags.tail, flags.sinceSeconds)
	cmdResult := "kubectl " + strings.Join(args, " ")
	output, err := ExecCmdReturnOutput("bash", "-c", cmdResult)
	checkError(err)

	byteOutput := []byte(output)
	var response logResponseLoki
	err = json.Unmarshal(byteOutput, &response)
	checkError(err)
	fmt.Println("the response is")
	fmt.Println(response)

	fileName := "./logs/"
	fileName += namespace + "_" + toMatch
	if container != emptyString {
		fileName = fileName + "_" + container
	}
	fileName = fileName + ".log"

	f, err := os.Create(fileName)
	checkError(err)
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("%v", response))
	checkError(err)
	err = f.Sync()
	checkError(err)
}

func showLogsFromKubectl(namespace, toMatch, container string) {
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) {
			output, err := ExecCmdReturnOutput("kubectl", BuildLogCommandArgs(KUBECONFIG, namespace, pod.Name, container, flags.tail, flags.sinceSeconds)...)
			checkError(err)
			fmt.Println(output)
		}
	}
}

func saveLogsFromKubectl(namespace, toMatch, container string) {
	fileName := "./logs/"
	fileName += namespace + "_" + toMatch
	if container != emptyString {
		fileName = fileName + "_" + container
	}
	fileName = fileName + ".log"
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) {
			args := BuildLogCommandArgs(KUBECONFIG, namespace, pod.Name, container, flags.tail, flags.sinceSeconds)
			cmdResult := "kubectl " + strings.Join(args, " ")
			err := ExecCmdSaveOutputFile(nil, cmdResult, fileName)
			checkError(err)
		}
	}
}

func showLogsFromLoki(namespace, toMatch, container string) {
	output, err := ExecCmdReturnOutput("kubectl", BuildLokiCommandArgs(KUBECONFIG, namespace, toMatch, container, flags.tail, flags.sinceSeconds)...)
	checkError(err)

	byteOutput := []byte(output)
	var response logResponseLoki
	err = json.Unmarshal(byteOutput, &response)
	checkError(err)

	fmt.Println(response)
}

//BuildLogCommandArgs build kubectl command to get logs
func BuildLogCommandArgs(kubeconfig string, namespace, podName, container string, tail int64, sinceSeconds time.Duration) []string {
	args := []string{
		"logs",
		"--kubeconfig=" + kubeconfig,
		podName,
	}

	if container != emptyString {
		args = append(args, []string{"-c", container}...)
	}

	args = append(args, []string{"-n", namespace}...)

	if tail != -1 {
		args = append(args, fmt.Sprintf("--tail=%d", tail))
	}
	if sinceSeconds != 0 {
		args = append(args, fmt.Sprintf("--since=%vs", sinceSeconds.Seconds()))
	}

	return args
}

//BuildLokiCommandArgs build kubect command to get logs from loki
func BuildLokiCommandArgs(kubeconfig string, namespace, podName, container string, tail int64, sinceSeconds time.Duration) []string {
	args := []string{
		"--kubeconfig=" + kubeconfig,
		"exec",
		"loki-0",
		"-n",
		namespace,
		"--",
		"wget",
		"--header",
		"'X-Scope-OrgID: operator'",
		"http://localhost:3100/loki/api/v1/query_range",
		"-O-",
	}

	lokiQuery := fmt.Sprintf("{pod_name=~\"%s.*\"}", podName)
	command := fmt.Sprintf("--post-data='query=%s", lokiQuery)

	if container != emptyString {
		command += fmt.Sprintf("&&query={container_name=~\"%s.*\"", container)
	}
	if tail != 0 {
		command += fmt.Sprintf("&&limit=%d", tail)
	}
	if sinceSeconds == 0 {
		sinceSeconds = fourteenDaysInSeconds * time.Second
	}
	sinceNanoSec := sinceSeconds.Nanoseconds()
	now := time.Now().UnixNano()

	command += fmt.Sprintf("&&start=%d&&end=%d", now-sinceNanoSec, now)
	command += "'"

	args = append(args, command)
	fmt.Println("")
	fmt.Println(strings.Join(args, " "))
	return args
}

// logPodGarden print logfiles for garden pods
func logPodGarden(toMatch, namespace string) {
	var err error
	Client, err = clientToTarget("garden")
	checkError(err)
	showLogsFromKubectl(namespace, toMatch, emptyString)
}

// logPodSeed print logfiles for Seed pods
func logPodSeed(toMatch, namespace string, container string) {
	var err error
	Client, err = clientToTarget(TargetKindSeed)
	checkError(err)
	if container != emptyString {
		showLogsFromKubectl(namespace, toMatch, container)
	} else {
		showLogsFromKubectl(namespace, toMatch, emptyString)
	}
}

func saveLogPodSeed(toMatch, namespace string, container string) {
	var err error
	Client, err = clientToTarget(TargetKindSeed)
	checkError(err)
	if container != emptyString {
		saveLogsFromKubectl(namespace, toMatch, container)
	} else {
		saveLogsFromKubectl(namespace, toMatch, emptyString)
	}
}

// logPodShoot print logfiles for shoot pods
func logPodShoot(toMatch, namespace string, container string) {
	var err error
	Client, err = clientToTarget(TargetKindShoot)
	checkError(err)
	if container != emptyString {
		showLogsFromKubectl(namespace, toMatch, container)
	} else {
		showLogsFromKubectl(namespace, toMatch, emptyString)
	}
}

func saveLogPodShoot(toMatch, namespace string, container string) {
	var err error
	Client, err = clientToTarget(TargetKindShoot)
	checkError(err)
	if container != emptyString {
		container = " -c " + container
		saveLogsFromKubectl(namespace, toMatch, container)
	} else {
		saveLogsFromKubectl(namespace, toMatch, emptyString)
	}
}

// logPodGardenImproved print logfiles for garden pods
func logPodGardenImproved(targetReader TargetReader, podName string) {
	var target Target
	ReadTarget(pathTarget, &target)
	Client, err := clientToTarget("garden")
	checkError(err)
	pods, err := Client.CoreV1().Pods("garden").List(metav1.ListOptions{})
	checkError(err)
	project, err := GetTargetName(targetReader, "project")
	checkError(err)
	shootName, err := GetTargetName(targetReader, "shoot")
	checkError(err)

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, podName) {
			output, err := ExecCmdReturnOutput("kubectl", BuildLogCommandArgs(KUBECONFIG, "garden", pod.Name, emptyString, flags.tail, flags.sinceSeconds)...)
			if err != nil {
				fmt.Println("Cmd was unsuccessful")
				os.Exit(2)
			}
			lines := strings.Split("time="+output, `time=`)
			for _, line := range lines {
				if strings.Contains(line, ("shoot=" + project + "/" + shootName)) {
					fmt.Print(line)
				}
			}
		}
	}
}

// logsGardenerApiserver prints the logfile of the garndener-api-server
func logsGardenerApiserver() {
	var target Target
	ReadTarget(pathTarget, &target)
	logPodGarden("gardener-apiserver", "garden")
}

// logsGardenerControllerManager prints the logfile of the gardener-controller-manager
func logsGardenerControllerManager(targetReader TargetReader) {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) != 3 {
		logPodGarden("gardener-controller-manager", "garden")
	} else {
		logPodGardenImproved(targetReader, "gardener-controller-manager")
	}
}

// logsGardenerDashboard
func logsGardenerDashboard() {
	logPodGarden("gardener", "garden")
}

// logsAPIServer prints the logfile of the api-server
func logsAPIServer(targetReader TargetReader) {
	logPod(targetReader, "kube-apiserver", "seed", "kube-apiserver")
}

func saveLogsAPIServer(targetReader TargetReader) {
	saveLogPod(targetReader, "kube-apiserver", "seed", "kube-apiserver")
}

// logsScheduler prints the logfile of the scheduler
func logsScheduler(targetReader TargetReader) {
	logPod(targetReader, "kube-scheduler", "seed", emptyString)
}

func saveLogsScheduler(targetReader TargetReader) {
	saveLogPod(targetReader, "kube-scheduler", "seed", emptyString)
}

// logsAPIServer prints the logfile of the controller-manager
func logsControllerManager(targetReader TargetReader) {
	logPod(targetReader, "kube-controller-manager", "seed", emptyString)
}

func saveLogsControllerManager(targetReader TargetReader) {
	saveLogPod(targetReader, "kube-controller-manager", "seed", emptyString)
}

// logsVpnSeed prints the logfile of the vpn-seed container
func logsVpnSeed(targetReader TargetReader, shootTechnicalID string) {
	fmt.Println("-----------------------Kube-Apiserver")
	if shootTechnicalID == emptyString {
		shootTechnicalID = GetFromTargetInfo(targetReader, "shootTechnicalID")
		logPodSeed("kube-apiserver", shootTechnicalID, "vpn-seed")
	} else {
		logPodSeed("kube-apiserver", shootTechnicalID, "vpn-seed")
	}
}

// logsEtcdOpertor prints the logfile of the etcd-operator
func logsEtcdOpertor() {
	logPodGarden("etcd-operator", "kube-system")
}

// logsEtcdMain prints the logfile of etcd-main
func logsEtcdMain(targetReader TargetReader, containerName string) {
	logPod(targetReader, "etcd-main", "seed", containerName)
}

func saveLogsEtcdMain(targetReader TargetReader, containerName string) {
	saveLogPod(targetReader, "etcd-main", "seed", containerName)
}

// logsEtcdMainBackup prints logfiles of etcd-main-backup-sidecar pod
func logsEtcdMainBackup(targetReader TargetReader) {
	logPod(targetReader, "etcd-main-backup-sidecar", "seed", emptyString)
}

func saveLogsEtcdMainBackup(targetReader TargetReader) {
	saveLogPod(targetReader, "etcd-main-backup-sidecar", "seed", emptyString)
}

// logsEtcdEvents prints the logfile of etcd-events
func logsEtcdEvents(targetReader TargetReader, containerName string) {
	logPod(targetReader, "etcd-events-", "seed", containerName)
}

func saveLogsEtcdEvents(targetReader TargetReader, containerName string) {
	saveLogPod(targetReader, "etcd-events-", "seed", containerName)
}

// logsAddonManager prints the logfile of addon-manager
func logsAddonManager(targetReader TargetReader) {
	logPod(targetReader, "addon-manager", "seed", emptyString)
}

func saveLogsAddonManager(targetReader TargetReader) {
	saveLogPod(targetReader, "addon-manager", "seed", emptyString)
}

// logsVpnShoot prints the logfile of vpn-shoot
func logsVpnShoot() {
	fmt.Println("-----------------------vpn-shoot")
	logPodShoot("vpn-shoot", "kube-system", emptyString)
}

func saveLogsVpnShoot() {
	fmt.Println("-----------------------vpn-shoot")
	saveLogPodShoot("vpn-shoot", "kube-system", emptyString)
}

// logsMachineControllerManager prints the logfile of machine-controller-manager
func logsMachineControllerManager(targetReader TargetReader) {
	logPod(targetReader, "machine-controller-manager", "seed", emptyString)
}

func saveLogsMachineControllerManager(targetReader TargetReader) {
	saveLogPod(targetReader, "machine-controller-manager", "seed", emptyString)
}

// logsKubernetesDashboard prints the logfile of the dashboard
func logsKubernetesDashboard(targetReader TargetReader) {
	var target Target
	ReadTarget(pathTarget, &target)
	namespace := "kube-system"
	if IsTargeted(targetReader, "shoot") {
		var err error
		Client, err = clientToTarget("shoot")
		checkError(err)
	} else if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
		gardenName := target.Stack()[0].Name
		KUBECONFIG = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, "kubeconfig.yaml")
		config, err := clientcmd.BuildConfigFromFlags(emptyString, KUBECONFIG)
		checkError(err)
		Client, err = kubernetes.NewForConfig(config)
		checkError(err)
	} else if len(target.Target) == 2 && target.Target[1].Kind == "project" {
		fmt.Println("Project targeted")
		os.Exit(2)
	} else if len(target.Target) == 1 {
		var err error
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
			err := ExecCmd(nil, "kubectl logs --tail="+strconv.Itoa(int(flags.tail))+" "+pod.Name+" -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

func saveLogsKubernetesDashboard() {
	var target Target
	ReadTarget(pathTarget, &target)
	namespace := "kube-system"
	if len(target.Target) == 3 {
		var err error
		Client, err = clientToTarget("shoot")
		checkError(err)
	} else if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
		gardenName := target.Stack()[0].Name
		KUBECONFIG = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, "kubeconfig.yaml")
		config, err := clientcmd.BuildConfigFromFlags(emptyString, KUBECONFIG)
		checkError(err)
		Client, err = kubernetes.NewForConfig(config)
		checkError(err)
	} else if len(target.Target) == 2 && target.Target[1].Kind == "project" {
		fmt.Println("Project targeted")
		os.Exit(2)
	} else if len(target.Target) == 1 {
		var err error
		Client, err = clientToTarget("garden")
		checkError(err)
	} else {
		fmt.Println("No target")
		os.Exit(2)
	}
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	p, err := os.Getwd()
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "kubernetes-dashboard") {
			fileName := path.Join(p, "logs", pod.Name)
			err := ExecCmdSaveOutputFile(nil, "kubectl logs --tail="+strconv.Itoa(int(flags.tail))+" "+pod.Name+" -n "+namespace, fileName, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

// logsPrometheus prints the logfiles of prometheus pod
func logsPrometheus(targetReader TargetReader) {
	logPod(targetReader, "prometheus", "seed", "prometheus")
}

func saveLogsPrometheus(targetReader TargetReader) {
	saveLogPod(targetReader, "prometheus", "seed", "prometheus")
}

// logsGrafana prints the logfiles of grafana pod
func logsGrafana(targetReader TargetReader) {
	logPod(targetReader, "grafana", "seed", "grafana")
}

func saveLogsGrafana(targetReader TargetReader) {
	saveLogPod(targetReader, "grafana", "seed", "grafana")
}

func logsGardenlet() {
	logPodSeed("gardenlet", "garden", emptyString)
}

func saveLogsGardenlet() {
	saveLogPodSeed("gardenlet", "garden", emptyString)
}

// logsClusterAutoscaler prints the logfiles of cluster-autoscaler
func logsClusterAutoscaler(targetReader TargetReader) {
	logPod(targetReader, "cluster-autoscaler", "seed", "cluster-autoscaler")
}

func saveLogsClusterAutoscaler(targetReader TargetReader) {
	saveLogPod(targetReader, "cluster-autoscaler", "seed", "cluster-autoscaler")
}

// logsTerraform prints the logfiles of tf pod
func logsTerraform(toMatch string) {
	var latestTime int64
	var podName [100]string
	var podNamespace [100]string
	var err error
	Client, err = clientToTarget("seed")
	checkError(err)
	pods, err := Client.CoreV1().Pods(emptyString).List(metav1.ListOptions{})
	checkError(err)
	count := 0
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) && pod.Status.Phase == "Running" {
			if latestTime < pod.ObjectMeta.CreationTimestamp.Unix() {
				latestTime = pod.ObjectMeta.CreationTimestamp.Unix()
				podName[count] = pod.Name
				podNamespace[count] = pod.Namespace
				count++
			}
		}
	}
	if len(podName) == 0 || len(podNamespace) == 0 {
		fmt.Println("No running tf " + toMatch)
	} else {
		for i := 0; i < count; i++ {
			fmt.Println("gardenctl logs " + podName[i] + " namespace=" + podNamespace[i])
			err = ExecCmd(nil, "kubectl logs "+podName[i]+" -n "+podNamespace[i], false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

func saveLogsTerraform(toMatch string) {
	var latestTime int64
	var podName [100]string
	var podNamespace [100]string
	var err error
	Client, err = clientToTarget("seed")
	checkError(err)
	pods, err := Client.CoreV1().Pods(emptyString).List(metav1.ListOptions{})
	checkError(err)
	count := 0
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) && pod.Status.Phase == "Running" {
			if latestTime < pod.ObjectMeta.CreationTimestamp.Unix() {
				latestTime = pod.ObjectMeta.CreationTimestamp.Unix()
				podName[count] = pod.Name
				podNamespace[count] = pod.Namespace
				count++
			}
		}
	}
	p, err := os.Getwd()
	checkError(err)
	if len(podName) == 0 || len(podNamespace) == 0 {
		fmt.Println("No running tf " + toMatch)
	} else {
		for i := 0; i < count; i++ {
			fileName := path.Join(p, "logs", podName[i])
			err := ExecCmdSaveOutputFile(nil, "kubectl logs "+podName[i]+" -n "+podNamespace[i], fileName, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

// logsTf prints the logfiles of tf job
func logsTfHelp() {
	fmt.Println("Command must be in the format: logs tf (infra|dns|ingress) shoot name")
}

// logsInfra prints the logfiles of tf infra job
func logsInfra(str string) {
	logsTerraform(str)
}

// logsDNS prints the logfiles of tf dns job
func logsDNS(str string) {
	logsTerraform(str)
}

// logsIngress prints the logfiles of tf ingress job
func logsIngress(str string) {
	logsTerraform(str)
}

type logFlags struct {
	sinceSeconds time.Duration
	sinceTime    string
	tail         int64
	loki         bool
}

func newLogsFlags() *logFlags {
	return &logFlags{
		tail: -1,
	}
}

type logResponseLoki struct {
	Data struct {
		Result []struct {
			Values [][]string `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

type logMessage struct {
	Log      string `json:"log"`
	Severity string `json:"severity"`
	Process  string `json:"pid"`
	Source   string `json:"source"`
}

func (msg logMessage) String() string {
	message := "\t"
	if msg.Severity != emptyString {
		message += (msg.Severity + "\t")
	}
	if msg.Process != emptyString {
		message += (msg.Process + "\t")
	}
	if msg.Source != emptyString {
		message += (msg.Source + "\t")
	}
	message += (msg.Log + "\n")

	return message
}

func (response logResponseLoki) String() string {
	results := response.Data.Result
	var allLogs strings.Builder
	valuesDelimeter := "------------------------------------------------------------------------------------------\n"

	for resultIndex := len(results) - 1; resultIndex >= 0; resultIndex-- {
		values := results[resultIndex].Values
		isThereLogs := false
		for valueIndex := len(values) - 1; valueIndex >= 0; valueIndex-- {
			time := parseTimeInRFC(values[valueIndex][0])
			log := parseLogMessage(values[valueIndex][1])
			allLogs.WriteString(time + log.String())
			isThereLogs = true
		}

		if isThereLogs {
			allLogs.WriteString(valuesDelimeter)
		}
	}

	return allLogs.String()
}

func parseTimeInRFC(unixTime string) string {
	intTime, err := strconv.ParseInt(unixTime, 10, 64)
	checkError(err)

	return time.Unix(0, intTime).String()
}

func parseLogMessage(logMsg string) logMessage {
	byteOutput := []byte(logMsg)
	var log logMessage
	err := json.Unmarshal(byteOutput, &log)
	checkError(err)

	return log
}
