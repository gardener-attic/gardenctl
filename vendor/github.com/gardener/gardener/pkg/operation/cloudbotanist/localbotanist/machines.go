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

package localbotanist

import (
	"github.com/gardener/gardener/pkg/operation"
	"k8s.io/apimachinery/pkg/util/sets"
)

// GetMachineClassInfo returns the name of the class kind, the plural of it and the name of the Helm chart which
// contains the machine class template.
func (b *LocalBotanist) GetMachineClassInfo() (classKind, classPlural, classChartName string) {
	classKind = ""
	classPlural = ""
	classChartName = ""
	return
}

// GenerateMachineClassSecretData generates the secret data for the machine class secret (except the userData field
// which is computed elsewhere).
func (b *LocalBotanist) GenerateMachineClassSecretData() map[string][]byte {
	return map[string][]byte{}
}

// GenerateMachineConfig generates the configuration values for the cloud-specific machine class Helm chart. It
// also generates a list of corresponding MachineDeployments. It returns the computed list of MachineClasses and
// MachineDeployments.
func (b *LocalBotanist) GenerateMachineConfig() ([]map[string]interface{}, operation.MachineDeployments, error) {
	return nil, nil, nil
}

// ListMachineClasses returns two sets of strings whereas the first contains the names of all machine
// classes, and the second the names of all referenced secrets.
func (b *LocalBotanist) ListMachineClasses() (sets.String, sets.String, error) {
	return nil, nil, nil
}

// CleanupMachineClasses deletes all machine classes which are not part of the provided list <existingMachineDeployments>.
func (b *LocalBotanist) CleanupMachineClasses(existingMachineDeployments operation.MachineDeployments) error {
	return nil
}
