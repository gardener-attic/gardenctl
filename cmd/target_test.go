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
	mockcmd "github.com/gardener/gardenctl/pkg/mock/cmd"
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	gardenfake "github.com/gardener/gardener/pkg/client/garden/clientset/versioned/fake"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Target command", func() {

	var (
		ctrl         *gomock.Controller
		configReader *mockcmd.MockConfigReader
		targetReader *mockcmd.MockTargetReader
		targetWriter *mockcmd.MockTargetWriter
		target       *mockcmd.MockTargetInterface
		command      *cobra.Command

		execute = func(command *cobra.Command, args []string) error {
			command.SetArgs(args)
			return command.Execute()
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		configReader = mockcmd.NewMockConfigReader(ctrl)
		targetReader = mockcmd.NewMockTargetReader(ctrl)
		targetWriter = mockcmd.NewMockTargetWriter(ctrl)
		target = mockcmd.NewMockTargetInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("with empty target", func() {
		It("targeting project", func() {
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			target.EXPECT().Stack().Return([]cmd.TargetMeta{})

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewTargetCmd(targetReader, targetWriter, configReader, ioStreams)
			err := execute(command, []string{"project", "bar"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no garden cluster targeted"))
		})

		It("targeting garden with wrong name", func() {
			gardenConfig := &cmd.GardenConfig{
				GardenClusters: []cmd.GardenClusterMeta{
					{
						Name: "prod",
					},
				},
			}
			configReader.EXPECT().ReadConfig(gomock.Any()).Return(gardenConfig)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewTargetCmd(targetReader, targetWriter, configReader, ioStreams)
			err := execute(command, []string{"garden", "foo"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no match for \"foo\""))
		})
	})

	Context("with garden target", func() {
		It("targeting project with wrong name", func() {
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			target.EXPECT().Stack().Return([]cmd.TargetMeta{
				{
					Kind: cmd.TargetKindGarden,
					Name: "prod",
				},
			})

			clientSet := gardenfake.NewSimpleClientset()
			target.EXPECT().GardenerClient().Return(clientSet, nil)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewTargetCmd(targetReader, targetWriter, configReader, ioStreams)
			err := execute(command, []string{"project", "foo"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no match for \"foo\""))
		})

		It("targeting project with correct name", func() {
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target).Times(2)
			target.EXPECT().Stack().Return([]cmd.TargetMeta{
				{
					Kind: cmd.TargetKindGarden,
					Name: "prod",
				},
			}).Times(2)

			clientSet := gardenfake.NewSimpleClientset(&v1beta1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myproject",
				},
			})
			target.EXPECT().GardenerClient().Return(clientSet, nil)

			target.EXPECT().SetStack([]cmd.TargetMeta{
				{
					Kind: cmd.TargetKindGarden,
					Name: "prod",
				},
				{
					Kind: cmd.TargetKindProject,
					Name: "myproject",
				},
			})
			targetWriter.EXPECT().WriteTarget(gomock.Any(), target)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewTargetCmd(targetReader, targetWriter, configReader, ioStreams)
			err := execute(command, []string{"project", "myproject"})

			Expect(err).NotTo(HaveOccurred())
		})
	})

	type targetCase struct {
		args        []string
		flags       []string
		expectedErr string
	}

	DescribeTable("validation",
		func(c targetCase) {
			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command := cmd.NewTargetCmd(targetReader, targetWriter, configReader, ioStreams)

			err := execute(command, c.args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(c.expectedErr))
		},
		Entry("with missing target kind", targetCase{
			args:        []string{},
			expectedErr: "command must be in the format: target <project|garden|seed|shoot> NAME",
		}),
		Entry("with 2 garden cluster names", targetCase{
			args:        []string{"garden", "prod-1", "prod-2"},
			expectedErr: "command must be in the format: target garden NAME",
		}),
		Entry("with missing project name", targetCase{
			args:        []string{"project"},
			expectedErr: "command must be in the format: target project NAME",
		}),
		Entry("with missing seed name", targetCase{
			args:        []string{"seed"},
			expectedErr: "command must be in the format: target seed NAME",
		}),
		Entry("with missing seed name", targetCase{
			args:        []string{"shoot"},
			expectedErr: "command must be in the format: target shoot NAME",
		}),
	)
})
