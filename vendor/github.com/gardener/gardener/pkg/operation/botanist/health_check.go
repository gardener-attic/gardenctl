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
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1/helper"
	machine "github.com/gardener/gardener/pkg/client/machine/clientset/versioned"
	"github.com/gardener/gardener/pkg/features"
	"github.com/gardener/gardener/pkg/operation/common"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/kubernetes/health"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	prometheusmodel "github.com/prometheus/common/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
)

func mustGardenRoleLabelSelector(gardenRoles ...string) labels.Selector {
	if len(gardenRoles) == 1 {
		labels.SelectorFromSet(map[string]string{common.GardenRole: gardenRoles[0]})
	}

	selector := labels.NewSelector()
	requirement, err := labels.NewRequirement(common.GardenRole, selection.In, gardenRoles)
	if err != nil {
		panic(err)
	}

	return selector.Add(*requirement)
}

var (
	controlPlaneSelector    = mustGardenRoleLabelSelector(common.GardenRoleControlPlane)
	systemComponentSelector = mustGardenRoleLabelSelector(common.GardenRoleSystemComponent)
	monitoringSelector      = mustGardenRoleLabelSelector(common.GardenRoleMonitoring)
	optionalAddonSelector   = mustGardenRoleLabelSelector(common.GardenRoleOptionalAddon)
	loggingSelector         = mustGardenRoleLabelSelector(common.GardenRoleLogging)
)

// Now determines the current time.
var Now = time.Now

// HealthChecker contains the condition thresholds.
type HealthChecker struct {
	conditionThresholds map[gardenv1beta1.ConditionType]time.Duration
}

func (b *HealthChecker) checkRequiredDeployments(condition *gardenv1beta1.Condition, requiredNames sets.String, objects []*appsv1.Deployment) *gardenv1beta1.Condition {
	actualNames := sets.NewString()
	for _, object := range objects {
		actualNames.Insert(object.Name)
	}

	if missingNames := requiredNames.Difference(actualNames); missingNames.Len() != 0 {
		return b.FailedCondition(
			condition,
			"DeploymentMissing",
			fmt.Sprintf("Missing required deployments: %v", missingNames.List()))
	}

	return nil
}

func (b *HealthChecker) checkDeployments(condition *gardenv1beta1.Condition, objects []*appsv1.Deployment) *gardenv1beta1.Condition {
	for _, object := range objects {
		if err := health.CheckDeployment(object); err != nil {
			return b.FailedCondition(
				condition,
				"DeploymentUnhealthy",
				fmt.Sprintf("Deployment %s is unhealthy: %v", object.Name, err.Error()))
		}
	}

	return nil
}

func (b *HealthChecker) checkRequiredStatefulSets(condition *gardenv1beta1.Condition, requiredNames sets.String, objects []*appsv1.StatefulSet) *gardenv1beta1.Condition {
	actualNames := sets.NewString()
	for _, object := range objects {
		actualNames.Insert(object.Name)
	}

	if missingNames := requiredNames.Difference(actualNames); missingNames.Len() != 0 {
		return b.FailedCondition(
			condition,
			"StatefulSetMissing",
			fmt.Sprintf("Missing required stateful sets: %v", missingNames.List()))
	}

	return nil
}

func (b *HealthChecker) checkStatefulSets(condition *gardenv1beta1.Condition, objects []*appsv1.StatefulSet) *gardenv1beta1.Condition {
	for _, object := range objects {
		if err := health.CheckStatefulSet(object); err != nil {
			return b.FailedCondition(
				condition,
				"StatefulSetUnhealthy",
				fmt.Sprintf("Stateful set %s is unhealthy: %v", object.Name, err.Error()))
		}
	}

	return nil
}

func (b *HealthChecker) checkNodes(condition *gardenv1beta1.Condition, objects []*corev1.Node) *gardenv1beta1.Condition {
	for _, object := range objects {
		if err := health.CheckNode(object); err != nil {
			return b.FailedCondition(
				condition,
				"NodeUnhealthy",
				fmt.Sprintf("Node %s is unhealthy: %v", object.Name, err))
		}
	}

	return nil
}

func (b *HealthChecker) checkMachineDeployments(condition *gardenv1beta1.Condition, objects []*machinev1alpha1.MachineDeployment) *gardenv1beta1.Condition {
	for _, object := range objects {
		if err := health.CheckMachineDeployment(object); err != nil {
			return b.FailedCondition(
				condition,
				"MachineDeploymentUnhealthy",
				fmt.Sprintf("Machine deployment %s is unhealthy: %v", object.Name, err))
		}
	}

	return nil
}

func (b *HealthChecker) checkRequiredDaemonSets(condition *gardenv1beta1.Condition, requiredNames sets.String, objects []*appsv1.DaemonSet) *gardenv1beta1.Condition {
	actualNames := sets.NewString()
	for _, object := range objects {
		actualNames.Insert(object.Name)
	}

	if missingNames := requiredNames.Difference(actualNames); missingNames.Len() != 0 {
		return b.FailedCondition(
			condition,
			"DaemonSetMissing",
			fmt.Sprintf("Missing required daemon sets: %v", missingNames.List()))
	}
	return nil
}

func (b *HealthChecker) checkDaemonSets(condition *gardenv1beta1.Condition, objects []*appsv1.DaemonSet) *gardenv1beta1.Condition {
	for _, object := range objects {
		if err := health.CheckDaemonSet(object); err != nil {
			return b.FailedCondition(
				condition,
				"DaemonSetUnhealthy",
				fmt.Sprintf("Daemon set %s is unhealthy: %v", object.Name, err.Error()))
		}
	}

	return nil
}

func shootHibernatedCondition(condition *gardenv1beta1.Condition) *gardenv1beta1.Condition {
	return helper.UpdatedCondition(condition, gardenv1beta1.ConditionTrue, "ConditionNotChecked", "Shoot cluster has been hibernated.")
}

// This is a hack to quickly do a cloud provider specific check for the required control plane deployments.
// As this will anyways change with the Gardener extensibility, for now, this will check for the only
// cloud provider where it differs (AWS). Once cloud provider specific code moves out, this will also have to
// be refactored / re-aligned.
func computeRequiredControlPlaneDeployments(shoot *gardenv1beta1.Shoot, seedCloudProvider gardenv1beta1.CloudProvider) (sets.String, error) {
	shootWantsClusterAutoscaler, err := helper.ShootWantsClusterAutoscaler(shoot)
	if err != nil {
		return nil, err
	}

	requiredControlPlaneDeployments := sets.NewString(common.RequiredControlPlaneDeployments.UnsortedList()...)
	if seedCloudProvider == gardenv1beta1.CloudProviderAWS {
		requiredControlPlaneDeployments.Insert(common.AWSLBReadvertiserDeploymentName)
	}

	if shootWantsClusterAutoscaler {
		requiredControlPlaneDeployments.Insert(common.ClusterAutoscalerDeploymentName)
	}

	return requiredControlPlaneDeployments, nil
}

// CheckControlPlane checks whether the control plane components in the given listers are complete and healthy.
func (b *HealthChecker) CheckControlPlane(
	shoot *gardenv1beta1.Shoot,
	namespace string,
	seedCloudProvider gardenv1beta1.CloudProvider,
	condition *gardenv1beta1.Condition,
	deploymentLister kutil.DeploymentLister,
	statefulSetLister kutil.StatefulSetLister,
) (*gardenv1beta1.Condition, error) {

	requiredControlPlaneDeployments, err := computeRequiredControlPlaneDeployments(shoot, seedCloudProvider)
	if err != nil {
		return nil, err
	}

	deployments, err := deploymentLister.Deployments(namespace).List(controlPlaneSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredDeployments(condition, requiredControlPlaneDeployments, deployments); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkDeployments(condition, deployments); exitCondition != nil {
		return exitCondition, nil
	}

	statefulSets, err := statefulSetLister.StatefulSets(namespace).List(controlPlaneSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredStatefulSets(condition, common.RequiredControlPlaneStatefulSets, statefulSets); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkStatefulSets(condition, statefulSets); exitCondition != nil {
		return exitCondition, nil
	}
	return nil, nil
}

// CheckSystemComponents checks whether the system components in the given listers are complete and healthy.
func (b *HealthChecker) CheckSystemComponents(
	namespace string,
	condition *gardenv1beta1.Condition,
	deploymentLister kutil.DeploymentLister,
	daemonSetLister kutil.DaemonSetLister,
) (*gardenv1beta1.Condition, error) {

	deploymentList, err := deploymentLister.Deployments(namespace).List(systemComponentSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredDeployments(condition, common.RequiredSystemComponentDeployments, deploymentList); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkDeployments(condition, deploymentList); exitCondition != nil {
		return exitCondition, nil
	}

	daemonSetList, err := daemonSetLister.DaemonSets(namespace).List(systemComponentSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredDaemonSets(condition, common.RequiredSystemComponentDaemonSets, daemonSetList); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkDaemonSets(condition, daemonSetList); exitCondition != nil {
		return exitCondition, nil
	}
	return nil, nil
}

// FailedCondition returns a progressing or false condition depending on the progressing threshold.
func (b *HealthChecker) FailedCondition(condition *gardenv1beta1.Condition, reason, message string) *gardenv1beta1.Condition {
	switch condition.Status {
	case gardenv1beta1.ConditionTrue:
		_, ok := b.conditionThresholds[condition.Type]
		if !ok {
			return helper.UpdatedCondition(condition, gardenv1beta1.ConditionFalse, reason, message)
		}

		return helper.UpdatedCondition(condition, gardenv1beta1.ConditionProgressing, reason, message)
	case gardenv1beta1.ConditionProgressing:
		threshold, ok := b.conditionThresholds[condition.Type]
		if !ok {
			return helper.UpdatedCondition(condition, gardenv1beta1.ConditionFalse, reason, message)
		}

		delta := Now().Sub(condition.LastTransitionTime.Time)
		if delta > threshold {
			return helper.UpdatedCondition(condition, gardenv1beta1.ConditionFalse, reason, message)
		}
		return helper.UpdatedCondition(condition, gardenv1beta1.ConditionProgressing, reason, message)
	}
	return helper.UpdatedCondition(condition, gardenv1beta1.ConditionFalse, reason, message)
}

// checkAPIServerAvailability checks if the API server of a Shoot cluster is reachable and measure the response time.
func (b *Botanist) checkAPIServerAvailability(checker *HealthChecker, condition *gardenv1beta1.Condition) *gardenv1beta1.Condition {
	// Try to reach the Shoot API server and measure the response time.
	now := Now()
	response := b.K8sShootClient.RESTClient().Get().AbsPath("/healthz").Do()
	responseDurationText := fmt.Sprintf("[response_time:%dms]", Now().Sub(now).Nanoseconds()/time.Millisecond.Nanoseconds())
	if response.Error() != nil {
		message := fmt.Sprintf("Request to Shoot API server /healthz endpoint failed. %s (%s)", responseDurationText, response.Error().Error())
		return checker.FailedCondition(condition, "HealthzRequestFailed", message)
	}

	// Determine the status code of the response.
	var statusCode int
	response.StatusCode(&statusCode)

	if statusCode != http.StatusOK {
		var body string
		bodyRaw, err := response.Raw()
		if err != nil {
			body = fmt.Sprintf("Could not parse response body: %s", err.Error())
		} else {
			body = string(bodyRaw)
		}
		message := fmt.Sprintf("Shoot API server /healthz endpoint endpoint check returned a non ok status code %d. %s (%s)", statusCode, responseDurationText, body)
		return checker.FailedCondition(condition, "HealthzRequestError", message)
	}

	message := fmt.Sprintf("Shoot API server /healthz endpoint responded with success status code. %s", responseDurationText)
	return helper.UpdatedCondition(condition, gardenv1beta1.ConditionTrue, "HealthzRequestSucceeded", message)
}

const (
	alertStatusFiring  = "firing"
	alertStatusPending = "pending"
	alertNameLabel     = "alertname"
	alertStateLabel    = "alertstate"
)

// checkAlerts checks whether firing or pending alerts exists by querying the Shoot Prometheus.
func (b *Botanist) checkAlerts(checker *HealthChecker, condition *gardenv1beta1.Condition) *gardenv1beta1.Condition {
	// Fetch firing and pending alerts from the Shoot cluster Prometheus.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	alertResultSet, err := b.MonitoringClient.Query(ctx, "ALERTS{alertstate=~'firing|pending'}", Now())
	if err != nil {
		return helper.UpdatedConditionUnknownErrorMessage(condition, fmt.Sprintf("Alerts can't be queried from Shoot Prometheus (%s).", err.Error()))
	}

	var (
		firingAlerts  []string
		pendingAlerts []string
	)

	switch alertResultSet.Type() {
	case prometheusmodel.ValVector:
		resultVector := alertResultSet.(prometheusmodel.Vector)
		for _, v := range resultVector {
			switch v.Metric[alertStateLabel] {
			case alertStatusFiring:
				firingAlerts = append(firingAlerts, string(v.Metric[alertNameLabel]))
			case alertStatusPending:
				pendingAlerts = append(pendingAlerts, string(v.Metric[alertNameLabel]))
			}
		}
	default:
		return helper.UpdatedConditionUnknownErrorMessage(condition, "Unexpected metrics format. Can't determine alerts.")
	}

	// Validate the alert results and update the conditions accordingly.
	var (
		message strings.Builder
		reason  string
		failed  bool
	)

	if len(firingAlerts) > 0 {
		reason = "FiringAlertsActive"
		failed = true
		message.WriteString(fmt.Sprintf("The following alerts are active: %v", strings.Join(firingAlerts, ", ")))
		if len(pendingAlerts) > 0 {
			reason = "FiringAndPendingAlertsActive"
		}
	} else {
		reason = "NoAlertsActive"
		failed = false
		message.WriteString("No active alerts")
		if len(pendingAlerts) > 0 {
			reason = "PendingAlertsActive"
		}
	}
	if len(pendingAlerts) > 0 {
		message.WriteString(fmt.Sprintf(". The following alerts might trigger soon: %v", strings.Join(pendingAlerts, ", ")))
	}
	if failed {
		return checker.FailedCondition(condition, reason, message.String())
	}
	return helper.UpdatedCondition(condition, gardenv1beta1.ConditionTrue, reason, message.String())
}

// CheckClusterNodes checks whether cluster nodes in the given listers are complete and healthy.
func (b *HealthChecker) CheckClusterNodes(
	namespace string,
	condition *gardenv1beta1.Condition,
	nodeLister kutil.NodeLister,
	machineDeploymentLister kutil.MachineDeploymentLister,
) (*gardenv1beta1.Condition, error) {
	nodeList, err := nodeLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkNodes(condition, nodeList); exitCondition != nil {
		return exitCondition, nil
	}

	machineDeploymentList, err := machineDeploymentLister.MachineDeployments(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	desiredMachines := 0
	for _, machineDeployment := range machineDeploymentList {
		if machineDeployment.DeletionTimestamp == nil {
			desiredMachines += int(machineDeployment.Spec.Replicas)
		}
	}

	if registeredNodes := len(nodeList); registeredNodes < desiredMachines {
		return b.FailedCondition(condition, "MissingNodes", fmt.Sprintf("Not enough worker nodes registered in the cluster (%d/%d).", registeredNodes, desiredMachines)), nil
	}
	if exitCondition := b.checkMachineDeployments(condition, machineDeploymentList); exitCondition != nil {
		return exitCondition, nil
	}
	return nil, nil
}

// CheckMonitoringSystemComponents checks whether the monitoring in the given listers are complete and healthy.
func (b *HealthChecker) CheckMonitoringSystemComponents(
	namespace string,
	condition *gardenv1beta1.Condition,
	daemonSetLister kutil.DaemonSetLister,
) (*gardenv1beta1.Condition, error) {

	daemonSetList, err := daemonSetLister.DaemonSets(namespace).List(monitoringSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredDaemonSets(condition, common.RequiredMonitoringShootDaemonSets, daemonSetList); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkDaemonSets(condition, daemonSetList); exitCondition != nil {
		return exitCondition, nil
	}
	return nil, nil
}

// CheckMonitoringControlPlane checks whether the monitoring in the given listers are complete and healthy.
func (b *HealthChecker) CheckMonitoringControlPlane(
	namespace string,
	condition *gardenv1beta1.Condition,
	deploymentLister kutil.DeploymentLister,
	statefulSetLister kutil.StatefulSetLister,
) (*gardenv1beta1.Condition, error) {

	deploymentList, err := deploymentLister.Deployments(namespace).List(monitoringSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredDeployments(condition, common.RequiredMonitoringSeedDeployments, deploymentList); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkDeployments(condition, deploymentList); exitCondition != nil {
		return exitCondition, nil
	}

	statefulSetList, err := statefulSetLister.StatefulSets(namespace).List(monitoringSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredStatefulSets(condition, common.RequiredMonitoringSeedStatefulSets, statefulSetList); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkStatefulSets(condition, statefulSetList); exitCondition != nil {
		return exitCondition, nil
	}
	return nil, nil
}

// CheckOptionalAddonsSystemComponents checks whether the addons in the given listers are healthy.
func (b *HealthChecker) CheckOptionalAddonsSystemComponents(
	namespace string,
	condition *gardenv1beta1.Condition,
	deploymentLister kutil.DeploymentLister,
	daemonSetLister kutil.DaemonSetLister,
) (*gardenv1beta1.Condition, error) {

	deploymentList, err := deploymentLister.Deployments(namespace).List(optionalAddonSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkDeployments(condition, deploymentList); exitCondition != nil {
		return exitCondition, nil
	}

	daemonSetList, err := daemonSetLister.DaemonSets(namespace).List(optionalAddonSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkDaemonSets(condition, daemonSetList); exitCondition != nil {
		return exitCondition, nil
	}
	return nil, nil
}

// CheckLoggingControlPlane checks whether the logging components in the given listers are complete and healthy.
func (b *HealthChecker) CheckLoggingControlPlane(
	namespace string,
	condition *gardenv1beta1.Condition,
	deploymentLister kutil.DeploymentLister,
	statefulSetLister kutil.StatefulSetLister,
) (*gardenv1beta1.Condition, error) {

	deploymentList, err := deploymentLister.Deployments(namespace).List(loggingSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredDeployments(condition, common.RequiredLoggingDeployments, deploymentList); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkDeployments(condition, deploymentList); exitCondition != nil {
		return exitCondition, nil
	}

	statefulSetList, err := statefulSetLister.StatefulSets(namespace).List(loggingSelector)
	if err != nil {
		return nil, err
	}

	if exitCondition := b.checkRequiredStatefulSets(condition, common.RequiredLoggingStatefulSets, statefulSetList); exitCondition != nil {
		return exitCondition, nil
	}
	if exitCondition := b.checkStatefulSets(condition, statefulSetList); exitCondition != nil {
		return exitCondition, nil
	}

	return nil, nil
}

// checkControlPlane checks whether the control plane of the Shoot cluster is healthy.
func (b *Botanist) checkControlPlane(
	checker *HealthChecker,
	condition *gardenv1beta1.Condition,
	seedDeploymentLister kutil.DeploymentLister,
	seedStatefulSetLister kutil.StatefulSetLister,
) (*gardenv1beta1.Condition, error) {

	if exitCondition, err := checker.CheckControlPlane(b.Shoot.Info, b.Shoot.SeedNamespace, b.Seed.CloudProvider, condition, seedDeploymentLister, seedStatefulSetLister); err != nil || exitCondition != nil {
		return exitCondition, err
	}
	if exitCondition, err := checker.CheckMonitoringControlPlane(b.Shoot.SeedNamespace, condition, seedDeploymentLister, seedStatefulSetLister); err != nil || exitCondition != nil {
		return exitCondition, err
	}
	if features.ControllerFeatureGate.Enabled(features.Logging) {
		if exitCondition, err := checker.CheckLoggingControlPlane(b.Shoot.SeedNamespace, condition, seedDeploymentLister, seedStatefulSetLister); err != nil || exitCondition != nil {
			return exitCondition, nil
		}
	}

	return helper.UpdatedCondition(condition, gardenv1beta1.ConditionTrue, "ControlPlaneRunning", "All control plane components are healthy."), nil
}

// checkSystemComponents checks whether the system components of a Shoot are running.
func (b *Botanist) checkSystemComponents(
	checker *HealthChecker,
	condition *gardenv1beta1.Condition,
	shootDeploymentLister kutil.DeploymentLister,
	shootDaemonSetLister kutil.DaemonSetLister,
) (*gardenv1beta1.Condition, error) {

	if exitCondition, err := checker.CheckSystemComponents(metav1.NamespaceSystem, condition, shootDeploymentLister, shootDaemonSetLister); err != nil || exitCondition != nil {
		return exitCondition, err
	}
	if exitCondition, err := checker.CheckMonitoringSystemComponents(metav1.NamespaceSystem, condition, shootDaemonSetLister); err != nil || exitCondition != nil {
		return exitCondition, err
	}
	if exitCondition, err := checker.CheckOptionalAddonsSystemComponents(metav1.NamespaceSystem, condition, shootDeploymentLister, shootDaemonSetLister); err != nil || exitCondition != nil {
		return exitCondition, err
	}

	return helper.UpdatedCondition(condition, gardenv1beta1.ConditionTrue, "SystemComponentsRunning", "All system components are healthy."), nil
}

// checkClusterNodes checks whether every node registered at the Shoot cluster is in "Ready" state, that
// as many nodes are registered as desired, and that every machine is running.
func (b *Botanist) checkClusterNodes(
	checker *HealthChecker,
	condition *gardenv1beta1.Condition,
	shootNodeLister kutil.NodeLister,
	seedMachineDeploymentLister kutil.MachineDeploymentLister,
) (*gardenv1beta1.Condition, error) {

	if exitCondition, err := checker.CheckClusterNodes(b.Shoot.SeedNamespace, condition, shootNodeLister, seedMachineDeploymentLister); err != nil || exitCondition != nil {
		return exitCondition, err
	}

	return helper.UpdatedCondition(condition, gardenv1beta1.ConditionTrue, "EveryNodeReady", "Every node registered to the cluster is ready"), nil
}

func makeDeploymentLister(clientset kubernetes.Interface, namespace string, options metav1.ListOptions) kutil.DeploymentLister {
	var (
		once  sync.Once
		items []*appsv1.Deployment
		err   error
	)

	return kutil.NewDeploymentLister(func() ([]*appsv1.Deployment, error) {
		once.Do(func() {
			var list *appsv1.DeploymentList
			list, err = clientset.AppsV1().Deployments(namespace).List(options)
			if err != nil {
				return
			}

			for _, item := range list.Items {
				it := item
				items = append(items, &it)
			}
		})
		return items, err
	})
}

func makeStatefulSetLister(clientset kubernetes.Interface, namespace string, options metav1.ListOptions) kutil.StatefulSetLister {
	var (
		once  sync.Once
		items []*appsv1.StatefulSet
		err   error

		onceBody = func() {
			var list *appsv1.StatefulSetList
			list, err = clientset.AppsV1().StatefulSets(namespace).List(options)
			if err != nil {
				return
			}

			for _, item := range list.Items {
				it := item
				items = append(items, &it)
			}
		}
	)

	return kutil.NewStatefulSetLister(func() ([]*appsv1.StatefulSet, error) {
		once.Do(onceBody)
		return items, err
	})
}

func makeDaemonSetLister(clientset kubernetes.Interface, namespace string, options metav1.ListOptions) kutil.DaemonSetLister {
	var (
		once  sync.Once
		items []*appsv1.DaemonSet
		err   error

		onceBody = func() {
			var list *appsv1.DaemonSetList
			list, err = clientset.AppsV1().DaemonSets(namespace).List(options)
			if err != nil {
				return
			}

			for _, item := range list.Items {
				it := item
				items = append(items, &it)
			}
		}
	)

	return kutil.NewDaemonSetLister(func() ([]*appsv1.DaemonSet, error) {
		once.Do(onceBody)
		return items, err
	})
}

func makeNodeLister(clientset kubernetes.Interface, options metav1.ListOptions) kutil.NodeLister {
	var (
		once  sync.Once
		items []*corev1.Node
		err   error

		onceBody = func() {
			var list *corev1.NodeList
			list, err = clientset.CoreV1().Nodes().List(options)
			if err != nil {
				return
			}

			for _, item := range list.Items {
				it := item
				items = append(items, &it)
			}
		}
	)

	return kutil.NewNodeLister(func() ([]*corev1.Node, error) {
		once.Do(onceBody)
		return items, err
	})
}

func makeMachineDeploymentLister(clientset machine.Interface, namespace string, options metav1.ListOptions) kutil.MachineDeploymentLister {
	var (
		once  sync.Once
		items []*machinev1alpha1.MachineDeployment
		err   error

		onceBody = func() {
			var list *machinev1alpha1.MachineDeploymentList
			list, err = clientset.MachineV1alpha1().MachineDeployments(namespace).List(options)
			if err != nil {
				return
			}

			for _, item := range list.Items {
				it := item
				items = append(items, &it)
			}
		}
	)

	return kutil.NewMachineDeploymentLister(func() ([]*machinev1alpha1.MachineDeployment, error) {
		once.Do(onceBody)
		return items, err
	})
}

func newConditionOrError(oldCondition, newCondition *gardenv1beta1.Condition, err error) *gardenv1beta1.Condition {
	if err != nil {
		return helper.UpdatedConditionUnknownError(oldCondition, err)
	}
	return newCondition
}

var (
	controlPlaneMonitoringLoggingSelector = mustGardenRoleLabelSelector(
		common.GardenRoleControlPlane,
		common.GardenRoleMonitoring,
		common.GardenRoleLogging,
	)
	systemComponentsOptionalAddonsSelector = mustGardenRoleLabelSelector(
		common.GardenRoleSystemComponent,
		common.GardenRoleOptionalAddon,
	)
	systemComponentsOptionalAddonsMonitoringSelector = mustGardenRoleLabelSelector(
		common.GardenRoleSystemComponent,
		common.GardenRoleOptionalAddon,
		common.GardenRoleMonitoring,
	)

	seedDeploymentListOptions        = metav1.ListOptions{LabelSelector: controlPlaneMonitoringLoggingSelector.String()}
	seedStatefulSetListOptions       = metav1.ListOptions{LabelSelector: controlPlaneMonitoringLoggingSelector.String()}
	seedMachineDeploymentListOptions = metav1.ListOptions{}

	shootDeploymentListOptions = metav1.ListOptions{LabelSelector: systemComponentsOptionalAddonsSelector.String()}
	shootDaemonSetListOptions  = metav1.ListOptions{LabelSelector: systemComponentsOptionalAddonsMonitoringSelector.String()}
	shootNodeListOptions       = metav1.ListOptions{}
)

// NewHealthChecker creates a new health checker.
func NewHealthChecker(conditionThresholds map[gardenv1beta1.ConditionType]time.Duration) *HealthChecker {
	return &HealthChecker{
		conditionThresholds: conditionThresholds,
	}
}

// HealthChecks conducts the health checks on all the given conditions.
func (b *Botanist) HealthChecks(initializeShootClients func() error, thresholdMappings map[gardenv1beta1.ConditionType]time.Duration, apiserverAvailability, controlPlane, nodes, systemComponents *gardenv1beta1.Condition) (*gardenv1beta1.Condition, *gardenv1beta1.Condition, *gardenv1beta1.Condition, *gardenv1beta1.Condition) {
	if b.Shoot.IsHibernated {
		return shootHibernatedCondition(apiserverAvailability), shootHibernatedCondition(controlPlane), shootHibernatedCondition(nodes), shootHibernatedCondition(systemComponents)
	}

	var (
		seedDeploymentLister  = makeDeploymentLister(b.K8sSeedClient.Kubernetes(), b.Shoot.SeedNamespace, seedDeploymentListOptions)
		seedStatefulSetLister = makeStatefulSetLister(b.K8sSeedClient.Kubernetes(), b.Shoot.SeedNamespace, seedStatefulSetListOptions)
		checker               = NewHealthChecker(thresholdMappings)
	)

	if err := initializeShootClients(); err != nil {
		message := fmt.Sprintf("Could not initialize Shoot client for health check: %+v", err)
		b.Logger.Error(message)
		apiserverAvailability = checker.FailedCondition(apiserverAvailability, "APIServerDown", "Could not reach API server during client initialization.")
		nodes = helper.UpdatedConditionUnknownErrorMessage(nodes, message)
		systemComponents = helper.UpdatedConditionUnknownErrorMessage(systemComponents, message)

		newControlPlane, err := b.checkControlPlane(checker, controlPlane, seedDeploymentLister, seedStatefulSetLister)
		controlPlane = newConditionOrError(controlPlane, newControlPlane, err)
		return apiserverAvailability, controlPlane, nodes, systemComponents
	}

	var (
		wg sync.WaitGroup

		seedMachineDeploymentLister = makeMachineDeploymentLister(b.K8sSeedClient.Machine(), b.Shoot.SeedNamespace, seedMachineDeploymentListOptions)

		shootDeploymentLister = makeDeploymentLister(b.K8sShootClient.Kubernetes(), metav1.NamespaceSystem, shootDeploymentListOptions)
		shootDaemonSetLister  = makeDaemonSetLister(b.K8sShootClient.Kubernetes(), metav1.NamespaceSystem, shootDaemonSetListOptions)
		shootNodeLister       = makeNodeLister(b.K8sShootClient.Kubernetes(), shootNodeListOptions)
	)

	wg.Add(4)
	go func() {
		defer wg.Done()
		apiserverAvailability = b.checkAPIServerAvailability(checker, apiserverAvailability)
	}()
	go func() {
		defer wg.Done()
		newControlPlane, err := b.checkControlPlane(checker, controlPlane, seedDeploymentLister, seedStatefulSetLister)
		controlPlane = newConditionOrError(controlPlane, newControlPlane, err)
	}()
	go func() {
		defer wg.Done()
		newNodes, err := b.checkClusterNodes(checker, nodes, shootNodeLister, seedMachineDeploymentLister)
		nodes = newConditionOrError(nodes, newNodes, err)
	}()
	go func() {
		defer wg.Done()
		newSystemComponents, err := b.checkSystemComponents(checker, systemComponents, shootDeploymentLister, shootDaemonSetLister)
		systemComponents = newConditionOrError(systemComponents, newSystemComponents, err)
	}()
	wg.Wait()

	return apiserverAvailability, controlPlane, nodes, systemComponents
}

// MonitoringHealthChecks performs the monitoring releated health checks.
func (b *Botanist) MonitoringHealthChecks(checker *HealthChecker, inactiveAlerts *gardenv1beta1.Condition) *gardenv1beta1.Condition {
	if b.Shoot.IsHibernated {
		return shootHibernatedCondition(inactiveAlerts)
	}
	if err := b.InitializeMonitoringClient(); err != nil {
		message := fmt.Sprintf("Could not initialize Shoot monitoring API client for health check: %+v", err)
		b.Logger.Error(message)
		return helper.UpdatedConditionUnknownErrorMessage(inactiveAlerts, message)
	}
	return b.checkAlerts(checker, inactiveAlerts)
}
