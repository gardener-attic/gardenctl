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
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	maxEsLogs             = 100000
	fourteenDaysInSeconds = 60 * 60 * 24 * 14
	esLogsPerRequest      = 10000
	emptyString           = ""
	unauthorized          = "Unauthorized"
)

//flags passed to the command
var flags *logFlags

func newCmdLogs() *cobra.Command {
	flags = newLogsFlags()
	cmd := &cobra.Command{
		Use:   "logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-main-backup|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)",
		Short: "Show and optionally follow logs of given component\n",
		PreRun: func(cmd *cobra.Command, args []string) {
			validateArgs(args)
			validateFlags(flags)
		},
		Run: func(cmd *cobra.Command, args []string) {
			runCommand(args)
		},
		ValidArgs: []string{"gardener-apiserver", "gardener-controller-manager", "gardener-dashboard", "api", "scheduler", "controller-manager", "etcd-operator", "etcd-main", "etcd-events", "addon-manager", "vpn-seed", "vpn-shoot", "auto-node-repair", "kubernetes-dashboard", "prometheus", "grafana", "alertmanager", "tf"},
		Aliases:   []string{"log"},
	}
	cmd.Flags().Int64Var(&flags.tail, "tail", 200, "Lines of recent log file to display. Defaults to 200 with no selector, if a selector is provided takes the number of specified lines (max 100 000 for elasticsearch).")
	cmd.Flags().DurationVar(&flags.sinceSeconds, "since", flags.sinceSeconds, "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")
	cmd.Flags().StringVar(&flags.sinceTime, "since-time", flags.sinceTime, "Only return logs after a specific date (RFC3339). Defaults to all logs. Only one of since-time / since may be used.")
	cmd.Flags().BoolVar(&flags.elasticsearch, "elasticsearch", flags.elasticsearch, "If the flag is set the logs are retrieved and shown from elasticsearch, otherwise from the kubelet.")
	return cmd
}

func validateArgs(args []string) {
	if len(args) < 1 || len(args) > 2 {
		fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress) flags(--elasticsearch|--tail|--since|--since-time|--timestamps)")
		os.Exit(2)
	}
	var t Target
	ReadTarget(pathTarget, &t)
	if len(t.Target) < 3 && (args[0] != "gardener-apiserver") && (args[0] != "gardener-controller-manager") && (args[0] != "tf") && (args[0] != "kubernetes-dashboard") {
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
	} else if flags.elasticsearch && flags.tail > maxEsLogs {
		fmt.Println(fmt.Sprintf("Maximum number of logs that can be fetched from elasticsearch is %d", maxEsLogs))
		os.Exit(2)
	}
}

func runCommand(args []string) {
	switch args[0] {
	case "gardener-apiserver":
		logsGardenerApiserver()
	case "gardener-controller-manager":
		logsGardenerControllerManager()
	case "gardener-dashboard":
		logsGardenerDashboard()
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
	case "kubernetes-dashboard":
		logsKubernetesDashboard()
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
			fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|kubernetes-dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)")
		}
	default:
		fmt.Println("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|auto-node-repair|kubernetes-dashboard|prometheus|grafana|alertmanager|tf (infra|dns|ingress)")
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
		Client, err = clientToTarget(TargetKindShoot)
		checkError(err)
	}
	if !flags.elasticsearch {
		showLogsFromKubectl(namespace, toMatch, container)
	} else {
		credentials, err := getCredentials(namespace)

		if err == nil {
			username := credentials.Data["username"]
			password := credentials.Data["password"]

			showLogsFromElasticsearch(namespace, toMatch, container, string(username), string(password))
		} else {
			showLogsFromElasticsearch(namespace, toMatch, container, emptyString, emptyString)
		}
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
			err := ExecCmd(nil, buildKubectlCommand(namespace, pod.Name, container), false)
			checkError(err)
		}
	}
}

func getCredentials(namespace string) (*v1.Secret, error) {
	oldConfig := KUBECONFIG
	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfigOfClusterType("seed"))
	checkError(err)
	clientset, err := k8s.NewForConfig(config)
	checkError(err)
	KUBECONFIG = oldConfig

	return clientset.CoreV1().Secrets(namespace).Get("logging-ingress-credentials", metav1.GetOptions{})
}

func showLogsFromElasticsearch(namespace, toMatch, container, username, password string) {
	output, err := ExecCmdReturnOutput("bash", "-c", buildElasticsearchCommand(namespace, toMatch, container, username, password))
	checkError(err)

	if output == unauthorized {
		fmt.Println("You have no permissions to read from elasticsearch")
		os.Exit(2)
	}

	responses := buildElasticsearchResponses(output, namespace, toMatch, container, username, password)
	var logs strings.Builder
	for i := len(responses) - 1; i >= 0; i-- {
		logs.WriteString(responses[i].String())
		if i != 0 {
			logs.WriteString("\n")
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 6, 4, 3, ' ', 0)
	fmt.Fprintln(w, logs.String())
	w.Flush()
}

func buildKubectlCommand(namespace, podName, container string) string {
	var command strings.Builder
	command.WriteString(fmt.Sprintf("kubectl logs --kubeconfig=%s %s%s -n %s ", KUBECONFIG, podName, container, namespace))
	if flags.tail != -1 {
		command.WriteString(fmt.Sprintf("--tail=%d ", flags.tail))
	}
	if flags.sinceSeconds != 0 {
		command.WriteString(fmt.Sprintf("--since=%vs ", flags.sinceSeconds.Seconds()))
	}

	return command.String()
}

func buildElasticsearchCommand(namespace, podName, container, username, password string) string {
	query := createElasticQuery(podName, container)
	bytes, err := json.Marshal(query)
	checkError(err)

	command := fmt.Sprintf("kubectl --kubeconfig=%s exec elasticsearch-logging-0 -n %s -- curl -X GET -H \"Content-Type:application/json\" localhost:9200/_all/_search?scroll=1m -d '%s'", KUBECONFIG, namespace, string(bytes))
	if username != emptyString && password != emptyString {
		command += fmt.Sprintf(" --user %s:%s", username, password)
	}

	return command
}

func buildElasticsearchScrollCommand(namespace, podName, container, scrollID, username, password string) string {
	request := scrollRequest{Scroll: "1m", ScrollID: scrollID}
	bytes, err := json.Marshal(request)
	checkError(err)

	scrollCommand := fmt.Sprintf("kubectl --kubeconfig=%s exec elasticsearch-logging-0 -n %s -- curl -X POST -H \"Content-Type:application/json\" localhost:9200/_search/scroll -d '%s'", KUBECONFIG, namespace, string(bytes))
	if username != emptyString && password != emptyString {
		scrollCommand += fmt.Sprintf(" --user %s:%s", username, password)
	}

	return scrollCommand
}

func buildElasticsearchResponses(output, namespace, toMatch, container, username, password string) []logResponse {
	responses := make([]logResponse, 0)

	byteOutput := []byte(output)
	var response logResponse
	err = json.Unmarshal(byteOutput, &response)
	checkError(err)

	responses = append(responses, response)

	scrollID := response.ScrollID
	logsToFetch := flags.tail - esLogsPerRequest
	for logsToFetch > 0 {
		output, err := ExecCmdReturnOutput("bash", "-c", buildElasticsearchScrollCommand(namespace, toMatch, container, scrollID, username, password))
		checkError(err)

		byteOutput := []byte(output)
		logResponse := new(logResponse)
		err = json.Unmarshal(byteOutput, logResponse)
		checkError(err)

		if int(logsToFetch) < len(logResponse.Hits.Hits) {
			start := esLogsPerRequest - logsToFetch
			logResponse.Hits.Hits = logResponse.Hits.Hits[start:]
		}

		responses = append(responses, *logResponse)

		scrollID = logResponse.ScrollID
		logsToFetch -= esLogsPerRequest
	}

	return responses
}

// logPodGarden print logfiles for garden pods
func logPodGarden(toMatch, namespace string) {
	Client, err = clientToTarget("garden")
	checkError(err)
	showLogsFromKubectl(namespace, toMatch, "")
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
			output, err := ExecCmdReturnOutput("bash", "-c", buildKubectlCommand("garden", pod.Name, ""))
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

// logsGardenerDashboard
func logsGardenerDashboard() {
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

// logsKubernetesDashboard prints the logfile of the dashboard
func logsKubernetesDashboard() {
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
	ScrollID string `json:"_scroll_id"`
	Hits     struct {
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
	Range struct {
		Timestamp rangeTimestamp `json:"@timestamp"`
	} `json:"range"`
}

type rangeTimestamp struct {
	Gte string `json:"gte"`
}

type podNameMatchPhrase struct {
	MatchPhrase struct {
		Value string `json:"kubernetes.pod_name"`
	} `json:"match_phrase"`
}

type containerNameMatchPhrase struct {
	MatchPhrase struct {
		Value string `json:"kubernetes.container_name"`
	} `json:"match_phrase"`
}

type mustQuery struct {
	MatchPhrases []interface{} `json:"must"`
}

type query struct {
	Bool struct {
		MatchPhrases []interface{} `json:"must"`
	} `json:"bool"`
}

type requestQuery struct {
	Query  query    `json:"query"`
	Size   int64    `json:"size"`
	Source []string `json:"_source"`
	Sort   struct {
		Timestamp string `json:"@timestamp"`
	} `json:"sort"`
}

type scrollRequest struct {
	Scroll   string `json:"scroll"`
	ScrollID string `json:"scroll_id"`
}

func newRequestQuery(size int64, r rng, sources []string, matchPhrases []interface{}) *requestQuery {
	q := new(requestQuery)
	q.Query.Bool.MatchPhrases = append(q.Query.Bool.MatchPhrases, matchPhrases...)
	q.Sort.Timestamp = "desc"
	q.Size = size
	q.Source = append(q.Source, sources...)
	return q
}

func createElasticQuery(podName, containerName string) requestQuery {
	matchPhrases := buildMatchPhrases(podName, containerName)
	sources := []string{"@timestamp", "severity", "source", "log"}

	if flags.sinceSeconds == 0 {
		flags.sinceSeconds = fourteenDaysInSeconds * time.Second
	}

	timestamp := rangeTimestamp{Gte: fmt.Sprintf("now-%ds", int64(flags.sinceSeconds.Seconds()))}
	rng := rng{
		Range: struct {
			Timestamp rangeTimestamp `json:"@timestamp"`
		}{
			Timestamp: timestamp,
		}}
	matchPhrases = append(matchPhrases, rng)

	min := flags.tail
	if min > esLogsPerRequest {
		min = esLogsPerRequest
	}

	return *newRequestQuery(min, rng, sources, matchPhrases)
}

func buildMatchPhrases(podName, containerName string) []interface{} {
	matchPhrases := make([]interface{}, 0)
	podNameMatchPhrase := podNameMatchPhrase{
		MatchPhrase: struct {
			Value string `json:"kubernetes.pod_name"`
		}{
			Value: podName,
		}}
	matchPhrases = append(matchPhrases, podNameMatchPhrase)
	if containerName != "" {
		containerNameMatchPhrase := containerNameMatchPhrase{
			MatchPhrase: struct {
				Value string `json:"kubernetes.container_name"`
			}{
				Value: containerName,
			}}
		matchPhrases = append(matchPhrases, containerNameMatchPhrase)
	}

	return matchPhrases
}

func (response logResponse) String() string {
	hits := response.Hits.Hits
	length := len(hits)
	output := make([]string, 0, length)
	for i := length - 1; i >= 0; i-- {
		source := hits[i].Source
		output = append(output, strings.TrimSpace(fmt.Sprintf("%v\t%v\t%v\t%v", source.Timestamp, source.Severity, source.Source, source.Log)))
	}
	return strings.Join(output, "\n")
}
