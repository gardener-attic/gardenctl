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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Infra", func() {
	rs := []string{"shoot--test-gdn--g2jbtq59ih-nodes"}
	terraformstate := `
  {
      "mode": "managed",
      "type": "aws_iam_instance_profile",
      "name": "nodes",
      "provider": "provider.aws",
      "instances": [
        {
          "schema_version": 0,
          "attributes": {
            "arn": "arn:aws:iam::783082953243:instance-profile/shoot--test-gdn--g2jbtq59ih-nodes",
            "create_date": "2020-08-28T16:38:05Z",
            "id": "shoot--test-gdn--g2jbtq59ih-nodes",
            "name": "shoot--test-gdn--g2jbtq59ih-nodes",
            "name_prefix": null,
            "path": "/",
            "role": "shoot--test-gdn--g2jbtq59ih-nodes",
            "roles": [],
            "unique_id": "AIPA3MU3BTIN7SQ6BYLYW"
          },
          "private": "bnVsbA==",
          "dependencies": [
            "aws_iam_role.nodes"
          ]
        }
      ]
    }
`
	Context("After calling getOrphanInfraResources", func() {
		It("should return with no errors", func() {
			err := getOrphanInfraResources(rs, terraformstate)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})