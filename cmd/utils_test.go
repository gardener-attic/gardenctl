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
	"fmt"
	"os"
	"strings"

	. "github.com/gardener/gardenctl/cmd"
	yaml "gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {

	var target Target
	var tm TargetMeta
	dumpPath := "/tmp"
	pathDir := dumpPath + "/testDir"
	pathFile := dumpPath + "/testDir/testfile"
	pathTarget := dumpPath + "/target"
	tm.Name = "garden-test"
	tm.Kind = TargetKindGarden
	target.Target = append(target.Target, tm)
	tm.Name = "seed-test"
	tm.Kind = TargetKindSeed
	target.Target = append(target.Target, tm)
	tm.Name = "shoot-test"
	tm.Kind = TargetKindShoot
	target.Target = append(target.Target, tm)

	file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
	content, err := yaml.Marshal(target)
	if err != nil {
		panic(err)
	}
	file.Write(content)
	err = file.Close()
	if err != nil {
		panic(err)
	}

	Context("After reading Home dir", func() {
		It("should be /root", func() {
			os.Setenv("HOME", "/root")
			dir := HomeDir()
			Expect(dir).To(Equal("/root"))
		})
	})

	Context("After creating a dir", func() {
		It("os.Stat should return err == nil", func() {
			CreateDir(pathDir, 0755)
			_, err = os.Stat(pathDir)
			Expect(err).To(BeNil())
		})
	})

	Context("Before creating a file", func() {
		It("CreateFile should return err and exists == false", func() {
			exists, _ := FileExists(pathFile + ".txt")
			Expect(exists).To(BeFalse())
		})
	})

	Context("After creating a file", func() {
		It("CreateFile should return err == nil and exists == true", func() {
			CreateFileIfNotExists(pathFile, 0644)
			exists, err := FileExists(pathFile)
			Expect(err).To(BeNil())
			Expect(exists).To(BeTrue())
		})
	})

	Context("After executing a shell command", func() {
		It("ExecCmd should return err == nil", func() {
			err := ExecCmd(nil, "sleep 1", false)
			Expect(err).To(BeNil())
		})
	})

	Context("After setting KUBECONFIG environment variable", func() {
		It("ExecCmdReturnOutput should return /tmp/kubeconfig as output", func() {
			output, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG=/tmp/kubeconfig; printenv KUBECONFIG")
			if err != nil {
				fmt.Println("Cmd was unsuccessful")
				os.Exit(2)
			}
			output = strings.TrimSpace(output)
			if err != nil {
				fmt.Println("Cmd was unsuccessful")
				os.Exit(2)
			}
			Expect(output).To(Equal("/tmp/kubeconfig"))
		})
	})

	Context("After targeting a shoot", func() {
		It("readTarget should return target stack with three elements", func() {
			ReadTarget(pathTarget, &target)
			Expect(len(target.Target)).To(Equal(3))
		})
	})

	Context("After creating target object", func() {
		It("name of garden cluster should be garden-test", func() {
			Expect(target.Target[0].Name).To(Equal("garden-test"))
		})
	})

	Context("After creating target object", func() {
		It("name of seed cluster should be garden-test", func() {
			Expect(target.Target[1].Name).To(Equal("seed-test"))
		})
	})

	Context("After creating target object", func() {
		It("name of shoot cluster should be garden-test", func() {
			Expect(target.Target[2].Name).To(Equal("shoot-test"))
		})
	})
})
