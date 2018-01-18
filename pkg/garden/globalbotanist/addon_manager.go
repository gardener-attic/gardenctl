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

package globalbotanist

import (
	"path/filepath"

	"github.com/gardener/gardenctl/pkg/chartrenderer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/gardenctl/pkg/garden"
)

// DeployKubeAddonManager deploys the Kubernetes Addon Manager which will use labelled Kubernetes resources in order
// to ensure that they exist in a cluster/reconcile them in case somebody changed something.
func (b *GlobalBotanist) DeployKubeAddonManager() error {
	name := "kube-addon-manager"
	cloudConfig, err := b.generateCloudConfigChart()
	if err != nil {
		return err
	}
	coreAddons, err := b.generateCoreAddonsChart()
	if err != nil {
		return err
	}
	admissionControls, err := b.generateAdmissionControlsChart()
	if err != nil {
		return err
	}
	optionalAddons, err := b.generateOptionalAddonsChart()
	if err != nil {
		return err
	}

	return b.
		Botanist.
		ApplyChartSeed(
			filepath.Join("charts", "seed-controlplane", "charts", name),
			name,
			b.ShootNamespace,
			nil,
			map[string]interface{}{
				"CloudConfigContent":       cloudConfig.Files,
				"CoreAddonsContent":        coreAddons.Files,
				"AdmissionControlsContent": admissionControls.Files,
				"OptionalAddonsContent":    optionalAddons.Files,
				"PodAnnotations": map[string]interface{}{
					"checksum/secret-kube-addon-manager": b.CheckSums[name],
				},
			},
		)
}

// generateCloudConfigChart renders the kube-addon-manager configuration for the cloud config user data.
// It will be stored as a Secret and mounted into the Pod. The configuration contains
// specially labelled Kubernetes manifests which will be created and periodically reconciled.
func (b *GlobalBotanist) generateCloudConfigChart() (*chartrenderer.RenderedChart, error) {
	var (
		kubeletSecret = b.Botanist.Secrets["kubelet"]
		workers       = []string{}
		cloudProvider = map[string]interface{}{
			"name": b.Botanist.Shoot.Spec.Infrastructure.Kind,
		}
	)

	for _, worker := range b.Botanist.Shoot.Spec.Workers {
		workers = append(workers, worker.Name)
	}

	userDataConfig := b.
		CloudBotanist.
		GenerateCloudConfigUserDataConfig()

	if userDataConfig.CloudConfig {
		cloudProviderConfig, err := b.
			CloudBotanist.
			GenerateCloudProviderConfig()
		if err != nil {
			return nil, err
		}
		cloudProvider["config"] = cloudProviderConfig
	}

	config := map[string]interface{}{
		"cloudProvider": cloudProvider,
		"kubernetes": map[string]interface{}{
			"caCert":     string(kubeletSecret.Data["ca.crt"]),
			"clusterDNS": garden.ComputeClusterIP(b.Botanist.Shoot.Spec.Networks.Services, 10),
			"kubelet": map[string]interface{}{
				"kubeconfig":    string(kubeletSecret.Data["kubeconfig"]),
				"networkPlugin": userDataConfig.NetworkPlugin,
				"parameters":    userDataConfig.KubeletParameters,
			},
			"nonMasqueradeCIDR": garden.ComputeNonMasqueradeCIDR(b.Botanist.Shoot.Spec.Networks.Services),
			"version":           b.Botanist.Shoot.Spec.KubernetesVersion,
		},
		"workers": workers,
	}

	if userDataConfig.RootCerts != "" {
		config["rootCerts"] = userDataConfig.RootCerts
	}

	return b.
		Botanist.
		ChartShootRenderer.
		Render(filepath.Join("charts", "shoot-cloud-config"), "shoot-cloud-config", metav1.NamespaceSystem, config)
}

// generateCoreAddonsChart renders the kube-addon-manager configuration for the core addons. It will be
// stored as a Secret (as it may contain credentials) and mounted into the Pod. The configuration contains
// specially labelled Kubernetes manifests which will be created and periodically reconciled.
func (b *GlobalBotanist) generateCoreAddonsChart() (*chartrenderer.RenderedChart, error) {
	kubeProxySecret := b.Secrets["kube-proxy"]
	sshKeyPairSecret := b.Secrets["vpn-ssh-keypair"]

	global := map[string]interface{}{
		"PodNetwork": b.Shoot.Spec.Networks.Pods,
	}
	kubeDNS := map[string]interface{}{
		"ClusterDNS": garden.ComputeClusterIP(b.Shoot.Spec.Networks.Services, 10),
	}
	kubeProxy := map[string]interface{}{
		"kubeconfig": kubeProxySecret.Data["kubeconfig"],
	}
	vpnShoot := map[string][]byte{
		"authorizedKeys": sshKeyPairSecret.Data["id_rsa.pub"],
	}
	calico, err := b.
		CloudBotanist.
		GenerateCalicoConfig()
	if err != nil {
		return nil, err
	}
	rbac := map[string]interface{}{
		"enabled": b.Shoot.Spec.Infrastructure.Kind == "gce",
	}
	return b.
		Botanist.
		ChartShootRenderer.
		Render(filepath.Join("charts", "shoot-core"), "shoot-core", metav1.NamespaceSystem, map[string]interface{}{
			"global":     global,
			"kube-dns":   kubeDNS,
			"kube-proxy": kubeProxy,
			"vpn-shoot":  vpnShoot,
			"calico":     calico,
			"rbac":       rbac,
		})
}

// generateOptionalAddonsChart renders the kube-addon-manager chart for the optional addons. It
// will be stored as a Secret (as it may contain credentials) and mounted into the Pod. The configuration
// contains specially labelled Kubernetes manifests which will be created and periodically reconciled.
func (b *GlobalBotanist) generateOptionalAddonsChart() (*chartrenderer.RenderedChart, error) {
	if b.Shoot.Spec.Addons == nil {
		return &chartrenderer.RenderedChart{}, nil
	}

	clusterAutoscaler, err := b.
		CloudBotanist.
		GenerateClusterAutoscalerConfig()
	if err != nil {
		return nil, err
	}
	heapster, err := b.
		Botanist.
		GenerateHeapsterConfig()
	if err != nil {
		return nil, err
	}
	helmTiller, err := b.
		Botanist.
		GenerateHelmTillerConfig()
	if err != nil {
		return nil, err
	}
	kubeLego, err := b.
		Botanist.
		GenerateKubeLegoConfig()
	if err != nil {
		return nil, err
	}
	kube2IAM, err := b.
		CloudBotanist.
		GenerateKube2IAMConfig()
	if err != nil {
		return nil, err
	}
	kubernetesDashboard, err := b.
		Botanist.
		GenerateKubernetesDashboardConfig()
	if err != nil {
		return nil, err
	}
	monocular, err := b.
		Botanist.
		GenerateMonocularConfig()
	if err != nil {
		return nil, err
	}
	nginxIngress, err := b.
		CloudBotanist.
		GenerateNginxIngressConfig()
	if err != nil {
		return nil, err
	}

	return b.
		Botanist.
		ChartShootRenderer.
		Render(filepath.Join("charts", "shoot-addons"), "addons", metav1.NamespaceSystem, map[string]interface{}{
			"cluster-autoscaler":   clusterAutoscaler,
			"heapster":             heapster,
			"helm-tiller":          helmTiller,
			"kube-lego":            kubeLego,
			"kube2iam":             kube2IAM,
			"kubernetes-dashboard": kubernetesDashboard,
			"monocular":            monocular,
			"nginx-ingress":        nginxIngress,
		})
}

// generateAdmissionControlsChart renders the kube-addon-manager configuration for the admission control
// extensions. It will be stored as a ConfigMap and mounted into the Pod. The configuration contains
// specially labelled Kubernetes manifests which will be created and periodically reconciled.
func (b *GlobalBotanist) generateAdmissionControlsChart() (*chartrenderer.RenderedChart, error) {
	config, err := b.CloudBotanist.GenerateAdmissionControlConfig()
	if err != nil {
		return nil, err
	}

	return b.
		Botanist.
		ChartShootRenderer.
		Render(filepath.Join("charts", "shoot-admission-controls"), "admission-controls", metav1.NamespaceSystem, config)
}
