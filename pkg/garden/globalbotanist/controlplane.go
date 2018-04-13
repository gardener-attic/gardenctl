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

package globalbotanist

import (
	"fmt"
	"path/filepath"

	"github.com/gardener/gardenctl/pkg/garden/botanist"
	"github.com/gardener/gardenctl/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

// DeployETCD deploys two etcd clusters (either via StatefulSets or via the etcd-operator). The first etcd cluster
// (called 'main') is used for all the data the Shoot Kubernetes cluster needs to store, whereas the second etcd
// cluster (called 'events') is only used to store the events data. The objectstore is also set up to store the backups.
func (b *GlobalBotanist) DeployETCD() error {
	secretData, err := b.
		CloudBotanist.
		GenerateEtcdBackupSecretData()
	if err != nil {
		return err
	}
	_, err = b.
		Botanist.
		K8sSeedClient.
		CreateSecret(b.ShootNamespace, botanist.BackupSecretName, corev1.SecretTypeOpaque, secretData, true)
	if err != nil {
		return err
	}
	backupCloudConfig, err := b.
		CloudBotanist.
		GenerateEtcdConfig(botanist.BackupSecretName)
	if err != nil {
		return err
	}

	for _, role := range []string{EtcdRoleMain, EtcdRoleEvents} {
		backupCloudConfig["role"] = role
		err = b.
			Botanist.
			ApplyChartSeed(
				filepath.Join("charts", "seed-controlplane", "charts", "etcd"),
				fmt.Sprintf("etcd-%s", role),
				b.ShootNamespace,
				nil,
				backupCloudConfig,
			)
		if err != nil {
			return err
		}
	}
	return err
}

// DeployCloudProviderConfig asks the Cloud Botanist to provide the cloud specific values for the cloud
// provider configuration. It will create a ConfigMap for it and store it in the Seed cluster.
func (b *GlobalBotanist) DeployCloudProviderConfig() error {
	name := "cloud-provider-config"
	cloudProviderConfig, err := b.
		CloudBotanist.
		GenerateCloudProviderConfig()
	if err != nil {
		return err
	}
	b.Botanist.CheckSums[name] = utils.ComputeSHA256Sum([]byte(cloudProviderConfig))

	return b.
		Botanist.
		ApplyChartSeed(
			filepath.Join("charts", "seed-controlplane", "charts", name),
			name,
			b.ShootNamespace,
			nil,
			map[string]interface{}{
				"CloudProviderConfig": cloudProviderConfig,
			},
		)
}

// DeployKubeAPIServer asks the Cloud Botanist to provide the cloud specific configuration values for the
// kube-apiserver deployment.
func (b *GlobalBotanist) DeployKubeAPIServer() error {
	name := "kube-apiserver"
	loadBalancer := b.Botanist.APIServerAddress
	loadBalancerIP, err := utils.WaitUntilDNSNameResolvable(loadBalancer)
	if err != nil {
		return err
	}

	defaultValues := map[string]interface{}{
		"AdvertiseAddress":  loadBalancerIP,
		"CloudProvider":     b.Shoot.Spec.Infrastructure.Kind,
		"KubernetesVersion": b.Shoot.Spec.KubernetesVersion,
		"PodNetwork":        b.Shoot.Spec.Networks.Pods,
		"NodeNetwork":       b.Shoot.Spec.Networks.Nodes,
		"ServiceNetwork":    b.Shoot.Spec.Networks.Services,
		"PodAnnotations": map[string]interface{}{
			"checksum/secret-ca":                        b.CheckSums["ca"],
			"checksum/secret-kube-apiserver":            b.CheckSums[name],
			"checksum/secret-kube-apiserver-kubelet":    b.CheckSums["kube-apiserver-kubelet"],
			"checksum/secret-kube-apiserver-basic-auth": b.CheckSums["kube-apiserver-basic-auth"],
			"checksum/secret-vpn-ssh-keypair":           b.CheckSums["vpn-ssh-keypair"],
			"checksum/secret-infrastructure":            b.CheckSums["infrastructure"],
			"checksum/configmap-cloud-provider-config":  b.CheckSums["cloud-provider-config"],
		},
	}
	cloudValues, err := b.
		CloudBotanist.
		GenerateKubeAPIServerConfig()
	if err != nil {
		return err
	}

	return b.
		Botanist.
		ApplyChartSeed(
			filepath.Join("charts", "seed-controlplane", "charts", name),
			name,
			b.ShootNamespace,
			defaultValues,
			cloudValues,
		)
}

// DeployKubeControllerManager asks the Cloud Botanist to provide the cloud specific configuration values for the
// kube-controller-manager deployment.
func (b *GlobalBotanist) DeployKubeControllerManager() error {
	name := "kube-controller-manager"
	defaultValues := map[string]interface{}{
		"CloudProvider":     b.Shoot.Spec.Infrastructure.Kind,
		"ClusterName":       b.ShootNamespace,
		"KubernetesVersion": b.Shoot.Spec.KubernetesVersion,
		"PodNetwork":        b.Shoot.Spec.Networks.Pods,
		"ServiceNetwork":    b.Shoot.Spec.Networks.Services,
		"ConfigureRoutes":   true,
		"PodAnnotations": map[string]interface{}{
			"checksum/secret-ca":                       b.CheckSums["ca"],
			"checksum/secret-kube-apiserver":           b.CheckSums["kube-apiserver"],
			"checksum/secret-kube-controller-manager":  b.CheckSums[name],
			"checksum/secret-infrastructure":           b.CheckSums["infrastructure"],
			"checksum/configmap-cloud-provider-config": b.CheckSums["cloud-provider-config"],
		},
	}
	cloudValues, err := b.
		CloudBotanist.
		GenerateKubeControllerManagerConfig()
	if err != nil {
		return err
	}

	return b.
		Botanist.
		ApplyChartSeed(
			filepath.Join("charts", "seed-controlplane", "charts", name),
			name,
			b.ShootNamespace,
			defaultValues,
			cloudValues,
		)
}

// DeployKubeScheduler asks the Cloud Botanist to provide the cloud specific configuration values for the
// kube-scheduler deployment.
func (b *GlobalBotanist) DeployKubeScheduler() error {
	name := "kube-scheduler"
	defaultValues := map[string]interface{}{
		"KubernetesVersion": b.Shoot.Spec.KubernetesVersion,
		"PodAnnotations": map[string]interface{}{
			"checksum/secret-kube-scheduler": b.CheckSums[name],
		},
	}
	cloudValues, err := b.
		CloudBotanist.
		GenerateKubeSchedulerConfig()
	if err != nil {
		return err
	}

	return b.
		Botanist.
		ApplyChartSeed(
			filepath.Join("charts", "seed-controlplane", "charts", name),
			name,
			b.ShootNamespace,
			defaultValues,
			cloudValues,
		)
}
