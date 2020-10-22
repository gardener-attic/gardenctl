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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Show command", func() {
	var (
		ctrl         *gomock.Controller
		targetReader *mockcmd.MockTargetReader
		target       *mockcmd.MockTargetInterface
		command      *cobra.Command
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		targetReader = mockcmd.NewMockTargetReader(ctrl)
		target = mockcmd.NewMockTargetInterface(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("with invalid number of args", func() {
		It("should return error", func() {
			targetReader.EXPECT().ReadTarget(gomock.Any()).Return(target).AnyTimes()
			target.EXPECT().Stack().Return([]cmd.TargetMeta{}).AnyTimes()
			command = cmd.NewShowCmd(targetReader)
			command.SetArgs([]string{})
			err := command.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Command must be in the format: show (operator|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main|etcd-events|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|tf (infra|dns|ingress)|cluster-autoscaler)"))
		})
	})
})
