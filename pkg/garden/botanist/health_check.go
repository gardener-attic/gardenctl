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
	gardenv1 "github.com/gardener/gardenctl/pkg/apis/garden/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InitCondition initializes a new ShootCondition with an Unknown status.
func InitCondition(conditionType gardenv1.ShootConditionType, reason, message string) *gardenv1.ShootCondition {
	if reason == "" {
		reason = "ConditionInitialized"
	}
	if message == "" {
		message = "The condition has been initialized but its semantic check has not been performed yet."
	}
	return &gardenv1.ShootCondition{
		Type:               conditionType,
		Status:             corev1.ConditionUnknown,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
}

// ModifyCondition updates the properties of one specific condition.
func ModifyCondition(condition *gardenv1.ShootCondition, status corev1.ConditionStatus, reason, message string) *gardenv1.ShootCondition {
	var update = false
	if status != (*condition).Status {
		update = true
		(*condition).Status = status
	}
	if reason != (*condition).Reason {
		update = true
		(*condition).Reason = reason
	}
	if message != (*condition).Message {
		update = true
		(*condition).Message = message
	}
	if update {
		(*condition).LastTransitionTime = metav1.Now()
	}
	return condition
}

// CheckConditionControlPlaneHealthy checks whether the control plane of the Shoot cluster is healthy,
// i.e. whether all containers running in the relevant namespace in the Seed cluster are healthy.
func (b *Botanist) CheckConditionControlPlaneHealthy(condition *gardenv1.ShootCondition) *gardenv1.ShootCondition {
	response, err := b.
		K8sShootClient.
		Curl("healthz")
	if err != nil {
		return ModifyCondition(condition, corev1.ConditionFalse, "KubeAPIServerNotHealthy", "Could not reach Shoot cluster kube-apiserver's /healthz endpoint: '"+err.Error()+"'")
	}
	var statusCode int
	response.StatusCode(&statusCode)
	if statusCode < 200 || statusCode >= 400 {
		return ModifyCondition(condition, corev1.ConditionFalse, "KubeAPIServerNotHealthy", "Shoot cluster kube-apiserver's /healthz endpoint indicates unhealthiness.")
	}
	podList, err := b.
		K8sSeedClient.
		ListPods(b.ShootNamespace, metav1.ListOptions{})
	if err != nil {
		return ModifyCondition(condition, corev1.ConditionUnknown, "FetchPodListFailed", err.Error())
	}
	if len(podList.Items) < 5 {
		return ModifyCondition(condition, corev1.ConditionFalse, "ControlPlaneIncomplete", "The control plane in the Shoot namespace is incomplete (Pod's are missing).")
	}
	for _, pod := range podList.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Running == nil && containerStatus.State.Terminated != nil && containerStatus.State.Terminated.Reason != "Completed" {
				return ModifyCondition(condition, corev1.ConditionFalse, "ContainerNotRunning", "Container "+containerStatus.Name+" of pod "+pod.ObjectMeta.Name+" is not in running state")
			}
		}
		if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
			return ModifyCondition(condition, corev1.ConditionFalse, "PodNotRunning", "Pod "+pod.ObjectMeta.Name+" is in phase "+string(pod.Status.Phase))
		}
	}
	return ModifyCondition(condition, corev1.ConditionTrue, "AllContainersInRunningState", "All pods running the Shoot namespace in the Seed cluster are healthy.")
}

// CheckConditionEveryNodeReady checks whether every node registered at the Shoot cluster is in "Ready" state and
// that no node known to the IaaS is not registered to the Shoot's kube-apiserver.
func (b *Botanist) CheckConditionEveryNodeReady(condition *gardenv1.ShootCondition, currentlyScaling bool, healthyInstances int) *gardenv1.ShootCondition {
	nodeList, err := b.
		K8sShootClient.
		ListNodes(metav1.ListOptions{})
	if err != nil {
		return ModifyCondition(condition, corev1.ConditionUnknown, "FetchNodeListFailed", err.Error())
	}
	if !currentlyScaling && healthyInstances > len(nodeList.Items) {
		return ModifyCondition(condition, corev1.ConditionFalse, "NodeMissing", "At least one healthy node known to the IaaS provider but not registered to the cluster.")
	}
	for _, node := range nodeList.Items {
		for _, nodeCondition := range node.Status.Conditions {
			if nodeCondition.Type == corev1.NodeReady && nodeCondition.Status != corev1.ConditionTrue {
				return ModifyCondition(condition, corev1.ConditionFalse, "NodeNotReady", "Node "+node.ObjectMeta.Name+" is not ready.")
			}
		}
	}
	return ModifyCondition(condition, corev1.ConditionTrue, "EveryNodeReady", "Every node registered to the cluster is ready.")
}

// CheckConditionSystemComponentsHealthy checks whether every container in the kube-system namespace of the Shoot cluster is in "Running"
// state and that the number of available replicas per deployment matches the number of actual replicas (i.e., the number of desired pods
// matches the number of actual running pods).
func (b *Botanist) CheckConditionSystemComponentsHealthy(condition *gardenv1.ShootCondition) *gardenv1.ShootCondition {
	// Check whether the number of availableReplicas matches the number of desired replicas.
	deploymentList, err := b.
		K8sShootClient.
		ListDeployments(metav1.NamespaceSystem, metav1.ListOptions{})
	if err != nil {
		return ModifyCondition(condition, corev1.ConditionUnknown, "FetchPodListFailed", err.Error())
	}
	for _, deployment := range deploymentList {
		if *deployment.Spec.Replicas != deployment.Status.AvailableReplicas {
			return ModifyCondition(condition, corev1.ConditionFalse, "NotAllPodsAvailable", "Deployment "+deployment.ObjectMeta.Name+" has not yet the desired number of available pods.")
		}
	}
	// Check whether the number of running containers matching the number of actual containers within the pods (i.e., everything is running).
	podList, err := b.
		K8sShootClient.
		ListPods(metav1.NamespaceSystem, metav1.ListOptions{})
	if err != nil {
		return ModifyCondition(condition, corev1.ConditionUnknown, "FetchPodListFailed", err.Error())
	}
	for _, pod := range podList.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Running == nil && containerStatus.State.Terminated != nil && containerStatus.State.Terminated.Reason != "Completed" {
				return ModifyCondition(condition, corev1.ConditionFalse, "ContainerNotRunning", "Container "+containerStatus.Name+" of pod "+pod.ObjectMeta.Name+" is not in running state.")
			}
		}
		if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
			return ModifyCondition(condition, corev1.ConditionFalse, "PodNotRunning", "Pod "+pod.ObjectMeta.Name+" is in phase "+string(pod.Status.Phase))
		}
	}
	return ModifyCondition(condition, corev1.ConditionTrue, "AllContainersInKubeSystemInRunningState", "Every container in the kube-system namespace of the Shoot cluster is running.")
}
