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
	"path/filepath"
	"strings"

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	yaml2 "github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get [(garden|project|seed|shoot|target) <name>]",
	Short: "Get single resource instance or target stack, e.g. CRD of a shoot (default: current target)\n",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			fmt.Println("Command must be in the format: get [(garden|project|seed|shoot|target) <name>]")
			os.Exit(2)
		}
		switch args[0] {
		case "project":
			if len(args) == 1 {
				getProject("")
			} else if len(args) == 2 {
				getProject(args[1])
			}
			tmp := KUBECONFIG
			Client, err = clientToTarget("garden")
			checkError(err)
			KUBECONFIG = tmp
		case "garden":
			if len(args) == 1 {
				getGarden("")
			} else if len(args) == 2 {
				getGarden(args[1])
			}
		case "seed":
			if len(args) == 1 {
				getSeed("")
			} else if len(args) == 2 {
				getSeed(args[1])
			}
		case "shoot":
			if len(args) == 1 {
				getShoot("")
			} else if len(args) == 2 {
				getShoot(args[1])
			}
		case "target":
			getTarget()
		default:
			fmt.Println("Command must be in the format: get [project|garden|seed|shoot|target] + <NAME>")
		}
	},
	ValidArgs: []string{"project", "garden", "seed", "shoot", "target"},
}

func init() {
}

// getProject lists
func getProject(name string) {
	var target Target
	if name == "" {
		targetFile, err := ioutil.ReadFile(pathTarget)
		checkError(err)
		err = yaml.Unmarshal(targetFile, &target)
		checkError(err)
		if len(target.Target) < 2 {
			fmt.Println("No project targeted")
			os.Exit(2)
		} else if len(target.Target) > 1 && target.Target[1].Kind == "project" {
			name = target.Target[1].Name
		} else if len(target.Target) > 1 && target.Target[1].Kind == "seed" {
			fmt.Println("Seed targeted, project expected")
		}
	}
	Client, err = clientToTarget("garden")
	checkError(err)
	namespace, err := Client.CoreV1().Namespaces().Get(name, metav1.GetOptions{})
	checkError(err)
	if outputFormat == "yaml" {
		j, err := json.Marshal(namespace)
		checkError(err)
		y, err := yaml2.JSONToYAML(j)
		checkError(err)
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.Marshal(namespace)
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, j, "", "  ")
		out.WriteTo(os.Stdout)
	}
}

// getGarden lists kubeconfig of garden cluster
func getGarden(name string) {
	var target Target
	if name == "" {
		targetFile, err := ioutil.ReadFile(pathTarget)
		checkError(err)
		err = yaml.Unmarshal(targetFile, &target)
		checkError(err)
		if len(target.Target) > 0 {
			name = target.Target[0].Name
		} else {
			fmt.Printf("No garden targeted\n")
			os.Exit(2)
		}
	}
	var gardenClusters GardenClusters
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenClusters)
	checkError(err)
	match := false
	for index, garden := range gardenClusters.GardenClusters {
		if garden.Name == name {
			pathToKubeconfig := gardenClusters.GardenClusters[index].KubeConfig
			if strings.Contains(pathToKubeconfig, "~") {
				pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(pathToKubeconfig, "~", "", 1)))
			}
			kubeconfig, err := ioutil.ReadFile(pathToKubeconfig)
			checkError(err)
			if outputFormat == "yaml" {
				fmt.Printf("%s", kubeconfig)
			} else if outputFormat == "json" {
				y, err := yaml2.YAMLToJSON([]byte(kubeconfig))
				checkError(err)
				var out bytes.Buffer
				json.Indent(&out, y, "", "  ")
				out.WriteTo(os.Stdout)
			}
			match = true
		}
	}
	if !match {
		fmt.Printf("No garden cluster found for %s\n", name)
	}
}

// getSeed lists kubeconfig of seed cluster
func getSeed(name string) {
	var target Target
	if name == "" {
		targetFile, err := ioutil.ReadFile(pathTarget)
		checkError(err)
		err = yaml.Unmarshal(targetFile, &target)
		checkError(err)
		if len(target.Target) > 1 && target.Target[1].Kind == "seed" {
			name = target.Target[1].Name
		} else if len(target.Target) > 1 && target.Target[1].Kind == "project" && len(target.Target) == 3 {
			name = getSeedForProject(target.Target[2].Name)
		} else {
			fmt.Println("No seed targeted or shoot targeted")
			os.Exit(2)
		}
	}
	Client, err = clientToTarget("garden")
	kubeSecret, err := Client.CoreV1().Secrets("garden").Get(name, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Seed not found")
		os.Exit(2)
	}
	if outputFormat == "yaml" {
		fmt.Printf("%s\n", kubeSecret.Data["kubeconfig"])
	} else if outputFormat == "json" {
		y, err := yaml2.YAMLToJSON([]byte(kubeSecret.Data["kubeconfig"]))
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, y, "", "  ")
		out.WriteTo(os.Stdout)
	}

}

// getShoot lists kubeconfig of shoot
func getShoot(name string) {
	var target Target
	if name == "" {
		targetFile, err := ioutil.ReadFile(pathTarget)
		checkError(err)
		err = yaml.Unmarshal(targetFile, &target)
		checkError(err)
		if len(target.Target) > 2 {
			name = target.Target[2].Name
		} else {
			fmt.Println("No shoot targeted")
			os.Exit(2)
		}
	}
	Client, err = clientToTarget("garden")
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	var matchedShoots []v1beta1.Shoot
	for _, item := range shootList.Items {
		if item.Name == name {
			matchedShoots = append(matchedShoots, item)
		}
	}
	if len(matchedShoots) < 1 {
		fmt.Println("Shoot not found")
	} else if len(matchedShoots) == 1 {
		kubeSecret, err := Client.CoreV1().Secrets("garden").Get(*matchedShoots[0].Spec.Cloud.Seed, metav1.GetOptions{})
		checkError(err)
		pathSeed := pathSeedCache + "/" + *matchedShoots[0].Spec.Cloud.Seed
		os.MkdirAll(pathSeed, os.ModePerm)
		err = ioutil.WriteFile(pathSeed+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
		checkError(err)
		KUBECONFIG = pathSeed + "/kubeconfig.yaml"
		namespace := "shoot-" + matchedShoots[0].Namespace + "-" + matchedShoots[0].Name
		pathToKubeconfig := pathGardenHome + "/cache/seeds" + "/" + *matchedShoots[0].Spec.Cloud.Seed + "/" + "kubeconfig.yaml"
		config, err := clientcmd.BuildConfigFromFlags("", pathToKubeconfig)
		checkError(err)
		client, err := k8s.NewForConfig(config)
		checkError(err)
		kubeSecret, err = client.CoreV1().Secrets(namespace).Get("kubecfg", metav1.GetOptions{})
		checkError(err)
		if outputFormat == "yaml" {
			fmt.Printf("%s\n", kubeSecret.Data["kubeconfig"])
		} else if outputFormat == "json" {
			y, err := yaml2.YAMLToJSON([]byte(kubeSecret.Data["kubeconfig"]))
			checkError(err)
			var out bytes.Buffer
			json.Indent(&out, y, "", "  ")
			out.WriteTo(os.Stdout)
		}
	} else if len(matchedShoots) > 1 {
		fmt.Println("Multiple matches, target a seed or project first")
	}
}

// getTarget prints target stack
func getTarget() {
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	var t Target
	yaml.Unmarshal(targetFile, &t)
	if len(t.Target) == 0 {
		fmt.Println("Target stack is empty")
		os.Exit(2)
	} else if outputFormat == "yaml" {
		y, err := yaml.Marshal(t)
		checkError(err)
		os.Stdout.Write(y)
	} else if outputFormat == "json" {
		j, err := json.Marshal(t)
		checkError(err)
		var out bytes.Buffer
		json.Indent(&out, j, "", "  ")
		out.WriteTo(os.Stdout)
	}
}
