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
	"fmt"
	"io/ioutil"
	"os"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download tf (infra|internal-dns|external-dns|ingress|backup)",
	Short: "Download terraform configuration/state for local execution for the targeted shoot",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 || !(args[1] == "infra" || args[1] == "internal-dns" || args[1] == "external-dns" || args[1] == "ingress" || args[1] == "backup") {
			fmt.Println("Command must be in the format: download tf + (infra|internal-dns|external-dns|ingress|backup)")
			os.Exit(2)
		}
		switch args[0] {
		case "tf":
			downloadTerraformFiles(args[1])
			checkError(err)
		default:
			fmt.Println("Command must be in the format: download tf + (infra|internal-dns|external-dns|ingress|backup)")
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
	ReadTarget(pathTarget, &target)
	Client, err = clientToTarget("garden")
	if len(target.Target) < 3 && (option == "infra" || option == "internal-dns" || option == "external-dns" || option == "ingress" || option == "backup") {
		fmt.Println("No Shoot targeted")
		os.Exit(2)
	} else if len(target.Target) < 3 {
		fmt.Println("Command must be in the format: download tf + (infra|internal-dns|external-dns|ingress|backup)")
		os.Exit(2)
	} else {
		Client, err = clientToTarget("garden")
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
		checkError(err)
		shootList, err := gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		var ind int
		for index, shoot := range shootList.Items {
			if shoot.Name == target.Target[2].Name && (shoot.Namespace == target.Target[1].Name || *shoot.Spec.Cloud.Seed == target.Target[1].Name) {
				ind = index
				break
			}
		}
		namespace = shootList.Items[ind].Status.TechnicalID
		fmt.Println(namespace)
		seed, err := gardenClientset.GardenV1beta1().Seeds().Get(*shootList.Items[ind].Spec.Cloud.Seed, metav1.GetOptions{})
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
		Client, err = k8s.NewForConfig(config)
		checkError(err)
	}
	cmTfConfig, err := Client.CoreV1().ConfigMaps(namespace).Get((target.Target[2].Name + "." + option + ".tf-config"), metav1.GetOptions{})
	checkError(err)
	cmTfState, err := Client.CoreV1().ConfigMaps(namespace).Get((target.Target[2].Name + "." + option + ".tf-state"), metav1.GetOptions{})
	checkError(err)
	secret, err := Client.CoreV1().Secrets(namespace).Get((target.Target[2].Name + "." + option + ".tf-vars"), metav1.GetOptions{})
	checkError(err)
	pathTerraform := ""
	if target.Target[1].Kind == "project" {
		CreateDir(pathGardenHome+"/cache/projects/"+target.Target[1].Name+"/"+target.Target[2].Name+"/terraform", 0751)
		pathTerraform = "cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/terraform"

	} else if target.Target[1].Kind == "seed" {
		CreateDir(pathGardenHome+"/cache/seeds/"+target.Target[1].Name+"/"+target.Target[2].Name+"/terraform", 0751)
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
