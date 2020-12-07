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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewAwsCmd returns a new aws command.
func NewAwsCmd(targetReader TargetReader) *cobra.Command {
	return &cobra.Command{
		Use:                "aws <args>",
		Short:              "e.g. \"gardenctl aws ec2 describe-security-groups\"",
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := targetReader.ReadTarget(pathTarget)
			if !CheckShootIsTargeted(target) {
				return errors.New("no shoot targeted")
			}
			if !CheckToolInstalled("aws") {
				fmt.Println("Please go to https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html for how to install aws cli")
				os.Exit(2)
			}

			arguments := strings.Join(os.Args[2:], " ")
			fmt.Println(operate("aws", arguments))

			return nil
		},
	}
}
