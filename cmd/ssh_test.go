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
	. "github.com/onsi/gomega"
)

var _ = Describe("SSH command", func() {

	var (
		ctrl    *gomock.Controller
		reader  *mockcmd.MockTargetReader
		target  *mockcmd.MockTargetInterface
		command *cobra.Command

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
		Context("when target is not shoot", func() {
			It("should return error", func() {
				reader.EXPECT().ReadTarget(gomock.Any()).Return(target)
				target.EXPECT().Stack().Return([]cmd.TargetMeta{})

				ioStreams, _, _, _ := cmd.NewTestIOStreams()
				command = cmd.NewSSHCmd(reader, ioStreams)
				err := execute(command, []string{})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("no shoot targeted"))
			})
		})
	})
})
