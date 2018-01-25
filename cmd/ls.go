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

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getCmd represents the get command
var lsCmd = &cobra.Command{
	Use:   "ls [issues|projects|gardens|seeds|shoots]",
	Short: "List all resource instances, e.g. list of shoots",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			fmt.Println("Command must be in the format: ls [issues|projects|gardens|seeds|shoots]")
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
			if len(target.Target) == 1 {
				tmp := KUBECONFIG
				Client, err = clientToTarget("garden")
				getProjectsWithShoots()
				KUBECONFIG = tmp
			} else if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
				getShoots()
			} else if len(target.Target) == 2 && target.Target[1].Kind == "project" {
				tmp := KUBECONFIG
				Client, err = clientToTarget("garden")
				getShoots()
				KUBECONFIG = tmp
			}
		case "issues":
			Client, err = clientToTarget("garden")
			checkError(err)
			getIssues()
		default:
			fmt.Println("Command must be in the format: ls [issues|projects|gardens|seeds|shoots]")
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
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
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

// getShoots lists all available shoots
func getShoots() {
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	var seeds Seeds
	var projects Projects
	if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
		Client, err = clientToTarget("garden")
		k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
		checkError(err)
		k8sGardenClient.SetGardenClientset(gardenClientset)
		shootList, err := k8sGardenClient.GetGardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		var sm SeedMeta
		sm.Seed = target.Target[1].Name
		for _, item := range shootList.Items {
			if *item.Spec.Cloud.Seed == target.Target[1].Name {
				sm.Shoots = append(sm.Shoots, item.Name)
			}
		}
		seeds.Seeds = append(seeds.Seeds, sm)
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
	} else if len(target.Target) == 2 && target.Target[1].Kind == "project" {
		k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
		checkError(err)
		k8sGardenClient.SetGardenClientset(gardenClientset)
		shootList, err := k8sGardenClient.GetGardenClientset().GardenV1beta1().Shoots(target.Target[1].Name).List(metav1.ListOptions{})
		checkError(err)
		var pm ProjectMeta
		pm.Project = target.Target[1].Name
		for _, item := range shootList.Items {
			if item.Namespace == target.Target[1].Name {
				pm.Shoots = append(pm.Shoots, item.Name)
			}
		}
		projects.Projects = append(projects.Projects, pm)
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
}

// getIssues lists broken shoot clusters
func getIssues() {
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
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
				statusMeta.LastError = item.Status.LastError.Description
				statusMeta.LastOperation = lastOperationMeta
				im.Health = state
				im.Project = item.Namespace
				im.Seed = *item.Spec.Cloud.Seed
				im.Shoot = item.Name
				im.Status = statusMeta
				issues.Issues = append(issues.Issues, im)
			}
		} else {
			lastOperationMeta.Description = "Not processed (!)"
			statusMeta.LastOperation = lastOperationMeta
			im.Status = statusMeta
			im.Project = item.Namespace
			im.Seed = *item.Spec.Cloud.Seed
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
