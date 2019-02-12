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
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

const (
	maxEsLogsPerQuery     = 10000
	fourteenDaysInSeconds = 60 * 60 * 24 * 14
)

//flags passed to the command
var flags *logFlags

func newCmdLogs() *cobra.Command {
	flags = newLogsFlags()
	cmd := &cobra.Command{
		Use:   "logs (gardener-apiserver|gardener-controller-manager|ui|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-main-backup|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)",
		Short: "Show and optionally follow logs of given component\n",
		Long:  ``,
		PreRun: func(cmd *cobra.Command, args []string) {
			validateArgs(args)
			validateFlags(flags)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runCommand(args)
		},
		ValidArgs: []string{"gardener-apiserver", "gardener-controller-manager", "ui", "api", "scheduler", "controller-manager", "etcd-operator", "etcd-main", "etcd-events", "addon-manager", "vpn-seed", "vpn-shoot", "auto-node-repair", "dashboard", "prometheus", "grafana", "alertmanager", "tf"},
		Aliases:   []string{"log"},
	}
	cmd.Flags().Int64Var(&flags.tail, "tail", 200, "Lines of recent log file to display. Defaults to 200 with no selector, if a selector is provided takes the number of specified lines.")
	cmd.Flags().DurationVar(&flags.sinceSeconds, "since", flags.sinceSeconds, "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	cmd.Flags().StringVar(&flags.sinceTime, "since-time", flags.sinceTime, "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	cmd.Flags().BoolVar(&flags.elasticsearch, "elasticsearch", flags.elasticsearch, "If the flag is set the logs are retrieved and shown from elasticsearch, otherwise from the kubelet.")
	return cmd
}

func validateArgs(args []string) {
	if len(args) < 1 || len(args) > 2 {
		fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|ui|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress) flags(--elasticsearch|--tail|--since|--since-time|--timestamps)")
		os.Exit(2)
	}
	var t Target
	ReadTarget(pathTarget, &t)
	if len(t.Target) < 3 && (args[0] != "gardener-apiserver") && (args[0] != "gardener-controller-manager") && (args[0] != "tf") && (args[0] != "dashboard") {
		fmt.Println("No shoot targeted")
		os.Exit(2)
	} else if (len(t.Target) < 2 && (args[0] == "tf")) || len(t.Target) < 3 && (args[0] == "tf") && (t.Target[1].Kind != "seed") {
		fmt.Println("No seed or shoot targeted")
		os.Exit(2)
	} else if len(t.Target) == 0 {
		fmt.Println("Target stack is empty")
		os.Exit(2)
	}
}

func validateFlags(flags *logFlags) {
	if flags.sinceSeconds != 0 && flags.sinceTime != "" {
		fmt.Println("Logs command can not contains --since and --since-time in the same time")
		os.Exit(2)
	} else if flags.sinceTime != "" {
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
	}
}

func runCommand(args []string) {
	switch args[0] {
	case "gardener-apiserver":
		logsGardenerApiserver()
	case "gardener-controller-manager":
		logsGardenerControllerManager()
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
		if len(args) == 2 {
			logsEtcdMain(args[1])
		} else {
			logsEtcdMain("")
		}
	case "etcd-main-backup":
		logsEtcdMainBackup()
	case "etcd-events":
		if len(args) == 2 {
			logsEtcdEvents(args[1])
		} else {
			logsEtcdEvents("")
		}
	case "addon-manager":
		logsAddonManager()
	case "vpn-seed":
		logsVpnSeed()
	case "vpn-shoot":
		logsVpnShoot()
	case "machine-controller-manager":
		logsMachineControllerManager()
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
			fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|ui|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)")
		}
	default:
		fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|ui|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)")
	}
}

// logsCmd represents the logs command
var logsCmd = newCmdLogs()

func init() {
}

// showPod is an abstraction to show pods in seed cluster controlplane or kube-system namespace of shoot
func logPod(toMatch string, toTarget string, container string) {
	var target Target
	ReadTarget(pathTarget, &target)
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
	if !flags.elasticsearch {
		showLogsFromKubectl(namespace, toMatch, container)
	} else {
		showLogsFromElasticsearch(namespace, toMatch, container)
	}
}

func showLogsFromKubectl(namespace, toMatch, container string) {
	if container != "" {
		container = " -c " + container
	}
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) {
			err := ExecCmd(nil, buildKubectlCommand(namespace, pod.Name, container), false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

func showLogsFromElasticsearch(namespace, toMatch, container string) {
	commands := buildElasticsearchCommands(namespace, toMatch, container)
	for _, command := range commands {
		output, err := ExecCmdReturnOutput("bash", "-c", fmt.Sprintf("export KUBECONFIG=%s; %s", KUBECONFIG, command))
		checkError(err)
		byteOutput := []byte(output)
		response := new(logResponse)
		json.Unmarshal(byteOutput, &response)
		fmt.Println(response)
	}
}

func buildKubectlCommand(namespace, podName, container string) string {
	var command strings.Builder
	command.WriteString(fmt.Sprintf("kubectl logs %s%s -n %s ", podName, container, namespace))
	if flags.tail != -1 {
		command.WriteString(fmt.Sprintf("--tail=%d ", flags.tail))
	}
	if flags.sinceSeconds != 0 {
		command.WriteString(fmt.Sprintf("--since=%vs ", flags.sinceSeconds.Seconds()))
	}

	return command.String()
}

func buildElasticsearchCommands(namespace, podName, container string) []string {
	queries := createElasticQueries(podName, container)
	commands := []string{}
	for _, query := range queries {
		commands = append(commands, fmt.Sprintf("kubectl exec elasticsearch-logging-0 -n %s -- curl -X GET -H \"Content-Type:application/json\" localhost:9200/_all/_search -d '%s'", namespace, query.toJSON()))
	}
	return commands
}

// logPodGarden print logfiles for garden pods
func logPodGarden(toMatch, namespace string) {
	Client, err = clientToTarget("garden")
	checkError(err)
	pods, err := Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	checkError(err)
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, toMatch) {
			err := ExecCmd(nil, "kubectl logs --tail="+strconv.Itoa(int(flags.tail))+" "+pod.Name+" -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
			break
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
func logsGardenerControllerManager() {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) != 3 {
		logPodGarden("gardener-controller-manager", "garden")
	} else {
		logPodGardenImproved("gardener-controller-manager")
	}
}

// logPodGardenImproved print logfiles for garden pods
func logPodGardenImproved(podName string) {
	var target Target
	ReadTarget(pathTarget, &target)
	Client, err := clientToTarget("garden")
	checkError(err)
	pods, err := Client.CoreV1().Pods("garden").List(metav1.ListOptions{})
	checkError(err)
	projectName := getProjectForShoot()
	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, podName) {
			output, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -n garden")
			if err != nil {
				fmt.Println("Cmd was unsuccessful")
				os.Exit(2)
			}
			lines := strings.Split("time="+output, `time=`)
			for _, line := range lines {
				if strings.Contains(line, ("shoot=" + projectName + "/" + target.Target[2].Name)) {
					fmt.Printf(line)
				}
			}
		}
	}
}

// logsUI
func logsUI() {
	logPodGarden("gardener", "garden")
}

// logsAPIServer prints the logfile of the api-server
func logsAPIServer() {
	logPod("kube-apiserver", "seed", "kube-apiserver")
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
	logPod("kube-apiserver", "seed", "vpn-seed")
	fmt.Println("-----------------------Prometheus")
	logPod("prometheus", "seed", "vpn-seed")
}

// logsEtcdOpertor prints the logfile of the etcd-operator
func logsEtcdOpertor() {
	logPodGarden("etcd-operator", "kube-system")
}

// logsEtcdMain prints the logfile of etcd-main
func logsEtcdMain(containerName string) {
	logPod("etcd-main", "seed", containerName)
}

// logsEtcdMainBackup prints logfiles of etcd-main-backup-sidecar pod
func logsEtcdMainBackup() {
	logPod("etcd-main-backup-sidecar", "seed", "")
}

// logsEtcdEvents prints the logfile of etcd-events
func logsEtcdEvents(containerName string) {
	logPod("etcd-events-", "seed", containerName)
}

// logsAddonManager prints the logfile of addon-manager
func logsAddonManager() {
	logPod("addon-manager", "seed", "")
}

// logsVpnShoot prints the logfile of vpn-shoot
func logsVpnShoot() {
	logPod("vpn-shoot", "shoot", "")
}

// logsMachineControllerManager prints the logfile of machine-controller-manager
func logsMachineControllerManager() {
	logPod("machine-controller-manager", "seed", "")
}

// logsDashboard prints the logfile of the dashboard
func logsDashboard() {
	var target Target
	ReadTarget(pathTarget, &target)
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
			err := ExecCmd(nil, "kubectl logs --tail="+strconv.Itoa(int(flags.tail))+" "+pod.Name+" -n "+namespace, false, "KUBECONFIG="+KUBECONFIG)
			checkError(err)
		}
	}
}

// logsPrometheus prints the logfiles of prometheus pod
func logsPrometheus() {
	logPod("prometheus", "seed", "prometheus")
}

// logsGrafana prints the logfiles of grafana pod
func logsGrafana() {
	logPod("grafana", "seed", "grafana")
}

// logsAlertmanager prints the logfiles of alertmanager
func logsAlertmanager() {
	logPod("alertmanager", "seed", "alertmanager") // TODO: TWO PODS ARE RUNNING
}

// logsTerraform prints the logfiles of tf pod
func logsTerraform(toMatch string) {
	var latestTime int64
	var podName [100]string
	var podNamespace [100]string
	Client, err = clientToTarget("seed")
	checkError(err)
	pods, err := Client.CoreV1().Pods("").List(metav1.ListOptions{})
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

type logFlags struct {
	sinceSeconds  time.Duration
	sinceTime     string
	tail          int64
	elasticsearch bool
}

func newLogsFlags() *logFlags {
	return &logFlags{
		tail: -1,
	}
}

type logResponse struct {
	Hits struct {
		Hits []struct {
			Source struct {
				Timestamp time.Time `json:"@timestamp"`
				Log       string    `json:"log"`
				Severity  string    `json:"severity"`
				Source    string    `json:"source"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type rng struct {
	TimeField string
	Gte       int64
}

func (r *rng) toJSON() string {
	return fmt.Sprintf("\"range\":{\"%s\":{\"gte\":\"now-%ds\"}}", r.TimeField, r.Gte)
}

type term struct {
	Key   string
	Value string
}

func (t *term) toJSON() string {
	return fmt.Sprintf("\"term\":{\"%s\":\"%s\"}", t.Key, t.Value)
}

type boolQuery struct {
	rng  rng
	term []term
}

func (f *boolQuery) toJSON() string {
	var sb strings.Builder
	sb.WriteString("\"query\":{\"bool\":{\"filter\":[")
	for _, term := range f.term {
		sb.WriteString("{")
		sb.WriteString(term.toJSON())
		sb.WriteString("},")
	}
	sb.WriteString("{")
	sb.WriteString(f.rng.toJSON())
	sb.WriteString("}]}}")
	return sb.String()
}

type requestQuery struct {
	BoolQuery boolQuery
	From      int64
	Size      int64
	Source    []string
}

func (q *requestQuery) toJSON() string {
	var sb strings.Builder
	sb.WriteString("{")
	sb.WriteString(q.BoolQuery.toJSON() + ",")
	sb.WriteString(fmt.Sprintf("\"from\":%d,", q.From))
	sb.WriteString(fmt.Sprintf("\"size\":%d,", q.Size))
	sb.WriteString("\"sort\": {\"@timestamp\": \"desc\"},")
	sb.WriteString("\"_source\":[")
	for index, source := range q.Source {
		if index != 0 {
			sb.WriteString(", " + fmt.Sprintf("\"%s\"", source))
		} else {
			sb.WriteString(fmt.Sprintf("\"%s\"", source))
		}
	}
	sb.WriteString("]}")
	return sb.String()
}

func newRequestQuery(from, size int64, r rng, sources []string, terms []term) *requestQuery {
	q := new(requestQuery)
	q.BoolQuery.rng = r
	q.BoolQuery.term = append(q.BoolQuery.term, terms...)
	q.From = from
	q.Size = size
	q.Source = append(q.Source, sources...)
	return q
}

func createElasticQueries(podName, containerName string) []requestQuery {
	podTokens := strings.Split(podName, "-")
	containerTokens := strings.Split(containerName, "-")
	terms := setTerms(podTokens, containerTokens)
	sources := []string{"@timestamp", "severity", "source", "log"}
	rng := rng{TimeField: "@timestamp", Gte: int64(flags.sinceSeconds.Seconds())}
	if rng.Gte == 0 {
		rng.Gte = fourteenDaysInSeconds
	}

	numberOfRequests := int((flags.tail-1)/maxEsLogsPerQuery + 1)
	queries := buildQueries(numberOfRequests, rng, sources, terms)
	return queries
}

func setTerms(podTokens, containerTokens []string) []term {
	terms := []term{}
	for _, token := range podTokens {
		terms = append(terms, term{Key: "kubernetes.pod_name", Value: token})
	}
	for _, token := range containerTokens {
		if token != "" {
			terms = append(terms, term{Key: "kubernetes.container_name", Value: token})
		}
	}
	return terms
}

func buildQueries(numberOfRequests int, r rng, sources []string, terms []term) []requestQuery {
	queries := []requestQuery{}
	from := int64(0)
	if flags.tail == -1 {
		flags.tail = maxEsLogsPerQuery
	}
	for i := 0; i < numberOfRequests; i++ {
		size := flags.tail - from
		if size > maxEsLogsPerQuery {
			size = maxEsLogsPerQuery
		}
		queries = append(queries, *newRequestQuery(from, size, r, sources, terms))
		from += (maxEsLogsPerQuery + 1)
	}
	return queries
}

func (response logResponse) String() string {
	output := make([]string, 0)
	for _, hit := range response.Hits.Hits {
		source := hit.Source
		output = append(output, fmt.Sprintf("%v %v %v | %v", source.Timestamp, source.Severity, source.Source, source.Log))
	}

	return strings.Join(output[:], "\n")
}
