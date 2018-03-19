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

package componentconfig

import (
	"os"
)

func FindCloudProviderConfig(cloudProviderConfiguration []GardenOperatorCloudProviderConfiguration, name string) ([]string, GardenOperatorCloudProviderConfiguration) {
	var (
		cloudProviders      []string
		cloudProviderConfig = GardenOperatorCloudProviderConfiguration{}
	)
	for _, cloudProvider := range cloudProviderConfiguration {
		cloudProviders = append(cloudProviders, cloudProvider.Name)
		if cloudProvider.Name == name {
			cloudProviderConfig = cloudProvider
		}
	}
	return cloudProviders, cloudProviderConfig
}

func ApplyEnvironmentToConfig(config *GardenOperatorConfiguration) {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		config.ClientConnection.KubeConfigFile = kubeconfig
	}
	if watchNamespace := os.Getenv("WATCH_NAMESPACE"); watchNamespace != "" {
		config.Controller.WatchNamespace = &watchNamespace
		config.LeaderElection.LockObjectNamespace = watchNamespace
	}
}
