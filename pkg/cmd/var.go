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

package cmd

import (
	"path/filepath"

	"k8s.io/client-go/kubernetes"
)

const (
	// configuration for gardenctl
	suggestionsMinimumDistance int  = 2
	prefixMatching             bool = true
)

var (
	// Client is Clientset to use for specified cluster
	Client     *kubernetes.Clientset
	kubeconfig *string

	// KUBECONFIG contains path to file
	KUBECONFIG string
	garden     bool
	seed       bool
	project    bool

	// credentials
	username string
	password string

	// file pathes
	pathGardenConfig   string
	pathTarget         string
	pathDefault        = filepath.Join(HomeDir(), ".garden")
	pathDefaultSession = filepath.Join(HomeDir(), ".garden", "sessions")
)
