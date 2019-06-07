// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package cmd

import (
	"errors"

	"k8s.io/client-go/kubernetes"
)

// ReadTarget returns the current target.
func (r *GardenctlTargetReader) ReadTarget(targetPath string) TargetInterface {
	var target Target
	ReadTarget(targetPath, &target)
	return &target
}

// Stack return current target stack.
func (t *Target) Stack() []TargetMeta {
	return t.Target
}

// SetStack sets the current target stack to a new one.
func (t *Target) SetStack(stack []TargetMeta) {
	t.Target = stack
}

// Kind returns the current target kind.
func (t *Target) Kind() (TargetKind, error) {
	length := len(t.Target)
	switch length {
	case 1:
		return TargetKindGarden, nil
	case 2:
		if t.Target[1].Kind == "seed" {
			return TargetKindSeed, nil
		}

		return TargetKindProject, nil
	case 3:
		return TargetKindShoot, nil
	default:
		return "", errors.New("No target selected")
	}
}

// K8SClient returns a kubernetes client configured against the current target.
func (t *Target) K8SClient() (kubernetes.Interface, error) {
	var kind TargetKind
	if kind, err = t.Kind(); err != nil {
		return nil, err
	}

	return clientToTarget(kind)
}

// K8SClientToKind returns a kubernetes client configured against the given target <kind>.
func (t *Target) K8SClientToKind(kind TargetKind) (kubernetes.Interface, error) {
	return clientToTarget(kind)
}
