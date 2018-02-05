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
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

// openstackCmd represents the openstack command
var openstackCmd = &cobra.Command{
	Use:   "openstack <args>",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var t Target
		targetFile, err := ioutil.ReadFile(pathTarget)
		checkError(err)
		err = yaml.Unmarshal(targetFile, &t)
		checkError(err)
		if len(t.Target) < 3 {
			fmt.Println("No shoot targeted")
			os.Exit(2)
		}
		arguments := "openstack " + strings.Join(args[:], " ")
		operate("openstack", arguments)
	},
}

func init() {
}
