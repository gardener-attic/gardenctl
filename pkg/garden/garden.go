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

package garden

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/gardener/gardenctl/pkg/apis/componentconfig"
	gardenv1 "github.com/gardener/gardenctl/pkg/apis/garden/v1"
	"github.com/gardener/gardenctl/pkg/chartrenderer"
	"github.com/gardener/gardenctl/pkg/client/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// New creates a new Garden object. It requires the input of a Shoot-specific <logger>, an Kubernetes client <k8sClient>
// for the Garden cluster, a map of <secrets> holding at least the image pull secret (probably also default domain or SMTP
// alerting secrets), information <operator> about the Garden Operator, and the new <shoot> and old <oldShoot> Shoot objects.
func New(logger *logrus.Entry, config *componentconfig.GardenOperatorConfiguration, k8sClient kubernetes.Client, secretsMap map[string]*corev1.Secret, operator *gardenv1.GardenOperator, shoot, oldShoot *gardenv1.Shoot) *Garden {
	secrets := make(map[string]*corev1.Secret)
	for k, v := range secretsMap {
		secrets[k] = v
	}

	projectName := shoot.ObjectMeta.Namespace
	if strings.HasPrefix(shoot.ObjectMeta.Namespace, ProjectPrefix) {
		projectName = strings.SplitAfterN(shoot.ObjectMeta.Namespace, ProjectPrefix, 2)[1]
	}

	internalDomain := secrets[GardenRoleInternalDomain].Annotations[DNSDomain]

	garden := &Garden{
		Logger:                logger,
		Config:                config,
		K8sGardenClient:       k8sClient,
		ChartGardenRenderer:   chartrenderer.New(k8sClient),
		Secrets:               secrets,
		CheckSums:             make(map[string]string),
		Operator:              operator,
		Shoot:                 shoot,
		OldShoot:              oldShoot,
		ProjectName:           projectName,
		ShootNamespace:        fmt.Sprintf("shoot-%s-%s", shoot.ObjectMeta.Namespace, shoot.ObjectMeta.Name),
		InternalClusterDomain: fmt.Sprintf("api.internal.%s.%s.%s", shoot.ObjectMeta.Name, projectName, internalDomain),
	}

	if shoot.Spec.DNS.Domain != "" {
		garden.ExternalClusterDomain = fmt.Sprintf("api.%s", shoot.Spec.DNS.Domain)
	}

	return garden
}

// DetermineSeedCluster returns the name of the Seed cluster if the .spec.seedName field in the Shoot manifest
// is set. If it is not set, then the function tries to find an adequate Seed cluster for the given infrastructure
// kind and region, and returns the name of the Secret in the Garden cluster which contains the Seed cluster
// configuration. Moreover, it stores the Kubernetes Secret object on the Garden in order to re-use it later.
func (g *Garden) DetermineSeedCluster() (string, error) {
	// We determine the Seed cluster (if it is not already present in the Shoot spec) we want to deploy the Shoot
	// controlplane into.
	var seedName = g.Shoot.Spec.SeedName
	if seedName != "" {
		g.Logger.Infof("Seed cluster has been given in manifest: %s", seedName)
	} else {
		seedKeys := g.GetSecretKeysOfKind("seed-")
		for _, seedKey := range seedKeys {
			secret := g.Secrets[seedKey]
			if gardenv1.CloudProvider(secret.Labels[InfrastructureKind]) == g.Shoot.Spec.Infrastructure.Kind && secret.Labels[InfrastructureRegion] == g.Shoot.Spec.Infrastructure.Region {
				seedName = secret.ObjectMeta.Name
				g.Logger.Infof("Seed cluster has been determined: %s", seedName)
				break
			}
		}
		if seedName == "" {
			return "", errors.New("Failed to determine an adequate Seed cluster for this infrastructure kind and region")
		}
	}
	return seedName, nil
}

// GetSeedSecret reads the Secret with the name .spec.seedName in the Garden cluster. The Secret must be stored in the Garden
// namespace.
func (g *Garden) GetSeedSecret(seedName string) (*corev1.Secret, error) {
	if seedSecret, ok := g.Secrets[fmt.Sprintf("seed-%s", seedName)]; ok {
		return seedSecret, nil
	}
	return nil, fmt.Errorf("No Seed cluster secret with name '%s' found", seedName)
}

// GetInfrastructureSecret reads the Secret with the name .spec.infrastructure.secret in the Garden cluster. The Secret must
func (g *Garden) GetInfrastructureSecret() (*corev1.Secret, error) {
	if infrastructureSecret, ok := g.Secrets["infrastructure"]; ok {
		return infrastructureSecret, nil
	}
	secret, err := g.
		K8sGardenClient.
		GetSecret(g.Shoot.ObjectMeta.Namespace, g.Shoot.Spec.Infrastructure.Secret)
	if err != nil {
		return nil, err
	}
	g.Secrets["infrastructure"] = secret
	return secret, nil
}

// GetKubernetesVersion extracts the major and minor part of a Kubernetes version and returns it.
// For example, for the input <version> = 1.2.3 it would return 1.2
func (g *Garden) GetKubernetesVersion() string {
	version := g.Shoot.Spec.KubernetesVersion
	return version[:strings.LastIndex(version, ".")]
}

// GetSecretKeysOfKind returns a list of keys which are present in the Garden Secrets map and which
// are prefixed with <kind>.
func (g *Garden) GetSecretKeysOfKind(kind string) []string {
	return GetSecretKeysWithPrefix(kind, g.Secrets)
}

// GetImagePullSecretsMap returns all known image pull secrets as map whereas the key is "name" and
// the value is the respective name of the image pull secret. The map can be used to specify a list
// of image pull secrets on a Kubernetes PodTemplateSpec object.
func (g *Garden) GetImagePullSecretsMap() []map[string]interface{} {
	imagePullSecrets := []map[string]interface{}{}
	for _, key := range g.GetSecretKeysOfKind("image-pull") {
		imagePullSecrets = append(imagePullSecrets, map[string]interface{}{
			"name": g.Secrets[key].ObjectMeta.Name,
		})
	}
	return imagePullSecrets
}

// ReportShootProgress will update the last operation object in the Shoot manifest `status` section
// by the current progress of the Flow execution.
func (g *Garden) ReportShootProgress(progress int, currentFunctions string) {
	g.Shoot.Status.LastOperation.Description = "Currently executing " + currentFunctions
	g.Shoot.Status.LastOperation.Progress = progress
	g.Shoot.Status.LastOperation.LastUpdateTime = metav1.Now()

	newShoot, err := g.
		K8sGardenClient.
		UpdateShoot(g.Shoot)
	if err == nil {
		g.Shoot = newShoot
	}
}

// ApplyGarden takes a path to a template <templatePath> and two maps <defaultValues>, <additionalValues>, and
// renders the template based on the merged result of both value maps. The resulting manifest will be applied
// to the Garden cluster.
func (g *Garden) ApplyGarden(templatePath string, defaultValues, additionalValues map[string]interface{}) error {
	return Apply(g.K8sGardenClient, templatePath, defaultValues, additionalValues)
}

// ApplyChartGarden takes a path to a chart <chartPath>, name of the release <name>, release's namespace <namespace>
// and two maps <defaultValues>, <additionalValues>, and renders the template based on the merged result of both value maps.
// The resulting manifest will be applied to the Garden cluster.
func (g *Garden) ApplyChartGarden(chartPath, name, namespace string, defaultValues, additionalValues map[string]interface{}) error {
	return ApplyChart(g.K8sGardenClient, g.ChartGardenRenderer, chartPath, name, namespace, defaultValues, additionalValues)
}
