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
	yaml2 "github.com/ghodss/yaml"
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
			output := ""
			output += fmt.Sprintf("seeds:\n")
			for _, seed := range getSeeds() {
				output += fmt.Sprintf("- seed: %s\n", seed)
			}
			if outputFormat == "yaml" {
				fmt.Println(output)
			} else if outputFormat == "json" {
				y, err := yaml2.YAMLToJSON([]byte(output))
				checkError(err)
				var out bytes.Buffer
				json.Indent(&out, y, "", "  ")
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
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots("").List(metav1.ListOptions{})
	output := ""
	output += fmt.Sprintf("projects:\n")
	checkError(err)
	for _, project := range projectList.Items {
		output += fmt.Sprintf("- project: %s\n", project.Name)
		output += fmt.Sprintf("  shoots:\n")
		for _, item := range shootList.Items {
			if item.Namespace == project.Name {
				output += fmt.Sprintf("  - %s\n", item.Name)
			}
		}
	}
	if outputFormat == "yaml" {
		fmt.Println(output)
	} else if outputFormat == "json" {
		y, err := yaml2.YAMLToJSON([]byte(output))
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, y, "", "  ")
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
	output := ""
	output += fmt.Sprintf("gardens:\n")
	for _, garden := range gardenClusters.GardenClusters {
		output += fmt.Sprintf("- garden: %s\n", garden.Name)
	}
	if outputFormat == "yaml" {
		fmt.Println(output)
	} else if outputFormat == "json" {
		y, err := yaml2.YAMLToJSON([]byte(output))
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, y, "", "  ")
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
	output := ""
	if len(target.Target) == 2 && target.Target[1].Kind == "seed" {
		Client, err = clientToTarget("garden")
		k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
		checkError(err)
		k8sGardenClient.SetGardenClientset(gardenClientset)
		shootList, err := k8sGardenClient.GetGardenClientset().GardenV1().Shoots("").List(metav1.ListOptions{})
		output += fmt.Sprintf("seeds:\n")
		output += fmt.Sprintf("- seed: %s\n", target.Target[1].Name)
		output += fmt.Sprintf("  shoots:\n")
		for _, item := range shootList.Items {
			if item.Spec.SeedName == target.Target[1].Name {
				output += fmt.Sprintf("  - %s\n", item.Name)
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
		output += fmt.Sprintf("projects:\n")
		output += fmt.Sprintf("- project: %s\n", target.Target[1].Name)
		output += fmt.Sprintf("  shoots:\n")
		for _, item := range shootList.Items {
			output += fmt.Sprintf("  - %s\n", item.Name)
		}
	}
	if outputFormat == "yaml" {
		fmt.Println(output)
	} else if outputFormat == "json" {
		y, err := yaml2.YAMLToJSON([]byte(output))
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, y, "", "  ")
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
	output := ""
	output += fmt.Sprintf("issues:\n")
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
				output += fmt.Sprintf("- project: %s\n", item.Namespace)
				output += fmt.Sprintf("  seed: %s\n", item.Spec.SeedName)
				output += fmt.Sprintf("  shoot: %s\n", item.Name)
				output += fmt.Sprintf("  health: %s\n", state)
				output += fmt.Sprintf("  status: \n")
				output += fmt.Sprintf("    lastError: \"%s\"\n", strings.Replace(strings.Replace(strings.Replace(item.Status.LastError, ":", " ", -1), "\n", " ", -1), "\"", " ", -1))
				output += fmt.Sprintf("    lastOperation: \n")
				output += fmt.Sprintf("      description: \"%s\"\n", strings.Replace(strings.Replace(strings.Replace(item.Status.LastOperation.Description, ":", " ", -1), "\n", " ", -1), "\"", " ", -1))
				output += fmt.Sprintf("      lastUpdateTime: \"%s\"\n", item.Status.LastOperation.LastUpdateTime)
				output += fmt.Sprintf("      progress: %d\n", item.Status.LastOperation.Progress)
				output += fmt.Sprintf("      state: %s\n", item.Status.LastOperation.State)
				output += fmt.Sprintf("      type: %s\n", item.Status.LastOperation.Type)
			}
		} else {
			output += fmt.Sprintf("- project: %s\n", item.Namespace)
			output += fmt.Sprintf("  seed: %s\n", item.Spec.SeedName)
			output += fmt.Sprintf("  shoot: %s\n", item.Name)
			output += fmt.Sprintf("  health: None\n")
			output += fmt.Sprintf("  status: \n")
			output += fmt.Sprintf("    lastOperation: Not processed (!)\n")
		}
	}
	if outputFormat == "yaml" {
		fmt.Println(output)
	} else if outputFormat == "json" {
		y, err := yaml2.YAMLToJSON([]byte(output))
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, y, "", "  ")
		out.WriteTo(os.Stdout)
	}
}
