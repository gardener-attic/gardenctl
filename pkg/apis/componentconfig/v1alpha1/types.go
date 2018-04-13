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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClientConnectionConfiguration contains details for constructing a client.
type ClientConnectionConfiguration struct {
	// KubeConfigFile is the path to a kubeconfig file.
	KubeConfigFile string `json:"kubeconfig"`
	// AcceptContentTypes defines the Accept header sent by clients when connecting to
	// a server, overriding the default value of 'application/json'. This field will
	// control all connections to the server used by a particular client.
	AcceptContentTypes string `json:"acceptContentTypes"`
	// ContentType is the content type used when sending data to the server from this
	// client.
	ContentType string `json:"contentType"`
	// QPS controls the number of queries per second allowed for this connection.
	QPS float32 `json:"qps"`
	// Burst allows extra queries to accumulate when a client is exceeding its rate.
	Burst int32 `json:"burst"`
}

// ControllerReconciliationConfiguration contains details for the reconciliation
// settings of a controller.
type ControllerReconciliationConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
	// ConcurrentSyncs is the duration how often the caches of existing resources
	// are reconciled.
	ResyncPeriod metav1.Duration `json:"resyncPeriod"`
	// RetryDuration is the maximum duration how often a reconciliation will be retried
	// in case of errors.
	RetryDuration *metav1.Duration `json:"retryDuration"`
}

// LeaderElectionConfiguration defines the configuration of leader election
// clients for components that can run with leader election enabled.
type LeaderElectionConfiguration struct {
	// LeaderElect enables a leader election client to gain leadership
	// before executing the main loop. Enable this when running replicated
	// components for high availability.
	LeaderElect bool `json:"leaderElect"`
	// LeaseDuration is the duration that non-leader candidates will wait
	// after observing a leadership renewal until attempting to acquire
	// leadership of a led but unrenewed leader slot. This is effectively the
	// maximum duration that a leader can be stopped before it is replaced
	// by another candidate. This is only applicable if leader election is
	// enabled.
	LeaseDuration metav1.Duration `json:"leaseDuration"`
	// RenewDeadline is the interval between attempts by the acting master to
	// renew a leadership slot before it stops leading. This must be less
	// than or equal to the lease duration. This is only applicable if leader
	// election is enabled.
	RenewDeadline metav1.Duration `json:"renewDeadline"`
	// RetryPeriod is the duration the clients should wait between attempting
	// acquisition and renewal of a leadership. This is only applicable if
	// leader election is enabled.
	RetryPeriod metav1.Duration `json:"retryPeriod"`
	// ResourceLock indicates the resource object type that will be used to lock
	// during leader election cycles.
	ResourceLock string `json:"resourceLock"`
	// LockObjectNamespace defines the namespace of the lock object.
	LockObjectNamespace string `json:"lockObjectNamespace"`
	// LockObjectName defines the lock object name.
	LockObjectName string `json:"lockObjectName"`
}

// ServerConfiguration contains details for the HTTP server.
type ServerConfiguration struct {
	// BindAddress is the IP address on which to listen for the specified port.
	BindAddress string `json:"bindAddress"`
	// Port is the port on which to serve unsecured, unauthenticated access.
	Port int `json:"port"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GardenOperatorConfiguration defines the configuration for the Garden Operator.
type GardenOperatorConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection ClientConnectionConfiguration `json:"clientConnection"`
	// Controller defines the configuration of the controllers.
	Controller GardenOperatorControllerConfiguration `json:"controller"`
	// Constraints specifies the constraints applied to the Garden Operator.
	Constraints GardenConstraints `json:"constraints"`
	// GardenNamespace is the namespace in which the configuration and secrets for
	// the Garden Operator will be stored (e.g., secrets for the Seed clusters).
	// Not specifying this field means the same namespace the Garden Operator is
	// running in (only reasonable when the Garden Operator runs inside a Kubernetes
	// cluster).
	GardenNamespace string `json:"gardenNamespace"`
	// Images is a list of container images which are deployed by the Garden Operator.
	Images []GardenOperatorImagesConfiguration `json:"images"`
	// LeaderElection defines the configuration of leader election client.
	LeaderElection LeaderElectionConfiguration `json:"leaderElection"`
	// LogLevel is the level/severity for the logs. Must be one of [`info`,`debug`,
	// `error`].
	LogLevel string `json:"logLevel"`
	// Server defines the configuration of the HTTP server.
	Server ServerConfiguration `json:"server"`
}

// GardenConstraints contains all constraints enforced by the Garden Operator.
type GardenConstraints struct {
	// CloudProviders is a list of cloud provider specific configurations and allowed
	// values (e.g., for machine types).
	CloudProviders []GardenOperatorCloudProviderConfiguration `json:"cloudProviders"`
	// KubernetesVersions is the list of Kubernetes versions which are supported by
	// the Garden Operator.
	KubernetesVersions []string `json:"kubernetesVersions"`
	// DNSProviders is the list of valid values for the .spec.dns.kind fields.
	DNSProviders []string `json:"dnsProviders"`
}

// GardenOperatorCloudProviderConfiguration contains configuration for the different cloud
// providers which are supported by the Garden Operator.
type GardenOperatorCloudProviderConfiguration struct {
	// Name is the name of the cloud provider.
	Name string `json:"name"`
	// VolumeTypes is a list of valid values for the .spec.workers[*].volumeType fields in a
	// Shoot cluster specification.
	VolumeTypes []string `json:"volumeTypes"`
	// MachineTypes is a list of valid values for the .spec.workers[*].machineType fields in a
	// Shoot cluster specification.
	MachineTypes []string `json:"machineTypes"`
	// RootCerts
	RootCerts string `json:"rootCerts,omitempty"`
	// RootCerts
	LoadBalancerProvider string `json:"loadBalancerProvider,omitempty"`
	// RootCerts
	FloatingPoolName string `json:"floatingPoolName,omitempty"`
}

// GardenOperatorControllerConfiguration contains configuration for the controllers
// of the Garden Operator. Not only the usual reconciliation configuration is reflected,
// but also a health check period and a namespace which should be watched.
type GardenOperatorControllerConfiguration struct {
	// HealthCheckPeriod is the duration how often the health check of Shoot clusters
	// is performed (only if no operation is already running on them).
	HealthCheckPeriod metav1.Duration `json:"healthCheckPeriod"`
	// Reconciliation defines the reconciliation settings of the controllers.
	Reconciliation ControllerReconciliationConfiguration `json:"reconciliation"`
	// WatchNamespace defines the namespace which should be watched by the controller.
	WatchNamespace *string `json:"watchNamespace"`
}

// GardenOperatorImagesConfiguration contains configuration for the contaimer images and
// tags/versions which are used by the Garden Operator.
type GardenOperatorImagesConfiguration struct {
	// Name is an alias for the image.
	Name string `json:"name"`
	// Image is the name of the image (registry location and used tag/version).
	Image string `json:"image"`
}

const (
	// GardenOperatorDefaultLockObjectNamespace is the default lock namespace for leader election.
	GardenOperatorDefaultLockObjectNamespace string = "garden"

	// GardenOperatorDefaultLockObjectName is the default lock name for leader election.
	GardenOperatorDefaultLockObjectName = "gardenctl-leader-election"
)
