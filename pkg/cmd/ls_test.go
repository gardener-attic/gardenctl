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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ls command", func() {

	Describe("#NewLsCmd", func() {
		var (
			ctrl         *gomock.Controller
			targetReader *mockcmd.MockTargetReader
			configReader *mockcmd.MockConfigReader
			target       *mockcmd.MockTargetInterface
			command      *cobra.Command
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			targetReader = mockcmd.NewMockTargetReader(ctrl)
			configReader = mockcmd.NewMockConfigReader(ctrl)
			target = mockcmd.NewMockTargetInterface(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		Context("with invalid number of args", func() {
			It("should return error", func() {
				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewLsCmd(targetReader, configReader, ioStreams)
				command.SetArgs([]string{})
				err := command.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("command must be in the format: ls [gardens|projects|seeds|shoots|issues|namespaces]"))
			})
		})

		Context("list shoots", func() {
			It("should return error for empty target", func() {
				targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().Stack().Return([]cmd.TargetMeta{})

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewLsCmd(targetReader, configReader, ioStreams)
				command.SetArgs([]string{"shoots"})
				err := command.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("target stack is empty"))
			})
		})
	})

	Describe("#PrintGardenClusters", func() {
		var (
			ctrl         *gomock.Controller
			configReader *mockcmd.MockConfigReader
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			configReader = mockcmd.NewMockConfigReader(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should print Garden clusters", func() {
			gardenConfig := &cmd.GardenConfig{
				GardenClusters: []cmd.GardenClusterMeta{
					{
						Name: "prod-1",
					},
					{
						Name: "prod-2",
					},
				},
			}
			configReader.EXPECT().ReadConfig(gomock.Any()).Return(gardenConfig)

			ioStreams, _, out, _ := cmd.NewTestIOStreams()
			err := cmd.PrintGardenClusters(configReader, ioStreams.Out, "yaml")
			Expect(err).To(BeNil())
			Expect(out.String()).To(Equal("gardenClusters:\n- name: prod-1\n- name: prod-2\n"))
		})
	})
})
