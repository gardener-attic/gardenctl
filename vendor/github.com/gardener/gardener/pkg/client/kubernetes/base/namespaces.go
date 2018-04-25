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

package kubernetesbase

import (
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a new Namespace object.
func (c *Client) CreateNamespace(namespace *corev1.Namespace, updateIfExists bool) (*corev1.Namespace, error) {
	res, err := c.clientset.CoreV1().Namespaces().Create(namespace)
	if err != nil && apierrors.IsAlreadyExists(err) && updateIfExists {
		return c.UpdateNamespace(namespace)
	}
	return res, err
}

// UpdateNamespace updates an already existing Namespace object.
func (c *Client) UpdateNamespace(namespace *corev1.Namespace) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Update(namespace)
}

// GetNamespace returns a Namespace object.
func (c *Client) GetNamespace(name string) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
}

// ListNamespaces returns a list of namespaces. The selection can be restricted by passing a <selector>.
func (c *Client) ListNamespaces(selector metav1.ListOptions) (*corev1.NamespaceList, error) {
	return c.clientset.CoreV1().Namespaces().List(selector)
}

// DeleteNamespace deletes a namespace.
func (c *Client) DeleteNamespace(name string) error {
	deleteGracePeriod := int64(1)
	return c.clientset.CoreV1().Namespaces().Delete(name, &metav1.DeleteOptions{
		PropagationPolicy:  &propagationPolicy,
		GracePeriodSeconds: &deleteGracePeriod,
	})
}
