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
	"fmt"
	"path/filepath"

	"github.com/gardener/gardenctl/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// DeployNamespace creates a namespace in the Seed cluster which is used to deploy all the control plane
// components for the Shoot cluster. Moreover, the cloud provider configuration and all the secrets will be
// stored as ConfigMaps/Secrets.
func (b *Botanist) DeployNamespace() error {
	_, err := b.
		K8sSeedClient.
		CreateNamespace(b.ShootNamespace, true)
	return err
}

// DeleteNamespace deletes the namespace in the Seed cluster which holds the control plane components. The built-in
// garbage collection in Kubernetes will automatically delete all resources which belong to this namespace. This
// comprises volumes and load balancers as well.
func (b *Botanist) DeleteNamespace() error {
	err := b.
		K8sSeedClient.
		DeleteNamespace(b.ShootNamespace)
	if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
		return nil
	}
	return err
}

// DeployETCDOperator deploys the etcd-operator which is used to spin up etcd clusters by leveraging the CRD concept.
func (b *Botanist) DeployETCDOperator() error {
	namespace, err := b.
		K8sSeedClient.
		GetNamespace(b.ShootNamespace)
	if err != nil {
		return err
	}
	imagePullSecrets := b.GetImagePullSecretsMap()

	return b.ApplyChartSeed(
		filepath.Join("charts", "seed-controlplane", "charts", "etcd-operator"),
		"etcd-operator",
		b.ShootNamespace,
		nil,
		map[string]interface{}{
			"imagePullSecrets": imagePullSecrets,
			"namespace": map[string]interface{}{
				"uid": namespace.ObjectMeta.UID,
			},
		},
	)
}

// DeployKubeAPIServerService creates a Service of type 'LoadBalancer' in the Seed cluster which is used to expose the
// kube-apiserver deployment (of the Shoot cluster). It waits until the load balancer is available and stores the address
// on the Botanist's APIServerAddress attribute.
func (b *Botanist) DeployKubeAPIServerService() error {
	return b.ApplyChartSeed(
		filepath.Join("charts", "seed-controlplane", "charts", "kube-apiserver-service"),
		"kube-apiserver-service",
		b.ShootNamespace,
		nil,
		map[string]interface{}{
			"CloudProvider": b.Shoot.Spec.Infrastructure.Kind,
		},
	)
}

// DeleteKubeAddonManager deletes the kube-addon-manager deployment in the Seed cluster which holds the control plane. It
// needs to be deleted before trying to remove any resources in the Shoot cluster, othwewise it will automatically recreate
// them and block the infrastructure deletion.
func (b *Botanist) DeleteKubeAddonManager() error {
	err := b.
		K8sSeedClient.
		DeleteDeployment(b.ShootNamespace, "kube-addon-manager")
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// DeploySeedMonitoring will install the Helm release "seed-monitoring" in the Seed clusters. It comprises components
// to monitor the Shoot cluster whose control plane runs in the Seed cluster.
func (b *Botanist) DeploySeedMonitoring() error {
	version, _, err := b.
		K8sShootClient.
		GetVersion()
	if err != nil {
		return err
	}

	namespace, err := b.
		K8sSeedClient.
		GetNamespace(b.ShootNamespace)
	if err != nil {
		return err
	}

	alertManagerHost, err := b.GetSeedIngressFQDN("a")
	if err != nil {
		return err
	}
	grafanaHost, err := b.GetSeedIngressFQDN("g")
	if err != nil {
		return err
	}
	prometheusHost, err := b.GetSeedIngressFQDN("p")
	if err != nil {
		return err
	}

	kubecfgSecret := b.Secrets["kubecfg"]
	basicAuth := utils.CreateSHA1Secret(kubecfgSecret.Data["username"], kubecfgSecret.Data["password"])
	imagePullSecrets := b.GetImagePullSecretsMap()

	values := map[string]interface{}{
		"global": map[string]interface{}{
			"ShootKubeVersion": map[string]interface{}{
				"GitVersion": version.GitVersion,
			},
		},
		"alertmanager": map[string]interface{}{
			"ingress": map[string]interface{}{
				"basicAuthSecret": basicAuth,
				"host":            alertManagerHost,
			},
		},
		"grafana": map[string]interface{}{
			"ingress": map[string]interface{}{
				"basicAuthSecret": basicAuth,
				"host":            grafanaHost,
			},
		},
		"prometheus": map[string]interface{}{
			"replicaCount": 1,
			"networks": map[string]interface{}{
				"pods":     b.Shoot.Spec.Networks.Pods,
				"services": b.Shoot.Spec.Networks.Services,
				"nodes":    b.Shoot.Spec.Networks.Nodes,
			},
			"ingress": map[string]interface{}{
				"basicAuthSecret": basicAuth,
				"host":            prometheusHost,
			},
			"imagePullSecrets": imagePullSecrets,
			"namespace": map[string]interface{}{
				"uid": namespace.ObjectMeta.UID,
			},
			"podAnnotations": map[string]interface{}{
				"checksum/secret-prometheus":                b.CheckSums["prometheus"],
				"checksum/secret-kube-apiserver-basic-auth": b.CheckSums["kube-apiserver-basic-auth"],
				"checksum/secret-vpn-ssh-keypair":           b.CheckSums["vpn-ssh-keypair"],
			},
		},
	}

	alertingSMTPKeys := b.GetSecretKeysOfKind("alerting-smtp")
	if len(alertingSMTPKeys) > 0 {
		emailConfigs := []map[string]interface{}{}
		for _, key := range alertingSMTPKeys {
			secret := b.Secrets[key]
			emailConfigs = append(emailConfigs, map[string]interface{}{
				"to":            string(secret.Data["to"]),
				"from":          string(secret.Data["from"]),
				"smarthost":     string(secret.Data["smarthost"]),
				"auth_username": string(secret.Data["auth_username"]),
				"auth_identity": string(secret.Data["auth_identity"]),
				"auth_password": string(secret.Data["auth_password"]),
			})
		}
		values["alertmanager"].(map[string]interface{})["email_configs"] = emailConfigs
	}

	return b.ApplyChartSeed(
		filepath.Join("charts", "seed-monitoring"),
		fmt.Sprintf("%s-monitoring", b.ShootNamespace),
		b.ShootNamespace,
		nil,
		values,
	)
}

// DeleteSeedMonitoring will delete the monitoring stack from the Seed cluster to avoid phantom alerts
// during the deletion process. More precisely, the Alertmanager and Prometheus StatefulSets will be
// deleted.
func (b *Botanist) DeleteSeedMonitoring() error {
	err := b.
		K8sSeedClient.
		DeleteStatefulSet(b.ShootNamespace, "alertmanager")
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	err = b.
		K8sSeedClient.
		DeleteStatefulSet(b.ShootNamespace, "prometheus")
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
