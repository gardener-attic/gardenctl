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
	"github.com/gardener/gardenctl/pkg/garden"
	"github.com/gardener/gardenctl/pkg/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeployNginxIngressResources creates the respective wildcard DNS record for the nginx-ingress-controller.
func (b *Botanist) DeployNginxIngressResources() error {
	loadBalancerIngress, _, err := garden.GetLoadBalancerIngress(b.K8sShootClient, metav1.NamespaceSystem, "addons-nginx-ingress-controller")
	if err != nil {
		return err
	}
	name, err := b.GetShootIngressFQDN("*")
	if err != nil {
		return err
	}
	return b.DeployDNSRecord("ingress", name, loadBalancerIngress, false)
}

// DestroyNginxIngressResources destroys the nginx-ingress resources created by Terraform.
func (b *Botanist) DestroyNginxIngressResources() error {
	return b.DestroyDNSRecord("ingress", false)
}

// GenerateNginxIngressConfig generates the values which are required to render the chart of
// the nginx-ingress-controller properly.
func (b *Botanist) GenerateNginxIngressConfig() (map[string]interface{}, error) {
	return garden.GenerateAddonConfig(nil, b.Shoot.Spec.Addons.NginxIngress.Enabled), nil
}

// GenerateKubernetesDashboardConfig generates the values which are required to render the chart of
// the kubernetes-dashboard properly.
func (b *Botanist) GenerateKubernetesDashboardConfig() (map[string]interface{}, error) {
	return garden.GenerateAddonConfig(nil, b.Shoot.Spec.Addons.KubernetesDashboard.Enabled), nil
}

// GenerateKubeLegoConfig generates the values which are required to render the chart of
// kube-lego properly.
func (b *Botanist) GenerateKubeLegoConfig() (map[string]interface{}, error) {
	var (
		enabled = b.Shoot.Spec.Addons.KubeLego.Enabled
		values  map[string]interface{}
	)

	if enabled {
		values = map[string]interface{}{
			"config": map[string]interface{}{
				"LEGO_EMAIL": b.Shoot.Spec.Addons.KubeLego.Mail,
			},
		}
	}

	return garden.GenerateAddonConfig(values, b.Shoot.Spec.Addons.KubeLego.Enabled), nil
}

// GenerateMonocularConfig generates the values which are required to render the chart of
// monocular properly.
func (b *Botanist) GenerateMonocularConfig() (map[string]interface{}, error) {
	var (
		enabled = b.Shoot.Spec.Addons.Monocular.Enabled
		values  map[string]interface{}
	)

	if enabled {
		monocularHost, err := b.GetShootIngressFQDN("monocular")
		if err != nil {
			return nil, err
		}
		kubecfgSecret := b.Secrets["kubecfg"]
		basicAuth := utils.CreateSHA1Secret(kubecfgSecret.Data["username"], kubecfgSecret.Data["password"])
		_, err = b.
			K8sShootClient.
			CreateSecret(metav1.NamespaceSystem, "monocular-tls", corev1.SecretTypeTLS, b.Secrets["monocular-tls"].Data, true)
		if err != nil {
			return nil, err
		}
		values = map[string]interface{}{
			"ingress": map[string]interface{}{
				"basicAuthSecret": basicAuth,
				"hosts":           []string{monocularHost},
			},
		}
	}

	return garden.GenerateAddonConfig(values, enabled), nil
}

// GenerateHeapsterConfig generates the values which are required to render the chart of
// heapster properly.
func (b *Botanist) GenerateHeapsterConfig() (map[string]interface{}, error) {
	var (
		enabled = b.Shoot.Spec.Addons.Heapster.Enabled
		values  map[string]interface{}
	)

	if enabled {
		addonManagerLabels := map[string]interface{}{
			"addonmanager.kubernetes.io/mode": "EnsureExists",
		}
		values = map[string]interface{}{
			"labels": addonManagerLabels,
			"service": map[string]interface{}{
				"labels": addonManagerLabels,
			},
		}
	}

	return garden.GenerateAddonConfig(values, enabled), nil
}

// GenerateHelmTillerConfig generates the values which are required to render the chart of
// helm-tiller properly.
func (b *Botanist) GenerateHelmTillerConfig() (map[string]interface{}, error) {
	return garden.GenerateAddonConfig(nil, b.Shoot.Spec.Addons.Monocular.Enabled), nil
}
