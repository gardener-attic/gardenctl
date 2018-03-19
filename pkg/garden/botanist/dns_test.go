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

package botanist_test

import (
	"fmt"
	"strings"

	"github.com/gardener/gardenctl/pkg/garden"
	. "github.com/gardener/gardenctl/pkg/garden/botanist"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	gardenv1 "github.com/gardener/gardenctl/pkg/apis/garden/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("garden", func() {
	Describe("botanist", func() {
		Describe("dns", func() {
			Describe("#GetSeedIngressFQDN", func() {
				var (
					seedFQDN       = "seed.example.com"
					shootName      = "shoot"
					shootNamespace = "example"
					botanist       = &Botanist{
						SeedFQDN: seedFQDN,
						Garden: &garden.Garden{
							Shoot: &gardenv1.Shoot{
								ObjectMeta: metav1.ObjectMeta{
									Name:      shootName,
									Namespace: shootNamespace,
								},
							},
							ProjectName: shootNamespace,
						},
					}
				)

				It("should return an error", func() {
					fqdn, err := botanist.GetSeedIngressFQDN(strings.Repeat("0", 30))

					Expect(err).To(HaveOccurred())
					Expect(fqdn).To(BeZero())
				})

				It("should return a valid FQDN (without `garden` prefix)", func() {
					subDomain := "accesspoint"

					fqdn, err := botanist.GetSeedIngressFQDN(subDomain)

					Expect(err).NotTo(HaveOccurred())
					Expect(fqdn).To(Equal(fmt.Sprintf("%s.%s.%s.ingress.%s", subDomain, shootName, shootNamespace, seedFQDN)))
				})

				It("should return a valid FQDN (with `garden` prefix)", func() {
					subDomain := "accesspoint"
					oldNamespace := botanist.Shoot.ObjectMeta.Namespace
					botanist.Shoot.ObjectMeta.Namespace = "garden-" + oldNamespace

					fqdn, err := botanist.GetSeedIngressFQDN(subDomain)

					Expect(err).NotTo(HaveOccurred())
					Expect(fqdn).To(Equal(fmt.Sprintf("%s.%s.%s.ingress.%s", subDomain, shootName, shootNamespace, seedFQDN)))
					botanist.Shoot.ObjectMeta.Namespace = oldNamespace
				})
			})

			Describe("#GetShootIngressFQDN", func() {
				var (
					shootDomain = "shoot.example.com"
					botanist    = &Botanist{
						Garden: &garden.Garden{
							Shoot: &gardenv1.Shoot{
								Spec: gardenv1.ShootSpec{
									DNS: &gardenv1.DNS{
										Domain: shootDomain,
									},
								},
							},
						},
					}
				)

				It("should return an error", func() {
					fqdn, err := botanist.GetShootIngressFQDN(strings.Repeat("0", 40))

					Expect(err).To(HaveOccurred())
					Expect(fqdn).To(BeZero())
				})

				It("should return a valid FQDN", func() {
					subDomain := "accesspoint"

					fqdn, err := botanist.GetShootIngressFQDN(subDomain)

					Expect(err).NotTo(HaveOccurred())
					Expect(fqdn).To(Equal(fmt.Sprintf("%s.ingress.%s", subDomain, shootDomain)))
				})
			})

			Describe("#getSeedFQDN", func() {
				var (
					seedSecret                corev1.Secret
					secretDNSDomainAnnotation = ""
				)

				JustBeforeEach(func() {
					annotations := make(map[string]string)
					annotations[garden.DNSDomain] = secretDNSDomainAnnotation

					seedSecret = corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "default-domain",
							Annotations: annotations,
						},
					}
				})

				AfterEach(func() {
					seedSecret = corev1.Secret{}
					secretDNSDomainAnnotation = ""
				})

				Context("no DNS domain annotation given in Seed secret", func() {
					BeforeEach(func() {
						secretDNSDomainAnnotation = ""
					})

					It("should return an error", func() {
						fqdn, err := ExportGetSeedFQDN(&seedSecret)

						Expect(err).To(HaveOccurred())
						Expect(fqdn).To(BeZero())
					})
				})

				Context("too long DNS domain annotation in Seed secret", func() {
					BeforeEach(func() {
						secretDNSDomainAnnotation = strings.Repeat("0", 33)
					})

					It("should return an error", func() {
						fqdn, err := ExportGetSeedFQDN(&seedSecret)

						Expect(err).To(HaveOccurred())
						Expect(fqdn).To(BeZero())
					})
				})

				Context("correct DNS domain annotation in Seed secret", func() {
					BeforeEach(func() {
						secretDNSDomainAnnotation = "seed.example.com"
					})

					It("should return a valid FQDN", func() {
						fqdn, err := ExportGetSeedFQDN(&seedSecret)

						Expect(err).NotTo(HaveOccurred())
						Expect(fqdn).To(Equal(secretDNSDomainAnnotation))
					})
				})
			})
		})
	})
})
