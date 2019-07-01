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

var _ = Describe("Drop command", func() {

	var (
		ctrl         *gomock.Controller
		targetReader *mockcmd.MockTargetReader
		targetWriter *mockcmd.MockTargetWriter
		command      *cobra.Command

		execute = func(command *cobra.Command, args []string) error {
			command.SetArgs(args)
			return command.Execute()
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		targetReader = mockcmd.NewMockTargetReader(ctrl)
		targetWriter = mockcmd.NewMockTargetWriter(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("without args", func() {
		It("should return err when target stack is empty", func() {
			target := &cmd.Target{}
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)

			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewDropCmd(targetReader, targetWriter, ioStreams)
			err := execute(command, []string{})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("target stack is empty"))
		})

		It("should drop the last target", func() {
			var (
				target = &cmd.Target{
					Target: []cmd.TargetMeta{
						{
							Kind: cmd.TargetKindGarden,
							Name: "test-garden",
						},
						{
							Kind: cmd.TargetKindSeed,
							Name: "test-seed",
						},
					},
				}
				expectedTarget = &cmd.Target{
					Target: []cmd.TargetMeta{
						{
							Kind: cmd.TargetKindGarden,
							Name: "test-garden",
						},
					},
				}
			)

			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target)
			targetWriter.EXPECT().WriteTarget(gomock.Any(), expectedTarget)

			ioStreams, _, out, _ := cmd.NewTestIOStreams()
			command = cmd.NewDropCmd(targetReader, targetWriter, ioStreams)
			err := execute(command, []string{})

			Expect(err).NotTo(HaveOccurred())
			Expect(out.String()).To(Equal("Dropped seed test-seed\n"))
		})
	})

	Context("with >= 2 args", func() {
		It("should return error", func() {
			ioStreams, _, _, _ := cmd.NewTestIOStreams()
			command = cmd.NewDropCmd(targetReader, targetWriter, ioStreams)
			err := execute(command, []string{"project", "seed"})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("command must be in the format: gardenctl drop [(project|seed)]"))
		})
	})
})
