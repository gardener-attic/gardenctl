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
	Use:   "shell (node)",
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

func init() {
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
func shellToNode(nodename string) {
	typeName, err := getTargetType()
	checkError(err)
	Client, err = clientToTarget(typeName)
	checkError(err)
	nodes, err := Client.CoreV1().Nodes().List(metav1.ListOptions{})
	checkError(err)
	var hostname string = ""
	for _, n := range nodes.Items {
		host := n.Labels
		if strings.Contains(host["kubernetes.io/hostname"], nodename) {
			hostname = host["kubernetes.io/hostname"]
			break
		}
	}
	if hostname == "" {
		fmt.Println("Nodename not found")
		os.Exit(2)
	}
	manifest := strings.Replace(shellManifest, "HOSTNAME", hostname, -1)
	err = ExecCmd([]byte(manifest), "kubectl -n default apply -f -", false, "KUBECONFIG="+getKubeConfigOfClusterType(typeName))
	checkError(err)
	for true {
		pod, err := Client.CoreV1().Pods("default").Get("rootpod", metav1.GetOptions{})
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
	err = ExecCmd(nil, "kubectl -n default exec -it rootpod -- chroot /hostroot", false, "KUBECONFIG="+getKubeConfigOfClusterType(typeName))
	checkError(err)
	err = ExecCmd(nil, "kubectl -n default delete pod rootpod", false, "KUBECONFIG="+getKubeConfigOfClusterType(typeName))
	checkError(err)
}

var shellManifest = `
apiVersion: v1
kind: Pod
metadata:
  name: rootpod
spec:
  containers:
  - image: busybox
    name: root-container
    command:
    - sleep 
    - "10000000"
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
