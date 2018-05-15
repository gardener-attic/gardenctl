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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
		ReadTarget(pathTarget, &target)
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
		ReadTarget(pathTarget, &target)
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
		ReadTarget(pathTarget, &target)
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
	checkError(err)
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	seed, err := k8sGardenClient.GardenClientset().GardenV1beta1().Seeds().Get(name, metav1.GetOptions{})
	checkError(err)
	kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
	checkError(err)
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
		ReadTarget(pathTarget, &target)
		if len(target.Target) < 3 {
			fmt.Println("No shoot targeted")
			os.Exit(2)
		}
	} else if name != "" {
		ReadTarget(pathTarget, &target)
		if len(target.Target) < 2 {
			fmt.Println("No seed or project targeted")
			os.Exit(2)
		}
	}
	Client, err = clientToTarget("garden")
	checkError(err)
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	var ind int
	var namespace string
	for index, shoot := range shootList.Items {
		if (name == "") && (shoot.Name == target.Target[2].Name) && (shoot.Namespace == target.Target[1].Name || *shoot.Spec.Cloud.Seed == target.Target[1].Name) {
			ind = index
			namespace = strings.Replace("shoot-"+shootList.Items[ind].Namespace+"-"+target.Target[2].Name, "-garden", "", 1)
			break
		}
		if (name != "") && (shoot.Name == name) && (shoot.Namespace == target.Target[1].Name || *shoot.Spec.Cloud.Seed == target.Target[1].Name) {
			ind = index
			namespace = strings.Replace("shoot-"+shootList.Items[ind].Namespace+"-"+name, "-garden", "", 1)
			break
		}
	}
	seed, err := k8sGardenClient.GardenClientset().GardenV1beta1().Seeds().Get(*shootList.Items[ind].Spec.Cloud.Seed, metav1.GetOptions{})
	checkError(err)
	kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
	checkError(err)
	pathSeed := pathSeedCache + "/" + seed.Spec.SecretRef.Name
	os.MkdirAll(pathSeed, os.ModePerm)
	err = ioutil.WriteFile(pathSeed+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
	checkError(err)
	KUBECONFIG = pathSeed + "/kubeconfig.yaml"
	pathToKubeconfig := pathGardenHome + "/cache/seeds" + "/" + seed.Spec.SecretRef.Name + "/" + "kubeconfig.yaml"
	config, err := clientcmd.BuildConfigFromFlags("", pathToKubeconfig)
	checkError(err)
	Client, err := k8s.NewForConfig(config)
	checkError(err)
	kubeSecret, err = Client.CoreV1().Secrets(namespace).Get("kubecfg", metav1.GetOptions{})
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
}

// getTarget prints target stack
func getTarget() {
	var t Target
	ReadTarget(pathTarget, &t)
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
