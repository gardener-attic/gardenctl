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

package shoot

import (
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/storage/names"

	"github.com/gardener/gardener/pkg/api"
	"github.com/gardener/gardener/pkg/apis/garden"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/apis/garden/validation"
	"github.com/gardener/gardener/pkg/operation/common"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
)

type shootStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

// Strategy defines the storage strategy for Shoots.
var Strategy = shootStrategy{api.Scheme, names.SimpleNameGenerator}

func (shootStrategy) NamespaceScoped() bool {
	return true
}

func (shootStrategy) PrepareForCreate(ctx genericapirequest.Context, obj runtime.Object) {
	shoot := obj.(*garden.Shoot)

	shoot.Generation = 1
	shoot.Status = garden.ShootStatus{}

	finalizers := sets.NewString(shoot.Finalizers...)
	if !finalizers.Has(gardenv1beta1.GardenerName) {
		finalizers.Insert(gardenv1beta1.GardenerName)
	}
	shoot.Finalizers = finalizers.UnsortedList()
}

func (shootStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newShoot := obj.(*garden.Shoot)
	oldShoot := old.(*garden.Shoot)
	newShoot.Status = oldShoot.Status

	if mustIncreaseGeneration(oldShoot, newShoot) {
		newShoot.Generation = oldShoot.Generation + 1
	}

	if newShoot.Annotations != nil {
		delete(newShoot.Annotations, common.ShootOperation)
	}
}

func mustIncreaseGeneration(oldShoot, newShoot *garden.Shoot) bool {
	// The Shoot specification changes.
	if !apiequality.Semantic.DeepEqual(oldShoot.Spec, newShoot.Spec) {
		return true
	}

	// The deletion timestamp and the special confirmation annotation was set.
	if !common.CheckConfirmationDeletionTimestampValid(oldShoot.ObjectMeta) && common.CheckConfirmationDeletionTimestampValid(newShoot.ObjectMeta) {
		return true
	}

	// The shoot state was failed but the retry annotation was set.
	lastOperation := newShoot.Status.LastOperation
	if lastOperation != nil && lastOperation.State == garden.ShootLastOperationStateFailed {
		if val, ok := newShoot.Annotations[common.ShootOperation]; ok && val == "retry" {
			return true
		}
	}

	return false
}

func (shootStrategy) Validate(ctx genericapirequest.Context, obj runtime.Object) field.ErrorList {
	shoot := obj.(*garden.Shoot)
	return validation.ValidateShoot(shoot)
}

func (shootStrategy) Canonicalize(obj runtime.Object) {
}

func (shootStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (shootStrategy) ValidateUpdate(ctx genericapirequest.Context, newObj, oldObj runtime.Object) field.ErrorList {
	oldShoot, newShoot := oldObj.(*garden.Shoot), newObj.(*garden.Shoot)
	return validation.ValidateShootUpdate(newShoot, oldShoot)
}

func (shootStrategy) AllowUnconditionalUpdate() bool {
	return false
}

type shootStatusStrategy struct {
	shootStrategy
}

// StatusStrategy defines the storage strategy for the status subresource of Shoots.
var StatusStrategy = shootStatusStrategy{Strategy}

func (shootStatusStrategy) PrepareForUpdate(ctx genericapirequest.Context, obj, old runtime.Object) {
	newShoot := obj.(*garden.Shoot)
	oldShoot := old.(*garden.Shoot)
	newShoot.Spec = oldShoot.Spec
}

func (shootStatusStrategy) ValidateUpdate(ctx genericapirequest.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateShootStatusUpdate(obj.(*garden.Shoot), old.(*garden.Shoot))
}
