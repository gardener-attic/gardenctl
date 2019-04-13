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
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// NewShellCmd returns a new shell command.
func NewShellCmd(targetProvider TargetProviderAPI, ioStreams IOStreams) *cobra.Command {
	shellCmd := &cobra.Command{
		Use:   "shell (node|pod)",
		Short: "Shell to a node",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 2 {
				return errors.New("command must be in the format: gardenctl shell (node|pod)")
			}

			targetKind, err := targetProvider.FetchTargetKind()
			checkError(err)
			if targetKind == "project" {
				return errors.New("project targeted")
			}

			client, err := targetProvider.ClientToTarget(targetKind)
			checkError(err)
			if len(args) == 0 {
				return printNodes(client, ioStreams)
			}

			return shellToNode(client, targetKind, args[0], ioStreams)
		},
		SilenceUsage: true,
	}

	shellCmd.PersistentFlags().StringVarP(&Image, "image", "i", "busybox", "image type")

	return shellCmd
}

// Image specify the container image to use
var Image string

// printNodes print all nodes in k8s cluster
func printNodes(client k8s.Interface, ioStreams IOStreams) error {
	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, n := range nodes.Items {
		fmt.Fprintln(ioStreams.Out, n.Name)
	}
	return nil
}

// shellToNode creates a rootpod on node
func shellToNode(client k8s.Interface, targetKind, name string, ioStreams IOStreams) error {
	// check if the node name was a pod name and we should actually identify the node from the pod (node that runs the pod)
	pods, err := client.CoreV1().Pods("").List(metav1.ListOptions{})
	checkError(err)
	namespace := "default"
	for _, p := range pods.Items {
		if p.Name == name {
			name = p.Spec.NodeName
			namespace = p.Namespace
			break
		}
	}
	hostname := ""
	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, n := range nodes.Items {
		host := n.Labels
		if strings.Contains(host["kubernetes.io/hostname"], name) {
			hostname = host["kubernetes.io/hostname"]
			break
		}
	}
	if hostname == "" {
		return fmt.Errorf("node %q not found", name)
	}
	podName, err := ExecCmdReturnOutput("whoami")
	if err != nil {
		return errors.New("Cmd was unsuccessful")
	}
	podName = "rootpod-" + podName
	typeOfTarget, err := getTargetType()
	checkError(err)
	if typeOfTarget == "shoot" {
		namespace = "kube-system"
	}
	manifest := strings.Replace(shellManifest, "rootpod", podName, -1)
	manifest = strings.Replace(manifest, "default", namespace, -1)
	manifest = strings.Replace(manifest, "busybox", Image, -1)
	manifest = strings.Replace(manifest, "HOSTNAME", hostname, -1)
	err = ExecCmd([]byte(manifest), "kubectl -n "+namespace+" apply -f -", false, "KUBECONFIG="+getKubeConfigOfClusterType(targetKind))
	checkError(err)

	for true {
		pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("pod not found: %s\n", err)
		} else {
			ip := pod.Status.HostIP
			if ip != "" && pod.Status.Phase == "Running" {
				fmt.Printf("host ip found: %s\n", ip)
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
	time.Sleep(1000)
	err = ExecCmd(nil, "kubectl -n "+namespace+" exec -it "+podName+" -- chroot /hostroot /bin/bash", false, "KUBECONFIG="+getKubeConfigOfClusterType(targetKind))
	checkError(err)
	err = ExecCmd(nil, "kubectl -n "+namespace+" delete pod "+podName, false, "KUBECONFIG="+getKubeConfigOfClusterType(targetKind))
	checkError(err)
	return nil
}

var shellManifest = `
apiVersion: v1
kind: Pod
metadata:
  name: rootpod
  namespace: default
spec:
  containers:
  - image: busybox
    name: root-container
    command:
    - sleep 
    - "10000000"
    stdin: true
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /hostroot
      name: root-volume
  hostNetwork: true
  hostPID: true
  nodeSelector:
    kubernetes.io/hostname: "HOSTNAME"
  tolerations:
  - key: node-role.kubernetes.io/master
    operator: Exists
    effect: NoSchedule
  volumes:
  - name: root-volume
    hostPath:
      path: /
`
