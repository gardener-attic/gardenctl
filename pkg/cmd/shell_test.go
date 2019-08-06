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
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Shell command", func() {

	var (
		ctrl      *gomock.Controller
		reader    *mockcmd.MockTargetReader
		target    *mockcmd.MockTargetInterface
		clientSet *fake.Clientset
		command   *cobra.Command

		execute = func(command *cobra.Command, args []string) error {
			command.SetArgs(args)
			return command.Execute()
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		reader = mockcmd.NewMockTargetReader(ctrl)
		target = mockcmd.NewMockTargetInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("without args", func() {
		It("should list the node names", func() {
			clientSet = fake.NewSimpleClientset(&corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "minikube",
				},
			})
			reader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			target.EXPECT().Kind().Return(cmd.TargetKindShoot, nil)
			target.EXPECT().K8SClient().Return(clientSet, nil)

			ioStreams, _, out, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(reader, ioStreams)
			err := execute(command, []string{})

			Expect(err).NotTo(HaveOccurred())
			Expect(out.String()).To(Equal("minikube\n"))
		})

		Context("when project is targeted", func() {
			It("should return error", func() {
				reader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().Kind().Return(cmd.TargetKindProject, nil)

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewShellCmd(reader, ioStreams)
				err := execute(command, []string{})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("project targeted"))
			})
		})
	})

	Context("with non-existing node name", func() {
		It("should return error", func() {
			clientSet = fake.NewSimpleClientset()
			reader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			target.EXPECT().Kind().Return(cmd.TargetKindShoot, nil)
			target.EXPECT().K8SClient().Return(clientSet, nil)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(reader, ioStreams)
			err := execute(command, []string{"minikube"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("nodes \"minikube\" not found"))
		})
	})

	Context("when project is targeted", func() {
		It("should return error", func() {
			reader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			target.EXPECT().Kind().Return(cmd.TargetKindProject, nil)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(reader, ioStreams)
			err := execute(command, []string{"minikube"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("project targeted"))
		})
	})

	Context("with >= 2 args", func() {
		It("should return error", func() {
			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(reader, ioStreams)
			err := execute(command, []string{"minikube", "docker-for-mac"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("command must be in the format: gardenctl shell (node|pod)"))
		})
	})
})
