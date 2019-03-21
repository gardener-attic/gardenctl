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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	// Register clients
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var cachevar bool
var outputFormat string
var gardenConfig string
var pathGardenHome string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "gardenctl",
	Short: "g",
	Long:  ``,
}

const (
	bashCompletionFunc = `

__gardenctl_parse() {
	local gardenctl_out
	case $1 in
	garden)
	if gardenctl_out=$(gardenctl ls gardens | grep -w "\- name:" | awk '{print $3}' 2>/dev/null); then
		COMPREPLY+=( $( compgen -W "${gardenctl_out[*]}" -- "$cur" ) )
	fi
	;;
	seed)
	if gardenctl_out=$(gardenctl ls seeds | grep -w "\- seed:" | awk '{print $3}' 2>/dev/null); then
		COMPREPLY+=( $( compgen -W "${gardenctl_out[*]}" -- "$cur" ) )
	fi
	;;
	project)
	if gardenctl_out=$(gardenctl ls projects | grep -w "\- project:" | awk '{print $3}' 2>/dev/null); then
		COMPREPLY+=( $( compgen -W "${gardenctl_out[*]}" -- "$cur" ) )
	fi
	;;
	shoot)
	if gardenctl_out=$(gardenctl ls shoots | grep -v "projects:" | grep -v "\- project: " | grep -v "seeds:" | grep -v "\- seed: " | grep -v "shoots:" | awk '{print $2}' 2>/dev/null); then
		COMPREPLY+=( $( compgen -W "${gardenctl_out[*]}" -- "$cur" ) )
	fi
	;;
	*)
	;;
	esac
}

__gardenctl_custom_func() {
	case ${last_command} in
	gardenctl_target | gardenctl_get)
	if [[ ${#nouns[@]} -ne 0 ]]; then
		__gardenctl_parse "${nouns[${#nouns[@]} -1]}"
	fi
	;;
	esac
}
`
)

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
	CreateDir(pathGardenHome, 0751)
	CreateFileIfNotExists(pathTarget, 0644)
	gardenConfig = os.Getenv("GARDENCONFIG")
	if gardenConfig != "" {
		pathGardenConfig = gardenConfig
	}
	if _, err := os.Stat(pathGardenConfig); err != nil {
		CreateFileIfNotExists(pathGardenConfig, 0644)
	}
	CreateDir(pathGardenHome+"/cache", 0751)
	CreateDir(pathGardenHome+"/cache/seeds", 0751)
	CreateDir(pathGardenHome+"/cache/projects", 0751)
	GetGardenClusterKubeConfigFromConfig(pathGardenConfig, pathTarget)
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
	RootCmd.AddCommand(registerCmd, unregisterCmd)
	RootCmd.AddCommand(completionCmd)
	RootCmd.AddCommand(shellCmd)
	RootCmd.AddCommand(sshCmd)
	RootCmd.AddCommand(kubectlCmd, kaCmd, ksCmd, kgCmd, knCmd, awsCmd, azCmd, gcloudCmd, openstackCmd, aliyunCmd)
	RootCmd.AddCommand(infoCmd)
	RootCmd.AddCommand(versionCmd)
	RootCmd.SuggestionsMinimumDistance = suggestionsMinimumDistance
	RootCmd.BashCompletionFunction = bashCompletionFunc
	RootCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{if .HasAvailableFlags}}{{appendIfNotPresent .UseLine "[flags]"}}{{else}}{{.UseLine}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath}} [command]{{end}}{{if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}
{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{ if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimRightSpace}}{{end}}{{ if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimRightSpace}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsHelpCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{ if .HasAvailableSubCommands }}

Use "{{.CommandPath}} [command] --help" for more information about a command.

Configuration and KUBECONFIG file cache located $GARDENCTL_HOME or ~/.garden (default).
Gardenctl configuration file must be provided in $GARDENCONFIG or ~/.garden/config (default).

Find more information and an example configuration at https://github.com/gardener/gardenctl
{{end}}
`)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
}
