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

package garden_test

import (
	"fmt"

	. "github.com/gardener/gardenctl/pkg/garden"
	"github.com/gardener/gardenctl/pkg/test/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("garden", func() {
	Describe("terraformer", func() {
		var (
			garden  = mocks.NewMockGarden()
			purpose = "test"
		)

		Describe("#NewTerraformer", func() {
			It("should create a Terraformer object", func() {
				shootName := garden.Shoot.ObjectMeta.Name

				terraformer := NewTerraformer(garden, purpose)

				Expect(terraformer.Garden).To(Equal(garden))
				Expect(terraformer.Namespace).To(Equal(garden.Shoot.ObjectMeta.Namespace))
				Expect(terraformer.Purpose).To(Equal(purpose))
				Expect(terraformer.ConfigName).To(Equal(fmt.Sprintf("%s.%s.tf-config", shootName, purpose)))
				Expect(terraformer.VariablesName).To(Equal(fmt.Sprintf("%s.%s.tf-vars", shootName, purpose)))
				Expect(terraformer.StateName).To(Equal(fmt.Sprintf("%s.%s.tf-state", shootName, purpose)))
				Expect(terraformer.JobName).To(Equal(fmt.Sprintf("%s.%s.tf-job", shootName, purpose)))
			})
		})

		Describe("<Terraformer>", func() {
			var (
				terraformer          *Terraformer
				renderTemplateCalled bool
			)

			BeforeEach(func() {
				terraformer = NewTerraformer(garden, purpose)
				renderTemplateCalled = false
			})

			AfterEach(func() {
				terraformer = nil
				garden.K8sGardenClient.Bootstrap()
			})
		})
	})
})
