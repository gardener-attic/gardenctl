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

package globalbotanist

import (
	"github.com/gardener/gardenctl/pkg/garden"
	"github.com/gardener/gardenctl/pkg/garden/botanist"
)

const (
	// EtcdRoleMain is the constant defining the role for main etcd storing data about objects in Shoot.
	EtcdRoleMain = "main"

	// EtcdRoleEvents is the constant defining the role for etcd storing events in Shoot.
	EtcdRoleEvents = "events"
)

// GlobalBotanist is a struct which contains the "normal" Botanist as well as the CloudBotanist.
// It is used to execute the work for which input from both is required or functionalities from
// both must be used.
type GlobalBotanist struct {
	*garden.Garden
	Botanist      *botanist.Botanist
	CloudBotanist garden.CloudBotanist
}
