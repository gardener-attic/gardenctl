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
	"github.com/gardener/gardenctl/pkg/chartrenderer"
	"github.com/gardener/gardenctl/pkg/client/kubernetes"
	"github.com/gardener/gardenctl/pkg/garden"
	corev1 "k8s.io/api/core/v1"
)

const (
	// BackupSecretName defines the name of the secret containing the credentials which are required to
	// authenticate against the respective cloud provider (required by etcd-operator to store the backups
	// of Shoot clusters).
	BackupSecretName = "etcd-backup"
)

// Botanist is a struct which is initialized whenever an event on a Shoot resource has been triggered.
// It 'inherits' the attributes and methods from the Garden struct.
// * K8sSeedClient is a Kubernetes client for the Seed cluster.
// * K8sShootClient is a Kubernetes client for the Shoot cluster.
// * ChartSeedRenderer is a Helm chart renderer client respecting the Seed cluster's Kubernetes version.
// * ChartShootRenderer is a Helm chart renderer client respecting the Shoot cluster's Kubernetes version.
// * SeedFQDN is the FQDN for the Seed cluster.
// * APIServerIngresses is a list of ingresses (hostname or IP) which point to the Shoot's kube-apiserver Pod
//   in the Seed cluster.
// * APIServerAddress is the load balancer address (one of the APIServerIngress'es) which is actually used
//   for communicating with the kube-apiserver (and therefore also written in the kubeconfigs).
// * DefaultDomainSecret is a Kubernetes secret object which is only not nil if a default domain is used for the
//   Shoot's DNS domain.
type Botanist struct {
	*garden.Garden
	K8sSeedClient       kubernetes.Client
	K8sShootClient      kubernetes.Client
	ChartSeedRenderer   chartrenderer.ChartRenderer
	ChartShootRenderer  chartrenderer.ChartRenderer
	SeedFQDN            string
	APIServerIngresses  []corev1.LoadBalancerIngress
	APIServerAddress    string
	DefaultDomainSecret *corev1.Secret
}
