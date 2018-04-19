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

package utils

import (
	"fmt"

	"github.com/Masterminds/semver"
)

// CheckIfNewVersion returns true if the <version1> is newer than <version2>, and false otherwise.
// The comparison is based on semantic versions, i.e. <version1> and <version2> will be converted
// if needed.
func CheckIfNewVersion(version1, version2 string) (bool, error) {
	v1, err := convertToSemanticVersion(version1)
	if err != nil {
		return false, err
	}
	v2, err := convertToSemanticVersion(version2)
	if err != nil {
		return false, err
	}

	constraint, err := semver.NewConstraint(fmt.Sprintf("< %s", v1))
	if err != nil {
		return false, err
	}
	version, _ := semver.NewVersion(v2)
	if err != nil {
		return false, err
	}

	return constraint.Check(version), nil
}

// convertToSemanticVersion converts a version with missing "major" parts to a semantic version by settings
// the major part to zero.
func convertToSemanticVersion(v string) (string, error) {
	version, err := semver.NewVersion(v)
	if err != nil {
		return "", err
	}

	if version.Major() != 0 && version.Patch() == 0 {
		return fmt.Sprintf("0.%s", v), nil
	}
	return v, nil
}
