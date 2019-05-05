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
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Target command", func() {

	var (
		ctrl         *gomock.Controller
		configReader *mockcmd.MockConfigReader
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
		configReader = mockcmd.NewMockConfigReader(ctrl)
		targetReader = mockcmd.NewMockTargetReader(ctrl)
		target = mockcmd.NewMockTargetInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("with empty target", func() {
		It("targeting project", func() {
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			target.EXPECT().Stack().Return([]cmd.TargetMeta{})

			command = cmd.NewTargetCmd(targetReader, configReader)
			err := execute(command, []string{"project", "bar"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no garden cluster targeted"))
		})
	})

	Context("with target kind garden", func() {
		It("with wrong name", func() {
			gardenConfig := &cmd.GardenConfig{
				GardenClusters: []cmd.GardenClusterMeta{
					cmd.GardenClusterMeta{
						Name: "bar",
					},
				},
			}
			configReader.EXPECT().ReadConfig(gomock.Any()).Return(gardenConfig)

			command = cmd.NewTargetCmd(targetReader, configReader)
			err := execute(command, []string{"garden", "foo"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no match for \"foo\""))
		})
	})

	type targetCase struct {
		args        []string
		flags       []string
		expectedErr string
	}

	DescribeTable("validation",
		func(c targetCase) {
			command := cmd.NewTargetCmd(targetReader, configReader)

			err := execute(command, c.args)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(c.expectedErr))
		},
		Entry("with missing target kind", targetCase{
			args:        []string{},
			expectedErr: "command must be in the format: target <project|garden|seed|shoot> NAME",
		}),
		Entry("with missing garden cluster name", targetCase{
			args:        []string{"garden"},
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
