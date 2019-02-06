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

package utils

import (
	"strings"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ComputeOperationType checksthe <lastOperation> and determines whether is it is Create operation or reconcile operation
func ComputeOperationType(meta metav1.ObjectMeta, lastOperation *gardenv1beta1.LastOperation) gardenv1beta1.ShootLastOperationType {
	switch {
	case meta.DeletionTimestamp != nil:
		return gardenv1beta1.ShootLastOperationTypeDelete
	case lastOperation == nil:
		return gardenv1beta1.ShootLastOperationTypeCreate
	case (lastOperation.Type == gardenv1beta1.ShootLastOperationTypeCreate && lastOperation.State != gardenv1beta1.ShootLastOperationStateSucceeded):
		return gardenv1beta1.ShootLastOperationTypeCreate
	}
	return gardenv1beta1.ShootLastOperationTypeReconcile
}

const separator = ","

// AddTasks adds a task to the ShootTasks annotation of the passed map.
func AddTasks(existingAnnotations map[string]string, tasksToAdd ...string) {
	var tasks []string
	if len(existingAnnotations[common.ShootTasks]) > 0 {
		tasks = strings.Split(existingAnnotations[common.ShootTasks], separator)
	}
	for _, taskToAdd := range tasksToAdd {
		if utils.ValueExists(taskToAdd, tasks) {
			continue
		}
		tasks = append(tasks, taskToAdd)
	}
	existingAnnotations[common.ShootTasks] = strings.Join(tasks, separator)
}

// HasTask checks if the passed task is part of the ShootTasks annotation.
func HasTask(existingAnnotations map[string]string, taskToCheck string) bool {
	existingTasks, ok := existingAnnotations[common.ShootTasks]
	if !ok {
		return false
	}
	tasks := strings.Split(existingTasks, separator)
	return utils.ValueExists(taskToCheck, tasks)
}

// RemoveAllTasks removes the ShootTasks annotation from the passed map.
func RemoveAllTasks(existingAnnotations map[string]string) {
	delete(existingAnnotations, common.ShootTasks)
}
