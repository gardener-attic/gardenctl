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
	. "github.com/gardener/gardenctl/pkg/internal/history"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("History", func() {
	history := History{
		ConfigPath: "A",
		Items:      []string{"A", "B"},
	}

	Context("SetPath", func() {
		It("should be a return path", func() {
			a := &History{ConfigPath: "A"}
			out := SetPath("A")
			Expect(out).To(Equal(a))
			Expect(out).NotTo(BeNil())
		})
	})

	Context("List", func() {
		It("should be a return List order by Ascending", func() {
			out := history.List().Items
			Expect(out).To(Equal([]string{"A", "B"}))
		})
	})

	Context("Reverse", func() {
		It("should be a return Reverse order by Descending", func() {
			out := history.Reverse().Items
			Expect(out).Should(Equal([]string{"B", "A"}))
			Expect(out).NotTo(BeNil())
		})
	})

	Context("PromptItems", func() {
		It("should be a return Promp cmd, Garden/Project/Shoot when use target", func() {
			list := []string{`{"Cmd":"gardenctl target --garden live --project projectA --shoot shootA","garden":"live","project":"projectA","shoot":"shootA"}`}
			out := PromptItems(list)
			Expect(out[0].Cmd).To(Equal("gardenctl target --garden live --project projectA --shoot shootA"))
			Expect(out[0].Shoot).To(Equal("shootA"))
			Expect(out[0].Garden).To(Equal("live"))
			Expect(out[0].Project).To(Equal("projectA"))
		})

		It("should be a return Promp cmd, Promp Garden/Project/Shoot when use --server for Seed", func() {
			list := []string{`{"Cmd":"gardenctl target --server https://A.B.C.D.ondemand.com --seed soilA --namespace shoot--projectA--shootA","garden":"live","namespace":"shoot--projectA--shootA","seed":"soilA"}`}
			out := PromptItems(list)
			Expect(out[0].Cmd).To(Equal("gardenctl target --server https://A.B.C.D.ondemand.com --seed soilA --namespace shoot--projectA--shootA"))
			Expect(out[0].Seed).To(Equal("soilA"))
			Expect(out[0].Garden).To(Equal("live"))
			Expect(out[0].Namespace).To(Equal("shoot--projectA--shootA"))
		})

		It("should be a return Promp cmd, Promp Garden/Project/Shoot when use --server for non-Seed", func() {
			list := []string{`{"Cmd":"gardenctl target --server https://A.B.C.D.ondemand.com --project projectA --shoot shootA","garden":"live","project":"projectA","shoot":"shootA"}`}
			out := PromptItems(list)
			Expect(out[0].Cmd).To(Equal("gardenctl target --server https://A.B.C.D.ondemand.com --project projectA --shoot shootA"))
			Expect(out[0].Garden).To(Equal("live"))
			Expect(out[0].Project).To(Equal("projectA"))
			Expect(out[0].Shoot).To(Equal("shootA"))
		})

		It("should be a return Promp cmd, Promp Garden/Project/Shoot when use --dashboardUrl for Seed", func() {
			list := []string{`{"Cmd":"gardenctl target --dashboardUrl https://A.B.C.D.ondemand.com/namespace/garden/shoots/aws-euA/","garden":"live","project":"garden","shoot":"aws-euA"}`}
			out := PromptItems(list)
			Expect(out[0].Cmd).To(Equal("gardenctl target --dashboardUrl https://A.B.C.D.ondemand.com/namespace/garden/shoots/aws-euA/"))
			Expect(out[0].Garden).To(Equal("live"))
			Expect(out[0].Project).To(Equal("garden"))
			Expect(out[0].Shoot).To(Equal("aws-euA"))
		})

		It("should be a return Promp cmd, Promp Garden/Project/Shoot when use --dashboardUrl for non-Seed", func() {
			list := []string{`{"Cmd":"gardenctl target --dashboardUrl https://A.B.C.D.ondemand.com/namespace/garden-projectA/shoots/shootA/","garden":"live","project":"projectA","shoot":"shootA"}`}
			out := PromptItems(list)
			Expect(out[0].Cmd).To(Equal("gardenctl target --dashboardUrl https://A.B.C.D.ondemand.com/namespace/garden-projectA/shoots/shootA/"))
			Expect(out[0].Garden).To(Equal("live"))
			Expect(out[0].Project).To(Equal("projectA"))
			Expect(out[0].Shoot).To(Equal("shootA"))
		})
	})
})
