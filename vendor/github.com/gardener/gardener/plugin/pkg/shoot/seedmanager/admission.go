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

package seedmanager

import (
	"errors"
	"io"

	"github.com/gardener/gardener/pkg/apis/garden"
	"github.com/gardener/gardener/pkg/apis/garden/helper"
	admissioninitializer "github.com/gardener/gardener/pkg/apiserver/admission/initializer"
	gardeninformers "github.com/gardener/gardener/pkg/client/garden/informers/internalversion"
	gardenlisters "github.com/gardener/gardener/pkg/client/garden/listers/garden/internalversion"
	"github.com/gardener/gardener/pkg/operation/common"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apiserver/pkg/admission"
)

const (
	// PluginName is the name of this admission plugin.
	PluginName = "ShootSeedManager"
)

// Register registers a plugin.
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return New()
	})
}

// SeedManager contains listers and and admission handler.
type SeedManager struct {
	*admission.Handler
	seedLister gardenlisters.SeedLister
}

var _ = admissioninitializer.WantsInternalGardenInformerFactory(&SeedManager{})

// New creates a new SeedManager admission plugin.
func New() (*SeedManager, error) {
	return &SeedManager{
		Handler: admission.NewHandler(admission.Create, admission.Update),
	}, nil
}

// SetInternalGardenInformerFactory gets Lister from SharedInformerFactory.
func (h *SeedManager) SetInternalGardenInformerFactory(f gardeninformers.SharedInformerFactory) {
	h.seedLister = f.Garden().InternalVersion().Seeds().Lister()
}

// ValidateInitialization checks whether the plugin was correctly initialized.
func (h *SeedManager) ValidateInitialization() error {
	if h.seedLister == nil {
		return errors.New("missing seed lister")
	}
	return nil
}

// Admit tries to find an adequate Seed cluster for the given cloud provider profile and region,
// and writes the name into the Shoot specification. It also ensures that protected Seeds are
// only usable by Shoots in the garden namespace.
func (h *SeedManager) Admit(a admission.Attributes) error {
	// Wait until the caches have been synced
	if !h.WaitForReady() {
		return admission.NewForbidden(a, errors.New("not yet ready to handle request"))
	}

	// Ignore all kinds other than Shoot
	if a.GetKind().GroupKind() != garden.Kind("Shoot") {
		return nil
	}
	shoot, ok := a.GetObject().(*garden.Shoot)
	if !ok {
		return apierrors.NewBadRequest("could not convert resource into Shoot object")
	}

	// If the Shoot manifest already specifies a desired Seed cluster, then we check whether it is protected or not.
	// In case it is protected then we only allow Shoot resources to reference it which are part of the Garden namespace.
	// Also, we don't allow shoot to be created on the seed which is already marked to be deleted.
	if shoot.Spec.Cloud.Seed != nil {
		seed, err := h.seedLister.Get(*shoot.Spec.Cloud.Seed)
		if err != nil {
			return admission.NewForbidden(a, err)
		}

		if shoot.Namespace != common.GardenNamespace && seed.Spec.Protected != nil && *seed.Spec.Protected {
			return admission.NewForbidden(a, errors.New("forbidden to use a protected seed"))
		}

		if a.GetOperation() == admission.Create && seed.DeletionTimestamp != nil {
			return admission.NewForbidden(a, errors.New("forbidden to use a seed marked to be deleted"))
		}

		return nil
	}

	// If no Seed is referenced, we try to determine an adequate one.
	seed, err := determineSeed(shoot, h.seedLister)
	if err != nil {
		return admission.NewForbidden(a, err)
	}

	shoot.Spec.Cloud.Seed = &seed.Name
	return nil
}

// determineSeed returns an appropriate Seed cluster (or nil).
func determineSeed(shoot *garden.Shoot, lister gardenlisters.SeedLister) (*garden.Seed, error) {
	list, err := lister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, seed := range list {
		// We return the first matching seed cluster.
		if seed.DeletionTimestamp == nil && seed.Spec.Cloud.Profile == shoot.Spec.Cloud.Profile && seed.Spec.Cloud.Region == shoot.Spec.Cloud.Region && seed.Spec.Visible != nil && *seed.Spec.Visible && verifySeedAvailability(seed) {
			return seed, nil
		}
	}

	return nil, errors.New("failed to determine an adequate Seed cluster for this cloud profile and region")
}

func verifySeedAvailability(seed *garden.Seed) bool {
	if cond := helper.GetCondition(seed.Status.Conditions, garden.SeedAvailable); cond != nil {
		return cond.Status == corev1.ConditionTrue
	}
	return false
}
