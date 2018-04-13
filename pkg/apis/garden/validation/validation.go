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

package validation

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/gardener/gardenctl/pkg/apis/componentconfig"
	gardenv1 "github.com/gardener/gardenctl/pkg/apis/garden/v1"
	"github.com/gardener/gardenctl/pkg/utils"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateShoot(Shoot *gardenv1.Shoot, constraints componentconfig.GardenConstraints) field.ErrorList {
	allErrs := ValidateShootSpec(&Shoot.Spec, constraints, field.NewPath("spec"))
	return allErrs
}

func ValidateShootSpec(spec *gardenv1.ShootSpec, constraints componentconfig.GardenConstraints, fldPath *field.Path) field.ErrorList {
	var (
		allErrs                     = field.ErrorList{}
		cloudProviders              = constraints.CloudProviders
		supportedDNSKinds           = constraints.DNSProviders
		supportedKubernetesVersions = constraints.KubernetesVersions
	)

	infrastructure := spec.Infrastructure
	if infrastructure == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("infrastructure"), "must specify infrastructure section"))
		return allErrs
	}

	cloud := string(infrastructure.Kind)
	supportedInfrastructureKinds, cloudProviderConfig := componentconfig.FindCloudProviderConfig(cloudProviders, cloud)

	if len(infrastructure.Region) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("infrastructure", "region"), "must specify a region"))
	}
	if len(infrastructure.Secret) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("infrastructure", "secret"), "must specify a secret"))
	}
	if !utils.ValueExists(cloud, supportedInfrastructureKinds) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("infrastructure", "kind"), cloud, supportedInfrastructureKinds))
	}

	if spec.DNS.Kind == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("dns", "kind"), "must specify a dns kind"))
	}
	if !utils.ValueExists(string(spec.DNS.Kind), supportedDNSKinds) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("dns", "kind"), spec.DNS.Kind, supportedDNSKinds))
	}
	if spec.DNS.Kind == gardenv1.DNSUnmanaged {
		if spec.Addons.Monocular.Enabled {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("addons", "monocular", "enabled"), spec.Addons.Monocular.Enabled, fmt.Sprintf("`.spec.addons.monocular.enabled` must be false when `.spec.dns.kind` is '%s'", gardenv1.DNSUnmanaged)))
		}
		if spec.DNS.HostedZoneID != "" {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("dns", "hostedZoneID"), spec.DNS.HostedZoneID, fmt.Sprintf("`.spec.dns.hostedZoneID` must not be set when `.spec.dns.kind` is '%s'", gardenv1.DNSUnmanaged)))
		}
	} else {
		if spec.DNS.Domain == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("dns", "domain"), fmt.Sprintf("`.spec.dns.domain` may only be empty if `.spec.dns.kind` is '%s'", gardenv1.DNSUnmanaged)))
		}
	}

	if !utils.ValueExists(spec.KubernetesVersion, supportedKubernetesVersions) {
		allErrs = append(allErrs, field.NotSupported(fldPath.Child("kubernetesVersion"), spec.KubernetesVersion, supportedKubernetesVersions))
	}

	if len(spec.Workers) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("workers"), "must specify at least one worker group"))
	}
	workerNames := make(map[string]bool)
	for i, worker := range spec.Workers {
		var (
			idxPath           = fldPath.Child("workers").Index(i)
			namePath          = idxPath.Child("name")
			autoScalerMinPath = idxPath.Child("autoScalerMin")
			autoScalerMaxPath = idxPath.Child("autoScalerMax")
			machineTypePath   = idxPath.Child("machineType")
			volumeSizePath    = idxPath.Child("volumeSize")
			volumeTypePath    = idxPath.Child("volumeType")
		)
		if worker.Name == "" {
			allErrs = append(allErrs, field.Required(namePath, ""))
		}
		if workerNames[worker.Name] {
			allErrs = append(allErrs, field.Duplicate(namePath, worker.Name))
		}
		workerNames[worker.Name] = true
		if len(worker.Name) > 15 {
			allErrs = append(allErrs, field.Invalid(namePath, worker.Name, `.spec.workers[*].name must not be longer than 15 characters`))
		}
		if worker.MachineType == "" {
			allErrs = append(allErrs, field.Required(machineTypePath, ""))
		}
		match, _ := regexp.MatchString("^(\\d+)Gi$", worker.VolumeSize)
		if !match {
			allErrs = append(allErrs, field.Invalid(volumeSizePath, worker.VolumeSize, `.spec.workers[*].volumeSize must match the regular expression "^(\\d+)Gi$", e.g. 100Gi`))
		}
		if worker.AutoScalerMin <= 0 {
			allErrs = append(allErrs, field.Invalid(autoScalerMinPath, worker.AutoScalerMin, "`.spec.workers[*].autoScalerMin` must be greater than zero"))
		}
		if worker.AutoScalerMax <= 0 || worker.AutoScalerMax < worker.AutoScalerMin {
			allErrs = append(allErrs, field.Invalid(autoScalerMaxPath, worker.AutoScalerMax, "`.spec.workers[*].autoScalerMax` must be greater than zero and no lower than `.spec.workers[*].autoScalerMin`"))
		}
		supportedMachineTypes := cloudProviderConfig.MachineTypes
		if !utils.ValueExists(worker.MachineType, supportedMachineTypes) {
			allErrs = append(allErrs, field.NotSupported(machineTypePath, worker.MachineType, supportedMachineTypes))
		}
		supportedVolumeTypes := cloudProviderConfig.VolumeTypes
		if !utils.ValueExists(worker.VolumeType, supportedVolumeTypes) {
			allErrs = append(allErrs, field.NotSupported(volumeTypePath, worker.VolumeType, supportedVolumeTypes))
		}
	}

	if spec.Addons == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("addons"), "addons section must not be empty"))
	}

	podNetwork := spec.Networks.Pods
	if podNetwork != "" {
		for _, msg := range isValidCIDR(string(podNetwork)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "pods"), podNetwork, msg))
		}
	}
	serviceNetwork := spec.Networks.Services
	if serviceNetwork != "" {
		for _, msg := range isValidCIDR(string(serviceNetwork)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "services"), serviceNetwork, msg))
		}
	}
	nodeNetwork := spec.Networks.Nodes
	if nodeNetwork != "" {
		for _, msg := range isValidCIDR(string(nodeNetwork)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "nodes"), nodeNetwork, msg))
		}
	}

	switch infrastructure.Kind {
	case gardenv1.CloudProviderAWS:
		allErrs = append(allErrs, validateShootSpecAWS(spec, constraints, fldPath)...)
	case gardenv1.CloudProviderAzure:
		allErrs = append(allErrs, validateShootSpecAzure(spec, constraints, fldPath)...)
	case gardenv1.CloudProviderGCE:
		allErrs = append(allErrs, validateShootSpecGCE(spec, constraints, fldPath)...)
	case gardenv1.CloudProviderOpenStack:
		allErrs = append(allErrs, validateShootSpecOpenStack(spec, constraints, fldPath)...)
	}

	return allErrs
}

// Update validation

func ValidateShootUpdate(Shoot, oldShoot *gardenv1.Shoot, constraints componentconfig.GardenConstraints) field.ErrorList {
	allErrs := ValidateShootSpecUpdate(Shoot.Spec, oldShoot.Spec, constraints, field.NewPath("spec"))
	return allErrs
}

func ValidateShootSpecUpdate(spec, oldSpec gardenv1.ShootSpec, constraints componentconfig.GardenConstraints, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, ValidateShootSpec(&spec, constraints, fldPath)...)

	allErrs = append(allErrs, validateImmutableField(spec.Infrastructure.Kind, oldSpec.Infrastructure.Kind, fldPath.Child("infrastructure.kind"))...)
	allErrs = append(allErrs, validateImmutableField(spec.Infrastructure.Region, oldSpec.Infrastructure.Region, fldPath.Child("infrastructure.region"))...)
	allErrs = append(allErrs, validateImmutableField(spec.DNS.Domain, oldSpec.DNS.Domain, fldPath.Child("dns.domain"))...)
	allErrs = append(allErrs, validateImmutableField(spec.DNS.HostedZoneID, oldSpec.DNS.HostedZoneID, fldPath.Child("dns.hostedZoneID"))...)
	allErrs = append(allErrs, validateImmutableField(spec.KubernetesVersion, oldSpec.KubernetesVersion, fldPath.Child("kubernetesVersion"))...)
	allErrs = append(allErrs, validateImmutableField(spec.Networks, oldSpec.Networks, fldPath.Child("networks"))...)

	switch spec.Infrastructure.Kind {
	case gardenv1.CloudProviderAWS:
		allErrs = append(allErrs, validateShootSpecUpdateAWS(spec, oldSpec, fldPath)...)
	case gardenv1.CloudProviderAzure:
		allErrs = append(allErrs, validateShootSpecUpdateAzure(spec, oldSpec, fldPath)...)
	case gardenv1.CloudProviderGCE:
		allErrs = append(allErrs, validateShootSpecUpdateGCE(spec, oldSpec, fldPath)...)
	case gardenv1.CloudProviderOpenStack:
		allErrs = append(allErrs, validateShootSpecUpdateOpenStack(spec, oldSpec, fldPath)...)
	}

	return allErrs
}

// AWS specific validation

func validateShootSpecAWS(spec *gardenv1.ShootSpec, constraints componentconfig.GardenConstraints, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.Backup != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(spec.Backup.IntervalInSecond), fldPath.Child("backup.intervalInSecond"))...)
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(spec.Backup.Maximum), fldPath.Child("backup.maximum"))...)
	}

	vpc := spec.Infrastructure.VPC
	if vpc == nil {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "infrastructure", "vpc"), "must specify a vpc"))
		return allErrs
	}
	if (vpc.CIDR == "" && vpc.ID == "") || (vpc.CIDR != "" && vpc.ID != "") {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "infrastructure", "vpc"), vpc, "Must specify exactly one of `.spec.infrastructure.vpc.cidr` or `.spec.infrastructure.vpc.id`"))
	}
	if vpc.CIDR != "" {
		for _, msg := range isValidCIDR(string(vpc.CIDR)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "infrastructure", "vpc", "cidr"), vpc.CIDR, msg))
		}
	}

	zoneCount := len(spec.Zones)
	if zoneCount == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "zones"), "must specify zones"))
	}

	if len(spec.Networks.Workers) != zoneCount {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "networks", "workers"), "must specify as many worker networks as zones"))
	}
	if len(spec.Networks.Public) != zoneCount {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "networks", "public"), "must specify as many public networks as zones"))
	}
	if len(spec.Networks.Internal) != zoneCount {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "networks", "internal"), "must specify as many internal networks as zones"))
	}
	for _, workerNetwork := range spec.Networks.Workers {
		for _, msg := range isValidCIDR(string(workerNetwork)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "workers"), workerNetwork, msg))
		}
	}
	for _, publicNetwork := range spec.Networks.Public {
		for _, msg := range isValidCIDR(string(publicNetwork)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "public"), publicNetwork, msg))
		}
	}
	for _, internalNetwork := range spec.Networks.Internal {
		for _, msg := range isValidCIDR(string(internalNetwork)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "internal"), internalNetwork, msg))
		}
	}
	if vpc.ID != "" && spec.Networks.Nodes == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "networks", "nodes"), "`.spec.networks.nodes` must not be empty if you are using an existing VPC (specify the VPC CIDR here)"))
	}

	return allErrs
}

func validateShootSpecUpdateAWS(spec, oldSpec gardenv1.ShootSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateImmutableField(spec.Infrastructure.VPC, oldSpec.Infrastructure.VPC, fldPath.Child("infrastructure.vpc"))...)
	return allErrs
}

// Azure specific validation

func validateShootSpecAzure(spec *gardenv1.ShootSpec, constraints componentconfig.GardenConstraints, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.Backup != nil {
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(spec.Backup.IntervalInSecond), fldPath.Child("backup.intervalInSecond"))...)
		allErrs = append(allErrs, validation.ValidateNonnegativeField(int64(spec.Backup.Maximum), fldPath.Child("backup.maximum"))...)
	}

	vnet := spec.Infrastructure.VNet
	if vnet == nil {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "infrastructure", "vnet"), "must specify a vnet"))
		return allErrs
	}
	if len(vnet.CIDR) != 0 {
		for _, msg := range isValidCIDR(string(vnet.CIDR)) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "infrastructure", "vnet", "cidr"), vnet.CIDR, msg))
		}
	}
	if err := validateDomainCount(spec.Infrastructure.CountUpdateDomains, 5, 20); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "infrastructure", "countUpdateDomains"), spec.Infrastructure.CountUpdateDomains, err.Error()))
	}
	if err := validateDomainCount(spec.Infrastructure.CountFaultDomains, 2, 3); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "infrastructure", "countFaultDomains"), spec.Infrastructure.CountFaultDomains, err.Error()))
	}

	// TODO: Currently we will not allow deployments into existing resource groups or VNets although this functionallity
	// is already implemented, because the Azure cloud provider (v1.7.6) is not cleaning up self created resources properly.
	// This resources would be orphaned when the cluster will be deleted. We block these cases thereby that the Azure shoot
	// validation here will fail for those cases. To reenable these functionallity, remove the blocking validation + tests
	// and enable the old validation + corresponding test cases again.
	if len(spec.Infrastructure.ResourceGroupName) != 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "infrastructure", "resourceGroupName"), vnet.Name, "`.spec.infrastructure.ResourceGroupName` is not supported yet. Usage of existing resource group is not possible"))
	}
	if len(vnet.Name) != 0 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "infrastructure", "vnet", "name"), vnet.Name, "`.spec.infrastructure.vnet.name` is not supported yet. Usage of existing vnet is not possible"))
	}
	if len(vnet.CIDR) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "infrastructure", "vnet", "cidr"), "must specify a vnet cidr"))
	}

	for _, msg := range isValidCIDR(string(spec.Networks.Workers[0])) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "workers", "0"), spec.Networks.Workers[0], msg))
	}

	for i, worker := range spec.Workers {
		var (
			idxPath           = fldPath.Child("workers").Index(i)
			autoScalerMaxPath = idxPath.Child("autoScalerMax")
		)
		if worker.AutoScalerMax != worker.AutoScalerMin {
			allErrs = append(allErrs, field.Invalid(autoScalerMaxPath, worker.AutoScalerMax, fmt.Sprintf("`.spec.workers[%d].autoScalerMax` must be equal to `.spec.infrastructure.workers[%d].autoScalerMin`", i, i)))
		}
	}

	return allErrs
}

func validateShootSpecUpdateAzure(spec, oldSpec gardenv1.ShootSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// GCE specific validation

func validateShootSpecGCE(spec *gardenv1.ShootSpec, constraints componentconfig.GardenConstraints, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	vpc := spec.Infrastructure.VPC
	if vpc != nil {
		if vpc.CIDR != "" {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "infrastructure", "vpc", "cidr"), "`.spec.infrastructure.vpc.cidr` is not allowed on GCE"))
		}
		if vpc.ID != "" {
			allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "infrastructure", "vpc", "id"), "`.spec.infrastructure.vpc.id` is not allowed on GCE"))
		}
	}

	if len(spec.Zones) != 1 {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "zones"), "`.spec.zones` must contain exactly one element"))
	}

	for _, zone := range spec.Zones {
		if strings.Index(string(zone), spec.Infrastructure.Region) != 0 {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "zones"), zone, "zones must match to the infrastructure region"))
		}
	}

	if len(spec.Networks.Workers) != 1 {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "workers"), spec.Networks.Workers, "`.spec.networks.workers[]` must contain exactly one element"))
	}

	for _, msg := range isValidCIDR(string(spec.Networks.Workers[0])) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "workers", "0"), spec.Networks.Workers[0], msg))
	}

	return allErrs
}

func validateShootSpecUpdateGCE(spec, oldSpec gardenv1.ShootSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// OpenStack specific validation

func validateShootSpecOpenStack(spec *gardenv1.ShootSpec, constraints componentconfig.GardenConstraints, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(spec.Zones) != 1 {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "zones"), "`.spec.zones` must contain exactly one element"))
	}

	for _, msg := range isValidCIDR(string(spec.Networks.Workers[0])) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "networks", "workers", "0"), spec.Networks.Workers[0], msg))
	}

	for i, worker := range spec.Workers {
		var (
			idxPath           = fldPath.Child("workers").Index(i)
			autoScalerMaxPath = idxPath.Child("autoScalerMax")
		)
		if worker.AutoScalerMax != worker.AutoScalerMin {
			allErrs = append(allErrs, field.Invalid(autoScalerMaxPath, worker.AutoScalerMax, fmt.Sprintf("`.spec.workers[%d].autoScalerMax` must be equal to `.spec.infrastructure.workers[%d].autoScalerMin`", i, i)))
		}
	}

	return allErrs
}

func validateShootSpecUpdateOpenStack(spec, oldSpec gardenv1.ShootSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	return allErrs
}

// Helper functions

func validateImmutableField(newVal, oldVal interface{}, fldPath *field.Path) field.ErrorList {
	return validation.ValidateImmutableField(newVal, oldVal, fldPath)
}

// isValidCIDR tests that the argument is a valid CIDR range.
func isValidCIDR(value string) []string {
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		return []string{"must be a valid CIDR, (e.g. 10.250.0.0/16)"}
	}
	return nil
}

// validateDomainCount gets a domainCount value and checks if the value is in range between min and max.
func validateDomainCount(domainCount, min, max int) error {
	if domainCount == 0 {
		return fmt.Errorf("must be not 0 or empty")
	}
	if domainCount < min || domainCount > max {
		return fmt.Errorf("must contain a value between %v and %v", min, max)
	}
	return nil
}
