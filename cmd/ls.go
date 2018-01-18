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

	clientset "github.com/gardener/gardenctl/pkg/client/garden/clientset/versioned"
	"github.com/gardener/gardenctl/pkg/client/kubernetes"
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
			fmt.Printf("seeds:\n")
			for _, seed := range getSeeds() {
				fmt.Printf("- seed: %s\n", seed)
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

//getProjects lists the name of existing projects
func getProjects() {
	projectLabel := "garden.sapcloud.io/role=project"
	projectList, err := Client.CoreV1().Namespaces().List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s", projectLabel),
	})
	checkError(err)
	for _, project := range projectList.Items {
		fmt.Printf("%s\n", project.Name)
	}
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
	fmt.Printf("projects:\n")
	checkError(err)
	for _, project := range projectList.Items {
		fmt.Printf("- project: %s\n", project.Name)
		fmt.Printf("  shoots:\n")
		for _, item := range shootList.Items {
			if item.Namespace == project.Name {
				fmt.Printf("  - %s\n", item.Name)
			}
		}
	}
}

// getGardens lists all garden cluster in config
func getGardens() {
	var gardenClusters GardenClusters
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenClusters)
	checkError(err)
	fmt.Printf("gardens:\n")
	for _, garden := range gardenClusters.GardenClusters {
		fmt.Printf("- garden: %s\n", garden.Name)
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
	if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
		Client, err = clientToTarget("garden")
		k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
		checkError(err)
		k8sGardenClient.SetGardenClientset(gardenClientset)
		shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots("").List(metav1.ListOptions{})
		fmt.Printf("seeds:\n")
		fmt.Printf("- seed: %s\n", target.Target[1].Name)
		fmt.Printf("  shoots:\n")
		for _, item := range shootList.Items {
			if item.Spec.SeedName == target.Target[1].Name {
				fmt.Printf("  - %s\n", item.Name)
			}
		}
	} else if len(target.Target) == 2 && target.Target[1].Kind == "project" {
		k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
		checkError(err)
		k8sGardenClient.SetGardenClientset(gardenClientset)
		shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots(target.Target[1].Name).List(metav1.ListOptions{})
		checkError(err)
		fmt.Printf("projects:\n")
		fmt.Printf("- project: %s\n", target.Target[1].Name)
		fmt.Printf("  shoots:\n")
		for _, item := range shootList.Items {
			fmt.Printf("  - %s\n", item.Name)
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
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots("").List(metav1.ListOptions{})
	fmt.Printf("issues:\n")
	for _, item := range shootList.Items {
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
				fmt.Printf("- project: %s\n", item.Namespace)
				fmt.Printf("  seed: %s\n", item.Spec.SeedName)
				fmt.Printf("  shoot: %s\n", item.Name)
				fmt.Printf("  health: %s\n", state)
				fmt.Printf("  status: \n")
				fmt.Printf("    lastError: %s\n", item.Status.LastError)
				fmt.Printf("    lastOperation:\n")
				fmt.Printf("      description: %s\n", item.Status.LastOperation.Description)
				fmt.Printf("      lastUpdateTime: %s\n", item.Status.LastOperation.LastUpdateTime)
				fmt.Printf("      progress: %d\n", item.Status.LastOperation.Progress)
				fmt.Printf("      state: %s\n", item.Status.LastOperation.State)
				fmt.Printf("      type: %s\n", item.Status.LastOperation.Type)
			}
		} else {
			fmt.Printf("- project: %s\n", item.Namespace)
			fmt.Printf("  seed: %s\n", item.Spec.SeedName)
			fmt.Printf("  shoot: %s\n", item.Name)
			fmt.Printf("  health: None\n")
			fmt.Printf("  status: \n")
			fmt.Printf("    lastOperation: Not processed (!)\n")
		}
	}
}
