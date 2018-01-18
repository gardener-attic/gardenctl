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

package cmd

import (
	"path/filepath"

	"k8s.io/client-go/kubernetes"
)

// Client is Clientset to use for specified cluster
var Client *kubernetes.Clientset
var err error
var kubeconfig *string
var KUBECONFIG string
var masterURL *string
var garden bool
var seed bool
var project bool

// configuration for gardenctl
const suggestionsMinimumDistance int = 2
const prefixMatching bool = true

// credentials
var username string
var password string

// file pathes
var (
	pathSeedCache    string
	pathProjectCache string
	pathShootCache   string
	pathGardenConfig string
	pathTarget       string
	pathDefault      = filepath.Join(HomeDir(), ".garden")
)
