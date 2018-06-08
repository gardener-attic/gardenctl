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

package backupinfrastructure

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/apis/componentconfig"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	gardeninformers "github.com/gardener/gardener/pkg/client/garden/informers/externalversions/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	controllerutils "github.com/gardener/gardener/pkg/controller/utils"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/operation"
	botanistpkg "github.com/gardener/gardener/pkg/operation/botanist"
	cloudbotanistpkg "github.com/gardener/gardener/pkg/operation/cloudbotanist"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils/flow"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func (c *Controller) backupInfrastructureAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Logger.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.backupInfrastructureQueue.Add(key)
}

func (c *Controller) backupInfrastructureUpdate(oldObj, newObj interface{}) {
	var (
		newBackupInfrastructure    = newObj.(*gardenv1beta1.BackupInfrastructure)
		backupInfrastructureLogger = logger.NewFieldLogger(logger.Logger, "backupinfrastructure", fmt.Sprintf("%s/%s", newBackupInfrastructure.Namespace, newBackupInfrastructure.Name))
	)

	// If the generation did not change for an update event (i.e., no changes to the .spec section have
	// been made), we do not want to add the BackupInfrastructure to the queue. The periodic reconciliation is handled
	// elsewhere by adding the BackupInfrastructure to the queue to dedicated times.
	if newBackupInfrastructure.Generation == newBackupInfrastructure.Status.ObservedGeneration {
		backupInfrastructureLogger.Debug("Do not need to do anything as the Update event occurred due to .status field changes")
		return
	}

	c.backupInfrastructureAdd(newObj)
}

func (c *Controller) backupInfrastructureDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Logger.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.backupInfrastructureQueue.Add(key)
}

func (c *Controller) reconcileBackupInfrastructureKey(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	backupInfrastructure, err := c.backupInfrastructureLister.BackupInfrastructures(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		logger.Logger.Debugf("[BACKUPINFRASTRUCTURE RECONCILE] %s - skipping because BackupInfrastructure has been deleted", key)
		return nil
	}
	if err != nil {
		logger.Logger.Infof("[BACKUPINFRASTRUCTURE RECONCILE] %s - unable to retrieve object from store: %v", key, err)
		return err
	}

	backupInfrastructureLogger := logger.NewFieldLogger(logger.Logger, "backupinfrastructure", fmt.Sprintf("%s/%s", backupInfrastructure.Namespace, backupInfrastructure.Name))

	if backupInfrastructure.DeletionTimestamp != nil && !sets.NewString(backupInfrastructure.Finalizers...).Has(gardenv1beta1.GardenerName) {
		backupInfrastructureLogger.Debug("Do not need to do anything as the BackupInfrastructure does not have my finalizer")
		c.backupInfrastructureQueue.Forget(key)
		return nil
	}

	durationToNextSync := c.config.Controllers.BackupInfrastructure.SyncPeriod.Duration
	if reconcileErr := c.control.ReconcileBackupInfrastructure(backupInfrastructure, key); reconcileErr != nil {
		durationToNextSync = 15 * time.Second
	}
	c.backupInfrastructureQueue.AddAfter(key, durationToNextSync)
	backupInfrastructureLogger.Infof("Scheduled next reconciliation for BackupInfrastructure '%s' in %s", key, durationToNextSync)
	return nil
}

// ControlInterface implements the control logic for updating BackupInfrastructures. It is implemented as an interface to allow
// for extensions that provide different semantics. Currently, there is only one implementation.
type ControlInterface interface {
	// ReconcileBackupInfrastructure implements the control logic for BackupInfrastructure creation, update, and deletion.
	// If an implementation returns a non-nil error, the invocation will be retried using a rate-limited strategy.
	// Implementors should sink any errors that they do not wish to trigger a retry, and they may feel free to
	// exit exceptionally at any point provided they wish the update to be re-run at a later point in time.
	ReconcileBackupInfrastructure(backupInfrastructure *gardenv1beta1.BackupInfrastructure, key string) error
}

// NewDefaultControl returns a new instance of the default implementation ControlInterface that
// implements the documented semantics for BackupInfrastructures. You should use an instance returned from NewDefaultControl() for any
// scenario other than testing.
func NewDefaultControl(k8sGardenClient kubernetes.Client, k8sGardenInformers gardeninformers.Interface, secrets map[string]*corev1.Secret, imageVector imagevector.ImageVector, identity *gardenv1beta1.Gardener, config *componentconfig.ControllerManagerConfiguration, recorder record.EventRecorder, updater UpdaterInterface) ControlInterface {
	return &defaultControl{k8sGardenClient, k8sGardenInformers, secrets, imageVector, identity, config, recorder, updater}
}

type defaultControl struct {
	k8sGardenClient    kubernetes.Client
	k8sGardenInformers gardeninformers.Interface
	secrets            map[string]*corev1.Secret
	imageVector        imagevector.ImageVector
	identity           *gardenv1beta1.Gardener
	config             *componentconfig.ControllerManagerConfiguration
	recorder           record.EventRecorder
	updater            UpdaterInterface
}

func (c *defaultControl) ReconcileBackupInfrastructure(obj *gardenv1beta1.BackupInfrastructure, key string) error {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return err
	}

	var (
		backupInfrastructure       = obj.DeepCopy()
		backupInfrastructureLogger = logger.NewFieldLogger(logger.Logger, "backupinfrastructure", fmt.Sprintf("%s/%s", backupInfrastructure.Namespace, backupInfrastructure.Name))
		lastOperation              = backupInfrastructure.Status.LastOperation
		operationType              = controllerutils.ComputeOperationType(lastOperation)
	)

	logger.Logger.Infof("[BACKUPINFRASTRUCTURE RECONCILE] %s", key)
	backupInfrastructureJSON, _ := json.Marshal(backupInfrastructure)
	backupInfrastructureLogger.Debugf(string(backupInfrastructureJSON))

	op, err := operation.NewWithBackupInfrastructure(backupInfrastructure, backupInfrastructureLogger, c.k8sGardenClient, c.k8sGardenInformers, c.identity, c.secrets, c.imageVector)
	if err != nil {
		backupInfrastructureLogger.Errorf("Could not initialize a new operation: %s", err.Error())
		return err
	}

	// The deletionTimestamp labels a BackupInfrastructure as intended to get deleted. Before deletion,
	// it has to be ensured that no infrastructure resources are depending on the BackupInfrastructure anymore.
	// When this happens the controller will remove the finalizer from the BackupInfrastructure so that it can be garbage collected.
	if backupInfrastructure.DeletionTimestamp != nil {
		gracePeriod := time.Hour * 24 * time.Duration(*c.config.Controllers.BackupInfrastructure.DeletionGracePeriodDays)
		if time.Now().Sub(backupInfrastructure.DeletionTimestamp.Time) > gracePeriod {
			if updateErr := c.updateBackupInfrastructureStatus(op, gardenv1beta1.ShootLastOperationStateProcessing, gardenv1beta1.ShootLastOperationTypeDelete, "Deletion of Backup Infrastructure in progress.", 1, nil); updateErr != nil {
				backupInfrastructureLogger.Errorf("Could not update the BackupInfrastructure status after deletion start: %+v", updateErr)
				return updateErr
			}

			if deleteErr := c.deleteBackupInfrastructure(op); deleteErr != nil {
				c.recorder.Eventf(backupInfrastructure, corev1.EventTypeWarning, gardenv1beta1.EventDeleteError, "%s", deleteErr.Description)
				if updateErr := c.updateBackupInfrastructureStatus(op, gardenv1beta1.ShootLastOperationStateError, gardenv1beta1.ShootLastOperationTypeDelete, deleteErr.Description+" Operation will be retried.", 1, deleteErr); updateErr != nil {
					backupInfrastructureLogger.Errorf("Could not update the BackupInfrastructure status after deletion error: %+v", updateErr)
					return updateErr
				}
				return errors.New(deleteErr.Description)
			}
			if updateErr := c.updateBackupInfrastructureStatus(op, gardenv1beta1.ShootLastOperationStateSucceeded, gardenv1beta1.ShootLastOperationTypeDelete, "Backup Infrastructure has been successfully deleted.", 100, nil); updateErr != nil {
				backupInfrastructureLogger.Errorf("Could not update the BackupInfrastructure status after deletion successful: %+v", updateErr)
				return updateErr
			}
			return c.removeFinalizer(op)
		}

		if updateErr := c.updateBackupInfrastructureStatus(op, gardenv1beta1.ShootLastOperationStatePending, gardenv1beta1.ShootLastOperationTypeDelete, fmt.Sprintf("Deletion of backup infrastructure is scheduled for %s", backupInfrastructure.DeletionTimestamp.Time.Add(gracePeriod)), 1, nil); updateErr != nil {
			backupInfrastructureLogger.Errorf("Could not update the BackupInfrastructure status after suspending deletion: %+v", updateErr)
			return updateErr
		}
		return nil
	}

	// When a BackupInfrastructure deletion timestamp is not set we need to create/reconcile the backup infrastructure.
	if updateErr := c.updateBackupInfrastructureStatus(op, gardenv1beta1.ShootLastOperationStateProcessing, operationType, "Reconciliation of Backup Infrastructure state in progress.", 1, nil); updateErr != nil {
		backupInfrastructureLogger.Errorf("Could not update the BackupInfrastructure status after reconciliation start: %+v", updateErr)
		return updateErr
	}
	if reconcileErr := c.reconcileBackupInfrastructure(op); reconcileErr != nil {
		c.recorder.Eventf(backupInfrastructure, corev1.EventTypeWarning, gardenv1beta1.EventReconcileError, "%s", reconcileErr.Description)
		if updateErr := c.updateBackupInfrastructureStatus(op, gardenv1beta1.ShootLastOperationStateError, operationType, reconcileErr.Description+" Operation will be retried.", 1, reconcileErr); updateErr != nil {
			backupInfrastructureLogger.Errorf("Could not update the BackupInfrastructure status after reconciliation error: %+v", updateErr)
			return updateErr
		}
		return errors.New(reconcileErr.Description)
	}
	if updateErr := c.updateBackupInfrastructureStatus(op, gardenv1beta1.ShootLastOperationStateSucceeded, operationType, "Backup Infrastructure has been successfully reconciled.", 100, nil); updateErr != nil {
		backupInfrastructureLogger.Errorf("Could not update the Shoot status after reconciliation success: %+v", updateErr)
		return updateErr
	}

	return nil
}

// reconcileBackupInfrastructure reconciles a BackupInfrastructure state.
func (c *defaultControl) reconcileBackupInfrastructure(o *operation.Operation) *gardenv1beta1.LastError {
	// We create botanists (which will do the actual work).
	botanist, err := botanistpkg.New(o)
	if err != nil {
		return formatError("Failed to create a Botanist", err)
	}
	seedCloudBotanist, err := cloudbotanistpkg.New(o, common.CloudPurposeSeed)
	if err != nil {
		return formatError("Failed to create a Seed CloudBotanist", err)
	}

	var (
		defaultRetry = 30 * time.Second

		f                     = flow.New("Backup Infrastructure creation").SetProgressReporter(o.ReportBackupInfrastructureProgress).SetLogger(o.Logger)
		deployBackupNamespace = f.AddTask(botanist.DeployBackupNamespaceFromBackupInfrastructure, defaultRetry)
		_                     = f.AddTask(seedCloudBotanist.DeployBackupInfrastructure, 0, deployBackupNamespace)
	)
	if e := f.Execute(); e != nil {
		e.Description = fmt.Sprintf("Failed to reconcile Backup Infrastructure state: %s", e.Description)
		return e
	}

	o.Logger.Infof("Successfully reconciled Backup Infrastructure state '%s'", o.BackupInfrastructure.Name)
	return nil
}

// deleteBackupInfrastructure deletes a BackupInfrastructure entirely.
func (c *defaultControl) deleteBackupInfrastructure(o *operation.Operation) *gardenv1beta1.LastError {
	// We create botanists (which will do the actual work).
	botanist, err := botanistpkg.New(o)
	if err != nil {
		return formatError("Failed to create a Botanist", err)
	}

	// We first check whether the namespace in the Seed cluster does exist - if it does not, then we assume that
	// all resources have already been deleted. We can delete the BackupInfrastructure resource as a consequence.
	namespace, err := botanist.K8sSeedClient.GetNamespace(common.GenerateBackupNamespaceName(o.BackupInfrastructure.Name))
	if apierrors.IsNotFound(err) {
		o.Logger.Infof("Did not find '%s' namespace in the Seed cluster - nothing to be done", common.GenerateBackupNamespaceName(o.BackupInfrastructure.Name))
		return nil
	}
	if err != nil {
		return formatError("Failed to retrieve the backup namespace in the Seed cluster", err)
	}

	seedCloudBotanist, err := cloudbotanistpkg.New(o, common.CloudPurposeSeed)
	if err != nil {
		return formatError("Failed to create a Seed CloudBotanist", err)
	}

	// We check whether the Backup namespace in the Seed cluster is already in a terminating state, i.e. whether
	// we have tried to delete it in a previous run. In that case, we do not need to cleanup backup infrastructure resource because
	// that would have already been done.
	var (
		cleanupBackupInfrastructureResources = namespace.Status.Phase != corev1.NamespaceTerminating
		defaultRetry                         = 30 * time.Second

		f                           = flow.New("Backup infrastructure deletion").SetProgressReporter(o.ReportBackupInfrastructureProgress).SetLogger(o.Logger)
		destroyBackupInfrastructure = f.AddTaskConditional(seedCloudBotanist.DestroyBackupInfrastructure, 0, cleanupBackupInfrastructureResources)
		deleteBackupNamespace       = f.AddTask(botanist.DeleteBackupNamespace, defaultRetry, destroyBackupInfrastructure)
		_                           = f.AddTask(botanist.WaitUntilBackupNamespaceDeleted, 0, deleteBackupNamespace)
	)
	if e := f.Execute(); e != nil {
		e.Description = fmt.Sprintf("Failed to delete Backup Infrastructure: %s", e.Description)
		return e
	}

	o.Logger.Infof("Successfully deleted Backup Infrastructure '%s'", o.BackupInfrastructure.Name)
	return nil
}

func (c *defaultControl) updateBackupInfrastructureStatus(o *operation.Operation, state gardenv1beta1.ShootLastOperationState, operationType gardenv1beta1.ShootLastOperationType, operationDescription string, progress int, lastError *gardenv1beta1.LastError) error {
	if state == gardenv1beta1.ShootLastOperationStateError && o.BackupInfrastructure.Status.LastOperation != nil {
		progress = o.BackupInfrastructure.Status.LastOperation.Progress
	}
	o.BackupInfrastructure.Status.LastOperation = &gardenv1beta1.LastOperation{
		Type:           operationType,
		State:          state,
		Progress:       progress,
		Description:    operationDescription,
		LastUpdateTime: metav1.Now(),
	}
	o.BackupInfrastructure.Status.LastError = lastError
	o.BackupInfrastructure.Status.ObservedGeneration = o.BackupInfrastructure.Generation

	newBackupInfrastructure, err := c.updater.UpdateBackupInfrastructureStatus(o.BackupInfrastructure)
	if err == nil {
		o.BackupInfrastructure = newBackupInfrastructure
	}
	return err
}

func (c *defaultControl) removeFinalizer(op *operation.Operation) error {
	backupInfrastructureFinalizers := sets.NewString(op.BackupInfrastructure.Finalizers...)
	backupInfrastructureFinalizers.Delete(gardenv1beta1.GardenerName)
	op.BackupInfrastructure.Finalizers = backupInfrastructureFinalizers.UnsortedList()

	newBackupInfrastructure, err := c.k8sGardenClient.GardenClientset().GardenV1beta1().BackupInfrastructures(op.BackupInfrastructure.Namespace).Update(op.BackupInfrastructure)
	if err != nil {
		op.Logger.Errorf("Could not remove finalizer of the BackupInfrastructure: %+v", err.Error())
		return err
	}
	op.BackupInfrastructure = newBackupInfrastructure

	// Wait until the above modifications are reflected in the cache to prevent unwanted reconcile
	// operations (sometimes the cache is not synced fast enough).
	return wait.PollImmediate(time.Second, 30*time.Second, func() (bool, error) {
		backupInfrastructure, err := c.k8sGardenInformers.BackupInfrastructures().Lister().BackupInfrastructures(op.BackupInfrastructure.Namespace).Get(op.BackupInfrastructure.Name)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if !sets.NewString(backupInfrastructure.Finalizers...).Has(gardenv1beta1.GardenerName) {
			return true, nil
		}
		return false, nil
	})
}

func formatError(message string, err error) *gardenv1beta1.LastError {
	return &gardenv1beta1.LastError{
		Description: fmt.Sprintf("%s (%s)", message, err.Error()),
	}
}
