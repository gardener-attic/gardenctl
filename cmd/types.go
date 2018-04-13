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

package cmd

// Target contains the current target
type Target struct {
	Target []TargetMeta `yaml:"target,omitempty" json:"target,omitempty"`
}

// TargetMeta contains kind and name of target
type TargetMeta struct {
	Kind string `yaml:"kind,omitempty" json:"kind,omitempty"`
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
}

// Projects contains list of all projects
type Projects struct {
	Projects []ProjectMeta `yaml:"projects,omitempty" json:"projects,omitempty"`
}

// ProjectMeta contains project and shoots of project
type ProjectMeta struct {
	Project string   `yaml:"project,omitempty" json:"project,omitempty"`
	Shoots  []string `yaml:"shoots,omitempty" json:"shoots,omitempty"`
}

// Seeds contains list of all seeds
type Seeds struct {
	Seeds []SeedMeta `yaml:"seeds,omitempty" json:"seeds,omitempty"`
}

// SeedMeta contains shoots per seed
type SeedMeta struct {
	Seed   string   `yaml:"seed,omitempty" json:"seed,omitempty"`
	Shoots []string `yaml:"shoots,omitempty" json:"shoots,omitempty"`
}

// GardenClusters contains all gardenclusters
type GardenClusters struct {
	GardenClusters []GardenClusterMeta `yaml:"gardenClusters,omitempty" json:"gardenClusters,omitempty"`
}

// GardenClusterMeta contains name and path to kubeconfig of gardencluster
type GardenClusterMeta struct {
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
	KubeConfig string `yaml:"kubeConfig,omitempty" json:"kubeConfig,omitempty"`
}

// Issues contains all projects with issues
type Issues struct {
	Issues []IssuesMeta `yaml:"issues,omitempty" json:"issues,omitempty"`
}

// IssuesMeta contains project related informations
type IssuesMeta struct {
	Project string     `yaml:"project,omitempty" json:"project,omitempty"`
	Seed    string     `yaml:"seed,omitempty" json:"seed,omitempty"`
	Shoot   string     `yaml:"shoot,omitempty" json:"shoot,omitempty"`
	Health  string     `yaml:"health,omitempty" json:"health,omitempty"`
	Status  StatusMeta `yaml:"status,omitempty" json:"status,omitempty"`
}

// StatusMeta contains status for a project
type StatusMeta struct {
	LastError     string            `yaml:"lastError,omitempty" json:"lastError,omitempty"`
	LastOperation LastOperationMeta `yaml:"lastOperation,omitempty" json:"lastOperation,omitempty"`
}

// LastOperationMeta contains information about last operation
type LastOperationMeta struct {
	Description    string `yaml:"description,omitempty" json:"description,omitempty"`
	LastUpdateTime string `yaml:"lastUpdateTime,omitempty" json:"lastUpdateTime,omitempty"`
	Progress       int    `yaml:"progress,omitempty" json:"progress,omitempty"`
	State          string `yaml:"state,omitempty" json:"state,omitempty"`
	Type           string `yaml:"type,omitempty" json:"type,omitempty"`
}
