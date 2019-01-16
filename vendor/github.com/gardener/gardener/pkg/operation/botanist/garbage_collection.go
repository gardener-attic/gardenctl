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
	"strings"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PerformGarbageCollectionSeed performs garbage collection in the Shoot namespace in the Seed cluster,
// i.e., it deletes old machine sets which have a desired=actual=0 replica count.
func (b *Botanist) PerformGarbageCollectionSeed() error {
	podList, err := b.K8sSeedClient.ListPods(b.Shoot.SeedNamespace, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if err := b.deleteStalePods(b.K8sSeedClient, podList); err != nil {
		return err
	}

	machineSetList, err := b.K8sSeedClient.Machine().MachineV1alpha1().MachineSets(b.Shoot.SeedNamespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, machineSet := range machineSetList.Items {
		if machineSet.Spec.Replicas == 0 && machineSet.Status.Replicas == 0 {
			b.Logger.Debugf("Deleting MachineSet %s as the number of desired and actual replicas is 0.", machineSet.Name)
			err := b.K8sSeedClient.Machine().MachineV1alpha1().MachineSets(machineSet.Namespace).Delete(machineSet.Name, nil)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return err
			}
		}
	}
	return nil
}

// PerformGarbageCollectionShoot performs garbage collection in the kube-system namespace in the Shoot
// cluster, i.e., it deletes evicted pods (mitigation for https://github.com/kubernetes/kubernetes/issues/55051).
func (b *Botanist) PerformGarbageCollectionShoot() error {
	podList, err := b.K8sShootClient.ListPods(metav1.NamespaceSystem, metav1.ListOptions{})
	if err != nil {
		return err
	}

	return b.deleteStalePods(b.K8sShootClient, podList)
}

func (b *Botanist) deleteStalePods(client kubernetes.Interface, podList *corev1.PodList) error {
	var result error

	for _, pod := range podList.Items {
		if strings.Contains(pod.Status.Reason, "Evicted") {
			b.Logger.Debugf("Deleting pod %s as its reason is %s.", pod.Name, pod.Status.Reason)
			if err := client.DeletePod(pod.Namespace, pod.Name); err != nil && !apierrors.IsNotFound(err) {
				result = multierror.Append(result, err)
			}
			continue
		}

		if common.ShouldObjectBeRemoved(&pod, common.GardenerDeletionGracePeriod) {
			b.Logger.Debugf("Deleting stuck terminating pod %q", pod.Name)
			if err := client.DeletePodForcefully(pod.Namespace, pod.Name); err != nil && !apierrors.IsNotFound(err) {
				result = multierror.Append(result, err)
			}
		}
	}

	return result
}
