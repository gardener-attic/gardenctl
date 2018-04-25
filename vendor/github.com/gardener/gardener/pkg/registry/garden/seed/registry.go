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

package seed

import (
	"github.com/gardener/gardener/pkg/apis/garden"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

// Registry is an interface for things that know how to store Seeds.
type Registry interface {
	ListSeeds(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (*garden.SeedList, error)
	WatchSeeds(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (watch.Interface, error)
	GetSeed(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (*garden.Seed, error)
	CreateSeed(ctx genericapirequest.Context, seed *garden.Seed, createValidation rest.ValidateObjectFunc) (*garden.Seed, error)
	UpdateSeed(ctx genericapirequest.Context, seed *garden.Seed, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (*garden.Seed, error)
	DeleteSeed(ctx genericapirequest.Context, name string) error
}

// storage puts strong typing around storage calls
type storage struct {
	rest.StandardStorage
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched
// types will panic.
func NewRegistry(s rest.StandardStorage) Registry {
	return &storage{s}
}

func (s *storage) ListSeeds(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (*garden.SeedList, error) {
	obj, err := s.List(ctx, options)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.SeedList), err
}

func (s *storage) WatchSeeds(ctx genericapirequest.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return s.Watch(ctx, options)
}

func (s *storage) GetSeed(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (*garden.Seed, error) {
	obj, err := s.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.Seed), nil
}

func (s *storage) CreateSeed(ctx genericapirequest.Context, seed *garden.Seed, createValidation rest.ValidateObjectFunc) (*garden.Seed, error) {
	obj, err := s.Create(ctx, seed, createValidation, false)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.Seed), nil
}

func (s *storage) UpdateSeed(ctx genericapirequest.Context, seed *garden.Seed, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (*garden.Seed, error) {
	obj, _, err := s.Update(ctx, seed.Name, rest.DefaultUpdatedObjectInfo(seed), createValidation, updateValidation)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.Seed), nil
}

func (s *storage) DeleteSeed(ctx genericapirequest.Context, name string) error {
	_, _, err := s.Delete(ctx, name, nil)
	return err
}
