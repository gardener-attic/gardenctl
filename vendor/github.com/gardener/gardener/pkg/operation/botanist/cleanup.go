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

package botanist

import (
	"fmt"
	"sync"
	"time"

	kubernetesbase "github.com/gardener/gardener/pkg/client/kubernetes/base"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var exceptions = map[string]map[string]bool{
	kubernetesbase.CustomResourceDefinitions: {
		"felixconfigurations.crd.projectcalico.org":   true,
		"bgppeers.crd.projectcalico.org":              true,
		"bgpconfigurations.crd.projectcalico.org":     true,
		"ippools.crd.projectcalico.org":               true,
		"clusterinformations.crd.projectcalico.org":   true,
		"globalnetworkpolicies.crd.projectcalico.org": true,
		"networkpolicies.crd.projectcalico.org":       true,
		"hostendpoints.crd.projectcalico.org":         true,
	},
	kubernetesbase.DaemonSets: {
		fmt.Sprintf("%s/calico-node", metav1.NamespaceSystem): true,
		fmt.Sprintf("%s/kube-proxy", metav1.NamespaceSystem):  true,
	},
	kubernetesbase.Deployments: {
		fmt.Sprintf("%s/kube-dns", metav1.NamespaceSystem): true,
	},
	kubernetesbase.Namespaces: {
		metav1.NamespacePublic:  true,
		metav1.NamespaceSystem:  true,
		metav1.NamespaceDefault: true,
	},
	kubernetesbase.Services: {
		fmt.Sprintf("%s/kubernetes", metav1.NamespaceDefault): true,
	},
}

// CleanKubernetesResources deletes all the Kubernetes resources in the Shoot cluster
// other than those stored in the exceptions map. It will check whether all the Kubernetes resources
// in the Shoot cluster other than those stored in the exceptions map have been deleted.
// It will return an error in case it has not finished yet, and nil if all resources are gone.
func (b *Botanist) CleanKubernetesResources() error {
	var (
		wg     sync.WaitGroup
		errors []error
	)

	if err := b.K8sShootClient.CleanupResources(exceptions); err != nil {
		return err
	}

	for resource, apiGroupPath := range b.K8sShootClient.GetResourceAPIGroups() {
		wg.Add(1)
		go func(apiGroupPath []string, resource string) {
			defer wg.Done()
			if err := b.waitForAPIGroupCleanedUp(apiGroupPath, resource); err != nil {
				errors = append(errors, err)
			}
		}(apiGroupPath, resource)
	}

	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("Error(s) while waiting for Kubernetes resource cleanup: %+v", errors)
}

// CleanCustomResourceDefinitions deletes all the CRDs in the Kubernetes cluster (which
// will delete the existing custom resources, recursively). It will wait until all resources
// have been cleaned up.
func (b *Botanist) CleanCustomResourceDefinitions() error {
	var (
		apiGroups       = b.K8sShootClient.GetResourceAPIGroups()
		resource        = kubernetesbase.CustomResourceDefinitions
		crdAPIGroupPath = apiGroups[resource]
	)

	if err := b.K8sShootClient.CleanupAPIGroupResources(exceptions, resource, crdAPIGroupPath); err != nil {
		return err
	}
	return b.waitForAPIGroupCleanedUp(crdAPIGroupPath, resource)
}

func (b *Botanist) waitForAPIGroupCleanedUp(apiGroupPath []string, resource string) error {
	if err := wait.PollImmediate(5*time.Second, 10*time.Minute, func() (bool, error) {
		return b.K8sShootClient.CheckResourceCleanup(exceptions, resource, apiGroupPath)
	}); err != nil {
		return fmt.Errorf("Error while waiting for cleanup of '%s' resources: '%s'", resource, err.Error())
	}
	return nil
}
