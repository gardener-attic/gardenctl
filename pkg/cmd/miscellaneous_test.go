// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package cmd_test

import (
	"os"

	"github.com/gardener/gardenctl/pkg/cmd"
	. "github.com/gardener/gardenctl/pkg/cmd"
	mockcmd "github.com/gardener/gardenctl/pkg/mock/cmd"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorefake "github.com/gardener/gardener/pkg/client/core/clientset/versioned/fake"
	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Miscellaneous", func() {
	var gardenConf GardenConfig
	var target Target
	dumpPath := "/tmp"
	pathTarget := dumpPath + "/target2"
	pathGardenConfig := dumpPath + "/gardenconfig"
	file, err := os.OpenFile(pathGardenConfig, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	gConfig := `
githubURL: https://github.croft.gardener.corp
gardenClusters:
- name: dev
  kubeConfig: /tmp/kubeconfig.yaml
- name: prod
  kubeConfig: /tmp/kubeconfig.yaml
`
	content := []byte(gConfig)
	_, err = file.Write(content)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
	file, err = os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
	content = []byte("")
	if err != nil {
		panic(err)
	}
	_, err = file.Write(content)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
	Context("After calling GetGardenClusterKubeConfigFromConfig", func() {
		It("First Garden Cluster should be set as default target Name if no garden cluster is specified", func() {
			GetGardenClusterKubeConfigFromConfig(pathGardenConfig, pathTarget)
			ReadTarget(pathTarget, &target)
			Expect(target.Target[0].Name).To(Equal("dev"))
		})
	})
	Context("After calling GetGardenClusters", func() {
		It("GardenCluster Name should be dev ", func() {
			GetGardenConfig(pathGardenConfig, &gardenConf)
			Expect(gardenConf.GardenClusters[0].Name).To(Equal("dev"))
			Expect(gardenConf.GardenClusters[1].Name).To(Equal("prod"))
		})
	})

	var _ = AfterSuite(func() {
		os.Remove(pathTarget)
		os.Remove(pathGardenConfig)
	})

	Describe("Miscellaneous", func() {
		var (
			ctrl         *gomock.Controller
			targetReader *mockcmd.MockTargetReader
			target       = &cmd.Target{
				Target: []cmd.TargetMeta{
					{
						Kind: cmd.TargetKindGarden,
						Name: "test-garden",
					},
				},
			}
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			targetReader = mockcmd.NewMockTargetReader(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		Context("IsTargeted Testing", func() {
			It("target seed should return False", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				Expect(IsTargeted(targetReader, "seed")).To(BeFalse())
			})

			It("target garden should return true", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				Expect(IsTargeted(targetReader, "garden")).To(BeTrue())
			})

			It("target empty should return true", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				Expect(IsTargeted(targetReader)).To(BeTrue())
			})
		})

		Context("GetTargetName garden", func() {
			It("should return err==nil", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				_, err := GetTargetName(targetReader, "garden")
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("Miscellaneous", func() {
		var (
			ctrl         *gomock.Controller
			targetReader *mockcmd.MockTargetReader
			target       *mockcmd.MockTargetInterface
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			targetReader = mockcmd.NewMockTargetReader(ctrl)
			target = mockcmd.NewMockTargetInterface(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		Context("Get Ojbect Testing", func() {
			seedName := "test-seed"
			nameSpace := "test-namespace"
			clientSet := gardencorefake.NewSimpleClientset(
				&gardencorev1beta1.Seed{
					ObjectMeta: metav1.ObjectMeta{
						Name: seedName,
					},
					Spec: gardencorev1beta1.SeedSpec{
						SecretRef: &corev1.SecretReference{
							Name:      "test-secret-name",
							Namespace: "test-namespace",
						},
					},
				},
				&gardencorev1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-shoot",
						Namespace: "test-namespace",
					},
					Spec: gardencorev1beta1.ShootSpec{
						SeedName: &seedName,
					},
				},
				&gardencorev1beta1.ShootList{
					Items: []gardencorev1beta1.Shoot{},
				},
				&gardencorev1beta1.Project{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-project",
					},
					Spec: gardencorev1beta1.ProjectSpec{
						Namespace: &nameSpace,
					},
				},
				&gardencorev1beta1.ProjectList{
					Items: []gardencorev1beta1.Project{},
				},
			)
			targetMeta := []cmd.TargetMeta{
				{
					Kind: cmd.TargetKindGarden,
					Name: "test-garden",
				},
				{
					Kind: cmd.TargetKindSeed,
					Name: "test-seed",
				},
				{
					Kind: cmd.TargetKindShoot,
					Name: "test-shoot",
				},
			}

			It("should pass on get Project Object", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target).AnyTimes()
				target.EXPECT().GardenerClient().Return(clientSet, nil).AnyTimes()
				_, err := GetProjectObject(targetReader, "test-project1")
				Expect(err.Error()).To(Equal("projects.core.gardener.cloud \"test-project1\" not found"))
				_, err = GetProjectObject(targetReader, "test-project")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass on get Seed Object", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target).AnyTimes()
				target.EXPECT().GardenerClient().Return(clientSet, nil).AnyTimes()
				_, err := GetSeedObject(targetReader, "test-seed1")
				Expect(err.Error()).To(Equal("seeds.core.gardener.cloud \"test-seed1\" not found"))
				_, err = GetSeedObject(targetReader, "test-seed")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass on get Shoot Object", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target).AnyTimes()
				target.EXPECT().Stack().Return(targetMeta).AnyTimes()
				target.EXPECT().GardenerClient().Return(clientSet, nil).AnyTimes()
				_, err := GetShootObject(targetReader, "test-shoot1")
				Expect(err.Error()).To(Equal("shoots.core.gardener.cloud \"test-shoot1\" not found"))
				_, err = GetShootObject(targetReader, "test-shoot")
				Expect(err).NotTo(HaveOccurred())
			})
		})

	})
})
