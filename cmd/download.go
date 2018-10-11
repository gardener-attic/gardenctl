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
	"strings"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download tf + (infra|internal-dns|external-dns|ingress|backup)\n  gardenctl download logs vpn\n ",
	Short: "Download terraform configuration/state for local execution for the targeted shoot or log files",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 || !(args[1] == "infra" || args[1] == "internal-dns" || args[1] == "external-dns" || args[1] == "ingress" || args[1] == "backup" || args[1] == "vpn") {
			fmt.Println("Command must be in the format:\n  download tf + (infra|internal-dns|external-dns|ingress|backup)\n  download logs vpn")
			os.Exit(2)
		}
		switch args[0] {
		case "tf":
			downloadTerraformFiles(args[1])
		case "logs":
			downloadLogs(args[1])
		default:
			fmt.Println("Command must be in the format:\n  download tf + (infra|internal-dns|external-dns|ingress|backup)\n  download logs vpn")
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
		fmt.Println("Command must be in the format:\n  download tf + (infra|internal-dns|external-dns|ingress|backup)\n  download logs vpn")
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

func downloadLogs(option string) {
	dir, err := os.Getwd()
	checkError(err)
	var target Target
	ReadTarget(pathTarget, &target)
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	shootList, err := gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	for _, shoot := range shootList.Items {
		seed, err := gardenClientset.GardenV1beta1().Seeds().Get(*shoot.Spec.Cloud.Seed, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Could not get seed")
			continue
		}
		kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Could not get kubeSecret")
			continue
		}
		pathSeed := pathSeedCache + "/" + seed.Spec.SecretRef.Name
		os.MkdirAll(pathSeed, os.ModePerm)
		err = ioutil.WriteFile(pathSeed+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
		if err != nil {
			fmt.Println("Could not write logs")
			continue
		}
		pathToKubeconfig := pathGardenHome + "/cache/seeds" + "/" + seed.Spec.SecretRef.Name + "/" + "kubeconfig.yaml"
		KUBECONFIG = pathToKubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", pathToKubeconfig)
		if err != nil {
			fmt.Println("Could not build config")
			continue
		}
		ClientSeed, err := k8s.NewForConfig(config)
		if err != nil {
			fmt.Println("Could not get seed client")
			continue
		}
		pods, err := ClientSeed.CoreV1().Pods(shoot.Status.TechnicalID).List(metav1.ListOptions{})
		if err != nil {
			fmt.Println("Shoot " + shoot.Name + " has no pods in " + shoot.Status.TechnicalID + " namespace")
			continue
		}
		CreateDir(dir+"/seeds/"+*shoot.Spec.Cloud.Seed+"/"+shoot.ObjectMeta.GetNamespace()+"/"+shoot.Name+"/logs/vpn", 0751)
		pathLogsSeeds := dir + "/seeds/" + *shoot.Spec.Cloud.Seed + "/" + shoot.ObjectMeta.GetNamespace() + "/" + shoot.Name + "/logs/vpn"
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "prometheus-0") {
				fmt.Println("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -c "+"vpn-seed"+" -n "+shoot.Status.TechnicalID)
				output, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -c "+"vpn-seed"+" -n "+shoot.Status.TechnicalID)
				if err != nil {
					fmt.Println("Could not execute cmd")
					continue
				}
				err = ioutil.WriteFile(pathLogsSeeds+"/vpn-seed-prometheus", []byte(output), 0644)
				if err != nil {
					fmt.Println("Could not write logs")
					continue
				}
			}
			if strings.Contains(pod.Name, "kube-apiserver") {
				fmt.Println("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -c "+"vpn-seed"+" -n "+shoot.Status.TechnicalID)
				output, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -c "+"vpn-seed"+" -n "+shoot.Status.TechnicalID)
				if err != nil {
					fmt.Println("Could not execute cmd")
					continue
				}
				err = ioutil.WriteFile(pathLogsSeeds+"/vpn-seed-"+pod.Name, []byte(output), 0644)
				if err != nil {
					fmt.Println("Could not write logs")
					continue
				}
			}
		}
		kubeSecretShoot, err := ClientSeed.CoreV1().Secrets(shoot.Status.TechnicalID).Get("kubecfg", metav1.GetOptions{})
		if err != nil {
			fmt.Println("Could not get kubeSecret")
			continue
		}
		pathShootKubeconfig := pathShootCache + "/" + seed.Name + "/" + shoot.Name
		os.MkdirAll(pathShootKubeconfig, os.ModePerm)
		err = ioutil.WriteFile(pathShootKubeconfig+"/kubeconfig.yaml", kubeSecretShoot.Data["kubeconfig"], 0644)
		if err != nil {
			fmt.Println("Could not write kubeconfig")
			continue
		}
		pathToKubeconfig = pathShootKubeconfig + "/" + "kubeconfig.yaml"
		KUBECONFIG = pathToKubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", pathToKubeconfig)
		if err != nil {
			fmt.Println("Could not build config")
			continue
		}
		ClientShoot, err := k8s.NewForConfig(config)
		pods, err = ClientShoot.CoreV1().Pods("kube-system").List(metav1.ListOptions{})
		if err != nil {
			fmt.Println("Shoot " + shoot.Name + " has no pods in kube-system namespace")
			continue
		}
		pathLogsShoots := dir + "/seeds/" + *shoot.Spec.Cloud.Seed + "/" + shoot.ObjectMeta.GetNamespace() + "/" + shoot.Name + "/logs/vpn"
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "vpn-shoot-") {
				fmt.Println("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -n "+"kube-system")
				output, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -n "+"kube-system")
				if err != nil {
					fmt.Println("Could not execute cmd")
					continue
				}
				err = ioutil.WriteFile(pathLogsShoots+"/"+pod.Name, []byte(output), 0644)
				if err != nil {
					fmt.Println("Could not write logs")
					continue
				}
			}
		}
	}
}
