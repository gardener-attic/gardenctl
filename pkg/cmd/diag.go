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
	"strconv"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// NewDiagCmd returns diagnostic information for a shoot.
func NewDiagCmd(reader TargetReader, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "diag",
		Short:        "Print shoot diagnostic information, e.g. \"gardenctl diag\"",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := reader.ReadTarget(pathTarget)
			if !CheckShootIsTargeted(target) {
				return errors.New("no shoot targeted")
			}

			shoot, err := FetchShootFromTarget(target)
			checkError(err)
			getShootInformation(shoot, target)
			return nil
		},
	}
	return cmd
}

//getShootInformation prints all information regarding a shoot
func getShootInformation(shoot *v1beta1.Shoot, target TargetInterface) {
	fmt.Println("The shoot diagnostic information are as follows:")
	fmt.Println()
	fmt.Println("Shoot: " + shoot.Name)
	fmt.Println("Kubernetes Version: " + shoot.Spec.Kubernetes.Version)
	fmt.Println("Created At: " + shoot.ObjectMeta.CreationTimestamp.String())
	annotationsMap := shoot.GetObjectMeta().GetAnnotations()
	fmt.Println("Created By: " + annotationsMap["gardener.cloud/created-by"])
	fmt.Println("Cloud Profile: " + shoot.Spec.CloudProfileName)
	fmt.Println("Region: " + shoot.Spec.Region)
	fmt.Println("Purpose: " + *shoot.Spec.Purpose)
	fmt.Println("Seed Name: " + *shoot.Status.SeedName)
	fmt.Println()

	fmt.Println("Last Operation:")
	fmt.Println()
	lastOperationMsg := []string{shoot.Status.LastOperation.Description, fmt.Sprintf("%v", shoot.Status.LastOperation.Type), fmt.Sprintf("%v", shoot.Status.LastOperation.State)}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Description", "Last Operation Type", "Last Operation Type State"})
	table.Append(lastOperationMsg)
	table.Render()
	fmt.Println()

	fmt.Println("Shoot Conditions: ")
	fmt.Println()
	data := [][]string{}
	for i := range shoot.Status.Conditions {
		condition := shoot.Status.Conditions[i]
		codesValue := condition.Codes
		codesString := fmt.Sprintf("%v", codesValue)
		data = append(data, []string{condition.Message, condition.LastTransitionTime.String(), codesString})
	}
	table = tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Message", "Last Transition Time", "Codes"})
	for _, v := range data {
		table.Append(v)
	}
	table.Render()
	fmt.Println()
	fmt.Println("Workers Groups:")
	fmt.Println()
	data = [][]string{}
	table = tablewriter.NewWriter(os.Stdout)
	for w := range shoot.Spec.Provider.Workers {
		worker := shoot.Spec.Provider.Workers[w]
		if worker.Volume != nil {
			data = append(data, []string{worker.Name, strconv.Itoa(int(worker.Minimum)), strconv.Itoa(int(worker.Maximum)), fmt.Sprintf("%v", worker.MaxUnavailable), fmt.Sprintf("%v", worker.MaxSurge), worker.Machine.Image.Name, *worker.Machine.Image.Version, worker.Machine.Type, fmt.Sprintf("%v", worker.Zones), fmt.Sprintf("%v", worker.Volume.Name), fmt.Sprintf("%v", worker.Volume.Type), worker.Volume.VolumeSize})
			table.SetHeader([]string{"Worker Name", "Min", "Max", "Max Unavailable", "Max Surge", "Image Name", "Image Version", "Image Type", "Zones", "Volume Name", "Volume Type", "Volume Size"})
		} else {
			data = append(data, []string{worker.Name, strconv.Itoa(int(worker.Minimum)), strconv.Itoa(int(worker.Maximum)), fmt.Sprintf("%v", worker.MaxUnavailable), fmt.Sprintf("%v", worker.MaxSurge), worker.Machine.Image.Name, *worker.Machine.Image.Version, worker.Machine.Type, fmt.Sprintf("%v", worker.Zones)})
			table.SetHeader([]string{"Worker Name", "Min", "Max", "Max Unavailable", "Max Surge", "Image Name", "Image Version", "Image Type", "Zones"})
		}
	}
	for _, v := range data {
		table.Append(v)
	}
	table.Render()

	if !shoot.Status.IsHibernated {
		var err error
		var shootClient kubernetes.Interface
		if shootClient, err = target.K8SClientToKind(TargetKindShoot); err != nil {
			checkError(err)
		}
		var nodes *corev1.NodeList
		if nodes, err = shootClient.CoreV1().Nodes().List(metav1.ListOptions{}); err != nil {
			checkError(err)
		}

		kubeconfig, err := ioutil.ReadFile(*kubeconfig)
		checkError(err)
		clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
		checkError(err)
		config, err := clientConfig.ClientConfig()
		checkError(err)
		metricsClientset, err := metricsv.NewForConfig(config)
		checkError(err)
		nodeMetricsList, err := metricsClientset.MetricsV1beta1().NodeMetricses().List(metav1.ListOptions{})
		checkError(err)
		fmt.Println()
		fmt.Println("Node Metrics:")
		fmt.Println()
		data = [][]string{}
		for _, metric := range nodeMetricsList.Items {
			cpuUsage, _ := metric.Usage.Cpu().AsInt64()
			memUsage, _ := metric.Usage.Memory().AsInt64()

			data = append(data, []string{metric.GetName(), fmt.Sprintf("%v", cpuUsage), fmt.Sprintf("%v", memUsage/1000)})
		}
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Node Name", "CPU Usage (Core)", "Memory Usage (MB)"})

		for _, v := range data {
			table.Append(v)
		}
		table.Render()
		fmt.Println()
		fmt.Println("System Components:")
		fmt.Println()
		systemPods, err := shootClient.CoreV1().Pods("kube-system").List(metav1.ListOptions{})
		checkError(err)
		data := [][]string{}
		for _, pod := range systemPods.Items {
			readyNumber := 0
			totalNumber := 0
			for pss := range pod.Status.ContainerStatuses {
				if pod.Status.ContainerStatuses[pss].Ready {
					readyNumber++
				}
				if pod.Status.ContainerStatuses[pss].State.Terminated == nil {
					totalNumber++
				}
			}
			podstatusPhase := string(pod.Status.Phase)
			podCreationTime := pod.GetCreationTimestamp()
			age := time.Since(podCreationTime.Time).Round(time.Second)
			timeString := fmt.Sprintf("%v", podCreationTime)

			data = append(data, []string{pod.GetName(), podstatusPhase, strconv.Itoa(readyNumber), strconv.Itoa(totalNumber), timeString, age.String()})
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Phase", "Ready", "Total", "Created", "Age"})
		for _, v := range data {
			table.Append(v)
		}
		table.Render()

		daemonSets, err := shootClient.AppsV1().DaemonSets("kube-system").List(metav1.ListOptions{})
		checkError(err)
		fmt.Println()
		fmt.Println("DaemonSets:")
		fmt.Println()
		data = [][]string{}
		for _, ds := range daemonSets.Items {
			data = append(data, []string{ds.GetName(), strconv.Itoa(int(ds.Status.DesiredNumberScheduled)), strconv.Itoa(int(ds.Status.NumberAvailable))})
		}
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Desired Number", "Current Number"})
		for _, v := range data {
			table.Append(v)
		}
		table.Render()

		fmt.Println()
		fmt.Println("Nodes:")
		fmt.Println()
		data = [][]string{}
		for _, n := range nodes.Items {
			addString := ""
			for a := range n.Status.Addresses {
				if n.Status.Addresses[a].Type == "InternalIP" {
					addString = n.Status.Addresses[a].Address
				}
			}
			cpuCount, _ := n.Status.Capacity.Cpu().AsInt64()
			memoryCount, _ := n.Status.Capacity.Memory().AsInt64()
			data = append(data, []string{n.Name, n.Spec.ProviderID, addString, strconv.Itoa(int(cpuCount)), strconv.Itoa(int(memoryCount / 1000))})
		}
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Node Name", "Provider ID", "Address", "CPU Cores", "Memory (MB)"})
		for _, v := range data {
			table.Append(v)
		}
		table.Render()
		fmt.Println()
		fmt.Println("BlockingDisruptionBudgets:")
		fmt.Println()
		dbds, err := shootClient.PolicyV1beta1().PodDisruptionBudgets("kube-system").List(metav1.ListOptions{})
		checkError(err)
		data = [][]string{}
		for dbdsIndex := range dbds.Items {
			pdb := dbds.Items[dbdsIndex]
			data = append(data, []string{pdb.Spec.Selector.MatchLabels["k8s-app"], fmt.Sprintf("%v", pdb.Spec.MinAvailable), fmt.Sprintf("%v", pdb.Spec.MaxUnavailable)})
		}
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Min Available", "Max Unavailable"})
		for _, v := range data {
			table.Append(v)
		}
		table.Render()
		fmt.Println()
		fmt.Println("MutatingWebhookConfigurations:")
		fmt.Println()
		data = [][]string{}
		mutatingWebhookConfigurations, err := shootClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(metav1.ListOptions{})
		checkError(err)
		for mwc := range mutatingWebhookConfigurations.Items {
			whList := mutatingWebhookConfigurations.Items[mwc].Webhooks
			for whIndex := range whList {
				wh := whList[whIndex]
				data = append(data, []string{wh.Name})
			}
		}
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"MutatingWebhookConfiguration Name"})

		for _, v := range data {
			table.Append(v)
		}
		table.Render()
		seedClient, err := target.K8SClientToKind(TargetKindSeed)
		checkError(err)
		controlPlanePods, err := seedClient.CoreV1().Pods(shoot.Status.TechnicalID).List(metav1.ListOptions{})
		checkError(err)
		fmt.Println()
		fmt.Println("Control Plane Pods:")
		fmt.Println()
		data = [][]string{}
		for _, pod := range controlPlanePods.Items {
			readyNumber := 0
			totalNumber := 0
			for pss := range pod.Status.ContainerStatuses {
				if pod.Status.ContainerStatuses[pss].Ready {
					readyNumber++
				}
				if pod.Status.ContainerStatuses[pss].State.Terminated == nil {
					totalNumber++
				}
			}
			podstatusPhase := string(pod.Status.Phase)
			podCreationTime := pod.GetCreationTimestamp()
			age := time.Since(podCreationTime.Time).Round(time.Second)
			data = append(data, []string{pod.GetName(), podstatusPhase, strconv.Itoa(readyNumber), strconv.Itoa(totalNumber), fmt.Sprintf("%v", podCreationTime), age.String()})
		}
		table = tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Phase", "Ready Number", "Total Number", "Creation Time", "Age"})

		for _, v := range data {
			table.Append(v)
		}
		table.Render()

	} else {
		fmt.Println()
		fmt.Println("This shoot is now in hibernating status")
		fmt.Println("Information like Nodes/Metrics/PDBs/Web hooks/etc will not be displayed")
	}

}
