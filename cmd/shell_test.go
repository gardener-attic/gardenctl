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
	"github.com/gardener/gardenctl/cmd"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Shell command", func() {

	var (
		ctrl      *gomock.Controller
		tp        *cmd.MockTargetProviderAPI
		clientSet *fake.Clientset
		command   *cobra.Command
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		tp = cmd.NewMockTargetProviderAPI(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	execute := func(command *cobra.Command, args []string) error {
		command.SetArgs(args)
		return command.Execute()
	}

	Context("without args", func() {
		It("should list the node names", func() {
			clientSet = fake.NewSimpleClientset(&v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "minikube",
				},
			})
			tp.EXPECT().FetchTargetKind().Return("shoot", nil)
			tp.EXPECT().ClientToTarget(gomock.Eq("shoot")).Return(clientSet, nil)

			ioStreams, _, out, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(tp, ioStreams)
			err := execute(command, []string{})

			Expect(err).NotTo(HaveOccurred())
			Expect(out.String()).To(Equal("minikube\n"))
		})

		Context("when project is targeted", func() {
			It("should return error", func() {
				tp.EXPECT().FetchTargetKind().Return("project", nil)

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewShellCmd(tp, ioStreams)
				err := execute(command, []string{})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("project targeted"))
			})
		})
	})

	Context("with non-existing node name", func() {
		It("should return error", func() {
			clientSet = fake.NewSimpleClientset()
			tp.EXPECT().FetchTargetKind().Return("shoot", nil)
			tp.EXPECT().ClientToTarget(gomock.Eq("shoot")).Return(clientSet, nil)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(tp, ioStreams)
			err := execute(command, []string{"minikube"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("node \"minikube\" not found"))
		})
	})

	Context("when project is targeted", func() {
		It("should return error", func() {
			tp.EXPECT().FetchTargetKind().Return("project", nil)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(tp, ioStreams)
			err := execute(command, []string{"minikube"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("project targeted"))
		})
	})

	Context("with >= 2 args", func() {
		It("should return error", func() {
			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewShellCmd(tp, ioStreams)
			err := execute(command, []string{"minikube", "docker-for-mac"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("command must be in the format: gardenctl shell (node|pod)"))
		})
	})
})
