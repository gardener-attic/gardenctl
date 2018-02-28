// Copyright 2018 The Gardener Authors.
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

package kubernetesv17

import (
	kubernetesv16 "github.com/gardener/gardener/pkg/client/kubernetes/v16"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	propagationPolicy    = metav1.DeletePropagationForeground
	gracePeriodSeconds   = int64(300)
	defaultDeleteOptions = metav1.DeleteOptions{
		PropagationPolicy:  &propagationPolicy,
		GracePeriodSeconds: &gracePeriodSeconds,
	}
)

// Client inherits all the attributes and methods of the v1.6 client.
// Please see the documentation of the base client for further details.
type Client struct {
	*kubernetesv16.Client
}
