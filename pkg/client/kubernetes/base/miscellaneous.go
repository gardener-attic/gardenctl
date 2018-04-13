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

package kubernetesbase

import (
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/rest"
)

// Curl performs an HTTP GET request to the API server and returns the result.
func (c *Client) Curl(path string) (*rest.Result, error) {
	res := c.
		RESTClient.
		Get().
		AbsPath(path).
		Do()
	err := res.Error()
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetVersion queries the version of the API server and returns the version struct, and the version
// in the form '<major>.<minor>'.
func (c *Client) GetVersion() (*version.Info, string, error) {
	serverVersion, err := c.
		Clientset.
		Discovery().
		ServerVersion()
	if err != nil {
		return nil, "", err
	}
	r, _ := regexp.Compile(`^(\d)+`)
	serverVersion.Minor = r.FindString(serverVersion.Minor)
	serverVersion.Major = r.FindString(serverVersion.Major)
	version := serverVersion.Major + "." + serverVersion.Minor
	c.version = version
	return serverVersion, version, nil
}

// Version returns the version of the Kubernetes cluster the client is acting on as integer, i.e. it
// converts <major.minor> to the concatenation of both. If an error occurs during conversion, 0 is returned.
func (c *Client) Version() int {
	if c.version == "" {
		c.GetVersion()
	}
	version, err := strconv.Atoi(strings.Replace(c.version, ".", "", -1))
	if err != nil {
		return 0
	}
	return version
}
