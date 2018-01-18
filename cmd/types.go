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

package cmd

// Target contains the current target
type Target struct {
	Target []TargetMeta `yaml:"target"`
}

// TargetMeta contains kind and name of target
type TargetMeta struct {
	Kind string `yaml:"kind"`
	Name string `yaml:"name"`
}

// Projects contains list of all projects
type Projects struct {
	Projects []ProjectMeta `yaml:"projects"`
}

// ProjectMeta contains project and shoots of project
type ProjectMeta struct {
	Project string   `yaml:"project"`
	Shoots  []string `yaml:"shoots"`
}

// Seeds contains list of all seeds
type Seeds struct {
	Seeds []SeedMeta `yaml:"seeds"`
}

// SeedMeta contains shoots per seed
type SeedMeta struct {
	Seed   string   `yaml:"seed"`
	Shoots []string `yaml:"shoots"`
}

// GardenClusters contains all gardenclusters
type GardenClusters struct {
	GardenClusters []GardenClusterMeta `yaml:"gardenClusters"`
}

// GardenClusterMeta contains name and path to kubeconfig of gardencluster
type GardenClusterMeta struct {
	Name       string `yaml:"name"`
	KubeConfig string `yaml:"kubeConfig"`
}
