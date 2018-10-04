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
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:   "shell (node|pod)",
	Short: "Shell to a node\n",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 2 {
			fmt.Println("Command must be in the format: shell (node)")
			os.Exit(2)
		} else if len(args) == 0 {
			printNodes()
		} else {
			shellToNode(args[0])
		}
	},
}

// Image specify the container image to use
var Image string

func init() {
	shellCmd.PersistentFlags().StringVarP(&Image, "image", "i", "busybox", "image type")
}

// printNodes print all nodes in k8s cluster
func printNodes() {
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)
	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, n := range nodes.Items {
		fmt.Println(n.Name)
	}
}

// shellToNode creates a rootpod on node
func shellToNode(name string) {
	namespace := "default"
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)
	// check if the node name was a pod name and we should actually identify the node from the pod (node that runs the pod)
	pods, err := Client.CoreV1().Pods("").List(metav1.ListOptions{})
	checkError(err)
	for _, p := range pods.Items {
		if p.Name == name {
			name = p.Spec.NodeName
			namespace = p.Namespace
			break
		}
	}
	hostname := ""
	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	for _, n := range nodes.Items {
		host := n.Labels
		if strings.Contains(host["kubernetes.io/hostname"], name) {
			hostname = host["kubernetes.io/hostname"]
			break
		}
	}
	if hostname == "" {
		fmt.Println("Nodename not found")
		os.Exit(2)
	}
	podName, err := ExecCmdReturnOutput("whoami")
	if err != nil {
		fmt.Println("Cmd was unsuccessful")
		os.Exit(2)
	}
	podName = "rootpod-" + podName
	typeOfTarget, err := getTargetType()
	checkError(err)
	if typeOfTarget == "shoot" {
		namespace = "kube-system"
	} else if typeOfTarget == "project" {
		fmt.Println("Project targeted")
		os.Exit(2)
	}
	manifest := strings.Replace(shellManifest, "rootpod", podName, -1)
	manifest = strings.Replace(manifest, "default", namespace, -1)
	manifest = strings.Replace(manifest, "busybox", Image, -1)
	manifest = strings.Replace(manifest, "HOSTNAME", hostname, -1)
	err = ExecCmd([]byte(manifest), "kubectl -n "+namespace+" apply -f -", false, "KUBECONFIG="+getKubeConfigOfClusterType(typeName))
	checkError(err)
	for true {
		pod, err := Client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
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
	err = ExecCmd(nil, "kubectl -n "+namespace+" exec -it "+podName+" -- chroot /hostroot /bin/bash", false, "KUBECONFIG="+getKubeConfigOfClusterType(typeName))
	checkError(err)
	err = ExecCmd(nil, "kubectl -n "+namespace+" delete pod "+podName, false, "KUBECONFIG="+getKubeConfigOfClusterType(typeName))
	checkError(err)
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
