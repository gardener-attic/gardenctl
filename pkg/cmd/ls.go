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
	"encoding/json"
	"errors"
	"fmt"
	"os"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencoreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewLsCmd returns a new ls command.
func NewLsCmd(targetReader TargetReader, configReader ConfigReader, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ls [gardens|projects|seeds|shoots|issues|namespaces]",
		Short:        "List all resource instances, e.g. list of shoots|issues",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(args) < 1 || len(args) > 2 {
				return errors.New("command must be in the format: ls [gardens|projects|seeds|shoots|issues|namespaces]")
			}

			target := targetReader.ReadTarget(pathTarget)
			if (len(target.Stack()) == 0) && args[0] != "gardens" {
				return errors.New("target stack is empty")
			}
			switch args[0] {
			case "projects":
				return printProjectsWithShoots(target, ioStreams)
			case "gardens":
				PrintGardenClusters(configReader, outputFormat, ioStreams)
			case "seeds":
				clientset, err := target.GardenerClient()
				checkError(err)
				seedList := getSeeds(clientset)
				var seeds Seeds
				for _, seed := range seedList.Items {
					var sm SeedMeta
					sm.Seed = seed.Name
					seeds.Seeds = append(seeds.Seeds, sm)
				}
				if outputFormat == "yaml" {
					y, err := yaml.Marshal(seeds)
					checkError(err)
					os.Stdout.Write(y)
				} else if outputFormat == "json" {
					j, err := json.MarshalIndent(seeds, "", "  ")
					checkError(err)
					fmt.Fprint(ioStreams.Out, string(j))
				}
			case "shoots":
				if len(target.Stack()) == 1 {
					return printProjectsWithShoots(target, ioStreams)
				} else if len(target.Stack()) == 2 && target.Stack()[1].Kind == "seed" {
					getProjectsWithShootsForSeed(ioStreams)
				} else if len(target.Stack()) == 2 && target.Stack()[1].Kind == "project" {
					getSeedsWithShootsForProject(ioStreams)
				}
			case "issues":
				getIssues(target, ioStreams)
			case "namespaces":
				getNamespaces(ioStreams)
			default:
				return errors.New("command must be in the format: ls [gardens|projects|seeds|shoots|issues|namespaces]")
			}

			return nil
		},
		ValidArgs: []string{"issues", "projects", "gardens", "seeds", "shoots", "namespaces"},
	}

	return cmd
}

// printProjectsWithShoots lists list of projects with shoots
func printProjectsWithShoots(target TargetInterface, ioStreams IOStreams) error {
	gardenClientset, err := target.GardenerClient()
	if err != nil {
		return err
	}
	projectList, err := gardenClientset.CoreV1beta1().Projects().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	shootList, err := gardenClientset.CoreV1beta1().Shoots("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var projects Projects
	for _, project := range projectList.Items {
		var pm ProjectMeta
		for _, shoot := range shootList.Items {
			if shoot.Namespace == *project.Spec.Namespace {
				currentShoot := shoot.Name
				if shoot.Status.IsHibernated {
					currentShoot += " (Hibernated)"
				}
				pm.Shoots = append(pm.Shoots, currentShoot)
			}
		}
		pm.Project = project.Name
		projects.Projects = append(projects.Projects, pm)
	}

	if outputFormat == "yaml" {
		y, err := yaml.Marshal(projects)
		if err != nil {
			return err
		}
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.MarshalIndent(projects, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprint(ioStreams.Out, string(j))
	}

	return nil
}

// PrintGardenClusters prints all Garden cluster in the Garden config
func PrintGardenClusters(reader ConfigReader, outFormat string, ioStreams IOStreams) {
	config := reader.ReadConfig(pathGardenConfig)

	var gardens GardenClusters
	for _, garden := range config.GardenClusters {
		var gm GardenClusterMeta
		gm.Name = garden.Name
		gardens.GardenClusters = append(gardens.GardenClusters, gm)
	}
	if outFormat == "yaml" {
		y, err := yaml.Marshal(gardens)
		checkError(err)
		fmt.Fprint(ioStreams.Out, string(y))
	} else if outFormat == "json" {
		j, err := json.MarshalIndent(gardens, "", "  ")
		checkError(err)
		fmt.Fprint(ioStreams.Out, string(j))
	}
}

// getSeeds returns list of seeds
func getSeeds(clientset gardencoreclientset.Interface) *gardencorev1beta1.SeedList {
	seedList, err := clientset.CoreV1beta1().Seeds().List(metav1.ListOptions{})
	checkError(err)
	return seedList
}

// getProjectsWithShootsForSeed
func getProjectsWithShootsForSeed(ioStreams IOStreams) {
	var target Target
	ReadTarget(pathTarget, &target)
	var projects Projects
	gardenClientset, err := target.GardenerClient()
	checkError(err)
	projectList, err := gardenClientset.CoreV1beta1().Projects().List(metav1.ListOptions{})
	checkError(err)
	shootList, err := gardenClientset.CoreV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	for _, project := range projectList.Items {
		var pm ProjectMeta
		for _, shoot := range shootList.Items {
			if shoot.Namespace == *project.Spec.Namespace && shoot.Spec.SeedName != nil && target.Target[1].Name == *shoot.Spec.SeedName {
				currentShoot := shoot.Name
				if shoot.Status.IsHibernated {
					currentShoot += " (Hibernated)"
				}
				pm.Shoots = append(pm.Shoots, currentShoot)
			}
		}
		if len(pm.Shoots) > 0 {
			pm.Project = project.Name
			projects.Projects = append(projects.Projects, pm)
		}
	}
	if len(projects.Projects) == 0 {
		fmt.Printf("No shoots for %s\n", target.Target[1].Name)
		os.Exit(2)
	}
	if outputFormat == "yaml" {
		y, err := yaml.Marshal(projects)
		checkError(err)
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.MarshalIndent(projects, "", "  ")
		checkError(err)
		fmt.Fprint(ioStreams.Out, string(j))
	}
}

// getIssues lists broken shoot clusters
func getIssues(target TargetInterface, ioStreams IOStreams) {
	gardenClientset, err := target.GardenerClient()
	checkError(err)
	shootList, err := gardenClientset.CoreV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	var issues Issues
	for _, item := range shootList.Items {
		var im IssuesMeta
		var statusMeta StatusMeta
		var lastOperationMeta LastOperationMeta
		state := ""
		healthy := true
		hasIssue := true
		unknown := true
		if item.Status.LastOperation != nil {
			if len(item.Status.Conditions) > 0 {
				for _, condition := range item.Status.Conditions {
					if condition.Status == "True" {
						unknown = false
					}
					if condition.Status == "False" {
						unknown = false
						healthy = false
					}
				}
			}
			if (item.Status.LastOperation.Progress == 100) && (item.Status.LastOperation.State == "Succeeded") && ((item.Status.LastOperation.Type == "Create") || (item.Status.LastOperation.Type == "Reconcile")) {
				hasIssue = false
			}
			if unknown {
				state = "Unknown"
			} else if healthy {
				state = "Ready"
			} else {
				state = "NotReady"
			}
			if !hasIssue && !healthy {
				hasIssue = true
			}
			if hasIssue {
				if item.Status.LastOperation != nil {
					lastOperationMeta.Description = item.Status.LastOperation.Description
					lastOperationMeta.LastUpdateTime = item.Status.LastOperation.LastUpdateTime.String()
					lastOperationMeta.Progress = item.Status.LastOperation.Progress
					lastOperationMeta.State = string(item.Status.LastOperation.State)
					lastOperationMeta.Type = string(item.Status.LastOperation.Type)
				}
				if item.Status.LastErrors != nil {
					for _, lastError := range item.Status.LastErrors {
						statusMeta.LastErrors = append(statusMeta.LastErrors, lastError.GetDescription())
					}
				}
				statusMeta.LastOperation = lastOperationMeta
				im.Health = state
				im.Project = item.Namespace
				im.Seed = *item.Spec.SeedName
				im.Shoot = item.Name
				im.Status = statusMeta
				issues.Issues = append(issues.Issues, im)
			}
		} else {
			lastOperationMeta.Description = "Not processed (!)"
			statusMeta.LastOperation = lastOperationMeta
			im.Status = statusMeta
			im.Project = item.Namespace
			im.Seed = *item.Spec.SeedName
			im.Shoot = item.Name
			im.Health = "None"
			issues.Issues = append(issues.Issues, im)
		}
	}
	if outputFormat == "yaml" {
		y, err := yaml.Marshal(issues)
		checkError(err)
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.MarshalIndent(issues, "", "  ")
		checkError(err)
		fmt.Fprint(ioStreams.Out, string(j))
	}
}

// getSeedsWithShootsForProject
func getSeedsWithShootsForProject(ioStreams IOStreams) {
	var target Target
	ReadTarget(pathTarget, &target)

	gardenClientset, err := target.GardenerClient()
	checkError(err)

	projectName := target.Target[1].Name
	project, err := gardenClientset.CoreV1beta1().Projects().Get(projectName, metav1.GetOptions{})
	checkError(err)

	projectNamespace := project.Spec.Namespace
	shootList, err := gardenClientset.CoreV1beta1().Shoots(*projectNamespace).List(metav1.ListOptions{})
	checkError(err)

	var seeds, seedsFiltered Seeds
	seedList := getSeeds(gardenClientset)
	for _, seed := range seedList.Items {
		var sm SeedMeta
		sm.Seed = seed.Name
		seeds.Seeds = append(seeds.Seeds, sm)
	}
	for _, shoot := range shootList.Items {
		for index, seed := range seeds.Seeds {
			if seed.Seed == *shoot.Spec.SeedName {
				currentShoot := shoot.Name
				if shoot.Status.IsHibernated {
					currentShoot += " (Hibernated)"
				}
				seeds.Seeds[index].Shoots = append(seeds.Seeds[index].Shoots, currentShoot)
			}
		}
	}
	for _, seed := range seeds.Seeds {
		if len(seed.Shoots) > 0 {
			seedsFiltered.Seeds = append(seedsFiltered.Seeds, seed)
		}
	}
	if len(seedsFiltered.Seeds) == 0 {
		fmt.Printf("Project %s is empty\n", target.Target[1].Name)
		os.Exit(2)
	}
	if outputFormat == "yaml" {
		y, err := yaml.Marshal(seedsFiltered)
		checkError(err)
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.MarshalIndent(seedsFiltered, "", "  ")
		checkError(err)
		fmt.Fprint(ioStreams.Out, string(j))
	}
}

//getNamespaces get all namespaces based on current kubeconfig
func getNamespaces(ioStreams IOStreams) {
	currentConfig := getKubeConfigOfCurrentTarget()

	out, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+currentConfig+"; kubectl get ns")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(out))
}
