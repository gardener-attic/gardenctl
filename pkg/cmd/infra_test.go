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
	. "github.com/gardener/gardenctl/pkg/cmd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Infra", func() {
	var rs = []string{"vpc-03cb057da4ded427f"}
	terraformstate := `
},
    "vpc_id": {
      "value": "vpc-03cb057da4ded427f",
      "type": "string"
    }
  },
`
	Context("Calling GetOrphanInfraResources", func() {
		It("should return err == nil", func() {
			err := GetOrphanInfraResources(rs, terraformstate)
			Expect(err).To(BeNil())
		})
	})
})
