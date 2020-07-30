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

import (
	gardencoreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

// TargetReader reads the current target.
type TargetReader interface {
	ReadTarget(targetPath string) TargetInterface
}

// TargetWriter writes the current target.
type TargetWriter interface {
	WriteTarget(targetPath string, target TargetInterface) error
}

// KubeconfigReader reads kubeconfig from given path.
type KubeconfigReader interface {
	ReadKubeconfig(kubeconfigPath string) ([]byte, error)
}

// KubeconfigWriter writes kubeconfig to given path.
type KubeconfigWriter interface {
	Write(path string, kubeconfig []byte) error
}

// GardenctlTargetReader implements TargetReader.
type GardenctlTargetReader struct{}

// GardenctlTargetWriter implements TargetWriter.
type GardenctlTargetWriter struct{}

// GardenctlKubeconfigReader implements KubeconfigReader.
type GardenctlKubeconfigReader struct{}

// GardenctlKubeconfigWriter implements KubeconfigWriter.
type GardenctlKubeconfigWriter struct{}

// TargetInterface defines target operations.
type TargetInterface interface {
	Stack() []TargetMeta
	SetStack([]TargetMeta)
	Kind() (TargetKind, error)
	K8SClient() (kubernetes.Interface, error)
	K8SClientToKind(TargetKind) (kubernetes.Interface, error)
	GardenerClient() (gardencoreclientset.Interface, error)
}

// Target contains the current target.
type Target struct {
	Target []TargetMeta `yaml:"target,omitempty" json:"target,omitempty"`
}

// TargetKind is a valid value for target kind.
type TargetKind string

// These are valid target kinds.
const (
	// TargetKindGarden points to garden cluster.
	TargetKindGarden TargetKind = "garden"
	// TargetKindProject points to project.
	TargetKindProject TargetKind = "project"
	// TargetKindSeed points to seed cluster.
	TargetKindSeed TargetKind = "seed"
	// TargetKindShoot points to shoot cluster.
	TargetKindShoot TargetKind = "shoot"
	// TargetKindNamespace points to namespace.
	TargetKindNamespace TargetKind = "namespace"
)

// TargetMeta contains kind and name of target.
type TargetMeta struct {
	Kind TargetKind `yaml:"kind,omitempty" json:"kind,omitempty"`
	Name string     `yaml:"name,omitempty" json:"name,omitempty"`
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

// ConfigReader reads the configuration.
type ConfigReader interface {
	ReadConfig(configPath string) *GardenConfig
}

// GardenConfigReader implements ConfigReader.
type GardenConfigReader struct{}

//GardenConfig contains config for gardenctl
type GardenConfig struct {
	Email          string              `yaml:"email,omitempty" json:"email,omitempty"`
	GithubURL      string              `yaml:"githubURL,omitempty" json:"githubURL,omitempty"`
	GardenClusters []GardenClusterMeta `yaml:"gardenClusters,omitempty" json:"gardenClusters,omitempty"`
}

// GardenClusters contains all gardenclusters
type GardenClusters struct {
	GardenClusters []GardenClusterMeta `yaml:"gardenClusters,omitempty" json:"gardenClusters,omitempty"`
}

// GardenClusterMeta contains name and path to kubeconfig of gardencluster
type GardenClusterMeta struct {
	Name               string              `yaml:"name,omitempty" json:"name,omitempty"`
	KubeConfig         string              `yaml:"kubeConfig,omitempty" json:"kubeConfig,omitempty"`
	DashboardURL       string              `yaml:"dashboardUrl,omitempty" json:"dashboardUrl,omitempty"`
	AccessRestrictions []AccessRestriction `yaml:"accessRestrictions,omitempty" json:"accessRestrictions,omitempty"`
}

// AccessRestrictionsOption contains key / notifyIf / msg
type AccessRestrictionsOption struct {
	Key      string `yaml:"key,omitempty" json:"key,omitempty"`
	NotifyIf bool   `yaml:"notifyIf,omitempty" json:"notifyIf,omitempty"`
	Msg      string `yaml:"msg,omitempty" json:"msg,omitempty"`
}

// AccessRestriction contains key / notifyIf / msg / options
type AccessRestriction struct {
	Key      string                     `yaml:"key,omitempty" json:"key,omitempty"`
	NotifyIf bool                       `yaml:"notifyIf,omitempty" json:"notifyIf,omitempty"`
	Msg      string                     `yaml:"msg,omitempty" json:"msg,omitempty"`
	Options  []AccessRestrictionsOption `yaml:"options,omitempty" json:"options,omitempty"`
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
	LastErrors    []string          `yaml:"lastErrors,omitempty" json:"lastErrors,omitempty"`
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
