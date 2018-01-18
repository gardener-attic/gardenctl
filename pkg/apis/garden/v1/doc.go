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

// +k8s:deepcopy-gen=package,register
// +k8s:conversion-gen=github.com/gardener/gardenctl/pkg/apis/garden
// +k8s:openapi-gen=true
// +k8s:defaulter-gen=TypeMeta

// Package v1 defines all of the versioned (v1) definitions
// of the shoot model.
// +groupName=garden.sapcloud.io
package v1
