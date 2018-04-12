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

package botanist

import (
	"time"

	"github.com/gardener/gardenctl/pkg/garden"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitUntilKubeAPIServerServiceIsReady waits until the external load balancer of the kube-apiserver has
// been created (i.e., its ingress information has been updated in the service status).
func (b *Botanist) WaitUntilKubeAPIServerServiceIsReady() error {
	var e error
	err := wait.PollImmediate(5*time.Second, 600*time.Second, func() (bool, error) {
		loadBalancerIngress, serviceStatusIngress, err := garden.GetLoadBalancerIngress(b.K8sSeedClient, b.ShootNamespace, "kube-apiserver")
		if err != nil {
			e = err
			b.Logger.Info("Waiting until the kube-apiserver service is ready...")
			return false, nil
		}
		b.APIServerAddress = loadBalancerIngress
		b.APIServerIngresses = serviceStatusIngress
		return true, nil
	})
	if err != nil {
		return e
	}
	return nil
}

// WaitUntilKubeAPIServerIsReady waits until the kube-apiserver pod has a condition in its status which
// marks that it is ready.
func (b *Botanist) WaitUntilKubeAPIServerIsReady() error {
	return wait.PollImmediate(5*time.Second, 300*time.Second, func() (bool, error) {
		podList, err := b.
			K8sSeedClient.
			ListPods(b.ShootNamespace, metav1.ListOptions{
				LabelSelector: "app=kubernetes,role=apiserver",
			})
		if err != nil {
			return false, err
		}
		if len(podList.Items) == 0 {
			b.Logger.Info("Waiting until the kube-apiserver deployment gets created...")
			return false, nil
		}

		apiserver := &podList.Items[len(podList.Items)-1]
		for _, containerStatus := range apiserver.Status.ContainerStatuses {
			if containerStatus.Name == "kube-apiserver" && containerStatus.Ready {
				return true, nil
			}
		}
		b.Logger.Info("Waiting until the kube-apiserver deployment is ready...")
		return false, nil
	})
}

// WaitUntilVPNConnectionExists waits until a port forward connection to the vpn-shoot pod in the kube-system
// namespace of the Shoot cluster can be established.
func (b *Botanist) WaitUntilVPNConnectionExists() error {
	return wait.PollImmediate(5*time.Second, 900*time.Second, func() (bool, error) {
		var vpnPod *corev1.Pod
		podList, err := b.
			K8sShootClient.
			ListPods(metav1.NamespaceSystem, metav1.ListOptions{
				LabelSelector: "app=vpn-shoot",
			})
		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodRunning {
				vpnPod = &pod
				break
			}
		}
		if vpnPod == nil {
			b.Logger.Info("Waiting until a running vpn-shoot pod exists in the Shoot cluster...")
			return false, nil
		}
		ok, err := b.
			K8sShootClient.
			CheckForwardPodPort(vpnPod.ObjectMeta.Namespace, vpnPod.ObjectMeta.Name, 0, 22)
		if err == nil && ok {
			b.Logger.Info("VPN connection has been established.")
			return true, nil
		}
		b.Logger.Info("Waiting until the VPN connection has been established...")
		return false, nil
	})
}

// WaitUntilNamespaceDeleted waits until the namespace of the Shoot cluster within the Seed cluster is deleted.
func (b *Botanist) WaitUntilNamespaceDeleted() error {
	return wait.PollImmediate(5*time.Second, 900*time.Second, func() (bool, error) {
		_, err := b.
			K8sSeedClient.
			GetNamespace(b.ShootNamespace)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		b.Logger.Info("Waiting until the Shoot namespace has been cleaned up and deleted in the Seed cluster...")
		return false, nil
	})
}

// WaitUntilKubeAddonManagerDeleted waits until the kube-addon-manager deployment within the Seed cluster has
// been deleted.
func (b *Botanist) WaitUntilKubeAddonManagerDeleted() error {
	return wait.PollImmediate(5*time.Second, 600*time.Second, func() (bool, error) {
		_, err := b.
			K8sSeedClient.
			GetDeployment(b.ShootNamespace, "kube-addon-manager")
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		b.Logger.Info("Waiting until the kube-addon-manager has been deleted in the Seed cluster...")
		return false, nil
	})
}
