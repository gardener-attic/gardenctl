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

package cloudprofile

import (
	"context"

	"github.com/gardener/gardener/pkg/apis/garden"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
)

// Registry is an interface for things that know how to store CloudProfiles.
type Registry interface {
	ListCloudProfiles(ctx context.Context, options *metainternalversion.ListOptions) (*garden.CloudProfileList, error)
	WatchCloudProfiles(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error)
	GetCloudProfile(ctx context.Context, name string, options *metav1.GetOptions) (*garden.CloudProfile, error)
	CreateCloudProfile(ctx context.Context, cloudProfile *garden.CloudProfile, createValidation rest.ValidateObjectFunc) (*garden.CloudProfile, error)
	UpdateCloudProfile(ctx context.Context, cloudProfile *garden.CloudProfile, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (*garden.CloudProfile, error)
	DeleteCloudProfile(ctx context.Context, name string) error
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

func (s *storage) ListCloudProfiles(ctx context.Context, options *metainternalversion.ListOptions) (*garden.CloudProfileList, error) {
	obj, err := s.List(ctx, options)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.CloudProfileList), err
}

func (s *storage) WatchCloudProfiles(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return s.Watch(ctx, options)
}

func (s *storage) GetCloudProfile(ctx context.Context, name string, options *metav1.GetOptions) (*garden.CloudProfile, error) {
	obj, err := s.Get(ctx, name, options)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.CloudProfile), nil
}

func (s *storage) CreateCloudProfile(ctx context.Context, cloudProfile *garden.CloudProfile, createValidation rest.ValidateObjectFunc) (*garden.CloudProfile, error) {
	obj, err := s.Create(ctx, cloudProfile, createValidation, false)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.CloudProfile), nil
}

func (s *storage) UpdateCloudProfile(ctx context.Context, cloudProfile *garden.CloudProfile, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc) (*garden.CloudProfile, error) {
	obj, _, err := s.Update(ctx, cloudProfile.Name, rest.DefaultUpdatedObjectInfo(cloudProfile), createValidation, updateValidation)
	if err != nil {
		return nil, err
	}

	return obj.(*garden.CloudProfile), nil
}

func (s *storage) DeleteCloudProfile(ctx context.Context, name string) error {
	_, _, err := s.Delete(ctx, name, nil)
	return err
}
