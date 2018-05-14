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
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"

	yaml "gopkg.in/yaml.v2"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetGardenClusters sets GardenCluster struct
func GetGardenClusters(pathGardenConfig string, gardenClusters *GardenClusters) {
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenClusters)
	if err != nil {
		fmt.Println("Invalid gardenctl configuration")
		os.Exit(2)
	}
}

// GetGardenClusterKubeConfigFromConfig return kubeconfig of garden cluster if exists
func GetGardenClusterKubeConfigFromConfig(pathGardenConfig, pathTarget string) {
	var gardenClusters GardenClusters
	var target Target
	i, err := os.Stat(pathTarget)
	checkError(err)
	if i.Size() == 0 {
		// if no garden cluster is selected take the first as default cluster
		i, err := os.Stat(pathGardenConfig)
		if i.Size() == 0 {
			fmt.Println("Please provide a gardenctl configuration before usage")
			return
		}
		GetGardenClusters(pathGardenConfig, &gardenClusters)
		target.Target = []TargetMeta{{"garden", gardenClusters.GardenClusters[0].Name}}
		file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
		checkError(err)
		defer file.Close()
		content, err := yaml.Marshal(target)
		checkError(err)
		file.Write(content)
	}
}

// clientToTarget returns the client to target e.g. garden, seed
func clientToTarget(target string) (*k8s.Clientset, error) {
	switch target {
	case "garden":
		KUBECONFIG = getKubeConfigOfClusterType("garden")
	case "seed":
		KUBECONFIG = getKubeConfigOfClusterType("seed")
	case "shoot":
		KUBECONFIG = getKubeConfigOfClusterType("shoot")
	}
	var pathToKubeconfig = ""
	if kubeconfig == nil {
		if home := HomeDir(); home != "" {
			if target == "seed" || target == "shoot" {
				kubeconfig = flag.String("kubeconfig", getKubeConfigOfCurrentTarget(), "(optional) absolute path to the kubeconfig file")
			} else {
				if strings.Contains(getGardenKubeConfig(), "~") {
					pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(getGardenKubeConfig(), "~", "", 1)))
				} else {
					pathToKubeconfig = getGardenKubeConfig()
				}
				kubeconfig = flag.String("kubeconfig", pathToKubeconfig, "(optional) absolute path to the kubeconfig file")
			}
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
		masterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
		flag.Parse()
	} else {
		flag.Set("kubeconfig", KUBECONFIG)
	}
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	checkError(err)
	// create the clientset
	clientset, err := k8s.NewForConfig(config)
	checkError(err)
	return clientset, err
}

// nameOfTargetedCluster returns the full clustername of the currently targeted cluster
func nameOfTargetedCluster() (clustername string) {
	clustername = ExecCmdReturnOutput("kubectl config current-context", "KUBECONFIG="+KUBECONFIG)
	return clustername
}

// getShootClusterName returns the clustername of the shoot cluster
func getShootClusterName() (clustername string) {
	clustername = ""
	file, _ := os.Open(getKubeConfigOfCurrentTarget())
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "current-context:") {
			clustername = strings.TrimPrefix(scanner.Text(), "current-context: ")
		}
	}
	// retrieve full clustername
	Client, err := clientToTarget("seed")
	checkError(err)
	namespaces, err := Client.CoreV1().Namespaces().List(metav1.ListOptions{})
	checkError(err)
	for _, namespace := range namespaces.Items {
		if strings.HasSuffix(namespace.Name, clustername) {
			clustername = namespace.Name
			break
		}
	}
	return clustername
}

// getCredentials returns username and password for url login
func getCredentials() (username, password string) {
	_, err := clientToTarget("shoot")
	checkError(err)
	output := ExecCmdReturnOutput("kubectl config view", "KUBECONFIG="+KUBECONFIG)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "password:") {
			password = strings.TrimPrefix(scanner.Text(), "    password: ")
		} else if strings.Contains(scanner.Text(), "username:") {
			username = strings.TrimPrefix(scanner.Text(), "    username: ")
		}
	}
	return username, password
}

// getSeedNamespaceNameForShoot returns namespace name
func getSeedNamespaceNameForShoot(shootName string) (namespaceSeed string) {
	var target Target
	ReadTarget(pathTarget, &target)
	Client, err = clientToTarget("garden")
	checkError(err)
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	var ind int
	for index, shoot := range shootList.Items {
		if shoot.Name == shootName && (shoot.Namespace == target.Target[1].Name || *shoot.Spec.Cloud.Seed == target.Target[1].Name) {
			ind = index
			break
		}
	}
	return strings.Replace("shoot-"+shootList.Items[ind].Namespace+"-"+shootName, "-garden", "", 1)
}

// returns projectName for Shoot
func getProjectForShoot() (projectName string) {
	var target Target
	ReadTarget(pathTarget, &target)
	if target.Target[1].Kind == "project" {
		projectName = target.Target[1].Name
	} else {
		Client, err = clientToTarget("garden")
		checkError(err)
		k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
		checkError(err)
		gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
		checkError(err)
		k8sGardenClient.SetGardenClientset(gardenClientset)
		shootList, err := k8sGardenClient.GardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		for _, shoot := range shootList.Items {
			if shoot.Name == target.Target[2].Name && *shoot.Spec.Cloud.Seed == target.Target[1].Name {
				projectName = shoot.Namespace
				break
			}
		}
	}
	return projectName
}
