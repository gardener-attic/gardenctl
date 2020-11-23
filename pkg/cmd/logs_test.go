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

	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logs and kubecmd command", func() {

	var (
		ctrl         *gomock.Controller
		targetReader *mockcmd.MockTargetReader
		command      *cobra.Command
		execute      = func(command *cobra.Command, args []string) error {
			command.SetArgs(args)
			return command.Execute()
		}
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		targetReader = mockcmd.NewMockTargetReader(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	var ()

	Context("with < 1 args", func() {
		It("should return error", func() {
			command = cmd.NewLogsCmd(targetReader)
			err := execute(command, []string{})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|gardenlet|tf (infra|dns|ingress)|cluster-autoscaler flags(--loki|--tail|--since|--since-time|--timestamps)"))
		})
	})

	Context("kubectl commands", func() {

		It("should build kubectl command", func() {

			expected := "logs --kubeconfig=/path/to/configfile myPod -c myContainer -n myns --tail=200 --since=3e-07s"
			command := cmd.BuildLogCommandArgs("/path/to/configfile", "myns", "myPod", "myContainer", 200, 300)
			join := strings.Join(command, " ")
			Expect(expected).To(Equal(join))
		})

		It("should build kubectl command", func() {

			//test `normalizeTimestamp` first - replace timestamp with some predefined values
			expected := `--kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget --header X-Scope-OrgID: operator http://localhost:3100/loki/api/v1/query_range -O- --post-data query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=1603184413805314000&&end=1604394013805314000`
			expectedNorm := `--kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget --header X-Scope-OrgID: operator http://localhost:3100/loki/api/v1/query_range -O- --post-data query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=101010&&end=202020`
			norm := normalizeTimestamp(expected)
			Expect(expectedNorm).To(Equal(norm))

			//test real command builder
			args := cmd.BuildLokiCommandArgs("/path/to/configfile", "myns", "nginx-pod", "mycontainer", 200, 0)
			command := strings.Join(args, " ")
			normCommand := normalizeTimestamp(command)
			Expect(expectedNorm).To(Equal(normCommand))
			Expect(len(args)).To(Equal(13))
		})
	})

	Context("versions comparison", func() {
		It("should be greater than Loki version release", func() {
			Expect(cmd.VersionGreaterThanLokiRelease("1.13.0-dev-38d42e28ec51d5b8728fcade4ae5b50f3d3eaca1")).To(BeTrue())
			Expect(cmd.VersionGreaterThanLokiRelease("1.13.0")).To(BeTrue())
		})

		It("should be earlier than Loki version release", func() {
			Expect(cmd.VersionGreaterThanLokiRelease("1.8.0-dev-38d42e28ec51d5b8728fcade4ae5b50f3d3eaca1")).To(BeFalse())
			Expect(cmd.VersionGreaterThanLokiRelease("1.7.0")).To(BeFalse())
		})
	})
})

func normalizeTimestamp(command string) string {
	var re = regexp.MustCompile(`(?m).*start=(\d+)&&end=(\d+)`)

	timestamps := re.FindAllStringSubmatch(command, -1)[0]
	command = strings.Replace(command, timestamps[1], "101010", 1)
	command = strings.Replace(command, timestamps[2], "202020", 1)

	return command
}
