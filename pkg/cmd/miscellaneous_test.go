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

package cmd_test

import (
	"os"

	. "github.com/gardener/gardenctl/pkg/cmd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Miscellaneous", func() {
	var gardenConf GardenConfig
	var target Target
	dumpPath := "/tmp"
	pathTarget := dumpPath + "/target2"
	pathGardenConfig := dumpPath + "/gardenconfig"
	file, err := os.OpenFile(pathGardenConfig, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	gConfig := `
githubURL: https://github.croft.gardener.corp
gardenClusters:
- name: dev
  kubeConfig: /tmp/kubeconfig.yaml
- name: prod
  kubeConfig: /tmp/kubeconfig.yaml
`
	content := []byte(gConfig)
	_, err = file.Write(content)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
	file, err = os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
	content = []byte("")
	if err != nil {
		panic(err)
	}
	_, err = file.Write(content)
	if err != nil {
		panic(err)
	}
	err = file.Close()
	if err != nil {
		panic(err)
	}
	Context("After calling GetGardenClusterKubeConfigFromConfig", func() {
		It("First Garden Cluster should be set as default target Name if no garden cluster is specified", func() {
			GetGardenClusterKubeConfigFromConfig(pathGardenConfig, pathTarget)
			ReadTarget(pathTarget, &target)
			Expect(target.Target[0].Name).To(Equal("dev"))
		})
	})
	Context("After calling GetGardenClusters", func() {
		It("GardenCluster Name should be dev ", func() {
			GetGardenConfig(pathGardenConfig, &gardenConf)
			Expect(gardenConf.GardenClusters[0].Name).To(Equal("dev"))
			Expect(gardenConf.GardenClusters[1].Name).To(Equal("prod"))
		})
	})

	var _ = AfterSuite(func() {
		os.Remove(pathTarget)
		os.Remove(pathGardenConfig)
	})
})
