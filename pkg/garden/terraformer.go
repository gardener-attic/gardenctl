// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package garden

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gardener/gardenctl/pkg/utils"
	batch_v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var chartPath = filepath.Join("charts", "garden-terraformer", "charts")

// NewTerraformer takes a Garden object <garden> and a string <purpose> which describes for what the
// Terraformer is used, and returns a Terraformer struct with initialized values for the namespace
// and the names which will be used for all the stored resources like ConfigMaps/Secrets.
func NewTerraformer(garden *Garden, purpose string) *Terraformer {
	prefix := fmt.Sprintf("%s.%s", garden.Shoot.ObjectMeta.Name, purpose)
	return &Terraformer{
		Garden:        garden,
		Namespace:     garden.Shoot.ObjectMeta.Namespace,
		Purpose:       purpose,
		ConfigName:    prefix + TerraformerConfigSuffix,
		VariablesName: prefix + TerraformerVariablesSuffix,
		StateName:     prefix + TerraformerStateSuffix,
		PodName:       prefix + TerraformerPodSuffix,
		JobName:       prefix + TerraformerJobSuffix,
	}
}

// SetVariablesEnvironment sets the provided <tfvarsEnvironment> on the Terraformer object.
func (t *Terraformer) SetVariablesEnvironment(tfvarsEnvironment []map[string]interface{}) *Terraformer {
	t.VariablesEnvironment = tfvarsEnvironment
	return t
}

// DefineConfig creates a ConfigMap for the tf state (if it does not exist, otherwise it won't update it),
// as well as a ConfigMap for the tf configuration (if it does not exist, otherwise it will update it).
// The tfvars are stored in a Secret as the contain confidental information like credentials.
func (t *Terraformer) DefineConfig(chartName string, values map[string]interface{}) *Terraformer {
	values["names"] = map[string]interface{}{
		"configuration": t.ConfigName,
		"variables":     t.VariablesName,
		"state":         t.StateName,
	}
	values["initializeEmptyState"] = t.IsStateEmpty()

	err := utils.Retry(t.Logger, 60*time.Second, func() (bool, error) {
		err := t.ApplyChartGarden(
			filepath.Join(chartPath, chartName),
			chartName,
			t.Namespace,
			nil,
			values,
		)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Logger.Errorf("Could not create the Terraform ConfigMaps/Secrets: %s", err.Error())
	} else {
		t.ConfigurationDefined = true
	}

	return t
}

// Apply executes the Terraform Job by running the 'terraform apply' command.
func (t *Terraformer) Apply() error {
	if !t.ConfigurationDefined {
		return errors.New("Terraformer configuration has not been defined, cannot execute the Terraform scripts")
	}
	return t.execute("apply")
}

// Destroy executes the Terraform Job by running the 'terraform destroy' command.
func (t *Terraformer) Destroy() error {
	err := t.execute("destroy")
	if err != nil {
		return err
	}
	return t.cleanupConfiguration()
}

// GetState returns the Terraform state as byte slice.
func (t *Terraformer) GetState() ([]byte, error) {
	configmap, err := t.
		K8sGardenClient.
		GetConfigMap(t.Namespace, t.StateName)
	if err != nil {
		return nil, err
	}
	return []byte(configmap.Data["terraform.tfstate"]), nil
}

// IsStateEmpty returns true if the Terraform state is empty, and false otherwise.
func (t *Terraformer) IsStateEmpty() bool {
	state, err := t.GetState()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true
		}
		return false
	}

	return len(state) == 0
}

// prepare checks whether all required ConfigMaps and Secrets exist. It returns the number of
// existing ConfigMaps/Secrets, or the error in case something unexpected happens.
func (t *Terraformer) prepare() (int, error) {
	// Check whether the required ConfigMaps and the Secret exist
	numberOfExistingResources := 3

	_, err := t.
		K8sGardenClient.
		GetConfigMap(t.Namespace, t.StateName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return -1, err
		}
		numberOfExistingResources--
	}
	_, err = t.
		K8sGardenClient.
		GetSecret(t.Namespace, t.VariablesName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return -1, err
		}
		numberOfExistingResources--
	}
	_, err = t.
		K8sGardenClient.
		GetConfigMap(t.Namespace, t.ConfigName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return -1, err
		}
		numberOfExistingResources--
	}
	if t.VariablesEnvironment == nil {
		return -1, errors.New("no Terraform variable environment provided")
	}

	// Clean up possible existing job/pod artifacts from previous runs
	jobPodList, err := t.listJobPods()
	if err != nil {
		return -1, err
	}
	err = t.cleanupJob(jobPodList)
	if err != nil {
		return -1, err
	}
	err = t.waitForCleanEnvironment()
	if err != nil {
		return -1, err
	}
	return numberOfExistingResources, nil
}

// deployTerraformer renders the Terraformer chart which contains the Job/Pod manifest.
func (t *Terraformer) deployTerraformer(values map[string]interface{}) error {
	return t.ApplyChartGarden(
		filepath.Join(chartPath, "terraformer"),
		"terraformer",
		t.Namespace,
		nil,
		values,
	)
}

// execute creates a Terraform Job which runs the provided scriptName (apply or destroy), waits for the Job to be completed
// (either successful or not), prints its logs, deletes it and returns whether it was successful or not.
func (t *Terraformer) execute(scriptName string) error {
	var (
		exitCode  int32 = 1     // Exit code of the Terraform validation pod
		succeeded       = true  // Success status of the Terraform execution job
		execute         = false // Should we skip the rest of the function depending on whether all ConfigMaps/Secrets exist/do not exist?
		skipPod         = false // Should we skip the execution of the Terraform Pod (validation of the Terraform config)?
		skipJob         = false // Should we skip the execution of the Terraform Job (actual execution of the Terraform config)?
	)

	// We should retry the preparation check in order to allow the kube-apiserver to actually create the ConfigMaps.
	err := utils.Retry(t.Logger, 30*time.Second, func() (bool, error) {
		numberOfExistingResources, err := t.prepare()
		if err != nil {
			return false, err
		}
		if numberOfExistingResources == 0 {
			t.Logger.Debug("All ConfigMaps/Secrets do not exist, can not execute the Terraform Job.")
			return true, nil
		} else if numberOfExistingResources == 3 {
			t.Logger.Debug("All ConfigMaps/Secrets exist, will execute the Terraform Job.")
			execute = true
			return true, nil
		} else {
			t.Logger.Error("Can not execute Terraform Job as ConfigMaps/Secrets are missing!")
			return false, nil
		}
	})
	if err != nil {
		return err
	}
	if !execute {
		return nil
	}

	// In case of scriptName == 'destroy', we need to first check whether the Terraform state contains
	// something at all. If it does not contain anything, then the 'apply' could never be executed, probably
	// because of syntax errors. In this case, we want to skip the Terraform job (as it wouldn't do anything
	// anyway) and just delete the related ConfigMaps/Secrets.
	if scriptName == "destroy" {
		skipPod = true
		skipJob = t.IsStateEmpty()
	}

	values := map[string]interface{}{
		"terraformVariablesEnvironment": t.VariablesEnvironment,
		"names": map[string]interface{}{
			"configuration": t.ConfigName,
			"variables":     t.VariablesName,
			"state":         t.StateName,
			"pod":           t.PodName,
			"job":           t.JobName,
		},
	}

	if !skipPod {
		values["kind"] = "Pod"
		values["script"] = "validate"

		// Create Terraform Pod which validates the Terraform configuration
		err := t.deployTerraformer(values)
		if err != nil {
			return err
		}

		// Wait for the Terraform validation Pod to be completed
		exitCode = t.waitForPod()
		skipJob = exitCode == 0 || exitCode == 1
		if exitCode == 0 {
			t.Logger.Debug("Terraform validation succeeded but there is no difference between state and actual resources.")
		} else if exitCode == 1 {
			t.Logger.Debug("Terraform validation failed, will not start the job.")
			succeeded = false
		} else {
			t.Logger.Debug("Terraform validation has been successful.")
		}
	}

	if !skipJob {
		values["kind"] = "Job"
		values["script"] = scriptName

		// Create Terraform Job which executes the provided scriptName
		err := t.deployTerraformer(values)
		if err != nil {
			return err
		}

		// Wait for the Terraform Job to be completed
		succeeded = t.waitForJob()
	}

	// Retrieve the logs of the Pods belonging to the completed Job
	jobPodList, err := t.listJobPods()
	if err != nil {
		t.Logger.Errorf("Could not retrieve list of pods belonging to Terraform job '%s': %s", t.JobName, err.Error())
		jobPodList = &corev1.PodList{}
	}

	logList, err := t.retrievePodLogs(jobPodList)
	if err != nil {
		t.Logger.Errorf("Could not retrieve the logs of the pods belonging to Terraform job '%s': %s", t.JobName, err.Error())
		logList = map[string]string{}
	}
	for podName, podLogs := range logList {
		t.Logger.Infof("Logs of Pod '%s' belonging to Terraform job '%s':\n%s", podName, t.JobName, podLogs)
	}

	// Delete the Terraform Job and all its belonging Pods
	err = t.cleanupJob(jobPodList)
	if err != nil {
		return err
	}

	// Evaluate whether the execution was successful or not
	if !succeeded {
		errorMessage := "Terraform execution job could not be completed."
		terraformErrors := retrieveTerraformErrors(logList)
		if terraformErrors != nil {
			errorMessage += fmt.Sprintf(" The following issues have been found in the logs:\n\n%s", strings.Join(terraformErrors, "\n\n"))
		}
		return errors.New(errorMessage)
	}
	return nil
}

// listJobPods lists all pods which have a label 'job-name' whose value is equal to the Terraformer job name.
func (t *Terraformer) listJobPods() (*corev1.PodList, error) {
	return t.
		K8sGardenClient.
		ListPods(t.Namespace, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", t.JobName),
		})
}

// waitForCleanEnvironment waits until no Terraform Job and Pod(s) exist for the current instance
// of the Terraformer.
func (t *Terraformer) waitForCleanEnvironment() error {
	return wait.PollImmediate(5*time.Second, 120*time.Second, func() (bool, error) {
		_, err := t.
			K8sGardenClient.
			GetJob(t.Namespace, t.JobName)
		if !apierrors.IsNotFound(err) {
			if err != nil {
				return false, err
			}
			t.Logger.Infof("Waiting until no Terraform Job with name '%s' exist any more...", t.JobName)
			return false, nil
		}

		jobPodList, err := t.listJobPods()
		if err != nil {
			return false, err
		}
		if len(jobPodList.Items) != 0 {
			t.Logger.Infof("Waiting until no Terraform Pods with label 'job-name=%s' exist any more...", t.JobName)
			return false, nil
		}

		return true, nil
	})
}

// waitForPod waits for the Terraform validation Pod to be completed (either successful or failed).
// It checks the Pod status field to identify the state.
func (t *Terraformer) waitForPod() int32 {
	// 'terraform plan' returns exit code 2 if the plan succeeded and there is a diff
	// If we can't read the terminated state of the container we simply force that the Terraform
	// job gets created.
	var exitCode int32 = 2

	wait.PollImmediate(5*time.Second, 120*time.Second, func() (bool, error) {
		t.Logger.Infof("Waiting for Terraform validation Pod '%s' to be completed...", t.PodName)
		pod, err := t.
			K8sGardenClient.
			GetPod(t.Namespace, t.PodName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logger.Warn("Terraform validation Pod disappeared unexpectedly, somebody must have manually deleted it!")
				return true, nil
			}
			exitCode = 1 // 'terraform plan' exit code for "errors"
			return false, err
		}
		// Check whether the Job has been successful (at least one succeeded Pod)
		phase := pod.Status.Phase
		if phase == corev1.PodSucceeded || phase == corev1.PodFailed {
			containerStateTerminated := pod.Status.ContainerStatuses[0].State.Terminated
			if containerStateTerminated != nil {
				exitCode = containerStateTerminated.ExitCode
			}
			return true, nil
		}
		return false, nil
	})
	return exitCode
}

// waitForJob waits for the Terraform Job to be completed (either successful or failed). It checks the
// Job status field to identify the state.
func (t *Terraformer) waitForJob() bool {
	var succeeded = false
	wait.PollImmediate(5*time.Second, 3600*time.Second, func() (bool, error) {
		t.Logger.Infof("Waiting for Terraform Job '%s' to be completed...", t.JobName)
		job, err := t.
			K8sGardenClient.
			GetJob(t.Namespace, t.JobName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logger.Warn("Terraform Job disappeared unexpectedly, somebody must have manually deleted it!")
				return true, nil
			}
			return false, err
		}
		// Check whether the Job has been successful (at least one succeeded Pod)
		if job.Status.Succeeded >= 1 {
			succeeded = true
			return true, nil
		}
		// Check whether the Job is still running at all
		for _, cond := range job.Status.Conditions {
			if cond.Type == batch_v1.JobComplete || cond.Type == batch_v1.JobFailed {
				return true, nil
			}
		}
		return false, nil
	})
	return succeeded
}

// retrievePodLogs fetches the logs of the created Pods by the Terraform Job and returns them as a map whose
// keys are pod names and whose values are the corresponding logs.
func (t *Terraformer) retrievePodLogs(jobPodList *corev1.PodList) (map[string]string, error) {
	var logList = map[string]string{}
	for _, jobPod := range jobPodList.Items {
		name := jobPod.ObjectMeta.Name
		namespace := jobPod.ObjectMeta.Namespace

		logsBuffer, err := t.
			K8sGardenClient.
			GetPodLogs(namespace, name, &corev1.PodLogOptions{})
		if err != nil {
			t.Logger.Warnf("Could not retrieve the logs of Terraform job pod %s: '%v'", name, err)
			continue
		}

		logList[name] = logsBuffer.String()
	}
	return logList, nil
}

// cleanupJob deletes the Terraform Job and all belonging Pods from the Garden cluster.
func (t *Terraformer) cleanupJob(jobPodList *corev1.PodList) error {
	// Delete the Terraform Job
	err := t.
		K8sGardenClient.
		DeleteJob(t.Namespace, t.JobName)
	if err == nil {
		t.Logger.Infof("Deleted Terraform Job '%s'", t.JobName)
	} else {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}

	// Delete the belonging Terraform Pods
	for _, jobPod := range jobPodList.Items {
		err = t.
			K8sGardenClient.
			DeletePod(jobPod.ObjectMeta.Namespace, jobPod.ObjectMeta.Name)
		if err == nil {
			t.Logger.Infof("Deleted Terraform Job Pod '%s'", jobPod.ObjectMeta.Name)
		} else {
			if !apierrors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}

// cleanupConfiguration deletes the two ConfigMaps which store the Terraform configuration and state. It also deletes
// the Secret which stores the Terraform variables.
func (t *Terraformer) cleanupConfiguration() error {
	t.Logger.Infof("Deleting Terraform variables Secret '%s'", t.VariablesName)
	err := t.
		K8sGardenClient.
		DeleteSecret(t.Namespace, t.VariablesName)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	t.Logger.Infof("Deleting Terraform configuration ConfigMap '%s'", t.ConfigName)
	err = t.
		K8sGardenClient.
		DeleteConfigMap(t.Namespace, t.ConfigName)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	t.Logger.Infof("Deleting Terraform state ConfigMap '%s'", t.StateName)
	err = t.
		K8sGardenClient.
		DeleteConfigMap(t.Namespace, t.StateName)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

// retrieveTerraformErrors gets a map <logList> whose keys are pod names and whose values are the corresponding logs,
// and it parses the logs for Terraform errors. If none are found, it will return nil, and otherwhise the list of
// found errors as string slice.
func retrieveTerraformErrors(logList map[string]string) []string {
	var foundErrors = map[string]string{}
	var errorList = []string{}

	for podName, output := range logList {
		errorMessage := findTerraformErrors(output)
		_, ok := foundErrors[errorMessage]

		// Add the errorMessage to the list of found errors (only if it does not already exist).
		if errorMessage != "" && !ok {
			foundErrors[errorMessage] = podName
		}
	}

	for errorMessage, podName := range foundErrors {
		errorList = append(errorList, fmt.Sprintf("-> Pod '%s' reported:\n%s", podName, errorMessage))
	}

	if len(errorList) > 0 {
		return errorList
	}
	return nil
}

// findTerraformErrors gets the <output> of a Terraform run and parses it to find the occurred
// errors (which will be returned). If no errors occurred, an empty string will be returned.
func findTerraformErrors(output string) string {
	var (
		regexTerraformError = regexp.MustCompile(`(?:Error [^:]*|Errors): *([\s\S]*)`)
		regexUUID           = regexp.MustCompile(`(?i)[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}`)
		regexMultiNewline   = regexp.MustCompile(`\n{2,}`)

		errorMessage = output
		valid        = []string{}
	)

	// Strip optional explaination how Terraform behaves in case of errors.
	suffixIndex := strings.Index(errorMessage, "\n\nTerraform does not automatically rollback")
	if suffixIndex != -1 {
		errorMessage = errorMessage[:suffixIndex]
	}

	// Search for errors in Terraform output.
	terraformErrorMatch := regexTerraformError.FindStringSubmatch(errorMessage)
	if len(terraformErrorMatch) > 1 {
		// Remove leading and tailing spaces and newlines.
		errorMessage = strings.TrimSpace(terraformErrorMatch[1])

		// Omit (request) uuid's to allow easy determination of duplicates.
		errorMessage = regexUUID.ReplaceAllString(errorMessage, "<omitted>")

		// Sort the occurred errors alphabetically
		lines := strings.Split(errorMessage, "*")
		sort.Strings(lines)

		// Only keep the lines beginning with ' ' (actual errors)
		for _, line := range lines {
			if strings.HasPrefix(line, " ") {
				valid = append(valid, line)
			}
		}
		errorMessage = "*" + strings.Join(valid, "\n*")

		// Strip multiple newlines to one newline
		errorMessage = regexMultiNewline.ReplaceAllString(errorMessage, "\n")

		// Remove leading and tailing spaces and newlines.
		errorMessage = strings.TrimSpace(errorMessage)

		return errorMessage
	}
	return ""
}
