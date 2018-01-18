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

package botanist

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gardener/gardenctl/pkg/chartrenderer"
	"github.com/gardener/gardenctl/pkg/client/kubernetes"
	gardenpkg "github.com/gardener/gardenctl/pkg/garden"
	corev1 "k8s.io/api/core/v1"
)

// New creates a new Botanist object. It requires the input of a Garden object <garden> and a name of a
// Secret containing a Kubeconfig for a Seed cluster <seedName>. The Garden Kubernetes client will be used
// to read the infrastructure credentials and the Seed Secret containing the Kubeconfig.
// What else will be initialized is a Kubernetes client for the Seed cluster is a Chart renderer which will
// be attached to the returned Botanist object.
func New(garden *gardenpkg.Garden, seedName string) (*Botanist, error) {
	_, err := garden.GetInfrastructureSecret()
	if err != nil {
		return nil, err
	}
	seedSecret, err := garden.GetSeedSecret(seedName)
	if err != nil {
		return nil, err
	}
	k8sSeedClient, err := kubernetes.NewClientFromSecretObject(seedSecret)
	if err != nil {
		return nil, err
	}

	// Check whether the Kubernetes version of the Seed cluster is at least 1.7.
	if k8sSeedClient.Version() < 17 {
		return nil, errors.New("The Kubernetes version of the Seed cluster must be at least v1.7")
	}

	chartSeedRenderer := chartrenderer.New(k8sSeedClient)
	seedFQDN, err := getSeedFQDN(seedSecret)
	if err != nil {
		return nil, err
	}

	b := &Botanist{
		Garden:            garden,
		K8sSeedClient:     k8sSeedClient,
		ChartSeedRenderer: chartSeedRenderer,
		SeedFQDN:          seedFQDN,
	}

	// Determine all default domain secrets and check whether the used Shoot domain matches a default domain.
	defaultDomainKey := ""
	defaultDomainKeys := b.GetSecretKeysOfKind("default-domain-")
	for _, key := range defaultDomainKeys {
		defaultDomain := strings.SplitAfter(key, "default-domain-")[1]
		if strings.HasSuffix(b.Shoot.Spec.DNS.Domain, defaultDomain) {
			defaultDomainKey = fmt.Sprintf("default-domain-%s", defaultDomain)
			break
		}
	}
	if defaultDomainKey != "" {
		b.DefaultDomainSecret = b.Secrets[defaultDomainKey]
	}

	return b, nil
}

// InitializeShootClients will use the Seed Kubernetes client to read the gardenctl Secret in the Seed
// cluster which contains a Kubeconfig that can be used to authenticate against the Shoot cluster. With it,
// a Kubernetes client as well as a Chart renderer for the Shoot cluster will be initialized and attached to
// the already existing Botanist object.
func (b *Botanist) InitializeShootClients() error {
	k8sShootClient, err := kubernetes.NewClientFromSecret(b.K8sSeedClient, b.ShootNamespace, "gardenctl")
	if err != nil {
		return err
	}
	chartShootRenderer := chartrenderer.New(k8sShootClient)

	b.K8sShootClient = k8sShootClient
	b.ChartShootRenderer = chartShootRenderer
	return nil
}

// GenerateTerraformVariablesEnvironment takes a <secret> and a <keyValueMap> and builds an environment which
// can be injected into the Terraformer job/pod manifest. The keys of the <keyValueMap> will be prefixed with
// 'TF_VAR_' and the value will be used to extract the respective data from the <secret>.
func GenerateTerraformVariablesEnvironment(secret *corev1.Secret, keyValueMap map[string]string) []map[string]interface{} {
	m := []map[string]interface{}{}
	for key, value := range keyValueMap {
		m = append(m, map[string]interface{}{
			"name":  fmt.Sprintf("TF_VAR_%s", key),
			"value": string(secret.Data[value]),
		})
	}
	return m
}
