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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Info command", func() {

	var (
		ctrl         *gomock.Controller
		targetReader *mockcmd.MockTargetReader
		target       *mockcmd.MockTargetInterface
		command      *cobra.Command

		execute = func(command *cobra.Command, args []string) error {
			command.SetArgs(args)
			return command.Execute()
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		targetReader = mockcmd.NewMockTargetReader(ctrl)
		target = mockcmd.NewMockTargetInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("with empty target stack", func() {
		It("should return err", func() {
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(&cmd.Target{})

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewInfoCmd(targetReader, ioStreams)
			err := execute(command, []string{})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no garden cluster targeted"))
		})
	})

	Context("with garden target", func() {
		It("should write appropriate info", func() {
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			target.EXPECT().Stack().Return([]cmd.TargetMeta{
				{
					Kind: cmd.TargetKindGarden,
					Name: "prod",
				},
			})
			seedAws := "aws"
			seedGcp := "gcp"
			clientSet := gardenfake.NewSimpleClientset(
				&v1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{Name: "unscheduled"},
				},
				&v1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{Name: "aws"},
					Spec: v1beta1.ShootSpec{
						Cloud: v1beta1.Cloud{
							Seed: &seedAws,
						},
					},
				},
				&v1beta1.Shoot{
					ObjectMeta: metav1.ObjectMeta{Name: "gcp"},
					Spec: v1beta1.ShootSpec{
						Cloud: v1beta1.Cloud{
							Seed: &seedGcp,
						},
					},
				},
			)
			target.EXPECT().GardenerClient().Return(clientSet, nil)

			ioStreams, _, out, _ := cmd.NewTestIOStreams()
			command = cmd.NewInfoCmd(targetReader, ioStreams)
			err := execute(command, []string{})
			Expect(err).NotTo(HaveOccurred())

			actual := out.String()
			Expect(actual).To(ContainSubstring("Garden: prod"))
			Expect(actual).To(ContainSubstring("total: 3"))
			Expect(actual).To(ContainSubstring("unscheduled: 1"))
			Expect(actual).To(ContainSubstring("aws: 1"))
			Expect(actual).To(ContainSubstring("gcp: 1"))
		})
	})
})
