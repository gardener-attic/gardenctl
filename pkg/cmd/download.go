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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencoreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// NewDownloadCmd returns a new download command.
func NewDownloadCmd(targetReader TargetReader) *cobra.Command {
	return &cobra.Command{
		Use:   "download tf + (infra|internal-dns|external-dns|ingress|backup)\n  gardenctl download logs vpn\n ",
		Short: "Download terraform configuration/state for local execution for the targeted shoot or log files",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 || !(args[1] == "infra" || args[1] == "internal-dns" || args[1] == "external-dns" || args[1] == "ingress" || args[1] == "backup" || args[1] == "vpn") {
				return errors.New("Command must be in the format:\n  download tf + (infra|internal-dns|external-dns|ingress|backup)\n  download logs vpn")
			}
			switch args[0] {
			case "tf":
				path := downloadTerraformFiles(args[1], targetReader)
				fmt.Println("Downloaded to " + path)
			case "logs":
				downloadLogs(args[1], targetReader)
			default:
				fmt.Println("Command must be in the format:\n  download tf + (infra|internal-dns|external-dns|ingress|backup)\n  download logs vpn")
			}
			return nil
		},
		ValidArgs: []string{"tf"},
	}
}

// downloadTerraformFiles downloads the corresponding tf file
func downloadTerraformFiles(option string, targetReader TargetReader) string {
	namespace := ""
	target := targetReader.ReadTarget(pathTarget)
	// return path allow non operator download key file
	if getRole() == "user" {
		if (len(target.Stack()) < 3) || (len(target.Stack()) == 3 && target.Stack()[2].Kind == "namespace") {
			fmt.Println("No Shoot targeted")
			os.Exit(2)
		} else if target.Stack()[1].Kind == "seed" {
			fmt.Println("Currently target stack is garden/seed/shoot which doesn't allow user role to access shoot via seed")
			fmt.Println("Please target shoot via `gardenctl target shoot` directly or target shoot via project")
			os.Exit(2)
		}
		gardenName := target.Stack()[0].Name
		projectName := target.Stack()[1].Name
		shootName := target.Stack()[2].Name
		pathTerraform := filepath.Join(pathGardenHome, "cache", gardenName, "projects", projectName, shootName)
		return filepath.Join(pathGardenHome, pathTerraform)
	}

	var err error
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenName := target.Stack()[0].Name
	pathSeedCache := filepath.Join("cache", gardenName, "seeds")
	pathProjectCache := filepath.Join("cache", gardenName, "projects")
	if (len(target.Stack()) < 3 && (option == "infra" || option == "internal-dns" || option == "external-dns" || option == "ingress" || option == "backup")) || (len(target.Stack()) == 3 && target.Stack()[2].Kind == "namespace") {
		fmt.Println("No Shoot targeted")
		os.Exit(2)
	} else if len(target.Stack()) < 3 {
		fmt.Println("Command must be in the format:\n download tf + (infra|internal-dns|external-dns|ingress|backup)\n  download logs vpn")
		os.Exit(2)
	} else {
		var err error
		Client, err = clientToTarget("garden")
		checkError(err)
		gardenClientset, err := gardencoreclientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
		checkError(err)
		var shoot *gardencorev1beta1.Shoot
		if target.Stack()[1].Kind == "project" {
			project, err := gardenClientset.CoreV1beta1().Projects().Get(target.Stack()[1].Name, metav1.GetOptions{})
			checkError(err)
			shoot, err = gardenClientset.CoreV1beta1().Shoots(*project.Spec.Namespace).Get(target.Stack()[2].Name, metav1.GetOptions{})
			checkError(err)
		} else {
			shootList, err := gardenClientset.CoreV1beta1().Shoots("").List(metav1.ListOptions{})
			checkError(err)
			for index, s := range shootList.Items {
				if s.Name == target.Stack()[2].Name && *s.Spec.SeedName == target.Stack()[1].Name {
					shoot = &shootList.Items[index]
					break
				}
			}
		}
		namespace = shoot.Status.TechnicalID
		seed, err := gardenClientset.CoreV1beta1().Seeds().Get(*shoot.Spec.SeedName, metav1.GetOptions{})
		checkError(err)
		kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
		checkError(err)
		pathSeed := filepath.Join(pathGardenHome, pathSeedCache, seed.Spec.SecretRef.Name)
		pathToKubeconfig := filepath.Join(pathSeed, "kubeconfig.yaml")
		err = os.MkdirAll(pathSeed, os.ModePerm)
		checkError(err)
		err = ioutil.WriteFile(pathToKubeconfig, kubeSecret.Data["kubeconfig"], 0644)
		checkError(err)
		KUBECONFIG = pathToKubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", pathToKubeconfig)
		checkError(err)
		Client, err = k8s.NewForConfig(config)
		checkError(err)
	}
	cmTfConfig, err := Client.CoreV1().ConfigMaps(namespace).Get((target.Stack()[2].Name + "." + option + ".tf-config"), metav1.GetOptions{})
	checkError(err)
	cmTfState, err := Client.CoreV1().ConfigMaps(namespace).Get((target.Stack()[2].Name + "." + option + ".tf-state"), metav1.GetOptions{})
	checkError(err)
	secret, err := Client.CoreV1().Secrets(namespace).Get((target.Stack()[2].Name + "." + option + ".tf-vars"), metav1.GetOptions{})
	checkError(err)
	pathTerraform := ""
	if target.Stack()[1].Kind == "project" {
		CreateDir(filepath.Join(pathGardenHome, pathProjectCache, target.Stack()[1].Name, target.Stack()[2].Name, "terraform"), 0751)
		pathTerraform = filepath.Join("cache", gardenName, "projects", target.Stack()[1].Name, target.Stack()[2].Name, "terraform")

	} else if target.Stack()[1].Kind == "seed" {
		CreateDir(filepath.Join(pathGardenHome, pathSeedCache, target.Stack()[1].Name, target.Stack()[2].Name, "terraform"), 0751)
		pathTerraform = filepath.Join("cache", gardenName, "seeds", target.Stack()[1].Name, target.Stack()[2].Name, "terraform")
	}
	err = ioutil.WriteFile(filepath.Join(pathGardenHome, pathTerraform, "main.tf"), []byte(cmTfConfig.Data["main.tf"]), 0644)
	checkError(err)
	err = ioutil.WriteFile(filepath.Join(pathGardenHome, pathTerraform, "variables.tf"), []byte(cmTfConfig.Data["variables.tf"]), 0644)
	checkError(err)
	err = ioutil.WriteFile(filepath.Join(pathGardenHome, pathTerraform, "terraform.tfstate"), []byte(cmTfState.Data["terraform.tfstate"]), 0644)
	checkError(err)
	err = ioutil.WriteFile(filepath.Join(pathGardenHome, pathTerraform, "terraform.tfvars"), []byte(secret.Data["terraform.tfvars"]), 0644)
	checkError(err)
	return filepath.Join(pathGardenHome, pathTerraform)
}

func downloadLogs(option string, targetReader TargetReader) {
	dir, err := os.Getwd()
	checkError(err)
	target := targetReader.ReadTarget(pathTarget)
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenName := target.Stack()[0].Name
	pathSeedCache := filepath.Join("cache", gardenName, "seeds")
	gardenClientset, err := gardencoreclientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	shootList, err := gardenClientset.CoreV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	for _, shoot := range shootList.Items {
		seed, err := gardenClientset.CoreV1beta1().Seeds().Get(*shoot.Spec.SeedName, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Could not get seed")
			continue
		}
		kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
		if err != nil {
			fmt.Println("Could not get kubeSecret")
			continue
		}
		pathSeed := filepath.Join(pathGardenHome, pathSeedCache, seed.Spec.SecretRef.Name)
		err = os.MkdirAll(pathSeed, os.ModePerm)
		checkError(err)
		err = ioutil.WriteFile(filepath.Join(pathSeed, "kubeconfig.yaml"), kubeSecret.Data["kubeconfig"], 0644)
		if err != nil {
			fmt.Println("Could not write logs")
			continue
		}
		pathToKubeconfig := filepath.Join(pathGardenHome, pathSeedCache, seed.Spec.SecretRef.Name, "kubeconfig.yaml")
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
		CreateDir(filepath.Join(dir, "seeds", *shoot.Spec.SeedName, shoot.ObjectMeta.GetNamespace(), shoot.Name, "logs", "vpn"), 0751)
		pathLogsSeeds := filepath.Join(dir, "seeds", *shoot.Spec.SeedName, shoot.ObjectMeta.GetNamespace(), shoot.Name, "logs", "vpn")
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "prometheus-0") {
				fmt.Println("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -c "+"vpn-seed"+" -n "+shoot.Status.TechnicalID)
				output, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -c "+"vpn-seed"+" -n "+shoot.Status.TechnicalID)
				if err != nil {
					fmt.Println("Could not execute cmd")
					continue
				}
				err = ioutil.WriteFile(filepath.Join(pathLogsSeeds, "vpn-seed-prometheus"), []byte(output), 0644)
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
				err = ioutil.WriteFile(filepath.Join(pathLogsSeeds, "vpn-seed-", pod.Name), []byte(output), 0644)
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
		pathShootKubeconfig := filepath.Join(pathGardenHome, pathSeedCache, seed.Name, shoot.Name)
		err = os.MkdirAll(pathShootKubeconfig, os.ModePerm)
		checkError(err)
		err = ioutil.WriteFile(filepath.Join(pathShootKubeconfig, "kubeconfig.yaml"), kubeSecretShoot.Data["kubeconfig"], 0644)
		if err != nil {
			fmt.Println("Could not write kubeconfig")
			continue
		}
		pathToKubeconfig = filepath.Join(pathShootKubeconfig, "kubeconfig.yaml")
		KUBECONFIG = pathToKubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", pathToKubeconfig)
		if err != nil {
			fmt.Println("Could not build config")
			continue
		}
		clientShoot, err := k8s.NewForConfig(config)
		checkError(err)
		pods, err = clientShoot.CoreV1().Pods("kube-system").List(metav1.ListOptions{})
		if err != nil {
			fmt.Println("Shoot " + shoot.Name + " has no pods in kube-system namespace")
			continue
		}
		pathLogsShoots := filepath.Join(dir, "seeds", *shoot.Spec.SeedName, shoot.ObjectMeta.GetNamespace(), shoot.Name, "logs", "vpn")
		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, "vpn-shoot-") {
				fmt.Println("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -n "+"kube-system")
				output, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+KUBECONFIG+"; kubectl logs "+pod.Name+" -n "+"kube-system")
				if err != nil {
					fmt.Println("Could not execute cmd")
					continue
				}
				err = ioutil.WriteFile(filepath.Join(pathLogsShoots, pod.Name), []byte(output), 0644)
				if err != nil {
					fmt.Println("Could not write logs")
					continue
				}
			}
		}
	}
}
