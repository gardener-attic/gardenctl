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

	gardenv1 "github.com/gardener/gardenctl/pkg/apis/garden/v1"
	"github.com/gardener/gardenctl/pkg/garden"
	corev1 "k8s.io/api/core/v1"
)

// DetermineHostedZoneID returns the .spec.dns.hostedZoneID field, if it is set. If it is not set, the function
// will read the default domain secrets and compare their annotations with the specified Shoot domain. If both
// fit, it will return the Hosted Zone of the respective default domain.
func (b *Botanist) DetermineHostedZoneID() (string, error) {
	var (
		infrastructureSecret = b.Secrets["infrastructure"]
		domain               = b.Shoot.Spec.DNS.Domain
		hostedZoneID         = b.Shoot.Spec.DNS.HostedZoneID
		kind                 = b.Shoot.Spec.DNS.Kind
	)

	// If <kind> is 'unmanaged' then we do not create any DNS records. Consequently, we return an empty string for
	// the Hosted Zone ID.
	if kind == gardenv1.DNSUnmanaged {
		return "", nil
	}

	// If no Hosted Zone ID has been specified by the user, then we need to find an appropriate default domain
	// secret which can be used to create the relevant DNS records.
	if hostedZoneID == "" {
		if b.DefaultDomainSecret == nil {
			return "", fmt.Errorf("Could not determine a Hosted Zone for the given domain ('%s' does not fit to any default domain)", domain)
		}
		defaultDomainKind := b.DefaultDomainSecret.Annotations[garden.DNSKind]
		if gardenv1.DNSKind(defaultDomainKind) != kind {
			return "", fmt.Errorf("Could not determine a Hosted Zone for the given domain ('%s' is a default domain of kind '%s')", domain, defaultDomainKind)
		}
		hostedZoneID = b.DefaultDomainSecret.Annotations[garden.DNSHostedZoneID]
		b.Logger.Info("Hosted Zone ID has been determined (default domain is used): " + hostedZoneID)
		return hostedZoneID, nil
	}

	// If a Hosted Zone ID has been specified by the user and it is not a default id, we must ensure that the
	// infrastructure credentials provided by the user contain credentials for the respective DNS kind. We will
	// use them later to create a Terraform job which will create DNS records in the Hosted Zone.
	//if hostedZoneID != defaultDomainSecret.Annotations[garden.DNSHostedZoneID] {
	if b.DefaultDomainSecret == nil {
		switch kind {
		case gardenv1.DNSAWS:
			_, accessKeyFound := infrastructureSecret.Data["accessKeyID"]
			_, secretKeyFound := infrastructureSecret.Data["secretAccessKey"]
			if !accessKeyFound || !secretKeyFound {
				return "", errors.New("Specifying the `.spec.dns.hostedZoneID` field is only possible if the infrastructure secret contains AWS credentials")
			}
		}
	}

	b.Logger.Info("Hosted Zone ID has been given in manifest: " + hostedZoneID)
	return hostedZoneID, nil
}

// DeployInternalDomainDNSRecord deploys the DNS record for the internal cluster domain.
func (b *Botanist) DeployInternalDomainDNSRecord() error {
	return b.DeployDNSRecord(garden.TerraformerPurposeInternalDNS, b.InternalClusterDomain, b.APIServerAddress, true)
}

// DestroyInternalDomainDNSRecord destroys the DNS record for the internal cluster domain.
func (b *Botanist) DestroyInternalDomainDNSRecord() error {
	return b.DestroyDNSRecord(garden.TerraformerPurposeInternalDNS, true)
}

// DeployExternalDomainDNSRecord deploys the DNS record for the external cluster domain.
func (b *Botanist) DeployExternalDomainDNSRecord() error {
	return b.DeployDNSRecord(garden.TerraformerPurposeExternalDNS, b.ExternalClusterDomain, b.InternalClusterDomain, false)
}

// DestroyExternalDomainDNSRecord destroys the DNS record for the external cluster domain.
func (b *Botanist) DestroyExternalDomainDNSRecord() error {
	return b.DestroyDNSRecord(garden.TerraformerPurposeExternalDNS, false)
}

// DeployDNSRecord kicks off a Terraform job of name <alias> which deploys the DNS record for <name> which
// will point to <target>.
func (b *Botanist) DeployDNSRecord(terraformerPurpose, name, target string, purposeInternalDomain bool) error {
	var (
		tfvarsEnvironment []map[string]interface{}
		values            map[string]interface{}
		err               error
	)

	switch b.determineDNSKind(purposeInternalDomain) {
	case gardenv1.DNSAWS:
		tfvarsEnvironment, err = b.GenerateTerraformRoute53VariablesEnvironment(purposeInternalDomain)
		if err != nil {
			return err
		}
		values = b.GenerateTerraformRoute53Config(name, []string{target})
	default:
		return nil
	}

	return garden.
		NewTerraformer(b.Garden, terraformerPurpose).
		SetVariablesEnvironment(tfvarsEnvironment).
		DefineConfig("aws-route53", values).
		Apply()
}

// DestroyDNSRecord kicks off a Terraform job which destroys the DNS record.
func (b *Botanist) DestroyDNSRecord(terraformerPurpose string, purposeInternalDomain bool) error {
	var (
		tfvarsEnvironment []map[string]interface{}
		err               error
	)

	switch b.determineDNSKind(purposeInternalDomain) {
	case gardenv1.DNSAWS:
		tfvarsEnvironment, err = b.GenerateTerraformRoute53VariablesEnvironment(purposeInternalDomain)
		if err != nil {
			return err
		}
	}

	return garden.
		NewTerraformer(b.Garden, terraformerPurpose).
		SetVariablesEnvironment(tfvarsEnvironment).
		Destroy()
}

// GenerateTerraformRoute53VariablesEnvironment generates the environment containing the credentials which
// are required to validate/apply/destroy the Terraform configuration. These environment must contain
// Terraform variables which are prefixed with TF_VAR_.
func (b *Botanist) GenerateTerraformRoute53VariablesEnvironment(purposeInternalDomain bool) ([]map[string]interface{}, error) {
	var (
		accessKeyIDField     = "accessKeyID"
		secretAccessKeyField = "secretAccessKey"
		secret               = b.getDomainCredentials(purposeInternalDomain, accessKeyIDField, secretAccessKeyField)
		keyValueMap          = map[string]string{
			"ACCESS_KEY_ID":     accessKeyIDField,
			"SECRET_ACCESS_KEY": secretAccessKeyField,
		}
	)
	return GenerateTerraformVariablesEnvironment(secret, keyValueMap), nil
}

// GenerateTerraformRoute53Config creates the Terraform variables and the Terraform config (for the DNS record)
// and returns them (these values will be stored as a ConfigMap and a Secret in the Garden cluster.
func (b *Botanist) GenerateTerraformRoute53Config(name string, values []string) map[string]interface{} {
	targetType, _ := garden.IdentifyAddressType(values[0])

	return map[string]interface{}{
		"record": map[string]interface{}{
			"hostedZoneID": b.Shoot.Spec.DNS.HostedZoneID,
			"name":         name,
			"type":         targetType,
			"values":       values,
		},
	}
}

func (b *Botanist) determineDNSKind(purposeInternalDomain bool) gardenv1.DNSKind {
	if purposeInternalDomain {
		return gardenv1.DNSKind(b.Secrets[garden.GardenRoleInternalDomain].Annotations[garden.DNSKind])
	}
	return b.Shoot.Spec.DNS.Kind
}

func (b *Botanist) getDomainCredentials(purposeInternalDomain bool, requiredKeys ...string) *corev1.Secret {
	if purposeInternalDomain {
		return b.Secrets[garden.GardenRoleInternalDomain]
	}

	var (
		defaultDomainSecret  = b.DefaultDomainSecret
		infrastructureSecret = b.Secrets["infrastructure"]
	)

	for _, key := range requiredKeys {
		if _, ok := infrastructureSecret.Data[key]; !ok {
			return defaultDomainSecret
		}
	}
	return infrastructureSecret
}

// GetSeedIngressFQDN returns the fully qualified domain name of ingress sub-resource for the Seed cluster. The
// end result is '<subDomain>.<shoot-name>.<garden-namespace>.ingress.<seed-fqdn>'. It must not exceed 64
// characters in length (see RFC-5280 for details).
func (b *Botanist) GetSeedIngressFQDN(subDomain string) (string, error) {
	result := fmt.Sprintf("%s.%s.%s.ingress.%s", subDomain, b.Shoot.ObjectMeta.Name, b.ProjectName, b.SeedFQDN)
	if len(result) > 64 {
		return "", fmt.Errorf("The FQDN for '%s' cannot be longer than 64 characters", result)
	}
	return result, nil
}

// GetShootIngressFQDN returns the fully qualified domain name of ingress sub-resource for the Shoot cluster. The
// end result is '<subDomain>.ingress.<clusterDomain>'. It must not exceed 64 characters in length (see RFC-5280
// for details).
func (b *Botanist) GetShootIngressFQDN(subDomain string) (string, error) {
	result := fmt.Sprintf("%s.ingress.%s", subDomain, b.Shoot.Spec.DNS.Domain)
	if len(result) > 64 {
		return "", fmt.Errorf("The FQDN for '%s' cannot be longer than 64 characters", result)
	}
	return result, nil
}

func getSeedFQDN(secret *corev1.Secret) (string, error) {
	seedFQDN := secret.Annotations[garden.DNSDomain]
	if seedFQDN == "" {
		return "", fmt.Errorf("Seed cluster's secret '%s' does not have the '%s' annotation", secret.ObjectMeta.Name, garden.DNSDomain)
	}
	if len(seedFQDN) > 32 {
		return "", fmt.Errorf("Seed cluster's FQDN '%s' must not exceed 32 characters", seedFQDN)
	}
	return seedFQDN, nil
}
