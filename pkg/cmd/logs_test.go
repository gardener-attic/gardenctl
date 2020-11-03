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
	"github.com/spf13/cobra"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logs command", func() {

	var (
		command *cobra.Command

		execute = func(command *cobra.Command, args []string) error {
			command.SetArgs(args)
			return command.Execute()
		}
	)

	Context("with < 1 args", func() {
		It("should return error", func() {
			command = cmd.NewLogsCmd()
			err := execute(command, []string{})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Command must be in the format: logs (gardener-apiserver|gardener-controller-manager|gardener-dashboard|api|scheduler|controller-manager|etcd-operator|etcd-main[etcd backup-restore]|etcd-events[etcd backup-restore]|addon-manager|vpn-seed|vpn-shoot|machine-controller-manager|kubernetes-dashboard|prometheus|grafana|gardenlet|tf (infra|dns|ingress)|cluster-autoscaler flags(--loki|--tail|--since|--since-time|--timestamps)"))
		})
	})
})

func Test_buildLokiCommand(t *testing.T) {
	cmd.KUBECONFIG = "/path/to/configfile"
	command := cmd.BuildLokiCommand("myns", "nginx-pod", "mycontainer")

	expected := `kubectl --kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget 'http://localhost:3100/loki/api/v1/query_range' -O- --post-data='query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=1603184413805314000&&end=1604394013805314000'`
	expectedNorm := `kubectl --kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget 'http://localhost:3100/loki/api/v1/query_range' -O- --post-data='query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=101010&&end=202020'`

	norm := normalizeTimestamp(expected)
	if expectedNorm != norm {
		t.Error("wrong timestamps normalizations")
	}

	normCommand := normalizeTimestamp(command)
	if expectedNorm != normCommand {
		t.Error("wrong timestamps normalizations for generated command")
	}
}

func normalizeTimestamp(command string) string {
	var re = regexp.MustCompile(`(?m).*start=(\d+)&&end=(\d+)`)

	timestamps := re.FindAllStringSubmatch(command, -1)[0]
	command = strings.Replace(command, timestamps[1], "101010", 1)
	command = strings.Replace(command, timestamps[2], "202020", 1)

	return command
}
