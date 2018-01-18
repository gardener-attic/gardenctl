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
	"github.com/gardener/gardenctl/pkg/apis/componentconfig"
	gardenv1 "github.com/gardener/gardenctl/pkg/apis/garden/v1"
	"github.com/gardener/gardenctl/pkg/chartrenderer"
	"github.com/gardener/gardenctl/pkg/client/kubernetes"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	// CRDName is the name of the Shoot's custom resource definition.
	CRDName = "shoots.garden.sapcloud.io"

	// GardenOperator is the value in a Shoot's `.metadata.finalizers[]` array on which the operator will react
	// when performing a delete request on a Shoot resource.
	GardenOperator = "garden.sapcloud.io/operator"

	// GardenRole is the key for an annotation on a Kubernetes Secret object whose value must be either 'seed' or
	// 'shoot'.
	GardenRole = "garden.sapcloud.io/role"

	// GardenRoleSeed is the value of the GardenRole key indicating type 'seed'.
	GardenRoleSeed = "seed"

	// GardenRoleDefaultDomain is the value of the GardenRole key indicating type 'default-domain'.
	GardenRoleDefaultDomain = "default-domain"

	// GardenRoleInternalDomain is the value of the GardenRole key indicating type 'internal-domain'.
	GardenRoleInternalDomain = "internal-domain"

	// GardenRoleImagePull is the value of the GardenRole key indicating type 'image-pull'.
	GardenRoleImagePull = "image-pull"

	// GardenRoleAlertingSMTP is the value of the GardenRole key indicating type 'alerting-smtp'.
	GardenRoleAlertingSMTP = "alerting-smtp"

	// ConfirmationDeletionTimestamp is an annotation on a Shoot resource whose value must be set equal to the Shoot's
	// '.metadata.deletionTimestamp' value to trigger the deletion process of the Shoot cluster.
	ConfirmationDeletionTimestamp = "confirmation.garden.sapcloud.io/deletionTimestamp"

	// ProjectPrefix is the prefix of namespaces in the Garden cluster which is used for all projects created by the
	// Gardener UI.
	ProjectPrefix = "garden-"

	// DNSKind is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// DNS provider.
	DNSKind = "dns.garden.sapcloud.io/kind"

	// DNSDomain is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// domain name.
	DNSDomain = "dns.garden.sapcloud.io/domain"

	// DNSHostedZoneID is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// DNS Hosted Zone.
	DNSHostedZoneID = "dns.garden.sapcloud.io/hostedZoneID"

	// InfrastructureKind is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// cloud provider (e.g., 'aws', 'azure', ...).
	InfrastructureKind = "infrastructure.garden.sapcloud.io/kind"

	// InfrastructureRegion is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// data center region for the given cloud provider.
	InfrastructureRegion = "infrastructure.garden.sapcloud.io/region"

	// DefaultPodNetworkCIDR is the default CIDR for the pod network.
	DefaultPodNetworkCIDR = gardenv1.CIDR("100.96.0.0/11")

	// DefaultServiceNetworkCIDR is the default CIDR for the service network.
	DefaultServiceNetworkCIDR = gardenv1.CIDR("100.64.0.0/13")
)

// Garden is a struct which is initialized whenever an event on a Shoot resource has been triggered.
// * Logger is a logger with an additional field containing the name of the Shoot resource.
// * Config is the Garden Operator component configuration.
// * K8sGardenClient is a Kubernetes client for the Garden cluster.
// * ChartGardenRenderer is a Helm chart renderer client respecting the Seed cluster's Kubernetes version.
// * CheckSums is a map which may be used to store sha256 checksums for Secrets or ConfigMaps which will
//   be injected into a Pod template (to establish automatic pod restart on changes).
// * Secrets is a map which may be used to store frequently used Kubernetes Secret objects.
// * Operator is a struct with information about name, id and version of the Garden operator.
// * OldShoot is only set in case of 'Update' events (otherwise it is nil).
// * Shoot is the Shoot object the event has been triggered for. In case of an 'Update' event, it is
//   the new Shoot resource.
// * ProjectName is the name of the Gardener project of the Shoot.
// * ShootNamespace is the concatenation of the Garden namespace in which the Shoot resource has been
//   created, a hyphen, and the Shoot resource name itself. E.g.: `d012345-shoot1`. It will be created
//   in the Seed cluster and is used to host the Shoot cluster's control plane.
// * InternalClusterDomain is the domain which points to the load balancer of the Shoot API server.
//   The user-defined cluster domain also points to this address.
// * ExternalClusterDomain is the user-defined cluster domain.
type Garden struct {
	Logger                *logrus.Entry
	Config                *componentconfig.GardenOperatorConfiguration
	K8sGardenClient       kubernetes.Client
	ChartGardenRenderer   chartrenderer.ChartRenderer
	CheckSums             map[string]string
	Secrets               map[string]*corev1.Secret
	Operator              *gardenv1.GardenOperator
	OldShoot              *gardenv1.Shoot
	Shoot                 *gardenv1.Shoot
	ProjectName           string
	ShootNamespace        string
	InternalClusterDomain string
	ExternalClusterDomain string
}

const (
	// TerraformerConfigSuffix is the suffix used for the ConfigMap which stores the Terraform configuration and variables declaration.
	TerraformerConfigSuffix = ".tf-config"

	// TerraformerVariablesSuffix is the suffix used for the Secret which stores the Terraform variables definition.
	TerraformerVariablesSuffix = ".tf-vars"

	// TerraformerStateSuffix is the suffix used for the ConfigMap which stores the Terraform state.
	TerraformerStateSuffix = ".tf-state"

	// TerraformerPodSuffix is the suffix used for the name of the Pod which validates the Terraform configuration.
	TerraformerPodSuffix = ".tf-pod"

	// TerraformerJobSuffix is the suffix used for the name of the Job which executes the Terraform configuration.
	TerraformerJobSuffix = ".tf-job"

	// TerraformerPurposeInfra is a constant for the complete Terraform setup with purpose 'infrastructure'.
	TerraformerPurposeInfra = "infra"

	// TerraformerPurposeInternalDNS is a constant for the complete Terraform setup with purpose 'internal cluster domain'
	TerraformerPurposeInternalDNS = "internal-dns"

	// TerraformerPurposeExternalDNS is a constant for the complete Terraform setup with purpose 'external cluster domain'.
	TerraformerPurposeExternalDNS = "external-dns"

	// TerraformerPurposeBackup is a constant for the complete Terraform setup with purpose 'etcd backup'.
	TerraformerPurposeBackup = "backup"

	// TerraformerPurposeKube2IAM is a constant for the complete Terraform setup with purpose 'kube2iam roles'.
	TerraformerPurposeKube2IAM = "kube2iam"

	// TerraformerPurposeIngress is a constant for the complete Terraform setup with purpose 'ingress'.
	TerraformerPurposeIngress = "ingress"
)

// Terraformer is a struct containing configuration parameters for the Terraform script it acts on.
// * Garden is a Garden object holding the Kubernetes client (the interaction of the Terraformer is
//   always with the Garden cluster).
// * Purpose is a one-word description depicting what the Terraformer does (e.g. 'infrastructure').
// * Namespace is the namespace in which the Terraformer will act (usually the Garden namespace).
// * ConfigName is the name of the ConfigMap containing the main Terraform file ('main.tf').
// * VariablesName is the name of the Secret containing the Terraform variables ('terraform.tfvars').
// * StateName is the name of the ConfigMap containing the Terraform state ('terraform.tfstate').
// * PodName is the name of the Pod which will validate the Terraform file.
// * JobName is the name of the Job which will execute the Terraform file.
// * VariablesEnvironment is a map of environment variables which will be injected in the resulting
//   Terraform job/pod. These variables should contain Terraform variables (i.e., must be prefixed
//   with TF_VAR_).
// * ConfigurationDefined indicates whether the required configuration ConfigMaps/Secrets have been
//   successfully defined.
type Terraformer struct {
	*Garden
	Purpose              string
	Namespace            string
	ConfigName           string
	VariablesName        string
	StateName            string
	PodName              string
	JobName              string
	VariablesEnvironment []map[string]interface{}
	ConfigurationDefined bool
}

// CloudBotanist is an interface which must be implemented by cloud-specific Botanists. The Cloud Botanist
// is responsible for all operations which require IaaS specific knowledge.
type CloudBotanist interface {
	// Infrastructure
	DeployInfrastructure() error
	DestroyInfrastructure() error
	DeployBackupInfrastructure() error
	DestroyBackupInfrastructure() error

	// Control Plane
	DeployAutoNodeRepair() error
	GenerateCloudProviderConfig() (string, error)
	GenerateCloudConfigUserDataConfig() *CloudConfigUserDataConfig
	GenerateKubeAPIServerConfig() (map[string]interface{}, error)
	GenerateKubeControllerManagerConfig() (map[string]interface{}, error)
	GenerateKubeSchedulerConfig() (map[string]interface{}, error)
	GenerateEtcdBackupSecretData() (map[string][]byte, error)
	GenerateEtcdBackupDefaults() *gardenv1.Backup
	GenerateEtcdConfig(string) (map[string]interface{}, error)

	// Addons
	DeployKube2IAMResources() error
	DestroyKube2IAMResources() error
	GenerateKube2IAMConfig() (map[string]interface{}, error)
	GenerateClusterAutoscalerConfig() (map[string]interface{}, error)
	GenerateAdmissionControlConfig() (map[string]interface{}, error)
	GenerateCalicoConfig() (map[string]interface{}, error)
	GenerateNginxIngressConfig() (map[string]interface{}, error)

	// Hooks
	ApplyCreateHook() error
	ApplyDeleteHook() error

	// Miscellaneous (Health check, ...)
	CheckIfClusterGetsScaled() (bool, int, error)
	DetermineNetworks() (gardenv1.CIDR, gardenv1.CIDR, gardenv1.CIDR)
}

// CloudConfigUserDataConfig is a struct containing cloud-specific configuration required to
// render the shoot-cloud-config chart properly.
type CloudConfigUserDataConfig struct {
	CloudConfig       bool
	KubeletParameters []string
	NetworkPlugin     string
	RootCerts         string
}
