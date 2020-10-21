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
	"io"
	"path/filepath"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewGetCmd returns a new get command.
func NewGetCmd(targetReader TargetReader, configReader ConfigReader,
	kubeconfigReader KubeconfigReader, kubeconfigWriter KubeconfigWriter, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "get [(garden|project|seed|shoot|target) <name>]",
		Short:        "Get single resource instance or target stack, e.g. CRD of a shoot (default: current target)\n",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) < 1 || len(args) > 2 {
				return errors.New("command must be in the format: get [(garden|project|seed|shoot|target) <name>]")
			}

			name := ""
			if len(args) == 2 {
				name = args[1]
			}

			switch args[0] {
			case "project":
				if IsTargeted(targetReader, "project") {
					err = printProjectKubeconfig(name, targetReader, ioStreams.Out, outputFormat)
					checkError(err)
				} else {
					return errors.New("no project targeted")
				}

			case "garden":
				if IsTargeted(targetReader, "garden") {
					err = printGardenKubeconfig(name, configReader, targetReader, kubeconfigReader, ioStreams.Out, outputFormat)
					checkError(err)
				} else {
					return errors.New("no garden targeted")
				}

			case "seed":
				if IsTargeted(targetReader, "seed") || IsTargeted(targetReader, "project", "shoot") {
					err = printSeedKubeconfig(name, targetReader, ioStreams.Out, outputFormat)
					checkError(err)
				} else {
					return errors.New("no seed targeted targeted or shoot targeted")
				}

			case "shoot":
				if IsTargeted(targetReader, "shoot") {
					err = printShootKubeconfig(name, targetReader, kubeconfigWriter, ioStreams.Out, outputFormat)
					checkError(err)
				} else {
					return errors.New("no shoot targeted")
				}

			case "target":
				if !IsTargeted(targetReader) {
					return errors.New("target stack is empty")
				}

				err = printTarget(targetReader, ioStreams.Out, outputFormat)
				if err != nil {
					return err
				}
			default:
				fmt.Fprint(ioStreams.Out, "command must be in the format: get [project|garden|seed|shoot|target] + <NAME>")
			}

			return nil
		},
		ValidArgs: []string{"project", "garden", "seed", "shoot", "target"},
	}

	return cmd
}

// printProjectKubeconfig lists
func printProjectKubeconfig(name string, targetReader TargetReader, writer io.Writer, outFormat string) error {
	var err error
	var project *v1beta1.Project
	if name == "" {
		project, err = GetTargetedProjectObject(targetReader)
	} else {
		project, err = GetProjectObject(targetReader, name)
	}

	if err != nil {
		return err
	}

	return PrintoutObject(project, writer, outFormat)
}

// printGardenKubeconfig lists kubeconfig of garden cluster
func printGardenKubeconfig(name string, configReader ConfigReader, targetReader TargetReader, kubeconfigReader KubeconfigReader, writer io.Writer, outFormat string) error {
	if name == "" {
		var err error
		name, err = GetTargetName(targetReader, "garden")
		checkError(err)
	}

	config := configReader.ReadConfig(pathGardenConfig)
	match := false
	for index, garden := range config.GardenClusters {
		if garden.Name == name {
			pathToKubeconfig := config.GardenClusters[index].KubeConfig
			pathToKubeconfig = TidyKubeconfigWithHomeDir(pathToKubeconfig)
			kubeconfig, err := kubeconfigReader.ReadKubeconfig(pathToKubeconfig)
			if err != nil {
				return err
			}

			return PrintoutObject(kubeconfig, writer, outFormat)
		}
	}
	if !match {
		return fmt.Errorf("no garden cluster found for %s", name)
	}

	return nil
}

// printSeedKubeconfig lists kubeconfig of seed cluster
func printSeedKubeconfig(name string, targetReader TargetReader, writer io.Writer, outFormat string) error {
	target := targetReader.ReadTarget(pathTarget)

	client, err := target.K8SClientToKind(TargetKindGarden)
	if err != nil {
		return err
	}
	var seed *v1beta1.Seed
	if name == "" {
		seed, err = GetTargetedSeedObject(targetReader)
		if err != nil {
			return err
		}
	} else {
		seed, err = GetSeedObject(targetReader, name)
		if err != nil {
			return err
		}
	}

	kubeSecret, err := client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return PrintoutObject(fmt.Sprintf("%s\n", kubeSecret.Data["kubeconfig"]), writer, outFormat)
}

// printShootKubeconfig lists kubeconfig of shoot
func printShootKubeconfig(name string, targetReader TargetReader, kubeconfigWriter KubeconfigWriter, writer io.Writer, outFormat string) error {
	target := targetReader.ReadTarget(pathTarget)

	client, err := target.K8SClientToKind(TargetKindGarden)
	if err != nil {
		return err
	}

	var shoot *v1beta1.Shoot
	if name == "" {
		shoot, err = GetTargetedShootObject(targetReader)
		checkError(err)
	} else {
		shoot, err = GetShootObject(targetReader, name)
		checkError(err)
	}

	namespace := shoot.Status.TechnicalID

	seed, err := GetTargetedSeedObject(targetReader)
	if err != nil {
		return err
	}
	kubeSecret, err := client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	gardenName, err := GetTargetName(targetReader, "garden")
	checkError(err)
	kubeconfigPath := filepath.Join(pathGardenHome, "cache", gardenName, "seeds", seed.Spec.SecretRef.Name, "kubeconfig.yaml")
	err = kubeconfigWriter.Write(kubeconfigPath, kubeSecret.Data["kubeconfig"])
	checkError(err)
	KUBECONFIG = kubeconfigPath

	seedClient, err := target.K8SClientToKind(TargetKindSeed)
	if err != nil {
		return err
	}

	kubeSecret, err = seedClient.CoreV1().Secrets(namespace).Get("kubecfg", metav1.GetOptions{})
	if err != nil {
		return err
	}

	return PrintoutObject(fmt.Sprintf("%s\n", kubeSecret.Data["kubeconfig"]), writer, outFormat)
}

// printTarget prints the target stack.
func printTarget(targetReader TargetReader, writer io.Writer, outFormat string) (err error) {
	target := targetReader.ReadTarget(pathTarget)
	return PrintoutObject(target, writer, outFormat)
}
