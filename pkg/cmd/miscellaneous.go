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
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencoreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetGardenConfig sets GardenConfig struct
func GetGardenConfig(pathGardenConfig string, gardenConfig *GardenConfig) {
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenConfig)
	if err != nil {
		fmt.Println("Invalid gardenctl configuration")
		os.Exit(2)
	}
}

// GetGardenClusterKubeConfigFromConfig return kubeconfig of garden cluster if exists
func GetGardenClusterKubeConfigFromConfig(pathGardenConfig, pathTarget string) {
	var gardenConfig GardenConfig
	var target Target
	i, err := os.Stat(pathTarget)
	checkError(err)
	if i.Size() == 0 {
		// if no garden cluster is selected take the first as default cluster
		i, err := os.Stat(pathGardenConfig)
		checkError(err)
		if i.Size() == 0 {
			fmt.Println("Please provide a gardenctl configuration before usage")
			return
		}
		GetGardenConfig(pathGardenConfig, &gardenConfig)
		target.Target = []TargetMeta{{"garden", gardenConfig.GardenClusters[0].Name}}
		file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
		checkError(err)
		defer file.Close()
		content, err := yaml.Marshal(target)
		checkError(err)
		_, err = file.Write(content)
		checkError(err)
	}
}

// clientToTarget returns the client to target e.g. garden, seed
// DEPRECATED: Use `target.K8SClientToKind()` instead.
func clientToTarget(target TargetKind) (*k8s.Clientset, error) {
	switch target {
	case TargetKindGarden:
		KUBECONFIG = getKubeConfigOfClusterType("garden")
	case TargetKindSeed:
		KUBECONFIG = getKubeConfigOfClusterType("seed")
	case TargetKindShoot:
		KUBECONFIG = getKubeConfigOfClusterType("shoot")
	}
	var pathToKubeconfig string
	if kubeconfig == nil {
		if home := HomeDir(); home != "" {
			if target == TargetKindSeed || target == TargetKindShoot {
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
		err := flag.Set("kubeconfig", KUBECONFIG)
		checkError(err)
	}
	kubeconfig, err := ioutil.ReadFile(*kubeconfig)
	checkError(err)
	clientConfig, err := clientcmd.NewClientConfigFromBytes(kubeconfig)
	checkError(err)
	rawConfig, err := clientConfig.RawConfig()
	checkError(err)
	if err := ValidateClientConfig(rawConfig); err != nil {
		return nil, err
	}
	config, err := clientConfig.ClientConfig()
	checkError(err)
	clientset, err := k8s.NewForConfig(config)
	return clientset, err
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

// getMonitoringCredentials returns username and password required for url login to the montiring tools
func getMonitoringCredentials() (username, password string) {
	var target Target
	ReadTarget(pathTarget, &target)
	shootName := target.Target[2].Name
	shootNamespace := getSeedNamespaceNameForShoot(shootName)
	var err error
	Client, err = clientToTarget("seed")
	checkError(err)
	secretName := "monitoring-ingress-credentials"
	monitoringSecret, err := Client.CoreV1().Secrets(shootNamespace).Get((secretName), metav1.GetOptions{})
	checkError(err)
	username = string(monitoringSecret.Data["username"][:])
	password = string(monitoringSecret.Data["password"][:])
	return username, password
}

// getLoggingCredentials returns username and password required for url login to the kibana dashboard
func getLoggingCredentials() (username, password string) {
	var target Target
	ReadTarget(pathTarget, &target)
	var namespace, secretName string
	if len(target.Target) == 2 {
		namespace = "garden"
		secretName = "seed-logging-ingress-credentials"
	} else if len(target.Target) == 3 {
		namespace = getSeedNamespaceNameForShoot(target.Target[2].Name)
		secretName = "logging-ingress-credentials"
	}
	var err error
	Client, err = clientToTarget("seed")
	checkError(err)
	monitoringSecret, err := Client.CoreV1().Secrets(namespace).Get((secretName), metav1.GetOptions{})
	checkError(err)
	username = string(monitoringSecret.Data["username"][:])
	password = string(monitoringSecret.Data["password"][:])
	return username, password
}

// getSeedNamespaceNameForShoot returns namespace name
func getSeedNamespaceNameForShoot(shootName string) (namespaceSeed string) {
	var target Target
	ReadTarget(pathTarget, &target)
	var err error
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenClientset, err := gardencoreclientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	var shoot *gardencorev1alpha1.Shoot
	if target.Stack()[1].Kind == "project" {
		project, err := gardenClientset.CoreV1alpha1().Projects().Get(target.Stack()[1].Name, metav1.GetOptions{})
		checkError(err)
		shoot, err = gardenClientset.CoreV1alpha1().Shoots(*project.Spec.Namespace).Get(target.Stack()[2].Name, metav1.GetOptions{})
		checkError(err)
	} else {
		shootList, err := gardenClientset.CoreV1alpha1().Shoots("").List(metav1.ListOptions{})
		checkError(err)
		for index, s := range shootList.Items {
			if s.Name == target.Stack()[2].Name && *s.Spec.SeedName == target.Stack()[1].Name {
				shoot = &shootList.Items[index]
				break
			}
		}
	}
	return shoot.Status.TechnicalID
}

// getProjectForShoot returns projectName for Shoot
func getProjectForShoot() (projectName string) {
	var target Target
	ReadTarget(pathTarget, &target)
	if target.Target[1].Kind == "project" {
		projectName = target.Target[1].Name
	} else {
		var err error
		Client, err = clientToTarget("garden")
		checkError(err)
		gardenClientset, err := gardencoreclientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
		checkError(err)
		shootList, err := gardenClientset.CoreV1alpha1().Shoots("").List(metav1.ListOptions{})
		checkError(err)
		for _, shoot := range shootList.Items {
			if shoot.Name == target.Target[2].Name && *shoot.Spec.SeedName == target.Target[1].Name {
				projectName = shoot.Namespace
				break
			}
		}
	}
	return projectName
}

// getTargetType returns error and name of type
func getTargetType() (TargetKind, error) {
	var target Target
	ReadTarget(pathTarget, &target)
	length := len(target.Target)
	switch length {
	case 1:
		return TargetKindGarden, nil
	case 2:
		if target.Target[1].Kind == "seed" {
			return TargetKindSeed, nil
		}

		return TargetKindProject, nil
	case 3:
		return TargetKindShoot, nil
	default:
		return "", errors.New("No target selected")
	}
}

func getEmail(githubURL string) string {
	if githubURL == "" {
		return "null"
	}
	res, err := ExecCmdReturnOutput("bash", "-c", "curl -ks "+githubURL+"/api/v3/users/"+os.Getenv("USER")+" | jq -r .email")
	if err != nil {
		fmt.Println("Cmd was unsuccessful")
		os.Exit(2)
	}
	fmt.Printf("used GitHub email: %s\n", res)
	return res
}

func getEmailFromConfig() string {
	var gardenConfig GardenConfig
	GetGardenConfig(pathGardenConfig, &gardenConfig)
	return gardenConfig.Email
}

func getGithubURL() string {
	var gardenConfig GardenConfig
	GetGardenConfig(pathGardenConfig, &gardenConfig)
	return gardenConfig.GithubURL
}

func capture() func() (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	done := make(chan error, 1)
	save := os.Stdout
	os.Stdout = w
	var buf strings.Builder
	go func() {
		_, err := io.Copy(&buf, r)
		r.Close()
		done <- err
	}()
	return func() (string, error) {
		os.Stdout = save
		w.Close()
		err := <-done
		return buf.String(), err
	}
}

func isIP(word string) bool {
	parts := strings.Split(word, ".")
	if len(parts) < 4 {
		return false
	}
	for _, x := range parts {
		if i, err := strconv.Atoi(x); err == nil {
			if i < 0 || i > 255 {
				return false
			}
		} else {
			return false
		}

	}
	return true
}
