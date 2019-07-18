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

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewTargetCmd returns a new target command.
func NewTargetCmd(targetReader TargetReader, targetWriter TargetWriter, configReader ConfigReader, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "target <project|garden|seed|shoot> NAME",
		Short:        "Set scope for next operations",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 4 {
				return errors.New("command must be in the format: target <project|garden|seed|shoot> NAME")
			}
			switch args[0] {
			case "garden":
				if len(args) == 1 {
					// Print Garden clusters
					PrintGardenClusters(configReader, "yaml", ioStreams)
					return nil
				} else if len(args) > 2 {
					return errors.New("command must be in the format: target garden NAME")
				}

				gardens := resolveNameGarden(configReader, args[1])
				if len(gardens) == 0 {
					return fmt.Errorf("no match for %q", args[1])
				} else if len(gardens) == 1 {
					targetGarden(targetWriter, gardens[0])
				} else if len(gardens) > 1 {
					fmt.Println("gardens:")
					for _, val := range gardens {
						fmt.Println("- garden: " + val)
					}
					os.Exit(2)
				}
			case "project":
				if len(args) != 2 {
					return errors.New("command must be in the format: target project NAME")
				}
				target := targetReader.ReadTarget(pathTarget)
				if len(target.Stack()) < 1 {
					return errors.New("no garden cluster targeted")
				}
				projects := resolveNameProject(target, args[1])
				if len(projects) == 0 {
					return fmt.Errorf("no match for %q", args[1])
				} else if len(projects) == 1 {
					targetProject(targetReader, targetWriter, projects[0])
				} else if len(projects) > 1 {
					fmt.Println("projects:")
					for _, val := range projects {
						fmt.Println("- project: " + val)
					}
					os.Exit(2)
				}
			case "seed":
				if len(args) != 2 {
					return errors.New("command must be in the format: target seed NAME")
				}
				target := targetReader.ReadTarget(pathTarget)
				if len(target.Stack()) < 1 {
					return errors.New("no garden cluster targeted")
				}
				seeds := resolveNameSeed(args[1])
				if len(seeds) == 0 {
					return fmt.Errorf("no match for %q", args[1])
				} else if len(seeds) == 1 {
					targetSeed(targetReader, targetWriter, seeds[0], true)
				} else if len(seeds) > 1 {
					fmt.Println("seeds:")
					for _, val := range seeds {
						fmt.Println("- seed: " + val)
					}
					os.Exit(2)
				}
			case "shoot":
				if len(args) != 2 {
					return errors.New("command must be in the format: target shoot NAME")
				}
				target := targetReader.ReadTarget(pathTarget)
				if len(target.Stack()) < 1 {
					return errors.New("no garden cluster targeted")
				}
				shoots := resolveNameShoot(target, args[1])
				if len(shoots) == 0 {
					return fmt.Errorf("no match for %q", args[1])
				} else if len(shoots) == 1 {
					targetShoot(targetWriter, shoots[0])
				} else if len(shoots) > 1 {
					fmt.Println("shoots:")
					for _, val := range shoots {
						fmt.Println("- shoot: " + val)
					}
					os.Exit(2)
				}
			default:
				target := targetReader.ReadTarget(pathTarget)
				if len(target.Stack()) < 1 {
					return errors.New("no garden cluster targeted")
				} else if garden && !seed && !project {
					gardens := resolveNameGarden(configReader, args[0])
					if len(gardens) == 0 {
						fmt.Println("No match for " + args[0])
						os.Exit(2)
					} else if len(gardens) == 1 {
						targetGarden(targetWriter, gardens[0])
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
						targetSeed(targetReader, targetWriter, seeds[0], true)
					} else if len(seeds) > 1 {
						fmt.Println("seeds:")
						for _, val := range seeds {
							fmt.Println("- seed: " + val)
						}
						os.Exit(2)
					}
					break
				} else if !garden && !seed && project {
					projects := resolveNameProject(target, args[0])
					if len(projects) == 0 {
						fmt.Println("No match for " + args[0])
						os.Exit(2)
					} else if len(projects) == 1 {
						targetProject(targetReader, targetWriter, projects[0])
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
						targetSeed(targetReader, targetWriter, args[0], true)
						os.Exit(0)
					}
				}
				gardenClientset, err := target.GardenerClient()
				checkError(err)
				projectList, err := gardenClientset.GardenV1beta1().Projects().List(metav1.ListOptions{})
				checkError(err)
				match := false
				for _, project := range projectList.Items {
					if args[0] == project.Name {
						targetProject(targetReader, targetWriter, args[0])
						match = true
						break
					}
				}
				KUBECONFIG = tmp
				if match {
					break
				}
				shoots := resolveNameShoot(target, args[0])
				if len(shoots) == 0 {
					fmt.Println("No match for " + args[0])
					os.Exit(2)
				} else if len(shoots) == 1 {
					targetShoot(targetWriter, shoots[0])
				} else if len(shoots) > 1 {
					fmt.Println("shoots:")
					for _, val := range shoots {
						fmt.Println("- shoot: " + val)
					}
					os.Exit(2)
				}
			}

			return nil
		},
		ValidArgs: []string{"project", "garden", "seed", "shoot"},
	}

	cmd.PersistentFlags().BoolVarP(&garden, "garden", "g", false, "target garden")
	cmd.PersistentFlags().BoolVarP(&seed, "seed", "s", false, "target seed")
	cmd.PersistentFlags().BoolVarP(&project, "project", "p", false, "target project")

	return cmd
}

// resolveNameProject resolves name to project
func resolveNameProject(target TargetInterface, name string) (matches []string) {
	gardenClientset, err := target.GardenerClient()
	checkError(err)
	if !strings.Contains(name, "*") {
		project, err := gardenClientset.GardenV1beta1().Projects().Get(name, metav1.GetOptions{})
		if err != nil {
			return []string{}
		}
		return []string{project.Name}
	}

	projectList, err := gardenClientset.GardenV1beta1().Projects().List(metav1.ListOptions{})
	checkError(err)
	matcher := ""
	for _, project := range projectList.Items {
		if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 2)
			if strings.Contains(project.Name, matcher) {
				matches = append(matches, project.Name)
				continue
			}
		}
		if strings.HasSuffix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasPrefix(project.Name, matcher) {
				matches = append(matches, project.Name)
				continue
			}
		}
		if strings.HasPrefix(name, "*") {
			matcher = strings.Replace(name, "*", "", 1)
			if strings.HasSuffix(project.Name, matcher) {
				matches = append(matches, project.Name)
			}
		}
	}
	return matches
}

// targetProject targets a project
func targetProject(targetReader TargetReader, targetWriter TargetWriter, name string) {
	target := targetReader.ReadTarget(pathTarget)
	new := target.Stack()[:1]
	new = append(new, TargetMeta{
		Kind: TargetKindProject,
		Name: name,
	})
	target.SetStack(new)
	err = targetWriter.WriteTarget(pathTarget, target)
	checkError(err)
}

// resolveNameGarden resolves name to garden
func resolveNameGarden(reader ConfigReader, name string) (matches []string) {
	config := reader.ReadConfig(pathGardenConfig)
	matcher := ""
	for _, garden := range config.GardenClusters {
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
func targetGarden(targetWriter TargetWriter, name string) {
	target := &Target{
		Target: []TargetMeta{
			{
				Kind: TargetKindGarden,
				Name: name,
			},
		},
	}

	err = targetWriter.WriteTarget(pathTarget, target)
	checkError(err)

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
func targetSeed(targetReader TargetReader, targetWriter TargetWriter, name string, cache bool) {
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	seed, err := gardenClientset.GardenV1beta1().Seeds().Get(name, metav1.GetOptions{})
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

	target := targetReader.ReadTarget(pathTarget)
	new := target.Stack()[:1]
	new = append(new, TargetMeta{
		Kind: TargetKindSeed,
		Name: name,
	})
	target.SetStack(new)

	err = targetWriter.WriteTarget(pathTarget, target)
	checkError(err)

	fmt.Println("KUBECONFIG=" + getKubeConfigOfCurrentTarget())
}

// resolveNameShoot resolves name to shoot
func resolveNameShoot(target TargetInterface, name string) (matches []string) {
	tmp := KUBECONFIG
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	var shootList *v1beta1.ShootList
	if len(target.Stack()) > 1 && target.Stack()[1].Kind == TargetKindProject {
		projectName := target.Stack()[1].Name
		project, err := gardenClientset.GardenV1beta1().Projects().Get(projectName, metav1.GetOptions{})
		checkError(err)

		projectNamespace := project.Spec.Namespace
		shootList, err = gardenClientset.GardenV1beta1().Shoots(*projectNamespace).List(metav1.ListOptions{})
		checkError(err)
	} else if len(target.Stack()) > 1 && target.Stack()[1].Kind == TargetKindSeed {
		shootList, err = gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		checkError(err)
		var filteredShoots []v1beta1.Shoot
		for _, shoot := range shootList.Items {
			if *shoot.Spec.Cloud.Seed == target.Stack()[1].Name {
				filteredShoots = append(filteredShoots, shoot)
			}
		}
		shootList.Items = filteredShoots
	} else {
		shootList, err = gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
		checkError(err)
	}

	matcher := ""
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
func targetShoot(targetWriter TargetWriter, name string) {
	Client, err = clientToTarget("garden")
	gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	shootList, err := gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
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
		} else if len(target.Target) == 3 && item.Name == name && *item.Spec.Cloud.Seed == target.Target[1].Name {
			matchedShoots = append(matchedShoots, item)
			seedName = *item.Spec.Cloud.Seed
		} else if len(target.Target) >= 2 && item.Name == name {
			matchedShoots = append(matchedShoots, item)
			seedName = *item.Spec.Cloud.Seed
		}
	}
	if len(matchedShoots) == 0 {
		fmt.Println("Shoot " + name + " not found")
	} else if len(matchedShoots) == 1 {
		gardenClientset, err := target.GardenerClient()
		checkError(err)
		project, err := getProjectByShootNamespace(gardenClientset, matchedShoots[0].Namespace)
		checkError(err)

		if len(target.Target) == 1 {
			target.Target = append(target.Target, TargetMeta{"project", project.Name})
			target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
		} else if len(target.Target) == 2 {
			drop(targetWriter)
			if target.Target[1].Kind == "seed" {
				target.Target[1].Kind = "seed"
				target.Target[1].Name = *matchedShoots[0].Spec.Cloud.Seed
			} else if target.Target[1].Kind == "project" {
				target.Target[1].Kind = "project"
				target.Target[1].Name = project.Name
			}
			target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
		} else if len(target.Target) == 3 {
			drop(targetWriter)
			drop(targetWriter)
			if len(target.Target) > 2 && target.Target[1].Kind == "seed" {
				target.Target = target.Target[:len(target.Target)-2]
				target.Target = append(target.Target, TargetMeta{"seed", *matchedShoots[0].Spec.Cloud.Seed})
				target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
			} else if len(target.Target) > 2 && target.Target[1].Kind == "project" {
				target.Target = target.Target[:len(target.Target)-2]
				target.Target = append(target.Target, TargetMeta{"project", project.Name})
				target.Target = append(target.Target, TargetMeta{"shoot", matchedShoots[0].Name})
			}
		}
		err = targetWriter.WriteTarget(pathTarget, &target)
		checkError(err)

		Client, err = clientToTarget("garden")
		checkError(err)
		seed, err := gardenClientset.GardenV1beta1().Seeds().Get(seedName, metav1.GetOptions{})
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

func getProjectByShootNamespace(gardenClientset clientset.Interface, shootNamespace string) (*v1beta1.Project, error) {
	projectList, err := gardenClientset.GardenV1beta1().Projects().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, project := range projectList.Items {
		if shootNamespace == *project.Spec.Namespace {
			return &project, nil
		}
	}

	return nil, fmt.Errorf("project with namespace %q not found", shootNamespace)
}

// getSeedForProject
func getSeedForProject(shootName string) (seedName string) {
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	shootList, err := gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	for _, item := range shootList.Items {
		if item.Name == shootName {
			seedName = *item.Spec.Cloud.Seed
		}
	}
	return seedName
}

// getKubeConfigOfClusterType return config of specified type
func getKubeConfigOfClusterType(clusterType TargetKind) (pathToKubeconfig string) {
	var target Target
	ReadTarget(pathTarget, &target)
	switch clusterType {
	case TargetKindGarden:
		if strings.Contains(getGardenKubeConfig(), "~") {
			pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(getGardenKubeConfig(), "~", "", 1)))
		} else {
			pathToKubeconfig = getGardenKubeConfig()
		}
	case TargetKindSeed:
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = pathGardenHome + "/cache/seeds/" + target.Target[1].Name + "/kubeconfig.yaml"
		} else {
			pathToKubeconfig = pathGardenHome + "/cache/seeds/" + getSeedForProject(target.Target[2].Name) + "/kubeconfig.yaml"
		}
	case TargetKindShoot:
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = pathGardenHome + "/cache/seeds/" + getSeedForProject(target.Target[2].Name) + "/" + target.Target[2].Name + "/kubeconfig.yaml"
		} else if target.Target[1].Kind == "project" {
			pathToKubeconfig = pathGardenHome + "/cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/kubeconfig.yaml"
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
		pathToKubeconfig = pathGardenHome + "/cache/seeds/" + target.Target[1].Name + "/kubeconfig.yaml"
	} else if len(target.Target) == 3 {
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = pathGardenHome + "/cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/kubeconfig.yaml"
		} else if target.Target[1].Kind == "project" {
			pathToKubeconfig = pathGardenHome + "/cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/kubeconfig.yaml"
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
