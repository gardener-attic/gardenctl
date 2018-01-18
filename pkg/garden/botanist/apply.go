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

import "github.com/gardener/gardenctl/pkg/garden"

// ApplySeed takes a path to a template <templatePath> and two maps <defaultValues>, <additionalValues>, and
// renders the template based on the merged result of both value maps. The resulting manifest will be applied
// to the Seed cluster.
func (b *Botanist) ApplySeed(templatePath string, defaultValues, additionalValues map[string]interface{}) error {
	return garden.Apply(b.K8sSeedClient, templatePath, defaultValues, additionalValues)
}

// ApplyShoot takes a path to a template <templatePath> and two maps <defaultValues>, <additionalValues>, and
// renders the template based on the merged result of both value maps. The resulting manifest will be applied
// to the Shoot cluster.
func (b *Botanist) ApplyShoot(templatePath string, defaultValues, additionalValues map[string]interface{}) error {
	return garden.Apply(b.K8sShootClient, templatePath, defaultValues, additionalValues)
}

// ApplyChartSeed takes a path to a chart <chartPath>, name of the release <name>, release's namespace <namespace>
// and two maps <defaultValues>, <additionalValues>, and renders the template based on the merged result of both value maps.
// The resulting manifest will be applied to the Seed cluster.
func (b *Botanist) ApplyChartSeed(chartPath, name, namespace string, defaultValues, additionalValues map[string]interface{}) error {
	return garden.ApplyChart(b.K8sSeedClient, b.ChartSeedRenderer, chartPath, name, namespace, defaultValues, additionalValues)
}

// ApplyChartShoot takes a path to a chart <chartPath>, name of the release <name>, release's namespace <namespace>
// and two maps <defaultValues>, <additionalValues>, and renders the template based on the merged result of both value maps.
// The resulting manifest will be applied to the Shoot cluster.
func (b *Botanist) ApplyChartShoot(chartPath, name, namespace string, defaultValues, additionalValues map[string]interface{}) error {
	return garden.ApplyChart(b.K8sShootClient, b.ChartShootRenderer, chartPath, name, namespace, defaultValues, additionalValues)
}
