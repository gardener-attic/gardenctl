package kubectl

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"regexp"
	"strings"
	"testing"
)

var _ = Describe("Build kubectl commands", func() {

	Context("kubectl commands", func() {

		It("should build kubectl command", func() {

			expected := "logs --kubeconfig=/path/to/configfile myPodmyContainer -n myns --tail=200 --since=3e-07s"
			command := BuildKubectlCommandArgs("/path/to/configfile", "myns", "myPod", "myContainer", 200, 300)
			join := strings.Join(command, " ")
			Expect(expected).To(Equal(join))
		})

		It("should build kubectl command", func() {

			//test `normalizeTimestamp` first - replace timestamp with some predefined values
			expected := `--kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget 'http://localhost:3100/loki/api/v1/query_range' -O- --post-data='query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=1603184413805314000&&end=1604394013805314000'`
			expectedNorm := `--kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget 'http://localhost:3100/loki/api/v1/query_range' -O- --post-data='query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=101010&&end=202020'`
			norm := normalizeTimestamp(expected)
			Expect(expectedNorm).To(Equal(norm))

			//test real command builder
			args := BuildLokiCommandArgs("/path/to/configfile", "myns", "nginx-pod", "mycontainer", 200, 0)
			command := strings.Join(args, " ")
			normCommand := normalizeTimestamp(command)
			Expect(expectedNorm).To(Equal(normCommand))
		})
	})
})

//all these tests will be deleted after the review except `normalizeTimestamp`
func Test_buildKubectlCommand(t *testing.T) {
	KUBECONFIG := "/path/to/configfile"
	command := BuildKubectlCommand(KUBECONFIG, "myns", "myPod", "myContainer", 200, 0)

	expected := "kubectl logs --kubeconfig=/path/to/configfile myPodmyContainer -n myns --tail=200 "

	if command != expected {
		t.Error("failed to build command")
	}
}

func Test_buildKubectlCommand2(t *testing.T) {
	KUBECONFIG := "/path/to/configfile"
	command := BuildKubectlCommand(KUBECONFIG, "myns", "myPod", "myContainer", 200, 300)

	expected := "kubectl logs --kubeconfig=/path/to/configfile myPodmyContainer -n myns --tail=200 --since=3e-07s "

	if command != expected {
		t.Error("failed to build command")
	}
}

func Test_buildKubectlCommand2Args(t *testing.T) {
	KUBECONFIG := "/path/to/configfile"
	command := BuildKubectlCommandArgs(KUBECONFIG, "myns", "myPod", "myContainer", 200, 300)

	expected := "logs --kubeconfig=/path/to/configfile myPodmyContainer -n myns --tail=200 --since=3e-07s"
	join := strings.Join(command, " ")

	if join != expected {
		t.Error("failed to build command")
	}
}

func Test_buildLokiCommand(t *testing.T) {
	KUBECONFIG := "/path/to/configfile"
	command := BuildLokiCommand(KUBECONFIG, "myns", "nginx-pod", "mycontainer", 200, 0)

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

func Test_buildLokiCommandArgs(t *testing.T) {
	KUBECONFIG := "/path/to/configfile"
	args := BuildLokiCommandArgs(KUBECONFIG, "myns", "nginx-pod", "mycontainer", 200, 0)

	command := strings.Join(args, " ")

	expected := `--kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget 'http://localhost:3100/loki/api/v1/query_range' -O- --post-data='query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=1603184413805314000&&end=1604394013805314000'`
	expectedNorm := `--kubeconfig=/path/to/configfile exec loki-0 -n myns -- wget 'http://localhost:3100/loki/api/v1/query_range' -O- --post-data='query={pod_name=~"nginx-pod.*"}&&query={container_name=~"mycontainer.*"&&limit=200&&start=101010&&end=202020'`

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
