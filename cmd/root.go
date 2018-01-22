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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var cachevar bool
var outputFormat string
var gardenConfig string
var pathGardenHome string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "g",
	Short: "",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	pathGardenHome = os.Getenv("GARDENCTL_HOME")
	if pathGardenHome == "" {
		pathGardenHome = pathDefault
	} else if strings.Contains(pathGardenHome, "~") {
		pathGardenHome = strings.Replace(pathGardenHome, "~", HomeDir(), 1)
	}
	pathSeedCache = filepath.Join(pathGardenHome, "cache", "seeds")
	pathProjectCache = filepath.Join(pathGardenHome, "cache", "projects")
	pathShootCache = filepath.Join(pathGardenHome, "cache", "shoots")
	pathGardenConfig = filepath.Join(pathGardenHome, "config")
	pathTarget = filepath.Join(pathGardenHome, "target")
	createDir(pathGardenHome, 0751)
	createDir(pathGardenHome+"/cache", 0751)
	createDir(pathGardenHome+"/cache/seeds", 0751)
	createDir(pathGardenHome+"/cache/projects", 0751)
	gardenConfig = os.Getenv("GARDENCONFIG")
	if gardenConfig != "" {
		pathGardenConfig = gardenConfig
	}
	getGardenClusterKubeConfigFromConfig()
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().BoolVarP(&cachevar, "no-cache", "n", false, "no caching")
	RootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "yaml", "output format yaml or json")
	cobra.EnableCommandSorting = false
	cobra.EnablePrefixMatching = prefixMatching
	RootCmd.AddCommand(lsCmd, targetCmd, dropCmd, getCmd)
	RootCmd.AddCommand(downloadCmd, showCmd, logsCmd)
	RootCmd.AddCommand(completionCmd)
	RootCmd.AddCommand(kubectlCmd, kaCmd, ksCmd, awsCmd, azCmd, gcloudCmd, openstackCmd)
	RootCmd.SuggestionsMinimumDistance = suggestionsMinimumDistance
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
}
