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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ls command", func() {

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
			cmd.PrintGardenClusters(configReader, "yaml", ioStreams)
			Expect(out.String()).To(Equal("gardenClusters:\n- name: prod-1\n- name: prod-2\n"))
		})
	})
})
