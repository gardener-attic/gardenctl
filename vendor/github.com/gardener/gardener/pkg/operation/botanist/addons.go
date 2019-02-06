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
	"github.com/gardener/gardener/pkg/operation/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureIngressDNSRecord creates the respective wildcard DNS record for the nginx-ingress-controller.
func (b *Botanist) EnsureIngressDNSRecord() error {
	if !b.Shoot.NginxIngressEnabled() || b.Shoot.IsHibernated {
		return b.DestroyIngressDNSRecord()
	}

	loadBalancerIngress, _, err := common.GetLoadBalancerIngress(b.K8sShootClient, metav1.NamespaceSystem, "addons-nginx-ingress-controller")
	if err != nil {
		return err
	}
	return b.DeployDNSRecord("ingress", b.Shoot.GetIngressFQDN("*"), loadBalancerIngress, false)
}

// DestroyIngressDNSRecord destroys the nginx-ingress resources created by Terraform.
func (b *Botanist) DestroyIngressDNSRecord() error {
	return b.DestroyDNSRecord("ingress", false)
}

// GenerateKubernetesDashboardConfig generates the values which are required to render the chart of
// the kubernetes-dashboard properly.
func (b *Botanist) GenerateKubernetesDashboardConfig() (map[string]interface{}, error) {
	return common.GenerateAddonConfig(nil, b.Shoot.KubernetesDashboardEnabled()), nil
}

// GenerateKubeLegoConfig generates the values which are required to render the chart of
// kube-lego properly.
func (b *Botanist) GenerateKubeLegoConfig() (map[string]interface{}, error) {
	var (
		enabled = b.Shoot.KubeLegoEnabled()
		values  map[string]interface{}
	)

	if enabled {
		values = map[string]interface{}{
			"config": map[string]interface{}{
				"LEGO_EMAIL": b.Shoot.Info.Spec.Addons.KubeLego.Mail,
			},
		}
	}

	return common.GenerateAddonConfig(values, enabled), nil
}
