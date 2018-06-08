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

package validation_test

import (
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/apis/garden"
	. "github.com/gardener/gardener/pkg/apis/garden/validation"
	"github.com/gardener/gardener/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("validation", func() {
	Describe("#ValidateCloudProfile", func() {
		var (
			metadata = metav1.ObjectMeta{
				Name: "profile",
			}
			dnsProviderConstraint = []garden.DNSProviderConstraint{
				{
					Name: garden.DNSUnmanaged,
				},
			}
			kubernetesVersionConstraint = garden.KubernetesConstraints{
				Versions: []string{"1.8.4"},
			}
			machineType = garden.MachineType{
				Name:   "machine-type-1",
				CPU:    resource.MustParse("2"),
				GPU:    resource.MustParse("0"),
				Memory: resource.MustParse("100Gi"),
			}
			machineTypesConstraint = []garden.MachineType{
				machineType,
			}
			openStackMachineTypesConstraint = []garden.OpenStackMachineType{
				{
					MachineType: machineType,
					VolumeType:  "default",
					VolumeSize:  resource.MustParse("20Gi"),
				},
			}
			volumeTypesConstraint = []garden.VolumeType{
				{
					Name:  "volume-type-1",
					Class: "super-premium",
				},
			}
			zonesConstraint = []garden.Zone{
				{
					Region: "my-region-",
					Names:  []string{"my-region-a"},
				},
			}

			invalidDNSProviders = []garden.DNSProviderConstraint{
				{
					Name: garden.DNSProvider("some-unsupported-provider"),
				},
			}
			invalidKubernetes  = []string{"1.8"}
			invalidMachineType = garden.MachineType{

				Name:   "",
				CPU:    resource.MustParse("-1"),
				GPU:    resource.MustParse("-1"),
				Memory: resource.MustParse("-100Gi"),
			}
			invalidMachineTypes = []garden.MachineType{
				invalidMachineType,
			}
			invalidOpenStackMachineTypes = []garden.OpenStackMachineType{
				{
					MachineType: invalidMachineType,
					VolumeType:  "",
					VolumeSize:  resource.MustParse("-100Gi"),
				},
			}
			invalidVolumeTypes = []garden.VolumeType{
				{
					Name:  "",
					Class: "",
				},
			}
			invalidZones = []garden.Zone{
				{
					Region: "",
					Names:  []string{""},
				},
			}
		)

		It("should forbid empty CloudProfile resources", func() {
			cloudProfile := &garden.CloudProfile{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       garden.CloudProfileSpec{},
			}

			errorList := ValidateCloudProfile(cloudProfile)

			Expect(len(errorList)).To(Equal(2))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("spec.aws/azure/gcp/openstack/local"),
			}))
		})

		Context("tests for AWS cloud profiles", func() {
			var (
				fldPath         = "aws"
				awsCloudProfile *garden.CloudProfile
			)

			BeforeEach(func() {
				awsCloudProfile = &garden.CloudProfile{
					ObjectMeta: metadata,
					Spec: garden.CloudProfileSpec{
						AWS: &garden.AWSProfile{
							Constraints: garden.AWSConstraints{
								DNSProviders: dnsProviderConstraint,
								Kubernetes:   kubernetesVersionConstraint,
								MachineImages: []garden.AWSMachineImageMapping{
									{
										Name: garden.MachineImageCoreOS,
										Regions: []garden.AWSRegionalMachineImage{
											{
												Name: "eu-west-1",
												AMI:  "ami-12345678",
											},
										},
									},
								},
								MachineTypes: machineTypesConstraint,
								VolumeTypes:  volumeTypesConstraint,
								Zones:        zonesConstraint,
							},
						},
					},
				}
			})

			It("should not return any errors", func() {
				errorList := ValidateCloudProfile(awsCloudProfile)

				Expect(len(errorList)).To(Equal(0))
			})

			It("should forbid ca bundles with unsupported format", func() {
				awsCloudProfile.Spec.CABundle = makeStringPointer("unsupported")

				errorList := ValidateCloudProfile(awsCloudProfile)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.caBundle"),
				}))
			})

			Context("dns provider constraints", func() {
				It("should enforce that at least one provider has been defined", func() {
					awsCloudProfile.Spec.AWS.Constraints.DNSProviders = []garden.DNSProviderConstraint{}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders", fldPath)),
					}))
				})

				It("should forbid unsupported providers", func() {
					awsCloudProfile.Spec.AWS.Constraints.DNSProviders = invalidDNSProviders

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders[0]", fldPath)),
					}))
				})
			})

			Context("kubernetes version constraints", func() {
				It("should enforce that at least one version has been defined", func() {
					awsCloudProfile.Spec.AWS.Constraints.Kubernetes.Versions = []string{}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions", fldPath)),
					}))
				})

				It("should forbid versions of a not allowed pattern", func() {
					awsCloudProfile.Spec.AWS.Constraints.Kubernetes.Versions = invalidKubernetes

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions[0]", fldPath)),
					}))
				})
			})

			Context("machine image validation", func() {
				It("should forbid an empty list of machine images", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineImages = []garden.AWSMachineImageMapping{}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages", fldPath)),
					}))
				})

				It("should forbid duplicate names in list of machine images", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineImages = []garden.AWSMachineImageMapping{
						{
							Name: garden.MachineImageCoreOS,
							Regions: []garden.AWSRegionalMachineImage{
								{
									Name: "my-region",
									AMI:  "ami-a1b2c3d4",
								},
							},
						},
						{
							Name: garden.MachineImageCoreOS,
							Regions: []garden.AWSRegionalMachineImage{
								{
									Name: "my-region",
									AMI:  "ami-a1b2c3d4",
								},
							},
						},
					}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[1]", fldPath)),
					}))
				})

				It("should forbid machine images names other than CoreOS", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineImages = []garden.AWSMachineImageMapping{
						{
							Name: garden.MachineImageName("some-unsupported-os"),
							Regions: []garden.AWSRegionalMachineImage{
								{
									Name: "my-region",
									AMI:  "ami-a1b2c3d4",
								},
							},
						},
					}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0]", fldPath)),
					}))
				})

				It("should forbid machine images with no regions", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineImages = []garden.AWSMachineImageMapping{
						{
							Name:    garden.MachineImageCoreOS,
							Regions: []garden.AWSRegionalMachineImage{},
						},
					}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].regions", fldPath)),
					}))
				})

				It("should forbid machine images with duplicate region names", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineImages = []garden.AWSMachineImageMapping{
						{
							Name: garden.MachineImageCoreOS,
							Regions: []garden.AWSRegionalMachineImage{
								{
									Name: "my-region",
									AMI:  "ami-a1b2c3d4",
								},
								{
									Name: "my-region",
									AMI:  "ami-a1b2c3d4",
								},
							},
						},
					}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].regions[1]", fldPath)),
					}))
				})

				It("should forbid machine images with invalid amis", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineImages = []garden.AWSMachineImageMapping{
						{
							Name: garden.MachineImageCoreOS,
							Regions: []garden.AWSRegionalMachineImage{
								{
									Name: "my-region",
									AMI:  "invalid-ami",
								},
							},
						},
					}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].regions[0].ami", fldPath)),
					}))
				})
			})

			Context("machine types validation", func() {
				It("should enforce that at least one machine type has been defined", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineTypes = []garden.MachineType{}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes", fldPath)),
					}))
				})

				It("should forbid machine types with unsupported property values", func() {
					awsCloudProfile.Spec.AWS.Constraints.MachineTypes = invalidMachineTypes

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(4))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].name", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].cpu", fldPath)),
					}))
					Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].gpu", fldPath)),
					}))
					Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].memory", fldPath)),
					}))
				})
			})

			Context("volume types validation", func() {
				It("should enforce that at least one volume type has been defined", func() {
					awsCloudProfile.Spec.AWS.Constraints.VolumeTypes = []garden.VolumeType{}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes", fldPath)),
					}))
				})

				It("should forbid volume types with unsupported property values", func() {
					awsCloudProfile.Spec.AWS.Constraints.VolumeTypes = invalidVolumeTypes

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes[0].name", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes[0].class", fldPath)),
					}))
				})
			})

			Context("zones validation", func() {
				It("should enforce that at least one zone has been defined", func() {
					awsCloudProfile.Spec.AWS.Constraints.Zones = []garden.Zone{}

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones", fldPath)),
					}))
				})

				It("should forbid zones with unsupported name values", func() {
					awsCloudProfile.Spec.AWS.Constraints.Zones = invalidZones

					errorList := ValidateCloudProfile(awsCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones[0].region", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones[0].names[0]", fldPath)),
					}))
				})
			})
		})

		Context("tests for Azure cloud profiles", func() {
			var (
				fldPath           = "azure"
				azureCloudProfile *garden.CloudProfile
			)

			BeforeEach(func() {
				azureCloudProfile = &garden.CloudProfile{
					ObjectMeta: metadata,
					Spec: garden.CloudProfileSpec{
						Azure: &garden.AzureProfile{
							Constraints: garden.AzureConstraints{
								DNSProviders: dnsProviderConstraint,
								Kubernetes:   kubernetesVersionConstraint,
								MachineImages: []garden.AzureMachineImage{
									{
										Name:      garden.MachineImageCoreOS,
										Publisher: "CoreOS",
										Offer:     "CoreOS",
										SKU:       "Beta",
										Version:   "coreos-1.6.4",
									},
								},
								MachineTypes: machineTypesConstraint,
								VolumeTypes:  volumeTypesConstraint,
							},
							CountFaultDomains: []garden.AzureDomainCount{
								{
									Region: "westeurope",
									Count:  0,
								},
							},
							CountUpdateDomains: []garden.AzureDomainCount{
								{
									Region: "westeurope",
									Count:  0,
								},
							},
						},
					},
				}
			})

			Context("dns provider constraints", func() {
				It("should enforce that at least one provider has been defined", func() {
					azureCloudProfile.Spec.Azure.Constraints.DNSProviders = []garden.DNSProviderConstraint{}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders", fldPath)),
					}))
				})

				It("should forbid unsupported providers", func() {
					azureCloudProfile.Spec.Azure.Constraints.DNSProviders = invalidDNSProviders

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders[0]", fldPath)),
					}))
				})
			})

			Context("kubernetes version constraints", func() {
				It("should enforce that at least one version has been defined", func() {
					azureCloudProfile.Spec.Azure.Constraints.Kubernetes.Versions = []string{}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions", fldPath)),
					}))
				})

				It("should forbid versions of a not allowed pattern", func() {
					azureCloudProfile.Spec.Azure.Constraints.Kubernetes.Versions = invalidKubernetes

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions[0]", fldPath)),
					}))
				})
			})

			Context("machine image validation", func() {
				It("should forbid an empty list of machine images", func() {
					azureCloudProfile.Spec.Azure.Constraints.MachineImages = []garden.AzureMachineImage{}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages", fldPath)),
					}))
				})

				It("should forbid duplicate names in list of machine images", func() {
					azureCloudProfile.Spec.Azure.Constraints.MachineImages = []garden.AzureMachineImage{
						{
							Name:      garden.MachineImageCoreOS,
							Publisher: "some-name",
							Offer:     "some-name",
							SKU:       "some-name",
							Version:   "some-name",
						},
						{
							Name:      garden.MachineImageCoreOS,
							Publisher: "some-name",
							Offer:     "some-name",
							SKU:       "some-name",
							Version:   "some-name",
						},
					}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[1]", fldPath)),
					}))
				})

				It("should forbid machine images names other than CoreOS", func() {
					azureCloudProfile.Spec.Azure.Constraints.MachineImages = []garden.AzureMachineImage{
						{
							Name:      garden.MachineImageName("some-unsupported-os"),
							Publisher: "some-name",
							Offer:     "some-name",
							SKU:       "some-name",
							Version:   "some-name",
						},
					}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0]", fldPath)),
					}))
				})

				It("should forbid machine images with empty image names", func() {
					azureCloudProfile.Spec.Azure.Constraints.MachineImages = []garden.AzureMachineImage{
						{
							Name:      garden.MachineImageCoreOS,
							Publisher: "",
							Offer:     "",
							SKU:       "",
							Version:   "",
						},
					}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(4))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].publisher", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].offer", fldPath)),
					}))
					Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].sku", fldPath)),
					}))
					Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].version", fldPath)),
					}))
				})
			})

			Context("machine types validation", func() {
				It("should enforce that at least one machine type has been defined", func() {
					azureCloudProfile.Spec.Azure.Constraints.MachineTypes = []garden.MachineType{}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes", fldPath)),
					}))
				})

				It("should forbid machine types with unsupported property values", func() {
					azureCloudProfile.Spec.Azure.Constraints.MachineTypes = invalidMachineTypes

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(4))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].name", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].cpu", fldPath)),
					}))
					Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].gpu", fldPath)),
					}))
				})
			})

			Context("volume types validation", func() {
				It("should enforce that at least one volume type has been defined", func() {
					azureCloudProfile.Spec.Azure.Constraints.VolumeTypes = []garden.VolumeType{}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes", fldPath)),
					}))
				})

				It("should forbid volume types with unsupported property values", func() {
					azureCloudProfile.Spec.Azure.Constraints.VolumeTypes = invalidVolumeTypes

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes[0].name", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes[0].class", fldPath)),
					}))
				})
			})

			Context("fault domain count validation", func() {
				It("should enforce that at least one fault domain count has been defined", func() {
					azureCloudProfile.Spec.Azure.CountFaultDomains = []garden.AzureDomainCount{}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.countFaultDomains", fldPath)),
					}))
				})

				It("should forbid fault domain count with unsupported format", func() {
					azureCloudProfile.Spec.Azure.CountFaultDomains = []garden.AzureDomainCount{
						{
							Region: "",
							Count:  -1,
						},
					}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.countFaultDomains[0].region", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.countFaultDomains[0].count", fldPath)),
					}))
				})
			})

			Context("update domain count validation", func() {
				It("should enforce that at least one update domain count has been defined", func() {
					azureCloudProfile.Spec.Azure.CountUpdateDomains = []garden.AzureDomainCount{}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.countUpdateDomains", fldPath)),
					}))
				})

				It("should forbid update domain count with unsupported format", func() {
					azureCloudProfile.Spec.Azure.CountUpdateDomains = []garden.AzureDomainCount{
						{
							Region: "",
							Count:  -1,
						},
					}

					errorList := ValidateCloudProfile(azureCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.countUpdateDomains[0].region", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.countUpdateDomains[0].count", fldPath)),
					}))
				})
			})
		})

		Context("tests for GCP cloud profiles", func() {
			var (
				fldPath         = "gcp"
				gcpCloudProfile *garden.CloudProfile
			)

			BeforeEach(func() {
				gcpCloudProfile = &garden.CloudProfile{
					ObjectMeta: metadata,
					Spec: garden.CloudProfileSpec{
						GCP: &garden.GCPProfile{
							Constraints: garden.GCPConstraints{
								DNSProviders: dnsProviderConstraint,
								Kubernetes:   kubernetesVersionConstraint,
								MachineImages: []garden.GCPMachineImage{
									{
										Name:  garden.MachineImageCoreOS,
										Image: "coreos-1.6.4",
									},
								},
								MachineTypes: machineTypesConstraint,
								VolumeTypes:  volumeTypesConstraint,
								Zones:        zonesConstraint,
							},
						},
					},
				}
			})

			Context("dns provider constraints", func() {
				It("should enforce that at least one provider has been defined", func() {
					gcpCloudProfile.Spec.GCP.Constraints.DNSProviders = []garden.DNSProviderConstraint{}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders", fldPath)),
					}))
				})

				It("should forbid unsupported providers", func() {
					gcpCloudProfile.Spec.GCP.Constraints.DNSProviders = invalidDNSProviders

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders[0]", fldPath)),
					}))
				})
			})

			Context("kubernetes version constraints", func() {
				It("should enforce that at least one version has been defined", func() {
					gcpCloudProfile.Spec.GCP.Constraints.Kubernetes.Versions = []string{}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions", fldPath)),
					}))
				})

				It("should forbid versions of a not allowed pattern", func() {
					gcpCloudProfile.Spec.GCP.Constraints.Kubernetes.Versions = invalidKubernetes

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions[0]", fldPath)),
					}))
				})
			})

			Context("machine image validation", func() {
				It("should forbid an empty list of machine images", func() {
					gcpCloudProfile.Spec.GCP.Constraints.MachineImages = []garden.GCPMachineImage{}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages", fldPath)),
					}))
				})

				It("should forbid duplicate names in list of machine images", func() {
					gcpCloudProfile.Spec.GCP.Constraints.MachineImages = []garden.GCPMachineImage{
						{
							Name:  garden.MachineImageCoreOS,
							Image: "some-image",
						},
						{
							Name:  garden.MachineImageCoreOS,
							Image: "some-image",
						},
					}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[1]", fldPath)),
					}))
				})

				It("should forbid machine images names other than CoreOS", func() {
					gcpCloudProfile.Spec.GCP.Constraints.MachineImages = []garden.GCPMachineImage{
						{
							Name:  garden.MachineImageName("some-unsupported-os"),
							Image: "some-image",
						},
					}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0]", fldPath)),
					}))
				})

				It("should forbid machine images with empty image names", func() {
					gcpCloudProfile.Spec.GCP.Constraints.MachineImages = []garden.GCPMachineImage{
						{
							Name:  garden.MachineImageCoreOS,
							Image: "",
						},
					}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].image", fldPath)),
					}))
				})
			})

			Context("machine types validation", func() {
				It("should enforce that at least one machine type has been defined", func() {
					gcpCloudProfile.Spec.GCP.Constraints.MachineTypes = []garden.MachineType{}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes", fldPath)),
					}))
				})

				It("should forbid machine types with unsupported property values", func() {
					gcpCloudProfile.Spec.GCP.Constraints.MachineTypes = invalidMachineTypes

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(4))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].name", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].cpu", fldPath)),
					}))
					Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].gpu", fldPath)),
					}))
					Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].memory", fldPath)),
					}))
				})
			})

			Context("volume types validation", func() {
				It("should enforce that at least one volume type has been defined", func() {
					gcpCloudProfile.Spec.GCP.Constraints.VolumeTypes = []garden.VolumeType{}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes", fldPath)),
					}))
				})

				It("should forbid volume types with unsupported property values", func() {
					gcpCloudProfile.Spec.GCP.Constraints.VolumeTypes = invalidVolumeTypes

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes[0].name", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.volumeTypes[0].class", fldPath)),
					}))
				})
			})

			Context("zones validation", func() {
				It("should enforce that at least one zone has been defined", func() {
					gcpCloudProfile.Spec.GCP.Constraints.Zones = []garden.Zone{}

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones", fldPath)),
					}))
				})

				It("should forbid zones with unsupported name values", func() {
					gcpCloudProfile.Spec.GCP.Constraints.Zones = invalidZones

					errorList := ValidateCloudProfile(gcpCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones[0].region", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones[0].names[0]", fldPath)),
					}))
				})
			})
		})

		Context("tests for OpenStack cloud profiles", func() {
			var (
				fldPath               = "openstack"
				openStackCloudProfile *garden.CloudProfile
			)

			BeforeEach(func() {
				openStackCloudProfile = &garden.CloudProfile{
					ObjectMeta: metadata,
					Spec: garden.CloudProfileSpec{
						OpenStack: &garden.OpenStackProfile{
							Constraints: garden.OpenStackConstraints{
								DNSProviders: dnsProviderConstraint,
								FloatingPools: []garden.OpenStackFloatingPool{
									{
										Name: "MY-POOL",
									},
								},
								Kubernetes: kubernetesVersionConstraint,
								LoadBalancerProviders: []garden.OpenStackLoadBalancerProvider{
									{
										Name: "haproxy",
									},
								},

								MachineImages: []garden.OpenStackMachineImage{
									{
										Name:  garden.MachineImageCoreOS,
										Image: "coreos-1.6.4",
									},
								},
								MachineTypes: openStackMachineTypesConstraint,
								Zones:        zonesConstraint,
							},
							KeyStoneURL: "http://url-to-keystone/v3",
						},
					},
				}
			})

			Context("dns provider constraints", func() {
				It("should enforce that at least one provider has been defined", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.DNSProviders = []garden.DNSProviderConstraint{}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders", fldPath)),
					}))
				})

				It("should forbid unsupported providers", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.DNSProviders = invalidDNSProviders

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.dnsProviders[0]", fldPath)),
					}))
				})
			})

			Context("floating pools constraints", func() {
				It("should enforce that at least one pool has been defined", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.FloatingPools = []garden.OpenStackFloatingPool{}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.floatingPools", fldPath)),
					}))
				})

				It("should forbid unsupported providers", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.FloatingPools = []garden.OpenStackFloatingPool{
						{
							Name: "",
						},
					}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.floatingPools[0].name", fldPath)),
					}))
				})
			})

			Context("kubernetes version constraints", func() {
				It("should enforce that at least one version has been defined", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.Kubernetes.Versions = []string{}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions", fldPath)),
					}))
				})

				It("should forbid versions of a not allowed pattern", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.Kubernetes.Versions = invalidKubernetes

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.kubernetes.versions[0]", fldPath)),
					}))
				})
			})

			Context("load balancer provider constraints", func() {
				It("should enforce that at least one provider has been defined", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.LoadBalancerProviders = []garden.OpenStackLoadBalancerProvider{}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.loadBalancerProviders", fldPath)),
					}))
				})

				It("should forbid unsupported providers", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.LoadBalancerProviders = []garden.OpenStackLoadBalancerProvider{
						{
							Name: "",
						},
					}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.loadBalancerProviders[0].name", fldPath)),
					}))
				})
			})

			Context("machine image validation", func() {
				It("should forbid an empty list of machine images", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.MachineImages = []garden.OpenStackMachineImage{}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages", fldPath)),
					}))
				})

				It("should forbid duplicate names in list of machine images", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.MachineImages = []garden.OpenStackMachineImage{
						{
							Name:  garden.MachineImageCoreOS,
							Image: "some-image",
						},
						{
							Name:  garden.MachineImageCoreOS,
							Image: "some-image",
						},
					}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeDuplicate),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[1]", fldPath)),
					}))
				})

				It("should forbid machine images names other than CoreOS", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.MachineImages = []garden.OpenStackMachineImage{
						{
							Name:  garden.MachineImageName("some-unsupported-os"),
							Image: "some-image",
						},
					}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeNotSupported),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0]", fldPath)),
					}))
				})

				It("should forbid machine images with empty image names", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.MachineImages = []garden.OpenStackMachineImage{
						{
							Name:  garden.MachineImageCoreOS,
							Image: "",
						},
					}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineImages[0].image", fldPath)),
					}))
				})
			})

			Context("machine types validation", func() {
				It("should enforce that at least one machine type has been defined", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.MachineTypes = []garden.OpenStackMachineType{}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes", fldPath)),
					}))
				})

				It("should forbid machine types with unsupported property values", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.MachineTypes = invalidOpenStackMachineTypes

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(6))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].volumeType", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].volumeSize", fldPath)),
					}))
					Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].name", fldPath)),
					}))
					Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].cpu", fldPath)),
					}))
					Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].gpu", fldPath)),
					}))
					Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeInvalid),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.machineTypes[0].memory", fldPath)),
					}))
				})
			})

			Context("zones validation", func() {
				It("should enforce that at least one zone has been defined", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.Zones = []garden.Zone{}

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones", fldPath)),
					}))
				})

				It("should forbid zones with unsupported name values", func() {
					openStackCloudProfile.Spec.OpenStack.Constraints.Zones = invalidZones

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(2))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones[0].region", fldPath)),
					}))
					Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.constraints.zones[0].names[0]", fldPath)),
					}))
				})
			})

			Context("keystone url validation", func() {
				It("should forbid keystone urls with unsupported format", func() {
					openStackCloudProfile.Spec.OpenStack.KeyStoneURL = ""

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.keyStoneURL", fldPath)),
					}))
				})
			})

			Context("dhcp domain validation", func() {
				It("should forbid not specifying a value when the key is present", func() {
					openStackCloudProfile.Spec.OpenStack.DHCPDomain = makeStringPointer("")

					errorList := ValidateCloudProfile(openStackCloudProfile)

					Expect(len(errorList)).To(Equal(1))
					Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
						"Type":  Equal(field.ErrorTypeRequired),
						"Field": Equal(fmt.Sprintf("spec.%s.dhcpDomain", fldPath)),
					}))
				})
			})
		})
	})

	Describe("#ValidateSeed", func() {
		var seed *garden.Seed

		BeforeEach(func() {
			seed = &garden.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "seed-1",
				},
				Spec: garden.SeedSpec{
					Cloud: garden.SeedCloud{
						Profile: "aws",
						Region:  "eu-west-1",
					},
					IngressDomain: "ingress.my-seed-1.example.com",
					SecretRef: corev1.SecretReference{
						Name:      "seed-aws",
						Namespace: "garden",
					},
					Networks: garden.SeedNetworks{
						Nodes:    garden.CIDR("10.250.0.0/16"),
						Pods:     garden.CIDR("100.96.0.0/11"),
						Services: garden.CIDR("100.64.0.0/13"),
					},
				},
			}
		})

		It("should not return any errors", func() {
			errorList := ValidateSeed(seed)

			Expect(len(errorList)).To(Equal(0))
		})

		It("should forbid Seed resources with empty metadata", func() {
			seed.ObjectMeta = metav1.ObjectMeta{}

			errorList := ValidateSeed(seed)

			Expect(len(errorList)).To(Equal(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			}))
		})

		It("should forbid Seed specification with empty or invalid keys", func() {
			seed.Spec.Cloud = garden.SeedCloud{}
			seed.Spec.IngressDomain = "invalid-domain-name"
			seed.Spec.SecretRef = corev1.SecretReference{}
			seed.Spec.Networks = garden.SeedNetworks{
				Nodes:    garden.CIDR("invalid-cidr"),
				Pods:     garden.CIDR("300.300.300.300/300"),
				Services: garden.CIDR("invalid-cidr"),
			}

			errorList := ValidateSeed(seed)

			Expect(len(errorList)).To(Equal(8))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.cloud.profile"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.cloud.region"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.ingressDomain"),
			}))
			Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.secretRef.name"),
			}))
			Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.secretRef.namespace"),
			}))
			Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.networks.nodes"),
			}))
			Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.networks.pods"),
			}))
			Expect(*errorList[7]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.networks.services"),
			}))
		})
	})

	Describe("#ValidateQuota", func() {
		var quota *garden.Quota

		BeforeEach(func() {
			quota = &garden.Quota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "quota-1",
					Namespace: "my-namespace",
				},
				Spec: garden.QuotaSpec{
					Scope: garden.QuotaScopeProject,
					Metrics: corev1.ResourceList{
						"cpu":    resource.MustParse("200"),
						"memory": resource.MustParse("4000Gi"),
					},
				},
			}
		})

		It("should not return any errors", func() {
			errorList := ValidateQuota(quota)

			Expect(len(errorList)).To(Equal(0))
		})

		It("should forbid Quota specification with empty or invalid keys", func() {
			quota.ObjectMeta = metav1.ObjectMeta{}
			quota.Spec.Scope = garden.QuotaScope("does-not-exist")
			quota.Spec.Metrics["key"] = resource.MustParse("-100")

			errorList := ValidateQuota(quota)

			Expect(len(errorList)).To(Equal(5))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("spec.scope"),
			}))
			Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.metrics[key]"),
			}))
			Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.metrics[key]"),
			}))
		})
	})

	Describe("#ValidateSecretBinding", func() {
		var secretBinding *garden.SecretBinding

		BeforeEach(func() {
			secretBinding = &garden.SecretBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "profile",
					Namespace: "garden",
				},
				SecretRef: corev1.SecretReference{
					Name:      "my-secret",
					Namespace: "my-namespace",
				},
			}
		})

		It("should not return any errors", func() {
			errorList := ValidateSecretBinding(secretBinding)

			Expect(len(errorList)).To(Equal(0))
		})

		It("should forbid empty SecretBinding resources", func() {
			secretBinding.ObjectMeta = metav1.ObjectMeta{}
			secretBinding.SecretRef = corev1.SecretReference{}

			errorList := ValidateSecretBinding(secretBinding)

			Expect(len(errorList)).To(Equal(3))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("secretRef.name"),
			}))
		})

		It("should forbid empty stated Quota names", func() {
			secretBinding.Quotas = []corev1.ObjectReference{
				{},
			}

			errorList := ValidateSecretBinding(secretBinding)

			Expect(len(errorList)).To(Equal(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("quotas[0].name"),
			}))
		})
	})

	Describe("#ValidateShoot, #ValidateShootUpdate", func() {
		var (
			shoot *garden.Shoot

			hostedZoneID = "ABCDEF1234"
			domain       = "my-cluster.example.com"

			nodeCIDR    = garden.CIDR("10.250.0.0/16")
			podCIDR     = garden.CIDR("100.96.0.0/11")
			serviceCIDR = garden.CIDR("100.64.0.0/13")
			invalidCIDR = garden.CIDR("invalid-cidr")

			invalidBackup = &garden.Backup{
				Schedule: "76 * * * *",
				Maximum:  0,
			}
			addon = garden.Addon{
				Enabled: true,
			}
			k8sNetworks = garden.K8SNetworks{
				Nodes:    &nodeCIDR,
				Pods:     &podCIDR,
				Services: &serviceCIDR,
			}
			invalidK8sNetworks = garden.K8SNetworks{
				Nodes:    &invalidCIDR,
				Pods:     &invalidCIDR,
				Services: &invalidCIDR,
			}
			worker = garden.Worker{
				Name:          "worker-name",
				MachineType:   "large",
				AutoScalerMin: 1,
				AutoScalerMax: 1,
			}
			invalidWorker = garden.Worker{
				Name:          "",
				MachineType:   "",
				AutoScalerMin: -1,
				AutoScalerMax: -2,
			}
			invalidWorkerName = garden.Worker{
				Name:          "not_compliant",
				MachineType:   "large",
				AutoScalerMin: 1,
				AutoScalerMax: 1,
			}
			invalidWorkerTooLongName = garden.Worker{
				Name:          "worker-name-is-too-long",
				MachineType:   "large",
				AutoScalerMin: 1,
				AutoScalerMax: 1,
			}
		)

		BeforeEach(func() {
			shoot = &garden.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "my-namespace",
				},
				Spec: garden.ShootSpec{
					Addons: &garden.Addons{
						Kube2IAM: &garden.Kube2IAM{
							Addon: addon,
							Roles: []garden.Kube2IAMRole{
								{
									Name:        "iam-role",
									Description: "some-text",
									Policy:      `{"some-valid": "json-document"}`,
								},
							},
						},
						KubernetesDashboard: &garden.KubernetesDashboard{
							Addon: addon,
						},
						ClusterAutoscaler: &garden.ClusterAutoscaler{
							Addon: addon,
						},
						NginxIngress: &garden.NginxIngress{
							Addon: addon,
						},
						Monocular: &garden.Monocular{
							Addon: addon,
						},
						KubeLego: &garden.KubeLego{
							Addon: addon,
							Mail:  "info@example.com",
						},
					},
					Backup: &garden.Backup{
						Schedule: "*/1 * * * *",
						Maximum:  2,
					},
					Cloud: garden.Cloud{
						Profile: "aws-profile",
						Region:  "eu-west-1",
						SecretBindingRef: corev1.LocalObjectReference{
							Name: "my-secret",
						},
						AWS: &garden.AWSCloud{
							Networks: garden.AWSNetworks{
								K8SNetworks: k8sNetworks,
								Internal:    []garden.CIDR{"10.250.0.0/16"},
								Public:      []garden.CIDR{"10.250.0.0/16"},
								Workers:     []garden.CIDR{"10.250.0.0/16"},
								VPC: garden.AWSVPC{
									CIDR: &nodeCIDR,
								},
							},
							Workers: []garden.AWSWorker{
								{
									Worker:     worker,
									VolumeSize: "20Gi",
									VolumeType: "default",
								},
							},
							Zones: []string{"eu-west-1a"},
						},
					},
					DNS: garden.DNS{
						Provider:     garden.DNSAWSRoute53,
						HostedZoneID: &hostedZoneID,
						Domain:       &domain,
					},
					Kubernetes: garden.Kubernetes{
						Version: "1.8.2",
						KubeAPIServer: &garden.KubeAPIServerConfig{
							OIDCConfig: &garden.OIDCConfig{
								CABundle:       makeStringPointer("-----BEGIN CERTIFICATE-----\nMIICRzCCAfGgAwIBAgIJALMb7ecMIk3MMA0GCSqGSIb3DQEBCwUAMH4xCzAJBgNV\nBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNVBAcMBkxvbmRvbjEYMBYGA1UE\nCgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1JVCBEZXBhcnRtZW50MRswGQYD\nVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwIBcNMTcwNDI2MjMyNjUyWhgPMjExNzA0\nMDIyMzI2NTJaMH4xCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xDzANBgNV\nBAcMBkxvbmRvbjEYMBYGA1UECgwPR2xvYmFsIFNlY3VyaXR5MRYwFAYDVQQLDA1J\nVCBEZXBhcnRtZW50MRswGQYDVQQDDBJ0ZXN0LWNlcnRpZmljYXRlLTAwXDANBgkq\nhkiG9w0BAQEFAANLADBIAkEAtBMa7NWpv3BVlKTCPGO/LEsguKqWHBtKzweMY2CV\ntAL1rQm913huhxF9w+ai76KQ3MHK5IVnLJjYYA5MzP2H5QIDAQABo1AwTjAdBgNV\nHQ4EFgQU22iy8aWkNSxv0nBxFxerfsvnZVMwHwYDVR0jBBgwFoAU22iy8aWkNSxv\n0nBxFxerfsvnZVMwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAANBAEOefGbV\nNcHxklaW06w6OBYJPwpIhCVozC1qdxGX1dg8VkEKzjOzjgqVD30m59OFmSlBmHsl\nnkVA6wyOSDYBf3o=\n-----END CERTIFICATE-----"),
								ClientID:       makeStringPointer("client-id"),
								GroupsClaim:    makeStringPointer("groups-claim"),
								GroupsPrefix:   makeStringPointer("groups-prefix"),
								IssuerURL:      makeStringPointer("https://some-endpoint.com"),
								UsernameClaim:  makeStringPointer("user-claim"),
								UsernamePrefix: makeStringPointer("user-prefix"),
							},
						},
					},
					Maintenance: &garden.Maintenance{
						AutoUpdate: &garden.MaintenanceAutoUpdate{
							KubernetesVersion: true,
						},
						TimeWindow: &garden.MaintenanceTimeWindow{
							Begin: "220000+0100",
							End:   "230000+0100",
						},
					},
				},
			}
		})

		It("should forbid shoots containing two consecutive hyphens", func() {
			shoot.ObjectMeta.Name = "sho--ot"

			errorList := ValidateShoot(shoot)

			Expect(len(errorList)).To(Equal(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("metadata.name"),
			}))
		})

		It("should forbid empty Shoot resources", func() {
			shoot := &garden.Shoot{
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       garden.ShootSpec{},
			}

			errorList := ValidateShoot(shoot)

			Expect(len(errorList)).To(Equal(3))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("spec.cloud.aws/azure/gcp/openstack/local"),
			}))
		})

		It("should forbid unsupported addon configuration", func() {
			shoot.Spec.Addons.Kube2IAM.Roles = []garden.Kube2IAMRole{
				{
					Name:        "",
					Description: "",
					Policy:      "invalid-json",
				},
			}
			shoot.Spec.Addons.KubeLego.Mail = "some-invalid-email"

			errorList := ValidateShoot(shoot)

			Expect(len(errorList)).To(Equal(4))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.addons.kube2iam.roles[0].name"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.addons.kube2iam.roles[0].description"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.addons.kube2iam.roles[0].policy"),
			}))
			Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.addons.kube-lego.mail"),
			}))
		})

		It("should forbid unsupported cloud specification (provider independent)", func() {
			shoot.Spec.Cloud.Profile = ""
			shoot.Spec.Cloud.Region = ""
			shoot.Spec.Cloud.SecretBindingRef = corev1.LocalObjectReference{
				Name: "",
			}
			shoot.Spec.Cloud.Seed = makeStringPointer("")

			errorList := ValidateShoot(shoot)

			Expect(len(errorList)).To(Equal(4))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.cloud.profile"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.cloud.region"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.cloud.secretBindingRef.name"),
			}))
			Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.cloud.seed"),
			}))
		})

		It("should forbid updating some cloud keys", func() {
			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.Cloud.Profile = "another-profile"
			newShoot.Spec.Cloud.Region = "another-region"
			newShoot.Spec.Cloud.SecretBindingRef = corev1.LocalObjectReference{
				Name: "another-reference",
			}
			newShoot.Spec.Cloud.Seed = makeStringPointer("another-seed")

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(len(errorList)).To(Equal(3))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.cloud.profile"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.cloud.region"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.cloud.seed"),
			}))
		})

		Context("AWS specific validation", func() {
			var (
				fldPath  = "aws"
				awsCloud *garden.AWSCloud
			)

			BeforeEach(func() {
				awsCloud = &garden.AWSCloud{
					Networks: garden.AWSNetworks{
						K8SNetworks: k8sNetworks,
						Internal:    []garden.CIDR{"10.250.0.0/16"},
						Public:      []garden.CIDR{"10.250.0.0/16"},
						Workers:     []garden.CIDR{"10.250.0.0/16"},
						VPC: garden.AWSVPC{
							CIDR: &nodeCIDR,
						},
					},
					Workers: []garden.AWSWorker{
						{
							Worker:     worker,
							VolumeSize: "20Gi",
							VolumeType: "default",
						},
					},
					Zones: []string{"eu-west-1a"},
				}
				shoot.Spec.Cloud.AWS = awsCloud
			})

			It("should not return any errors", func() {
				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(0))
			})

			It("should forbid invalid backup configuration", func() {
				shoot.Spec.Backup = invalidBackup

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.backup.schedule"),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.backup.maximum"),
				}))
			})

			It("should forbid invalid network configuration", func() {
				shoot.Spec.Cloud.AWS.Networks.Internal = []garden.CIDR{"one cidr", "another cidr"}
				shoot.Spec.Cloud.AWS.Networks.Public = []garden.CIDR{"invalid-cidr", "another cidr"}
				shoot.Spec.Cloud.AWS.Networks.Workers = []garden.CIDR{"invalid-cidr", "another cidr"}
				shoot.Spec.Cloud.AWS.Networks.K8SNetworks = invalidK8sNetworks
				shoot.Spec.Cloud.AWS.Networks.VPC = garden.AWSVPC{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(13))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.nodes", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.pods", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.services", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.internal", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.internal[0]", fldPath)),
				}))
				Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.internal[1]", fldPath)),
				}))
				Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.public", fldPath)),
				}))
				Expect(*errorList[7]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.public[0]", fldPath)),
				}))
				Expect(*errorList[8]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.public[1]", fldPath)),
				}))
				Expect(*errorList[9]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers", fldPath)),
				}))
				Expect(*errorList[10]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers[0]", fldPath)),
				}))
				Expect(*errorList[11]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers[1]", fldPath)),
				}))
				Expect(*errorList[12]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.vpc", fldPath)),
				}))
			})

			It("should forbid invalid VPC CIDRs", func() {
				shoot.Spec.Cloud.AWS.Networks.VPC.CIDR = &invalidCIDR

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.vpc.cidr", fldPath)),
				}))
			})

			It("should forbid an empty worker list", func() {
				shoot.Spec.Cloud.AWS.Workers = []garden.AWSWorker{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers", fldPath)),
				}))
			})

			It("should enforce unique worker names", func() {
				shoot.Spec.Cloud.AWS.Workers = []garden.AWSWorker{
					{
						Worker:     worker,
						VolumeSize: "20Gi",
						VolumeType: "default",
					},
					{
						Worker:     worker,
						VolumeSize: "20Gi",
						VolumeType: "default",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[1]", fldPath)),
				}))
			})

			It("should forbid invalid worker configuration", func() {
				shoot.Spec.Cloud.AWS.Workers = []garden.AWSWorker{
					{
						Worker:     invalidWorker,
						VolumeSize: "hugo",
						VolumeType: "",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(7))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].machineType", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMin", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeSize", fldPath)),
				}))
				Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeType", fldPath)),
				}))
			})

			It("should forbid worker pools with too less volume size", func() {
				shoot.Spec.Cloud.AWS.Workers = []garden.AWSWorker{
					{
						Worker:     worker,
						VolumeSize: "10Gi",
						VolumeType: "gp2",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeSize", fldPath)),
				}))
			})

			It("should forbid too long worker names", func() {
				shoot.Spec.Cloud.AWS.Workers[0].Worker = invalidWorkerTooLongName

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeTooLong),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid worker pools with names that are not DNS-1123 subdomain compliant", func() {
				shoot.Spec.Cloud.AWS.Workers = []garden.AWSWorker{
					{
						Worker:     invalidWorkerName,
						VolumeSize: "20Gi",
						VolumeType: "gp2",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid an empty zones list", func() {
				shoot.Spec.Cloud.AWS.Zones = []string{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.zones", fldPath)),
				}))
			})

			It("should forbid updating networks and zones", func() {
				newShoot := prepareShootForUpdate(shoot)
				cidr := garden.CIDR("255.255.255.255/32")
				newShoot.Spec.Cloud.AWS.Networks.Pods = &cidr
				newShoot.Spec.Cloud.AWS.Zones = []string{"another-zone"}

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.zones", fldPath)),
				}))
			})

			It("should forbid removing the AWS section", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Cloud.AWS = nil

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.cloud.aws/azure/gcp/openstack/local"),
				}))
			})
		})

		Context("Azure specific validation", func() {
			var (
				fldPath    = "azure"
				azureCloud *garden.AzureCloud
			)

			BeforeEach(func() {
				azureCloud = &garden.AzureCloud{
					Networks: garden.AzureNetworks{
						K8SNetworks: k8sNetworks,
						Workers:     garden.CIDR("10.250.0.0/16"),
						VNet: garden.AzureVNet{
							CIDR: &nodeCIDR,
						},
					},
					Workers: []garden.AzureWorker{
						{
							Worker:     worker,
							VolumeSize: "35Gi",
							VolumeType: "default",
						},
					},
				}
				shoot.Spec.Cloud.AWS = nil
				shoot.Spec.Cloud.Azure = azureCloud
			})

			It("should not return any errors", func() {
				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(0))
			})

			It("should forbid invalid backup configuration", func() {
				shoot.Spec.Backup = invalidBackup

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.backup.schedule"),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.backup.maximum"),
				}))
			})

			It("should forbid specifying a resource group configuration", func() {
				shoot.Spec.Cloud.Azure.ResourceGroup = &garden.AzureResourceGroup{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.resourceGroup.name", fldPath)),
				}))
			})

			It("should forbid specifying a vnet name", func() {
				shoot.Spec.Cloud.Azure.Networks.VNet = garden.AzureVNet{
					Name: makeStringPointer("existing-vnet"),
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.vnet.name", fldPath)),
				}))
			})

			It("should forbid invalid network configuration", func() {
				shoot.Spec.Cloud.Azure.Networks.Workers = invalidCIDR
				shoot.Spec.Cloud.Azure.Networks.K8SNetworks = invalidK8sNetworks
				shoot.Spec.Cloud.Azure.Networks.VNet = garden.AzureVNet{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(5))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.vnet.cidr", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.nodes", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.pods", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.services", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers", fldPath)),
				}))
			})

			It("should forbid invalid VNet CIDRs", func() {
				shoot.Spec.Cloud.Azure.Networks.VNet.CIDR = &invalidCIDR

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.vnet.cidr", fldPath)),
				}))
			})

			It("should forbid an empty worker list", func() {
				shoot.Spec.Cloud.Azure.Workers = []garden.AzureWorker{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers", fldPath)),
				}))
			})

			It("should enforce unique worker names", func() {
				shoot.Spec.Cloud.Azure.Workers = []garden.AzureWorker{
					{
						Worker:     worker,
						VolumeSize: "35Gi",
						VolumeType: "default",
					},
					{
						Worker:     worker,
						VolumeSize: "35Gi",
						VolumeType: "default",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[1]", fldPath)),
				}))
			})

			It("should forbid invalid worker configuration", func() {
				shoot.Spec.Cloud.Azure.Workers = []garden.AzureWorker{
					{
						Worker:     invalidWorker,
						VolumeSize: "hugo",
						VolumeType: "",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(8))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].machineType", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMin", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeSize", fldPath)),
				}))
				Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeType", fldPath)),
				}))
				Expect(*errorList[7]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
			})

			It("should forbid worker pools with too less volume size", func() {
				shoot.Spec.Cloud.Azure.Workers = []garden.AzureWorker{
					{
						Worker:     worker,
						VolumeSize: "30Gi",
						VolumeType: "gp2",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeSize", fldPath)),
				}))
			})

			It("should forbid worker volume sizes smaller than 35Gi", func() {
				shoot.Spec.Cloud.Azure.Workers = []garden.AzureWorker{
					{
						Worker:     worker,
						VolumeSize: "34Gi",
						VolumeType: "ok",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeSize", fldPath)),
				}))
			})

			It("should forbid too long worker names", func() {
				shoot.Spec.Cloud.Azure.Workers[0].Worker = invalidWorkerTooLongName

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeTooLong),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid worker pools with names that are not DNS-1123 subdomain compliant", func() {
				shoot.Spec.Cloud.Azure.Workers = []garden.AzureWorker{
					{
						Worker:     invalidWorkerName,
						VolumeSize: "35Gi",
						VolumeType: "ok",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid workers with auto scaling configured", func() {
				shoot.Spec.Cloud.Azure.Workers[0].AutoScalerMax = shoot.Spec.Cloud.Azure.Workers[0].AutoScalerMin + 1

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
			})

			It("should forbid updating resource group and zones", func() {
				newShoot := prepareShootForUpdate(shoot)
				cidr := garden.CIDR("255.255.255.255/32")
				newShoot.Spec.Cloud.Azure.Networks.Pods = &cidr
				newShoot.Spec.Cloud.Azure.ResourceGroup = &garden.AzureResourceGroup{
					Name: "another-group",
				}

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(3))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.resourceGroup", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.resourceGroup.name", fldPath)),
				}))
			})

			It("should forbid removing the Azure section", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Cloud.Azure = nil

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.cloud.aws/azure/gcp/openstack/local"),
				}))
			})
		})

		Context("GCP specific validation", func() {
			var (
				fldPath  = "gcp"
				gcpCloud *garden.GCPCloud
			)

			BeforeEach(func() {
				gcpCloud = &garden.GCPCloud{
					Networks: garden.GCPNetworks{
						K8SNetworks: k8sNetworks,
						Workers:     []garden.CIDR{"10.250.0.0/16"},
						VPC: &garden.GCPVPC{
							Name: "hugo",
						},
					},
					Workers: []garden.GCPWorker{
						{
							Worker:     worker,
							VolumeSize: "20Gi",
							VolumeType: "default",
						},
					},
					Zones: []string{"europe-west1-b"},
				}
				shoot.Spec.Cloud.AWS = nil
				shoot.Spec.Cloud.GCP = gcpCloud
				shoot.Spec.Backup = nil
			})

			It("should not return any errors", func() {
				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(0))
			})

			It("should forbid invalid network configuration", func() {
				shoot.Spec.Cloud.GCP.Networks.Workers = []garden.CIDR{"invalid-cidr", "another cidr"}
				shoot.Spec.Cloud.GCP.Networks.K8SNetworks = invalidK8sNetworks
				shoot.Spec.Cloud.GCP.Networks.VPC = &garden.GCPVPC{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(7))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.nodes", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.pods", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.services", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers[0]", fldPath)),
				}))
				Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers[1]", fldPath)),
				}))
				Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.vpc.name", fldPath)),
				}))
			})

			It("should forbid an empty worker list", func() {
				shoot.Spec.Cloud.GCP.Workers = []garden.GCPWorker{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers", fldPath)),
				}))
			})

			It("should enforce unique worker names", func() {
				shoot.Spec.Cloud.GCP.Workers = []garden.GCPWorker{
					{
						Worker:     worker,
						VolumeSize: "20Gi",
						VolumeType: "default",
					},
					{
						Worker:     worker,
						VolumeSize: "20Gi",
						VolumeType: "default",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[1]", fldPath)),
				}))
			})

			It("should forbid invalid worker configuration", func() {
				shoot.Spec.Cloud.GCP.Workers = []garden.GCPWorker{
					{
						Worker:     invalidWorker,
						VolumeSize: "hugo",
						VolumeType: "",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(7))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].machineType", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMin", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeSize", fldPath)),
				}))
				Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeType", fldPath)),
				}))
			})

			It("should forbid worker pools with too less volume size", func() {
				shoot.Spec.Cloud.GCP.Workers = []garden.GCPWorker{
					{
						Worker:     worker,
						VolumeSize: "19Gi",
						VolumeType: "default",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].volumeSize", fldPath)),
				}))
			})

			It("should forbid too long worker names", func() {
				shoot.Spec.Cloud.GCP.Workers[0].Worker = invalidWorkerTooLongName

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeTooLong),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid worker pools with names that are not DNS-1123 subdomain compliant", func() {
				shoot.Spec.Cloud.GCP.Workers = []garden.GCPWorker{
					{
						Worker:     invalidWorkerName,
						VolumeSize: "20Gi",
						VolumeType: "default",
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid an empty zones list", func() {
				shoot.Spec.Cloud.GCP.Zones = []string{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.zones", fldPath)),
				}))
			})

			It("should forbid updating networks and zones", func() {
				newShoot := prepareShootForUpdate(shoot)
				cidr := garden.CIDR("255.255.255.255/32")
				newShoot.Spec.Cloud.GCP.Networks.Pods = &cidr
				newShoot.Spec.Cloud.GCP.Zones = []string{"another-zone"}

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.zones", fldPath)),
				}))
			})

			It("should forbid removing the GCP section", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Cloud.GCP = nil

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.cloud.aws/azure/gcp/openstack/local"),
				}))
			})
		})

		Context("OpenStack specific validation", func() {
			var (
				fldPath        = "openstack"
				openStackCloud *garden.OpenStackCloud
			)

			BeforeEach(func() {
				openStackCloud = &garden.OpenStackCloud{
					FloatingPoolName:     "my-pool",
					LoadBalancerProvider: "haproxy",
					Networks: garden.OpenStackNetworks{
						K8SNetworks: k8sNetworks,
						Workers:     []garden.CIDR{"10.250.0.0/16"},
						Router: &garden.OpenStackRouter{
							ID: "router1234",
						},
					},
					Workers: []garden.OpenStackWorker{
						{
							Worker: worker,
						},
					},
					Zones: []string{"europe-1a"},
				}
				shoot.Spec.Cloud.AWS = nil
				shoot.Spec.Cloud.OpenStack = openStackCloud
				shoot.Spec.Backup = nil
			})

			It("should not return any errors", func() {
				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(0))
			})

			It("should forbid invalid floating pool name configuration", func() {
				shoot.Spec.Cloud.OpenStack.FloatingPoolName = ""

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.floatingPoolName", fldPath)),
				}))
			})

			It("should forbid invalid load balancer provider configuration", func() {
				shoot.Spec.Cloud.OpenStack.LoadBalancerProvider = ""

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.loadBalancerProvider", fldPath)),
				}))
			})

			It("should forbid invalid network configuration", func() {
				shoot.Spec.Cloud.OpenStack.Networks.Workers = []garden.CIDR{"invalid-cidr", "another cidr"}
				shoot.Spec.Cloud.OpenStack.Networks.K8SNetworks = invalidK8sNetworks
				shoot.Spec.Cloud.OpenStack.Networks.Router = &garden.OpenStackRouter{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(7))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.nodes", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.pods", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.services", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers[0]", fldPath)),
				}))
				Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.workers[1]", fldPath)),
				}))
				Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks.router.id", fldPath)),
				}))
			})

			It("should forbid an empty worker list", func() {
				shoot.Spec.Cloud.OpenStack.Workers = []garden.OpenStackWorker{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers", fldPath)),
				}))
			})

			It("should enforce unique worker names", func() {
				shoot.Spec.Cloud.OpenStack.Workers = []garden.OpenStackWorker{
					{
						Worker: worker,
					},
					{
						Worker: worker,
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[1]", fldPath)),
				}))
			})

			It("should forbid invalid worker configuration", func() {
				shoot.Spec.Cloud.OpenStack.Workers = []garden.OpenStackWorker{
					{
						Worker: invalidWorker,
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(6))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].machineType", fldPath)),
				}))
				Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMin", fldPath)),
				}))
				Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
				Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].autoScalerMax", fldPath)),
				}))
			})

			It("should forbid too long worker names", func() {
				shoot.Spec.Cloud.OpenStack.Workers = []garden.OpenStackWorker{
					{
						Worker: invalidWorkerTooLongName,
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeTooLong),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid worker pools with names that are not DNS-1123 subdomain compliant", func() {
				shoot.Spec.Cloud.OpenStack.Workers = []garden.OpenStackWorker{
					{
						Worker: invalidWorkerName,
					},
				}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.workers[0].name", fldPath)),
				}))
			})

			It("should forbid an empty zones list", func() {
				shoot.Spec.Cloud.OpenStack.Zones = []string{}

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.zones", fldPath)),
				}))
			})

			It("should forbid updating networks and zones", func() {
				newShoot := prepareShootForUpdate(shoot)
				cidr := garden.CIDR("255.255.255.255/32")
				newShoot.Spec.Cloud.OpenStack.Networks.Pods = &cidr
				newShoot.Spec.Cloud.OpenStack.Zones = []string{"another-zone"}

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.networks", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s.zones", fldPath)),
				}))
			})

			It("should forbid removing the OpenStack section", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.Cloud.OpenStack = nil

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal(fmt.Sprintf("spec.cloud.%s", fldPath)),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.cloud.aws/azure/gcp/openstack/local"),
				}))
			})
		})

		Context("dns section", func() {
			It("should forbid unsupported dns providers", func() {
				shoot.Spec.DNS.Provider = garden.DNSProvider("does-not-exist")

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotSupported),
					"Field": Equal("spec.dns.provider"),
				}))
			})

			It("should forbid empty hosted zone ids or domains", func() {
				shoot.Spec.DNS.HostedZoneID = makeStringPointer("")
				shoot.Spec.DNS.Domain = makeStringPointer("")

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.hostedZoneID"),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.domain"),
				}))
			})

			It("should forbid specifying no domain", func() {
				shoot.Spec.DNS.Provider = garden.DNSAWSRoute53
				shoot.Spec.DNS.Domain = nil

				errorList := ValidateShoot(shoot)
				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.dns.domain"),
				}))
			})

			It("should forbid specifying empty domain", func() {
				shoot.Spec.DNS.Provider = garden.DNSAWSRoute53
				shoot.Spec.DNS.Domain = makeStringPointer("")

				errorList := ValidateShoot(shoot)
				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.domain"),
				}))
			})

			It("should forbid specifying invalid domain", func() {
				shoot.Spec.DNS.Provider = garden.DNSAWSRoute53
				shoot.Spec.DNS.Domain = makeStringPointer("foo/bar.baz")

				errorList := ValidateShoot(shoot)
				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.domain"),
				}))
			})

			It("should forbid monocular when provider equals 'unmanaged'", func() {
				shoot.Spec.DNS.Provider = garden.DNSUnmanaged
				shoot.Spec.DNS.HostedZoneID = nil

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.addons.monocular.enabled"),
				}))
			})

			It("should forbid specifying a secret name when provider equals 'unmanaged'", func() {
				shoot.Spec.DNS.Provider = garden.DNSUnmanaged
				shoot.Spec.Addons.Monocular.Enabled = false
				shoot.Spec.DNS.HostedZoneID = nil
				shoot.Spec.DNS.SecretName = makeStringPointer("")

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.secretName"),
				}))
			})

			It("should forbid specifying a hosted zone id when provider equals 'unmanaged'", func() {
				shoot.Spec.DNS.Provider = garden.DNSUnmanaged
				shoot.Spec.Addons.Monocular.Enabled = false

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.hostedZoneID"),
				}))
			})

			It("should forbid updating the dns domain", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.Domain = makeStringPointer("another-domain.com")

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.domain"),
				}))
			})

			It("should forbid updating the dns hosted zone id", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.HostedZoneID = makeStringPointer("another-hosted-zone")

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.hostedZoneID"),
				}))
			})

			It("should forbid updating the dns provider", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.Provider = garden.DNSGoogleCloudDNS

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.dns.provider"),
				}))
			})

			It("should allow updating the dns domain", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Spec.DNS.SecretName = makeStringPointer("my-dns-secret")

				errorList := ValidateShootUpdate(newShoot, shoot)

				Expect(len(errorList)).To(Equal(0))
			})
		})

		It("should forbid unsupported kubernetes configuration", func() {
			shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.CABundle = makeStringPointer("")
			shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = makeStringPointer("")
			shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsClaim = makeStringPointer("")
			shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.GroupsPrefix = makeStringPointer("")
			shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.IssuerURL = makeStringPointer("")
			shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernameClaim = makeStringPointer("")
			shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.UsernamePrefix = makeStringPointer("")

			errorList := ValidateShoot(shoot)

			Expect(len(errorList)).To(Equal(7))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.caBundle"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.clientID"),
			}))
			Expect(*errorList[2]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.groupsClaim"),
			}))
			Expect(*errorList[3]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.groupsPrefix"),
			}))
			Expect(*errorList[4]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.issuerURL"),
			}))
			Expect(*errorList[5]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.usernameClaim"),
			}))
			Expect(*errorList[6]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.kubernetes.kubeAPIServer.oidcConfig.usernamePrefix"),
			}))
		})

		It("should forbid kubernetes version downgrades", func() {
			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.Kubernetes.Version = "1.7.2"

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(len(errorList)).To(Equal(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("spec.kubernetes.version"),
			}))
		})

		It("should forbid kubernetes version upgrades skipping a minor version", func() {
			newShoot := prepareShootForUpdate(shoot)
			newShoot.Spec.Kubernetes.Version = "1.10.1"

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(len(errorList)).To(Equal(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeForbidden),
				"Field": Equal("spec.kubernetes.version"),
			}))
		})

		Context("maintenance section", func() {
			It("should forbid not specifying the maintenance section", func() {
				shoot.Spec.Maintenance = nil

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.maintenance"),
				}))
			})

			It("should forbid not specifying the auto update section", func() {
				shoot.Spec.Maintenance.AutoUpdate = nil

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.maintenance.autoUpdate"),
				}))
			})

			It("should forbid not specifying the time window section", func() {
				shoot.Spec.Maintenance.TimeWindow = nil

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.maintenance.timeWindow"),
				}))
			})

			It("should forbid invalid formats for the time window begin and end values", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "invalidformat"
				shoot.Spec.Maintenance.TimeWindow.End = "invalidformat"

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(2))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.maintenance.timeWindow.begin"),
				}))
				Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.maintenance.timeWindow.end"),
				}))
			})

			It("should forbid time windows greater than 6 hours", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "145000+0100"
				shoot.Spec.Maintenance.TimeWindow.End = "215000+0100"

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.maintenance.timeWindow"),
				}))
			})

			It("should forbid time windows smaller than 30 minutes", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "225000+0100"
				shoot.Spec.Maintenance.TimeWindow.End = "231000+0100"

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.maintenance.timeWindow"),
				}))
			})

			It("should forbid time windows where end is before begin", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "225000+0100"
				shoot.Spec.Maintenance.TimeWindow.End = "224900+0100"

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeForbidden),
					"Field": Equal("spec.maintenance.timeWindow"),
				}))
			})

			It("should allow time windows which overlap over two days", func() {
				shoot.Spec.Maintenance.TimeWindow.Begin = "230000+0100"
				shoot.Spec.Maintenance.TimeWindow.End = "010000+0100"

				errorList := ValidateShoot(shoot)

				Expect(len(errorList)).To(Equal(0))
			})
		})

		It("should forbid updating the spec for shoots with deletion timestamp", func() {
			newShoot := prepareShootForUpdate(shoot)
			deletionTimestamp := metav1.NewTime(time.Now())
			shoot.ObjectMeta.DeletionTimestamp = &deletionTimestamp
			newShoot.ObjectMeta.DeletionTimestamp = &deletionTimestamp
			newShoot.Spec.Maintenance.AutoUpdate.KubernetesVersion = false

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(len(errorList)).To(Equal(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))
		})

		It("should allow updating the metadata for shoots with deletion timestamp", func() {
			newShoot := prepareShootForUpdate(shoot)
			deletionTimestamp := metav1.NewTime(time.Now())
			shoot.ObjectMeta.DeletionTimestamp = &deletionTimestamp
			newShoot.ObjectMeta.DeletionTimestamp = &deletionTimestamp
			newShoot.ObjectMeta.Labels = map[string]string{
				"new-key": "new-value",
			}

			errorList := ValidateShootUpdate(newShoot, shoot)

			Expect(len(errorList)).To(Equal(0))
		})
	})

	Describe("#ValidateShootStatus, #ValidateShootStatusUpdate", func() {
		var shoot *garden.Shoot

		BeforeEach(func() {
			shoot = &garden.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "shoot",
					Namespace: "my-namespace",
				},
				Spec:   garden.ShootSpec{},
				Status: garden.ShootStatus{},
			}
		})

		Context("uid checks", func() {
			It("should allow setting the uid", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Status.UID = types.UID("1234")

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(len(errorList)).To(Equal(0))
			})

			It("should forbid changing the uid", func() {
				newShoot := prepareShootForUpdate(shoot)
				shoot.Status.UID = types.UID("1234")
				newShoot.Status.UID = types.UID("1235")

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("status.uid"),
				}))
			})
		})

		Context("technical id checks", func() {
			It("should allow setting the technical id", func() {
				newShoot := prepareShootForUpdate(shoot)
				newShoot.Status.TechnicalID = "shoot--foo--bar"

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(len(errorList)).To(Equal(0))
			})

			It("should forbid changing the technical id", func() {
				newShoot := prepareShootForUpdate(shoot)
				shoot.Status.TechnicalID = "shoot-foo-bar"
				newShoot.Status.TechnicalID = "shoot--foo--bar"

				errorList := ValidateShootStatusUpdate(newShoot.Status, shoot.Status)

				Expect(len(errorList)).To(Equal(1))
				Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("status.technicalID"),
				}))
			})
		})
	})

	Describe("#ValidateBackupInfrastructure", func() {
		var backupInfrastructure *garden.BackupInfrastructure

		BeforeEach(func() {
			backupInfrastructure = &garden.BackupInfrastructure{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-backupinfrastructure",
					Namespace: "garden",
				},
				Spec: garden.BackupInfrastructureSpec{
					Seed:     "aws",
					ShootUID: types.UID(utils.ComputeSHA1Hex([]byte(fmt.Sprintf(fmt.Sprintf("shoot-%s-%s", "garden", "backup-infrastructure"))))),
				},
			}
		})

		It("should not return any errors", func() {
			errorList := ValidateBackupInfrastructure(backupInfrastructure)

			Expect(len(errorList)).To(Equal(0))
		})

		It("should forbid BackupInfrastructure resources with empty metadata", func() {
			backupInfrastructure.ObjectMeta = metav1.ObjectMeta{}

			errorList := ValidateBackupInfrastructure(backupInfrastructure)

			Expect(len(errorList)).To(Equal(2))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			}))
		})

		It("should forbid BackupInfrastructure specification with empty or invalid keys", func() {
			backupInfrastructure.Spec.Seed = ""
			backupInfrastructure.Spec.ShootUID = ""

			errorList := ValidateBackupInfrastructure(backupInfrastructure)

			Expect(len(errorList)).To(Equal(2))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.seed"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.shootUID"),
			}))
		})

		It("should forbid updating some keys", func() {
			newBackupInfrastructure := prepareBackupInfrastructureForUpdate(backupInfrastructure)
			newBackupInfrastructure.Spec.Seed = "another-seed"
			newBackupInfrastructure.Spec.ShootUID = "another-uid"

			errorList := ValidateBackupInfrastructureUpdate(newBackupInfrastructure, backupInfrastructure)

			Expect(len(errorList)).To(Equal(2))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.seed"),
			}))
			Expect(*errorList[1]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.shootUID"),
			}))
		})
	})
})

// Helper functions

func makeStringPointer(s string) *string {
	ptr := s
	return &ptr
}

func prepareShootForUpdate(shoot *garden.Shoot) *garden.Shoot {
	s := shoot.DeepCopy()
	s.ResourceVersion = "1"
	return s
}

func prepareBackupInfrastructureForUpdate(backupInfrastructure *garden.BackupInfrastructure) *garden.BackupInfrastructure {
	b := backupInfrastructure.DeepCopy()
	b.ResourceVersion = "1"
	return b
}
