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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerManagerConfiguration defines the configuration for the Gardener controller manager.
type ControllerManagerConfiguration struct {
	metav1.TypeMeta `json:",inline"`
	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection ClientConnectionConfiguration `json:"clientConnection"`
	// GardenerClientConnection specifies the kubeconfig file and client connection
	// settings for the garden-apiserver.
	// +optional
	GardenerClientConnection *ClientConnectionConfiguration `json:"gardenerClientConnection,omitempty"`
	// Controllers defines the configuration of the controllers.
	Controllers ControllerManagerControllerConfiguration `json:"controllers"`
	// LeaderElection defines the configuration of leader election client.
	LeaderElection LeaderElectionConfiguration `json:"leaderElection"`
	// LogLevel is the level/severity for the logs. Must be one of [info,debug,error].
	LogLevel string `json:"logLevel"`
	// Metrics defines the metrics configuration.
	Metrics MetricsConfiguration `json:"metrics"`
	// Server defines the configuration of the HTTP server.
	Server ServerConfiguration `json:"server"`
}

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

// ControllerManagerControllerConfiguration defines the configuration of the controllers.
type ControllerManagerControllerConfiguration struct {
	// CloudProfile defines the configuration of the CloudProfile controller.
	// +optional
	CloudProfile *CloudProfileControllerConfiguration `json:"cloudProfile,omitempty"`
	// SecretBinding defines the configuration of the SecretBinding controller.
	// +optional
	SecretBinding *SecretBindingControllerConfiguration `json:"secretBinding,omitempty"`
	// Quota defines the configuration of the Quota controller.
	// +optional
	Quota *QuotaControllerConfiguration `json:"quota,omitempty"`
	// Seed defines the configuration of the Seed controller.
	// +optional
	Seed *SeedControllerConfiguration `json:"seed,omitempty"`
	// Shoot defines the configuration of the Shoot controller.
	Shoot ShootControllerConfiguration `json:"shoot"`
	// ShootCare defines the configuration of the ShootCare controller.
	ShootCare ShootCareControllerConfiguration `json:"shootCare"`
	// ShootMaintenance defines the configuration of the ShootMaintenance controller.
	ShootMaintenance ShootMaintenanceControllerConfiguration `json:"shootMaintenance"`
	// ShootQuota defines the configuration of the ShootQuota controller.
	ShootQuota ShootQuotaControllerConfiguration `json:"shootQuota"`
}

// CloudProfileControllerConfiguration defines the configuration of the CloudProfile
// controller.
type CloudProfileControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
}

// SecretBindingControllerConfiguration defines the configuration of the
// SecretBinding controller.
type SecretBindingControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
}

// QuotaControllerConfiguration defines the configuration of the Quota controller.
type QuotaControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
}

// SeedControllerConfiguration defines the configuration of the Seed controller.
type SeedControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
}

// ShootControllerConfiguration defines the configuration of the CloudProfile
// controller.
type ShootControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
	// RespectSyncPeriodOverwrite determines whether a sync period overwrite of a
	// Shoot (via annotation) is respected or not. Defaults to false.
	// +optional
	RespectSyncPeriodOverwrite *bool `json:"respectSyncPeriodOverwrite,omitempty"`
	// RetryDuration is the maximum duration how often a reconciliation will be retried
	// in case of errors.
	RetryDuration metav1.Duration `json:"retryDuration"`
	// RetrySyncPeriod is the duration how fast Shoots with an errornous operation are
	// readded to the queue so that the operation can be retried. Defaults to 15s.
	// +optional
	RetrySyncPeriod *metav1.Duration `json:"retrySyncPeriod,omitempty"`
	// SyncPeriod is the duration how often the existing resources are reconciled.
	SyncPeriod metav1.Duration `json:"syncPeriod"`
	// WatchNamespace defines the namespace which should be watched by the controller.
	// +optional
	WatchNamespace *string `json:"watchNamespace,omitempty"`
}

// ShootCareControllerConfiguration defines the configuration of the ShootCare
// controller.
type ShootCareControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
	// SyncPeriod is the duration how often the existing resources are reconciled (how
	// often the health check of Shoot clusters is performed (only if no operation is
	// already running on them).
	SyncPeriod metav1.Duration `json:"syncPeriod"`
}

// ShootMaintenanceControllerConfiguration defines the configuration of the
// ShootMaintenance controller.
type ShootMaintenanceControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
	// SyncPeriod is the duration how often the existing resources are reconciled (how
	// often it is checked whether Shoot resources need maintenance).
	SyncPeriod metav1.Duration `json:"syncPeriod"`
}

// ShootQuotaControllerConfiguration defines the configuration of the
// ShootQuota controller.
type ShootQuotaControllerConfiguration struct {
	// ConcurrentSyncs is the number of workers used for the controller to work on
	// events.
	ConcurrentSyncs int `json:"concurrentSyncs"`
	// SyncPeriod is the duration how often the existing resources are reconciled
	// (how often Shoots referenced Quota is checked).
	SyncPeriod metav1.Duration `json:"syncPeriod"`
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

// MetricsConfiguration contains options to configure the metrics.
type MetricsConfiguration struct {
	// The interval defines how frequently metrics get scraped.
	Interval metav1.Duration `json:"interval"`
}

// ServerConfiguration contains details for the HTTP server.
type ServerConfiguration struct {
	// BindAddress is the IP address on which to listen for the specified port.
	BindAddress string `json:"bindAddress"`
	// Port is the port on which to serve unsecured, unauthenticated access.
	Port int `json:"port"`
}

const (
	// ControllerManagerDefaultLockObjectNamespace is the default lock namespace for leader election.
	ControllerManagerDefaultLockObjectNamespace = "garden"

	// ControllerManagerDefaultLockObjectName is the default lock name for leader election.
	ControllerManagerDefaultLockObjectName = "gardener-controller-manager-leader-election"
)
