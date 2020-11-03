package kubectl

import (
	"fmt"
	"strings"
	"time"
)

const (
	fourteenDaysInSeconds = 60 * 60 * 24 * 14
	emptyString           = ""
)

func getNamespaces() string {
	return "get ns"
}

func BuildKubectlCommand(kubeconfig string, namespace, podName, container string, tail int64, sinceSeconds time.Duration) string {
	var command strings.Builder
	command.WriteString(fmt.Sprintf("kubectl logs --kubeconfig=%s %s%s -n %s ", kubeconfig, podName, container, namespace))
	if tail != -1 {
		command.WriteString(fmt.Sprintf("--tail=%d ", tail))
	}
	if sinceSeconds != 0 {
		command.WriteString(fmt.Sprintf("--since=%vs ", sinceSeconds.Seconds()))
	}

	return command.String()
}

func BuildLokiCommand(kubeconfig string, namespace, podName, container string, tail int64, sinceSeconds time.Duration) string {
	lokiQuery := fmt.Sprintf("{pod_name=~\"%s.*\"}", podName)

	command := fmt.Sprintf("wget 'http://localhost:3100/loki/api/v1/query_range' -O- --post-data='query=%s", lokiQuery)

	if container != emptyString {
		command += fmt.Sprintf("&&query={container_name=~\"%s.*\"", container)
	}
	if tail != 0 {
		command += fmt.Sprintf("&&limit=%d", tail)
	}
	if sinceSeconds == 0 {
		sinceSeconds = fourteenDaysInSeconds * time.Second
	}
	sinceNanoSec := sinceSeconds.Nanoseconds()
	now := time.Now().UnixNano()

	command += fmt.Sprintf("&&start=%d&&end=%d", now-sinceNanoSec, now)
	command += "'"

	endCommand := fmt.Sprintf("kubectl --kubeconfig=%s exec loki-0 -n %s -- %s", kubeconfig, namespace, command)
	return endCommand
}
