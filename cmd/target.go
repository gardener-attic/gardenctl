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
	"path/filepath"
	"strings"

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// targetCmd represents the target command
var targetCmd = &cobra.Command{
	Use:   "target <project|garden|seed|shoot> + NAME",
	Short: `Set scope for next operations`,
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 4 {
			fmt.Println("Command must be in the format: target" + `	<project|garden|seed|shoot> + NAME`)
			os.Exit(2)
		}
		var t Target
		ReadTarget(pathTarget, &t)
		switch args[0] {
		case "garden":
			if len(args) != 2 {
				fmt.Println("Command must be in the format: target" + `	<project|garden|seed|shoot> + NAME`)
				os.Exit(2)
			}
			gardens := resolveNameGarden(args[1])
			if len(gardens) == 0 {
				fmt.Println("No match for " + args[1])
				os.Exit(2)
			} else if len(gardens) == 1 {
				targetGarden(gardens[0])
			} else if len(gardens) > 1 {
				fmt.Println("gardens:")
				for _, val := range gardens {
					fmt.Println("- garden: " + val)
				}
				os.Exit(2)
			}
		case "project":
			if len(args) != 2 {
				fmt.Println("Command must be in the format: target" + `	<project|garden|seed|shoot> + NAME`)
				os.Exit(2)
			}
			if len(t.Target) < 1 {
				fmt.Println("No garden cluster targeted")
				os.Exit(2)
			}
			projects := resolveNameProject(args[1])
			if len(projects) == 0 {
				fmt.Println("No match for " + args[1])
				os.Exit(2)
			} else if len(projects) == 1 {
				targetProject(projects[0])
			} else if len(projects) > 1 {
				fmt.Println("projects:")
				for _, val := range projects {
					fmt.Println("- project: " + val)
				}
				os.Exit(2)
			}
		case "seed":
			if len(args) != 2 {
				fmt.Println("Command must be in the format: target" + `	<project|garden|seed|shoot> + NAME`)
				os.Exit(2)
			}
			if len(t.Target) < 1 {
				fmt.Println("No garden cluster targeted")
				os.Exit(2)
			}
			seeds := resolveNameSeed(args[1])
			if len(seeds) == 0 {
				fmt.Println("No match for " + args[1])
				os.Exit(2)
			} else if len(seeds) == 1 {
				targetSeed(seeds[0], true)
			} else if len(seeds) > 1 {
				fmt.Println("seeds:")
				for _, val := range seeds {
					fmt.Println("- seed: " + val)
				}
				os.Exit(2)
			}
		case "shoot":
			if len(args) != 2 {
				fmt.Println("Command must be in the format: target" + `	<project|garden|seed|shoot> + NAME`)
				os.Exit(2)
			}
			if len(t.Target) < 1 {
				fmt.Println("No garden cluster targeted")
				os.Exit(2)
			}
			shoots := resolveNameShoot(args[1])
			if len(shoots) == 0 {
				fmt.Println("No match for " + args[1])
				os.Exit(2)
			} else if len(shoots) == 1 {
				targetShoot(shoots[0])
			} else if len(shoots) > 1 {
				fmt.Println("shoots:")
				for _, val := range shoots {
					fmt.Println("- shoot: " + val)
				}
				os.Exit(2)
			}
		default:
			if len(t.Target) < 1 {
				fmt.Println("No garden cluster targeted")
				os.Exit(2)
			} else if garden && !seed && !project {
				gardens := resolveNameGarden(args[0])
				if len(gardens) == 0 {
					fmt.Println("No match for " + args[0])
					os.Exit(2)
				} else if len(gardens) == 1 {
					targetGarden(gardens[0])
				} else if len(gardens) > 1 {
					fmt.Println("gardens:")
					for _, val := range gardens {
						fmt.Println("- garden: " + val)
					}
					os.Exit(2)
				}
				break
			} else if !garden && seed && !project {
				seeds := resolveNameSeed(args[0])
				if len(seeds) == 0 {
					fmt.Println("No match for " + args[0])
					os.Exit(2)
				} else if len(seeds) == 1 {
					targetSeed(seeds[0], true)
				} else if len(seeds) > 1 {
					fmt.Println("seeds:")
					for _, val := range seeds {
						fmt.Println("- seed: " + val)
					}
					os.Exit(2)
				}
				break
			} else if !garden && !seed && project {
				projects := resolveNameProject(args[0])
				if len(projects) == 0 {
					fmt.Println("No match for " + args[0])
					os.Exit(2)
				} else if len(projects) == 1 {
					targetProject(projects[0])
				} else if len(projects) > 1 {
					fmt.Println("projects:")
					for _, val := range projects {
						fmt.Println("- project: " + val)
					}
					os.Exit(2)
				}
				break
			}
			tmp := KUBECONFIG
			Client, err = clientToTarget("garden")
			checkError(err)
			seedList := getSeeds()
			for _, seed := range seedList.Items {
				if args[0] == seed.Name {
					targetSeed(args[0], true)
					os.Exit(0)
				}
			}
			projectLabel := "garden.sapcloud.io/role=project"
			projectList, err := Client.CoreV1().Namespaces().List(metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s", projectLabel),
			})
			checkError(err)
			match := false
			for _, project := range projectList.Items {
				if args[0] == project.Name {
					targetProject(args[0])
					match = true
					break
				}
				if ("garden-" + args[0]) == project.Name {
					targetProject("garden-" + args[0])
					match = true
					break
				}
			}
			KUBECONFIG = tmp
			if match {
				break
			}
			shoots := resolveNameShoot(args[0])
			if len(shoots) == 0 {
				fmt.Println("No match for " + args[0])
				os.Exit(2)
			} else if len(shoots) == 1 {
				targetShoot(shoots[0])
			} else if len(shoots) > 1 {
				fmt.Println("shoots:")
				for _, val := range shoots {
					fmt.Println("- shoot: " + val)
				}
				os.Exit(2)
			}
		}
	},
	ValidArgs: []string{"project", "garden", "seed", "shoot"},
}

func init() {
	targetCmd.PersistentFlags().BoolVarP(&garden, "garden", "g", false, "target garden")
	targetCmd.PersistentFlags().BoolVarP(&seed, "seed", "s", false, "target seed")
	targetCmd.PersistentFlags().BoolVarP(&project, "project", "p", false, "target project")
}

// resolveNameProject resolves name to project
func resolveNameProject(name string) (matches []string) {
	if !strings.HasPrefix(name, "garden-") {
		name = "garden-" + name
	}
	tmp := KUBECONFIG
	Client, err = clientToTarget("garden")
	checkError(err)
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
	matcher := ""
	for _, project := range projectList.Items {
		if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 2)
			if strings.Contains(project.Name, matcher) {
				matches = append(matches, project.Name)
			}
		} else if strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasPrefix(project.Name, matcher) {
				matches = append(matches, project.Name)
			}
		} else if strings.HasPrefix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasSuffix(project.Name, matcher) {
				matches = append(matches, project.Name)
			}
		} else {
			if project.Name == name {
				matches = append(matches, project.Name)
			}
		}
	}
	KUBECONFIG = tmp
	return matches
}

// targetProject targets a project
func targetProject(name string) {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) == 1 {
		target.Target = append(target.Target, TargetMeta{"project", name})
	} else if len(target.Target) == 2 {
		drop()
		target.Target[1].Kind = "project"
		target.Target[1].Name = name
	} else if len(target.Target) == 3 {
		drop()
		drop()
		if len(target.Target) > 2 {
			target.Target = target.Target[:len(target.Target)-2]
			target.Target = append(target.Target, TargetMeta{"project", name})
		}
	}
	file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
	checkError(err)
	content, err := yaml.Marshal(target)
	checkError(err)
	file.Write(content)
	file.Close()
}

// resolveNameGarden resolves name to garden
func resolveNameGarden(name string) (matches []string) {
	var gardenClusters GardenClusters
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenClusters)
	checkError(err)
	matcher := ""
	for _, garden := range gardenClusters.GardenClusters {
		if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 2)
			if strings.Contains(garden.Name, matcher) {
				matches = append(matches, garden.Name)
			}
		} else if strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasPrefix(garden.Name, matcher) {
				matches = append(matches, garden.Name)
			}
		} else if strings.HasPrefix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasSuffix(garden.Name, matcher) {
				matches = append(matches, garden.Name)
			}
		} else {
			if garden.Name == name {
				matches = append(matches, garden.Name)
			}
		}
	}
	return matches
}

// targetGarden targets kubeconfig file of garden cluster
func targetGarden(name string) {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) == 0 {
		target.Target = append(target.Target, TargetMeta{"garden", name})
	} else if len(target.Target) == 1 {
		drop()
		target.Target[0].Kind = "garden"
		target.Target[0].Name = name
	} else if len(target.Target) == 2 {
		drop()
		drop()
		target.Target = target.Target[:len(target.Target)-2]
		target.Target = append(target.Target, TargetMeta{"garden", name})
	} else if len(target.Target) == 3 {
		drop()
		drop()
		drop()
		if len(target.Target) > 2 {
			target.Target = target.Target[:len(target.Target)-3]
			target.Target = append(target.Target, TargetMeta{"garden", name})
		}
	}
	file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
	checkError(err)
	content, err := yaml.Marshal(target)
	checkError(err)
	file.Write(content)
	file.Close()
	fmt.Println("KUBECONFIG=" + getKubeConfigOfCurrentTarget())
}

// resolveNameSeed resolves name to seed
func resolveNameSeed(name string) (matches []string) {
	tmp := KUBECONFIG
	Client, err = clientToTarget("garden")
	checkError(err)
	matcher := ""
	seedList := getSeeds()
	for _, seed := range seedList.Items {
		if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 2)
			if strings.Contains(seed.Name, matcher) {
				matches = append(matches, seed.Name)
			}
		} else if strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasPrefix(seed.Name, matcher) {
				matches = append(matches, seed.Name)
			}
		} else if strings.HasPrefix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasSuffix(seed.Name, matcher) {
				matches = append(matches, seed.Name)
			}
		} else {
			if seed.Name == name {
				matches = append(matches, seed.Name)
			}
		}
	}
	KUBECONFIG = tmp
	return matches
}

// targetSeed targets kubeconfig file of seed cluster and updates target
func targetSeed(name string, cache bool) {
	Client, err = clientToTarget("garden")
	checkError(err)
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	seed, err := k8sGardenClient.GardenClientset().GardenV1beta1().Seeds().Get(name, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Seed not found")
		os.Exit(2)
	}
	kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
	checkError(err)
	pathSeed := pathSeedCache + "/" + name
	os.MkdirAll(pathSeed, os.ModePerm)
	err = ioutil.WriteFile(pathSeed+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
	checkError(err)
	KUBECONFIG = pathSeed + "/kubeconfig.yaml"
	if !cachevar && cache {
		err = ioutil.WriteFile(pathSeed+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
		checkError(err)
	}
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	if len(target.Target) == 1 {
		target.Target = append(target.Target, TargetMeta{"seed", name})
	} else if len(target.Target) == 2 {
		drop()
		target.Target[1].Kind = "seed"
		target.Target[1].Name = name
	} else if len(target.Target) == 3 {
		drop()
		drop()
		if len(target.Target) > 2 {
			target.Target = target.Target[:len(target.Target)-2]
			target.Target = append(target.Target, TargetMeta{"seed", name})
		}
	}
	file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
	checkError(err)
	content, err := yaml.Marshal(target)
	checkError(err)
	file.Write(content)
	file.Close()
	fmt.Println("KUBECONFIG=" + getKubeConfigOfCurrentTarget())
}

// resolveNameShoot resolves name to shoot
func resolveNameShoot(name string) (matches []string) {
	tmp := KUBECONFIG
	Client, err = clientToTarget("garden")
	checkError(err)
	matcher := ""
	var target Target
	ReadTarget(pathTarget, &target)
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	var shootList *v1beta1.ShootList
	if len(target.Target) > 1 && target.Target[1].Kind == "project" {
		shootList, err = k8sGardenClient.GardenClientset().GardenV1beta1().Shoots(target.Target[1].Name).List(metav1.ListOptions{})
		checkError(err)
	} else if len(target.Target) > 1 && target.Target[1].Kind == "seed" {
		shootList, err = k8sGardenClient.GardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		checkError(err)
		var filteredShoots []v1beta1.Shoot
		for _, shoot := range shootList.Items {
			if *shoot.Spec.Cloud.Seed == target.Target[1].Name {
				filteredShoots = append(filteredShoots, shoot)
			}
		}
		shootList.Items = filteredShoots
	} else {
		shootList, err = k8sGardenClient.GardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		checkError(err)
	}
	for _, shoot := range shootList.Items {
		shootName := shoot.Name
		if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 2)
			if strings.Contains(shootName, matcher) {
				matches = append(matches, shootName)
			}
		} else if strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasPrefix(shootName, matcher) {
				matches = append(matches, shootName)
			}
		} else if strings.HasPrefix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasSuffix(shootName, matcher) {
				matches = append(matches, shootName)
			}
		} else {
			if shootName == name {
				matches = append(matches, shootName)
			}
		}
	}
	KUBECONFIG = tmp
	return matches
}

// targetShoot targets shoot cluster with project as default value in stack
func targetShoot(name string) {
	Client, err = clientToTarget("garden")
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	var target Target
	ReadTarget(pathTarget, &target)
	var matchedShoots []v1beta1.Shoot
	seedName := ""
	for _, item := range shootList.Items {
		if len(target.Target) == 1 && item.Name == name {
			matchedShoots = append(matchedShoots, item)
			seedName = *item.Spec.Cloud.Seed
		} else if len(target.Target) == 2 && item.Name == name && *item.Spec.Cloud.Seed == target.Target[1].Name {
			seedName = target.Target[1].Name
			matchedShoots = append(matchedShoots, item)
		} else if len(target.Target) == 2 && item.Name == name && item.Namespace == target.Target[1].Name {
			seedName = *item.Spec.Cloud.Seed
			matchedShoots = append(matchedShoots, item)
		} else if len(target.Target) == 3 && item.Name == name && *item.Spec.Cloud.Seed == target.Target[1].Name {
			matchedShoots = append(matchedShoots, item)
			seedName = *item.Spec.Cloud.Seed
		} else if len(target.Target) == 3 && item.Name == name && item.Namespace == target.Target[1].Name {
			matchedShoots = append(matchedShoots, item)
			seedName = *item.Spec.Cloud.Seed
		}
	}
	if len(matchedShoots) == 0 {
		fmt.Println("Shoot " + name + " not found")
	} else if len(matchedShoots) == 1 {
		if len(target.Target) == 1 {
			target.Target = append(target.Target, TargetMeta{"project", matchedShoots[0].Namespace})
			target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
		} else if len(target.Target) == 2 {
			drop()
			if target.Target[1].Kind == "seed" {
				target.Target[1].Kind = "seed"
				target.Target[1].Name = *matchedShoots[0].Spec.Cloud.Seed
			} else if target.Target[1].Kind == "project" {
				target.Target[1].Kind = "project"
				target.Target[1].Name = matchedShoots[0].Namespace
			}
			target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
		} else if len(target.Target) == 3 {
			drop()
			drop()
			if len(target.Target) > 2 && target.Target[1].Kind == "seed" {
				target.Target = target.Target[:len(target.Target)-2]
				target.Target = append(target.Target, TargetMeta{"seed", *matchedShoots[0].Spec.Cloud.Seed})
				target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
			} else if len(target.Target) > 2 && target.Target[1].Kind == "project" {
				target.Target = target.Target[:len(target.Target)-2]
				target.Target = append(target.Target, TargetMeta{"project", matchedShoots[0].Namespace})
				target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
			}
		}
		file, err := os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
		checkError(err)
		content, err := yaml.Marshal(target)
		checkError(err)
		file.Write(content)
		file.Close()
		Client, err = clientToTarget("garden")
		checkError(err)
		seed, err := k8sGardenClient.GardenClientset().GardenV1beta1().Seeds().Get(seedName, metav1.GetOptions{})
		checkError(err)
		kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
		checkError(err)
		pathSeed := pathSeedCache + "/" + *matchedShoots[0].Spec.Cloud.Seed
		os.MkdirAll(pathSeed, os.ModePerm)
		err = ioutil.WriteFile(pathSeed+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
		checkError(err)
		KUBECONFIG = pathSeed + "/kubeconfig.yaml"
		namespace := matchedShoots[0].Status.TechnicalID

		Client, err = clientToTarget("seed")
		checkError(err)
		kubeSecret, err = Client.CoreV1().Secrets(namespace).Get("kubecfg", metav1.GetOptions{})
		checkError(err)
		if target.Target[1].Kind == "seed" {
			pathShootKubeconfig := pathSeedCache + "/" + target.Target[1].Name + "/" + name
			os.MkdirAll(pathShootKubeconfig, os.ModePerm)
			err = ioutil.WriteFile(pathShootKubeconfig+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
			checkError(err)
		} else if target.Target[1].Kind == "project" {
			pathProjectKubeconfig := pathProjectCache + "/" + target.Target[1].Name + "/" + name
			os.MkdirAll(pathProjectKubeconfig, os.ModePerm)
			err = ioutil.WriteFile(pathProjectKubeconfig+"/kubeconfig.yaml", kubeSecret.Data["kubeconfig"], 0644)
			checkError(err)
		}
		KUBECONFIG = getKubeConfigOfCurrentTarget()
		fmt.Println("KUBECONFIG=" + getKubeConfigOfCurrentTarget())
	} else if len(matchedShoots) > 1 {
		fmt.Println("Multiple Shoots found")
		fmt.Println("projects:")
		for _, shoot := range matchedShoots {
			fmt.Println("- project: " + shoot.Namespace)
			fmt.Println("  shoots: " + shoot.Name)
		}
		fmt.Println("Target a project first")
	}
}

// getSeedForProject
func getSeedForProject(shootName string) (seedName string) {
	Client, err = clientToTarget("garden")
	k8sGardenClient, err := kubernetes.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	for _, item := range shootList.Items {
		if item.Name == shootName {
			seedName = *item.Spec.Cloud.Seed
		}
	}
	return seedName
}

// getKubeConfigOfClusterType return config of specified type
func getKubeConfigOfClusterType(clusterType string) (pathToKubeconfig string) {
	var target Target
	ReadTarget(pathTarget, &target)
	switch clusterType {
	case "garden":
		if strings.Contains(getGardenKubeConfig(), "~") {
			pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(getGardenKubeConfig(), "~", "", 1)))
		} else {
			pathToKubeconfig = getGardenKubeConfig()
		}
	case "seed":
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = pathGardenHome + "/cache/seeds" + "/" + target.Target[1].Name + "/" + "kubeconfig.yaml"
		} else {
			pathToKubeconfig = pathGardenHome + "/cache/seeds" + "/" + getSeedForProject(target.Target[2].Name) + "/" + "kubeconfig.yaml"
		}
	case "shoot":
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = pathGardenHome + "/cache/seeds" + "/" + getSeedForProject(target.Target[2].Name) + "/" + target.Target[2].Name + "/" + "kubeconfig.yaml"
		} else if target.Target[1].Kind == "project" {
			pathToKubeconfig = pathGardenHome + "/cache/projects" + "/" + target.Target[1].Name + "/" + target.Target[2].Name + "/" + "kubeconfig.yaml"
		}
	}
	return pathToKubeconfig
}

// getKubeConfigOfCurrentTarget returns the path to the kubeconfig of current target
func getKubeConfigOfCurrentTarget() (pathToKubeconfig string) {
	var target Target
	ReadTarget(pathTarget, &target)
	if len(target.Target) == 1 {
		if strings.Contains(getGardenKubeConfig(), "~") {
			pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(getGardenKubeConfig(), "~", "", 1)))
		} else {
			pathToKubeconfig = getGardenKubeConfig()
		}
	} else if (len(target.Target) == 2) && (target.Target[1].Kind != "project") {
		pathToKubeconfig = pathGardenHome + "/cache/seeds" + "/" + target.Target[1].Name + "/" + "kubeconfig.yaml"
	} else if len(target.Target) == 3 {
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = pathGardenHome + "/cache/seeds" + "/" + target.Target[1].Name + "/" + target.Target[2].Name + "/" + "kubeconfig.yaml"
		} else if target.Target[1].Kind == "project" {
			pathToKubeconfig = pathGardenHome + "/cache/projects" + "/" + target.Target[1].Name + "/" + target.Target[2].Name + "/" + "kubeconfig.yaml"
		}
	}
	return pathToKubeconfig
}

// getGardenKubeConfig returns path to garden kubeconfig file
func getGardenKubeConfig() (pathToGardenKubeConfig string) {
	pathToGardenKubeConfig = ""
	var gardenClusters GardenClusters
	var target Target
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenClusters)
	checkError(err)
	ReadTarget(pathTarget, &target)
	for _, value := range gardenClusters.GardenClusters {
		if value.Name == target.Target[0].Name {
			pathToGardenKubeConfig = value.KubeConfig
		}
	}
	return pathToGardenKubeConfig
}
