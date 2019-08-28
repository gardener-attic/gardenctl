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
	"github.com/gardener/gardenctl/pkg/cmd"
	mockcmd "github.com/gardener/gardenctl/pkg/mock/cmd"
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	gardenfake "github.com/gardener/gardener/pkg/client/garden/clientset/versioned/fake"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Get command", func() {

	Describe("NewGetCmd", func() {
		var (
			ctrl             *gomock.Controller
			targetReader     *mockcmd.MockTargetReader
			configReader     *mockcmd.MockConfigReader
			kubeconfigReader *mockcmd.MockKubeconfigReader
			target           *mockcmd.MockTargetInterface
			command          *cobra.Command
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			targetReader = mockcmd.NewMockTargetReader(ctrl)
			configReader = mockcmd.NewMockConfigReader(ctrl)
			kubeconfigReader = mockcmd.NewMockKubeconfigReader(ctrl)
			target = mockcmd.NewMockTargetInterface(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		Context("with invalid number of args", func() {
			It("should return error", func() {
				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{})
				err := command.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("command must be in the format: get [(garden|project|seed|shoot|target) <name>]"))
			})
		})

		Context("missing target", func() {
			It("shout return error for missing shoot in the target", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().Stack().Return([]cmd.TargetMeta{})

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{"shoot"})
				err := command.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no shoot targeted"))
			})

			It("shout return error for missing target object", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().Stack().Return([]cmd.TargetMeta{})

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{"target"})
				err := command.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("target stack is empty"))
			})
		})

		Context("target shoot with valid target object", func() {
			seedName := "test-seed"

			k8sClientToGarden := kubernetesfake.NewSimpleClientset(
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
						Labels: map[string]string{
							cmd.ProjectName: "prod",
						},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret-name",
						Namespace: "test-namespace",
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubecfg",
						Namespace: "test-namespace",
					},
				})
			clientSet := gardenfake.NewSimpleClientset(
				&v1beta1.Seed{
					ObjectMeta: metav1.ObjectMeta{
						Name: seedName,
					},
					Spec: v1beta1.SeedSpec{
						SecretRef: corev1.SecretReference{
							Name:      "test-secret-name",
							Namespace: "test-namespace",
						},
					},
				},
				&v1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-shoot",
					},
					Spec: v1beta1.ShootSpec{

						Cloud: v1beta1.Cloud{
							Seed: &seedName,
						},
					},
				})

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
					Kind: cmd.TargetKindSeed,
					Name: "test-shoot",
				},
			}

			gardenConfig := &cmd.GardenConfig{
				GardenClusters: []cmd.GardenClusterMeta{
					{
						Name: "test-garden",
					},
				},
			}

			kubeconfig := []byte("test-kubeconfig")

			It("should pass on get garden", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				configReader.EXPECT().ReadConfig(gomock.Any()).Return(gardenConfig)
				kubeconfigReader.EXPECT().ReadKubeconfig(gomock.Any()).Return(kubeconfig, nil)
				target.EXPECT().Stack().Return(targetMeta).AnyTimes()

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{"garden"})
				err := command.Execute()

				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass on get seed", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().K8SClientToKind(cmd.TargetKindGarden).Return(k8sClientToGarden, nil)
				target.EXPECT().Stack().Return(targetMeta).AnyTimes()
				target.EXPECT().GardenerClient().Return(clientSet, nil)

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{"seed"})
				err := command.Execute()

				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass on get shoot", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().K8SClientToKind(cmd.TargetKindGarden).Return(k8sClientToGarden, nil)
				target.EXPECT().K8SClientToKind(cmd.TargetKindSeed).Return(k8sClientToGarden, nil)
				target.EXPECT().Stack().Return(targetMeta).AnyTimes()
				target.EXPECT().GardenerClient().Return(clientSet, nil)

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{"shoot"})
				err := command.Execute()

				Expect(err).NotTo(HaveOccurred())
			})

			It("should pass on get target", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().Stack().Return(targetMeta).AnyTimes()
				target.EXPECT().Stack().Return(targetMeta).AnyTimes()

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{"target"})
				err := command.Execute()

				Expect(err).NotTo(HaveOccurred())
			})

			It("should fail on get project", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().Stack().Return(targetMeta).AnyTimes()

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewGetCmd(targetReader, configReader, kubeconfigReader, ioStreams)
				command.SetArgs([]string{"project"})
				err := command.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("seed targeted, project expected"))
			})
		})
	})
})
