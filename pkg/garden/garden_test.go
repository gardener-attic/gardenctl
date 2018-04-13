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

package garden_test

import (
	"fmt"

	gardenv1 "github.com/gardener/gardenctl/pkg/apis/garden/v1"
	. "github.com/gardener/gardenctl/pkg/garden"
	"github.com/gardener/gardenctl/pkg/test/mocks"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("garden", func() {
	var (
		garden    *Garden
		logger    = mocks.NewMockLogger()
		k8sClient = mocks.NewMockKubernetesClient(false, false)
		secrets   = map[string]*corev1.Secret{
			"secret-1":               &corev1.Secret{},
			"secret-2":               &corev1.Secret{},
			GardenRoleInternalDomain: &corev1.Secret{},
		}
		operator = &gardenv1.GardenOperator{
			Name: "test-operator",
			ID:   "1234",
		}
		shootObj = mocks.NewMockShoot()
		shoot    = shootObj
		oldShoot = shoot
	)
	shootObj.Spec.SeedName = ""

	Describe("garden", func() {
		Describe("#New", func() {
			AfterEach(func() {
				shoot = shootObj
				oldShoot = shoot
			})

			BeforeEach(func() {
				shoot.ObjectMeta.Name = "shoot01"
			})

			It("should return a Garden object (without prefix in Shoot namespace)", func() {
				shoot.ObjectMeta.Namespace = "test-namespace"

				gardenObj := New(logger.Logger, nil, k8sClient, secrets, operator, &shoot, &oldShoot)

				Expect(gardenObj.Logger).To(Equal(logger.Logger))
				Expect(gardenObj.Operator.Name).To(Equal(operator.Name))
				Expect(gardenObj.Operator.ID).To(Equal(operator.ID))
				Expect(gardenObj.Shoot.Name).To(Equal(shoot.Name))
				Expect(gardenObj.OldShoot.Name).To(Equal(oldShoot.Name))
				Expect(gardenObj.ShootNamespace).To(Equal(fmt.Sprintf("shoot-%s-%s", shoot.ObjectMeta.Namespace, shoot.ObjectMeta.Name)))
				for name, secret := range gardenObj.Secrets {
					Expect(gardenObj.Secrets).To(HaveKeyWithValue(name, secret))
				}
			})
		})

		Describe("<Garden>", func() {
			BeforeEach(func() {
				garden = New(logger.Logger, nil, k8sClient, secrets, operator, &shoot, &oldShoot)
			})

			AfterEach(func() {
				garden = nil
			})

			Describe("#DetermineSeedCluster", func() {
				var seedName = "seed01"

				Context("seed cluster name not given in the Shoot manifest", func() {
					It("should return the name of a found seed cluster", func() {
						garden.Secrets[fmt.Sprintf("seed-%s", seedName)] = &corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name: seedName,
								Labels: map[string]string{
									InfrastructureKind:   string(garden.Shoot.Spec.Infrastructure.Kind),
									InfrastructureRegion: garden.Shoot.Spec.Infrastructure.Region,
								},
							},
						}

						name, err := garden.DetermineSeedCluster()

						Expect(err).NotTo(HaveOccurred())
						Expect(name).To(Equal(seedName))
					})

					It("should return an error (no adequate seed cluster found)", func() {
						delete(garden.Secrets, fmt.Sprintf("seed-%s", seedName))

						name, err := garden.DetermineSeedCluster()

						Expect(err).To(HaveOccurred())
						Expect(name).To(BeEmpty())
					})
				})

				Context("seed cluster name given in the Shoot manifest", func() {
					It("should return the name of the seed cluster", func() {
						garden.Shoot.Spec.SeedName = seedName

						name, err := garden.DetermineSeedCluster()

						Expect(err).NotTo(HaveOccurred())
						Expect(name).To(Equal(seedName))
					})
				})
			})

			Describe("#GetSeedSecret", func() {
				var seedName = "seed01"

				It("should return a secret (from the 'secrets' map)", func() {
					garden.Secrets[fmt.Sprintf("seed-%s", seedName)] = &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: seedName,
						},
					}

					secret, err := garden.GetSeedSecret(seedName)

					Expect(err).NotTo(HaveOccurred())
					Expect(secret.ObjectMeta.Name).To(Equal(seedName))
				})

				It("should return an error", func() {
					secret, err := garden.GetSeedSecret(seedName)

					Expect(err).To(HaveOccurred())
					Expect(secret).To(BeNil())
				})
			})

			Describe("#GetInfrastructureSecret", func() {
				var infrastructureSecretName = "infra"

				It("should return a secret (from the 'secrets' map)", func() {
					garden.Secrets["infrastructure"] = &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name: infrastructureSecretName,
						},
					}

					secret, err := garden.GetInfrastructureSecret()

					Expect(err).NotTo(HaveOccurred())
					Expect(secret.ObjectMeta.Name).To(Equal(infrastructureSecretName))
				})

				It("should return a secret (from the Garden cluster)", func() {
					secret, err := garden.GetInfrastructureSecret()

					Expect(err).NotTo(HaveOccurred())
					Expect(secret.ObjectMeta.Name).To(Equal(infrastructureSecretName))
					Expect(secret.ObjectMeta.Namespace).To(Equal("core"))
					Expect(garden.Secrets).To(HaveKeyWithValue("infrastructure", secret))
				})

				It("should return an error", func() {
					garden.K8sGardenClient = mocks.NewMockKubernetesClient(true, false)

					secret, err := garden.GetInfrastructureSecret()

					Expect(err).To(HaveOccurred())
					Expect(secret).To(BeNil())
				})
			})

			Describe("#GetKubernetesVersion", func() {
				It("should return the major and minor part of the Kubernetes version", func() {
					version := garden.GetKubernetesVersion()

					Expect(version).To(Equal("1.7"))
				})
			})
		})
	})
})
