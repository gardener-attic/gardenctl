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
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/spf13/cobra"
)

// NewHibernateCmd returns diagnostic information for a shoot.
func NewHibernateCmd(reader TargetReader, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "hibernate",
		Short:        "Hibernate a cluster if wakeup-able and shoot is wakeup",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := reader.ReadTarget(pathTarget)
			if !CheckShootIsTargeted(target) {
				return errors.New("no shoot targeted")
			}

			shoot, err := FetchShootFromTarget(target)
			checkError(err)
			hibernateShoot(shoot, reader)
			return nil
		},
	}
	return cmd
}

func hibernateShoot(shoot *v1beta1.Shoot, reader TargetReader) {
	if shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled != nil && *shoot.Spec.Hibernation.Enabled {
		fmt.Println("shoot already hibernated!")
		os.Exit(0)
	}

	//hibernate the shoot
	newShoot := shoot.DeepCopy()
	setHibernation(newShoot, true)
	patchedShoot, err := MergePatchShoot(shoot, newShoot, reader)
	checkError(err)
	*shoot = *patchedShoot

	time.Sleep(time.Second * 20)

	//wait for shoot to be reconciled in 20 mins
	err = waitShootReconciled(shoot, reader)
	checkError(err)
}
