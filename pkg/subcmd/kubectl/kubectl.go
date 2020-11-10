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

//BuildKubectlCommand this function will be removed after review
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

//BuildLogCommandArgs build kubectl command to get logs
func BuildLogCommandArgs(kubeconfig string, namespace, podName, container string, tail int64, sinceSeconds time.Duration) []string {
	args := []string{
		"logs",
		"--kubeconfig=" + kubeconfig,
		podName,
	}

	if container != emptyString {
		args = append(args, []string{"-c", container}...)
	}

	args = append(args, []string{"-n", namespace}...)

	if tail != -1 {
		args = append(args, fmt.Sprintf("--tail=%d", tail))
	}
	if sinceSeconds != 0 {
		args = append(args, fmt.Sprintf("--since=%vs", sinceSeconds.Seconds()))
	}

	return args
}

//BuildLokiCommand will be removed after review
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

//BuildLokiCommandArgs build kubect command to get logs from loki
func BuildLokiCommandArgs(kubeconfig string, namespace, podName, container string, tail int64, sinceSeconds time.Duration) []string {
	args := []string{
		"--kubeconfig=" + kubeconfig,
		"exec",
		"loki-0",
		"-n",
		namespace,
		"--",
		"wget",
		"'http://localhost:3100/loki/api/v1/query_range'",
		"-O-",
	}

	lokiQuery := fmt.Sprintf("{pod_name=~\"%s.*\"}", podName)
	command := fmt.Sprintf("--post-data='query=%s", lokiQuery)

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

	args = append(args, command)
	return args
}
