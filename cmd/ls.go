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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	clientset "github.com/gardener/gardenctl/pkg/client/garden/clientset/versioned"
	"github.com/gardener/gardenctl/pkg/client/kubernetes"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCmd represents the get command
var lsCmd = &cobra.Command{
	Use:   "ls [gardens|projects|seeds|shoots|issues]",
	Short: "List all resource instances, e.g. list of shoots|issues",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			fmt.Println("Command must be in the format: ls [gardens|projects|seeds|shoots|issues]")
			os.Exit(2)
		}
		var t Target
		targetFile, err := ioutil.ReadFile(pathTarget)
		checkError(err)
		err = yaml.Unmarshal(targetFile, &t)
		checkError(err)
		if (len(t.Target) == 0) && args[0] != "gardens" {
			fmt.Println("Target stack is empty")
			os.Exit(2)
		}
		switch args[0] {
		case "projects":
			tmp := KUBECONFIG
			Client, err = clientToTarget("garden")
			checkError(err)
			getProjectsWithShoots()
			KUBECONFIG = tmp
		case "gardens":
			getGardens()
		case "seeds":
			Client, err = clientToTarget("garden")
			checkError(err)
			var seeds Seeds
			for _, seed := range getSeeds() {
				var sm SeedMeta
				sm.Seed = seed
				seeds.Seeds = append(seeds.Seeds, sm)
			}
			if outputFormat == "yaml" {
				y, err := yaml.Marshal(seeds)
				checkError(err)
				os.Stdout.Write(y)
			} else if outputFormat == "json" {
				j, err := json.Marshal(seeds)
				checkError(err)
				var out bytes.Buffer
				json.Indent(&out, j, "", "  ")
				out.WriteTo(os.Stdout)
			}
		case "shoots":
			var target Target
			targetFile, err := ioutil.ReadFile(pathTarget)
			checkError(err)
			err = yaml.Unmarshal(targetFile, &target)
			checkError(err)
			tmp := KUBECONFIG
			Client, err = clientToTarget("garden")
			if len(target.Target) == 1 {
				getProjectsWithShoots()
			} else if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
				getProjectsWithShootsForSeed()
			} else if len(target.Target) == 2 && target.Target[1].Kind == "project" {
				getSeedsWithShootsForProject()
			}
			KUBECONFIG = tmp
		case "issues":
			Client, err = clientToTarget("garden")
			checkError(err)
			getIssues()
		default:
			fmt.Println("Command must be in the format: ls [gardens|projects|seeds|shoots|issues]")
		}
	},
	ValidArgs: []string{"issues", "projects", "gardens", "seeds", "shoots"},
}

func init() {
}

// getProjectsWithShoots lists list of projects with shoots
func getProjectsWithShoots() {
	projectLabel := "garden.sapcloud.io/role=project"
	projectList, err := Client.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s", projectLabel),
	})
	checkError(err)
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots("").List(metav1.ListOptions{})
	var projects Projects
	for _, project := range projectList.Items {
		var pm ProjectMeta
		for _, item := range shootList.Items {
			if item.Namespace == project.Name {
				pm.Shoots = append(pm.Shoots, item.Name)
			}
		}
		pm.Project = project.Name
		projects.Projects = append(projects.Projects, pm)
	}
	if outputFormat == "yaml" {
		y, err := yaml.Marshal(projects)
		checkError(err)
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.Marshal(projects)
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, j, "", "  ")
		out.WriteTo(os.Stdout)
	}
}

// getGardens lists all garden cluster in config
func getGardens() {
	var gardenClusters GardenClusters
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenClusters)
	checkError(err)
	var gardens GardenClusters
	for _, garden := range gardenClusters.GardenClusters {
		var gm GardenClusterMeta
		gm.Name = garden.Name
		gardens.GardenClusters = append(gardens.GardenClusters, gm)
	}
	if outputFormat == "yaml" {
		y, err := yaml.Marshal(gardens)
		checkError(err)
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.Marshal(gardens)
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, j, "", "  ")
		out.WriteTo(os.Stdout)
	}
}

// getSeeds returns the name of existing seeds
func getSeeds() (s []string) {
	var seeds []string
	secrets, err := Client.CoreV1().Secrets("garden").List(metav1.ListOptions{})
	checkError(err)
	for _, secret := range secrets.Items {
		if strings.HasPrefix(secret.Name, "seed-") {
			seeds = append(seeds, secret.Name)
		}
	}
	return seeds
}

// getProjectsWithShootsForSeed
func getProjectsWithShootsForSeed() {
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	var projects Projects
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	projectLabel := "garden.sapcloud.io/role=project"
	projectList, err := Client.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s", projectLabel),
	})
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	for _, project := range projectList.Items {
		var pm ProjectMeta
		for _, item := range shootList.Items {
			if item.Namespace == project.Name && target.Target[1].Name == item.Spec.SeedName {
				pm.Shoots = append(pm.Shoots, item.Name)
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
		j, err := json.Marshal(projects)
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, j, "", "  ")
		out.WriteTo(os.Stdout)
	}
}

// getIssues lists broken shoot clusters
func getIssues() {
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots("").List(metav1.ListOptions{})
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
				lastOperationMeta.Description = item.Status.LastOperation.Description
				lastOperationMeta.LastUpdateTime = fmt.Sprintf("%s", item.Status.LastOperation.LastUpdateTime)
				lastOperationMeta.Progress = item.Status.LastOperation.Progress
				lastOperationMeta.State = string(item.Status.LastOperation.State)
				lastOperationMeta.Type = string(item.Status.LastOperation.Type)
				statusMeta.LastError = item.Status.LastError
				statusMeta.LastOperation = lastOperationMeta
				im.Health = state
				im.Project = item.Namespace
				im.Seed = item.Spec.SeedName
				im.Shoot = item.Name
				im.Status = statusMeta
				issues.Issues = append(issues.Issues, im)
			}
		} else {
			lastOperationMeta.Description = "Not processed (!)"
			statusMeta.LastOperation = lastOperationMeta
			im.Status = statusMeta
			im.Project = item.Namespace
			im.Seed = item.Spec.SeedName
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
		j, err := json.Marshal(issues)
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, j, "", "  ")
		out.WriteTo(os.Stdout)
	}

}

// getSeedsWithShootsForProject
func getSeedsWithShootsForProject() {
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots(target.Target[1].Name).List(metav1.ListOptions{})
	var seeds, seedsFiltered Seeds
	for _, seed := range getSeeds() {
		var sm SeedMeta
		sm.Seed = seed
		seeds.Seeds = append(seeds.Seeds, sm)
	}
	for _, shoot := range shootList.Items {
		for index, seed := range seeds.Seeds {
			if seed.Seed == shoot.Spec.SeedName {
				seeds.Seeds[index].Shoots = append(seeds.Seeds[index].Shoots, shoot.Name)
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
		j, err := json.Marshal(seedsFiltered)
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, j, "", "  ")
		out.WriteTo(os.Stdout)
	}
}
