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

	"github.com/gardener/gardener/pkg/client/kubernetes"
	yaml "gopkg.in/yaml.v2"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download tf (infra|dns|ingress)",
	Short: "Download terraform configuration/state for local execution for the targeted shoot",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 || !(args[1] == "infra" || args[1] == "dns" || args[1] == "ingress") {
			fmt.Println("Command must be in the format: download tf + (infra|dns|ingress)")
			os.Exit(2)
		}
		switch args[0] {
		case "tf":
			downloadTerraformFiles(args[1])
			checkError(err)
		default:
			fmt.Println("Command must be in the format: download tf + (infra|dns|ingress)")
		}
	},
	ValidArgs: []string{"tf"},
}

func init() {
}

// downloadTerraformFiles downloads the corresponding tf file
func downloadTerraformFiles(option string) {
	namespace := ""
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	Client, err = clientToTarget("garden")
	if len(target.Target) < 3 && (option == "infra" || option == "dns" || option == "ingress") {
		fmt.Println("No Shoot targeted")
		os.Exit(2)
	} else if len(target.Target) < 3 {
		fmt.Println("Command must be in the format: download tf + (infra|dns|ingress)")
		os.Exit(2)
	} else if target.Target[1].Kind == "project" {
		namespace = target.Target[1].Name
	} else {
		k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
		checkError(err)
		k8sGardenClient.SetGardenClientset(gardenClientset)
		shootList, err := k8sGardenClient.GetGardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		for _, shoot := range shootList.Items {
			if shoot.Name == target.Target[2].Name && *shoot.Spec.Cloud.Seed == target.Target[1].Name {
				namespace = shoot.Namespace
			}
		}
	}
	cmTfConfig, err := Client.CoreV1().ConfigMaps(namespace).Get((target.Target[2].Name + "." + option + ".tf-config"), metav1.GetOptions{})
	checkError(err)
	cmTfState, err := Client.CoreV1().ConfigMaps(namespace).Get((target.Target[2].Name + "." + option + ".tf-state"), metav1.GetOptions{})
	checkError(err)
	secret, err := Client.CoreV1().Secrets(namespace).Get((target.Target[2].Name + "." + option + ".tf-vars"), metav1.GetOptions{})
	checkError(err)
	pathTerraform := ""
	if target.Target[1].Kind == "project" {
		createDir(pathGardenHome+"/cache/projects/"+target.Target[1].Name+"/"+target.Target[2].Name+"/terraform", 0751)
		pathTerraform = "cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/terraform"

	} else if target.Target[1].Kind == "seed" {
		createDir(pathGardenHome+"/cache/seeds/"+target.Target[1].Name+"/"+target.Target[2].Name+"/terraform", 0751)
		pathTerraform = "cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/terraform"
	}
	err = ioutil.WriteFile(pathGardenHome+"/"+pathTerraform+"/main.tf", []byte(cmTfConfig.Data["main.tf"]), 0644)
	checkError(err)
	err = ioutil.WriteFile(pathGardenHome+"/"+pathTerraform+"/variables.tf", []byte(cmTfConfig.Data["variables.tf"]), 0644)
	checkError(err)
	err = ioutil.WriteFile(pathGardenHome+"/"+pathTerraform+"/terraform.tfstate", []byte(cmTfState.Data["terraform.tfstate"]), 0644)
	checkError(err)
	err = ioutil.WriteFile(pathGardenHome+"/"+pathTerraform+"/terraform.tfvars", []byte(secret.Data["terraform.tfvars"]), 0644)
	checkError(err)
	fmt.Println("Downloaded to " + pathGardenHome + "/" + pathTerraform)
}
