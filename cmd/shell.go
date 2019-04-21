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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Represents the container image to use
var imageFlag string

// NewShellCmd returns a new shell command.
func NewShellCmd(targetProvider TargetProviderAPI, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "shell (node|pod)",
		Short:        "Shell to a node",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 2 {
				return errors.New("command must be in the format: gardenctl shell (node|pod)")
			}

			var targetKind TargetKind
			if targetKind, err = targetProvider.FetchTargetKind(); err != nil {
				return err
			}
			if targetKind == TargetKindProject {
				return errors.New("project targeted")
			}

			var client kubernetes.Interface
			if client, err = targetProvider.ClientToTarget(targetKind); err != nil {
				return err
			}
			if len(args) == 0 {
				return printNodes(client, ioStreams)
			}

			return shellToNode(client, targetKind, args[0], ioStreams)
		},
	}

	cmd.PersistentFlags().StringVarP(&imageFlag, "image", "i", "busybox", "image type")

	return cmd
}

// printNodes print all nodes in k8s cluster
func printNodes(client kubernetes.Interface, ioStreams IOStreams) (err error) {
	var nodes *corev1.NodeList
	if nodes, err = client.CoreV1().Nodes().List(metav1.ListOptions{}); err != nil {
		return err
	}

	for _, n := range nodes.Items {
		fmt.Fprintln(ioStreams.Out, n.Name)
	}

	return
}

// shellToNode creates a root pod on node
func shellToNode(client kubernetes.Interface, targetKind TargetKind, name string, ioStreams IOStreams) (err error) {
	// Check if the node name was a pod name and we should actually identify the node from the pod (node that runs the pod)
	var pods *corev1.PodList
	if pods, err = client.CoreV1().Pods("").List(metav1.ListOptions{}); err != nil {
		return err
	}
	namespace := "default"
	for _, p := range pods.Items {
		if p.Name == name {
			name = p.Spec.NodeName
			namespace = p.Namespace
			break
		}
	}

	var nodes *corev1.NodeList
	if nodes, err = client.CoreV1().Nodes().List(metav1.ListOptions{}); err != nil {
		return err
	}
	hostname := ""
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

	var podName string
	if podName, err = ExecCmdReturnOutput("whoami"); err != nil {
		return errors.New("Cmd was unsuccessful")
	}
	podName = "rootpod-" + podName
	if targetKind == TargetKindShoot {
		namespace = "kube-system"
	}

	// Apply root pod
	rootPod := buildRootPod(podName, namespace, imageFlag, hostname)
	if err = apply(client, rootPod); err != nil {
		return err
	}

	// Wait until root pod is running
	err = wait.PollImmediate(500*time.Millisecond, time.Minute, func() (bool, error) {
		pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintln(ioStreams.Out, err.Error())
			return false, nil
		}

		ip := pod.Status.HostIP
		if ip == "" || pod.Status.Phase != corev1.PodRunning {
			return false, nil
		}

		fmt.Fprintf(ioStreams.Out, "host ip found: %s\n", ip)
		return true, nil
	})
	if err != nil {
		return err
	}

	time.Sleep(1000)
	err = ExecCmd(nil, "kubectl -n "+namespace+" exec -it "+podName+" -- chroot /hostroot /bin/bash", false, "KUBECONFIG="+getKubeConfigOfClusterType(targetKind))
	if err != nil {
		return err
	}

	err = client.CoreV1().Pods(namespace).Delete(podName, &metav1.DeleteOptions{})
	return
}

func apply(client kubernetes.Interface, desired *corev1.Pod) (err error) {
	namespace := desired.Namespace
	current, err := client.CoreV1().Pods(namespace).Get(desired.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Pods(namespace).Create(desired)
		}
	} else {
		// Update the container image
		current.Spec.Containers[0].Image = desired.Spec.Containers[0].Image
		_, err = client.CoreV1().Pods(namespace).Update(current)
	}
	return
}

func buildRootPod(name, namespace, image, hostname string) *corev1.Pod {
	privileged := true
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:    "root-container",
					Image:   image,
					Command: []string{"sleep", "10000000"},
					Stdin:   true,
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					VolumeMounts: []corev1.VolumeMount{
						corev1.VolumeMount{
							Name:      "root-volume",
							MountPath: "/hostroot",
						},
					},
				},
			},
			HostNetwork: true,
			HostPID:     true,
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": hostname,
			},
			Tolerations: []corev1.Toleration{
				corev1.Toleration{
					Key:      "node-role.kubernetes.io/master",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "root-volume",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/",
						},
					},
				},
			},
		},
	}
}
