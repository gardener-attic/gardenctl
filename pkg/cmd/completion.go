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

	"github.com/spf13/cobra"
)

// NewCompletionCmd returns a new completion command.
func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <bash|zsh>",
		Short: "Generate bash or zsh completion script, e.g. \"gardenctl completion zsh\" generates completion for zsh",
	}

	cmd.AddCommand(NewBashCompletionCmd(), NewZshCompletionCmd())

	return cmd
}

// NewBashCompletionCmd returns a new bash completion command.
func NewBashCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RootCmd.GenBashCompletion(os.Stdout)
		},
	}
}

// NewZshCompletionCmd returns a new zsh completion command.
func NewZshCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RootCmd.GenZshCompletion(os.Stdout)
		},
	}
}
