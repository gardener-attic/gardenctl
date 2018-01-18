// Copyright 2018 The Gardener Authors.
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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ShootSingular is the singular form of the Shoot resource.
	ShootSingular = "Shoot"

	// ShootPlural is the plural form of the Shoot resource.
	ShootPlural = "Shoots"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Shoot represents a Shoot cluster whose control plane is deployed in a Seed cluster.
type Shoot struct {
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	metav1.TypeMeta   `json:",inline"`
	// +optional
	Spec ShootSpec `json:"spec,omitempty"`
	// +optional
	Status ShootStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ShootList is a collection of Shoots.
type ShootList struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Shoot `json:"items"`
}

// ShootSpec is the specification of a Shoot cluster.
type ShootSpec struct {
	// +optional
	Addons *Addons `json:"addons,omitempty"`
	// +optional
	Backup *Backup `json:"backup,omitempty"`
	// +optional
	DNS *DNS `json:"dns,omitempty"`
	// +optional
	Infrastructure *Infrastructure `json:"infrastructure,omitempty"`
	// +optional
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// +optional
	Networks *Networks `json:"networks,omitempty"`
	// +optional
	SeedName string `json:"seedName,omitempty"`
	// +optional
	Workers []Worker `json:"workers,omitempty"`
	// +optional
	Zones []Zone `json:"zones,omitempty"`
}

// ShootStatus represents the current state of a Shoot cluster.
type ShootStatus struct {
	// +optional
	Conditions []ShootCondition `json:"conditions,omitempty"`
	// +optional
	GardenOperator GardenOperator `json:"gardenOperator,omitempty"`
	// +optional
	LastOperation *LastOperation `json:"lastOperation,omitempty"`
	// +optional
	LastError string `json:"lastError,omitempty"`
	// +optional
	OperationStartTime *metav1.Time `json:"operationStartTime,omitempty"`
	// +optional
	UID types.UID `json:"uid,omitempty"`
}

/////////////////////////////////////
// Shoot Specification Definitions //
/////////////////////////////////////

// Addons is a collection of configuration for specific addons which are managed by the Garden operator.
type Addons struct {
	// +optional
	Kube2IAM Kube2IAM `json:"kube2iam,omitempty"`
	// +optional
	Heapster Heapster `json:"heapster,omitempty"`
	// +optional
	KubernetesDashboard KubernetesDashboard `json:"kubernetes-dashboard,omitempty"`
	// +optional
	ClusterAutoscaler ClusterAutoscaler `json:"cluster-autoscaler,omitempty"`
	// +optional
	NginxIngress NginxIngress `json:"nginx-ingress,omitempty"`
	// +optional
	Monocular Monocular `json:"monocular,omitempty"`
	// +optional
	KubeLego KubeLego `json:"kube-lego,omitempty"`
}

// Addon also enabling or disabling a specific addon and is used to derive from.
type Addon struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`
}

// HelmTiller describes configuration values for the helm-tiller addon.
type HelmTiller struct {
	Addon
}

// Heapster describes configuration values for the heapster addon.
type Heapster struct {
	Addon
}

// KubernetesDashboard describes configuration values for the kubernetes-dashboard addon.
type KubernetesDashboard struct {
	Addon
}

// ClusterAutoscaler describes configuration values for the cluster-autoscaler addon.
type ClusterAutoscaler struct {
	Addon
}

// NginxIngress describes configuration values for the nginx-ingress addon.
type NginxIngress struct {
	Addon
}

// Monocular describes configuration values for the monocular addon.
type Monocular struct {
	Addon
}

// KubeLego describes configuration values for the kube-lego addon.
type KubeLego struct {
	Addon

	// +optional
	Mail string `json:"email,omitempty"`
}

// Kube2IAM describes configuration values for the kube2iam addon.
type Kube2IAM struct {
	Addon
	// +optional
	Roles []Kube2IAMRole `json:"roles,omitempty"`
}

// Kube2IAMRole allows passing AWS IAM policies which will result in IAM roles.
type Kube2IAMRole struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Policy      string `json:"policy"`
}

// Backup holds information about the backup interval and maximum.
type Backup struct {
	// +optional
	IntervalInSecond int `json:"intervalInSecond,omitempty"`
	// +optional
	Maximum int `json:"maximum,omitempty"`
}

// DNS holds information about the kind, the hosted zone id and the domain.
type DNS struct {
	// +optional
	Kind DNSKind `json:"kind,omitempty"`
	// +optional
	HostedZoneID string `json:"hostedZoneID,omitempty"`
	// +optional
	Domain string `json:"domain,omitempty"`
}

// DNSKind is a string alias.
type DNSKind string

const (
	// DNSUnmanaged is a constant for the 'unmanaged' DNS kind.
	DNSUnmanaged DNSKind = "unmanaged"

	// DNSAWS is a constant for the 'aws' DNS kind.
	DNSAWS DNSKind = "aws"
)

// Infrastructure holds information about the kind, the region, the credentials, and any cloud provider
// specific configuration.
type Infrastructure struct {
	// +optional
	Kind CloudProvider `json:"kind,omitempty"`
	// +optional
	Region string `json:"region,omitempty"`
	// +optional
	Secret string `json:"secret,omitempty"`
	// +optional
	RootCerts string `json:"rootCerts,omitempty"`

	// Azure specifics

	// +optional
	ResourceGroupName string `json:"resourceGroupName,omitempty"`
	// +optional
	VNet *VNet `json:"vnet,omitempty"`
	// +optional
	CountFaultDomains int `json:"countFaultDomains,omitempty"`
	// +optional
	CountUpdateDomains int `json:"countUpdateDomains,omitempty"`

	// AWS specifics

	// +optional
	VPC *VPC `json:"vpc,omitempty"`

	// OpenStack specifics

	// +optional
	LoadBalancerNetwork string `json:"loadBalancerNetwork,omitempty"`
	// +optional
	LoadBalancerProvider string `json:"loadBalancerProvider,omitempty"`
	// +optional
	FloatingPoolName string `json:"floatingPoolName,omitempty"`
	// +optional
	RouterID string `json:"routerID,omitempty"`
	// +optional
	AdditionalDNS []string `json:"AdditionalDNS,omitempty"`
}

// CloudProvider is a string alias.
type CloudProvider string

const (
	// CloudProviderAWS is a constant for the AWS cloud provider.
	CloudProviderAWS CloudProvider = "aws"

	// CloudProviderAzure is a constant for the Azure cloud provider.
	CloudProviderAzure CloudProvider = "azure"

	// CloudProviderGCE is a constant for the GCE cloud provider.
	CloudProviderGCE CloudProvider = "gce"

	// CloudProviderOpenStack is a constant for the OpenStack cloud provider.
	CloudProviderOpenStack CloudProvider = "openstack"
)

// CIDR is a string alias.
type CIDR string

// VNet holds information about a name or a CIDR.
type VNet struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	CIDR CIDR `json:"cidr,omitempty"`
}

// VPC holds information about an id or a CIDR.
type VPC struct {
	// +optional
	ID string `json:"id,omitempty"`
	// +optional
	CIDR CIDR `json:"cidr,omitempty"`
	// +optional
	Name string `json:"name,omitempty"`
}

// Networks holds information about the network ranges for all the different networks.
type Networks struct {
	// +optional
	Pods CIDR `json:"pods,omitempty"`
	// +optional
	Services CIDR `json:"services,omitempty"`
	// +optional
	Nodes CIDR `json:"nodes,omitempty"`
	// +optional
	Workers []CIDR `json:"workers,omitempty"`
	// +optional
	Public []CIDR `json:"public,omitempty"`
	// +optional
	Internal []CIDR `json:"internal,omitempty"`
}

// Worker holds information about a specific worker group.
type Worker struct {
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	MachineType string `json:"machineType,omitempty"`
	// +optional
	VolumeType string `json:"volumeType,omitempty"`
	// +optional
	VolumeSize string `json:"volumeSize,omitempty"`
	// +optional
	AutoScalerMin int `json:"autoScalerMin,omitempty"`
	// +optional
	AutoScalerMax int `json:"autoScalerMax,omitempty"`
}

// Zone is a string alias.
type Zone string

//////////////////////////////
// Shoot Status Definitions //
//////////////////////////////

// GardenOperator holds the information about the GardenOperator
type GardenOperator struct {
	// +optional
	Version string `json:"version,omitempty"`
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	ID string `json:"id,omitempty"`
}

// ShootCondition holds the information about the condition of the Shoot cluster
type ShootCondition struct {
	// +optional
	Type ShootConditionType `json:"type,omitempty"`
	// +optional
	Status corev1.ConditionStatus `json:"status,omitempty"`
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

// ShootConditionType is a string alias.
type ShootConditionType string

const (
	// ShootControlPlaneHealthy is a constant for a condition type indicating the control plane health.
	ShootControlPlaneHealthy ShootConditionType = "ControlPlaneHealthy"

	// ShootEveryNodeReady is a constant for a condition type indicating the node health.
	ShootEveryNodeReady ShootConditionType = "EveryNodeReady"

	// ShootSystemComponentsHealthy is a constant for a condition type indicating the system components health.
	ShootSystemComponentsHealthy ShootConditionType = "SystemComponentsHealthy"
)

// LastOperation indicates the type and the state of the last operation, along with a description
// message and a progress indicator.
type LastOperation struct {
	Description    string                  `json:"description,omitempty"`
	LastUpdateTime metav1.Time             `json:"lastUpdateTime,omitempty"`
	Progress       int                     `json:"progress,omitempty"`
	State          ShootLastOperationState `json:"state,omitempty"`
	Type           ShootLastOperationType  `json:"type,omitempty"`
}

// ShootLastOperationType is a string alias.
type ShootLastOperationType string

const (
	// ShootLastOperationTypeCreate indicates a 'create' operation.
	ShootLastOperationTypeCreate ShootLastOperationType = "Create"

	// ShootLastOperationTypeReconcile indicates a 'reconcile' operation.
	ShootLastOperationTypeReconcile ShootLastOperationType = "Reconcile"

	// ShootLastOperationTypeUpdate indicates an 'update' operation.
	ShootLastOperationTypeUpdate ShootLastOperationType = "Update"

	// ShootLastOperationTypeDelete indicates a 'delete' operation.
	ShootLastOperationTypeDelete ShootLastOperationType = "Delete"
)

// ShootLastOperationState is a string alias.
type ShootLastOperationState string

const (
	// ShootLastOperationStateProcessing indicates that an operation is ongoing.
	ShootLastOperationStateProcessing ShootLastOperationState = "Processing"

	// ShootLastOperationStateSucceeded indicates that an operation has completed successfully.
	ShootLastOperationStateSucceeded ShootLastOperationState = "Succeeded"

	// ShootLastOperationStateError indicates that an operation is completed with errors and will be retried.
	ShootLastOperationStateError ShootLastOperationState = "Error"

	// ShootLastOperationStateFailed indicates that an operation is completed with errors and won't be retried.
	ShootLastOperationStateFailed ShootLastOperationState = "Failed"
)
