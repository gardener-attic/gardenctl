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
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

// ProjectName is they key of a label on namespaces whose value holds the project name.
const ProjectName = "project.garden.sapcloud.io/name"

var (
	pgarden    string
	pproject   string
	pseed      string
	pshoot     string
	pnamespace string
)

// NewTargetCmd returns a new target command.
func NewTargetCmd(targetReader TargetReader, targetWriter TargetWriter, configReader ConfigReader, ioStreams IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "target <project|garden|seed|shoot|namespace> NAME",
		Short:        "Set scope for next operations",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if pgarden != "" || pproject != "" || pseed != "" || pshoot != "" || pnamespace != "" {
				var arguments []string
				if pgarden != "" {
					arguments := append(arguments, "garden")
					arguments = append(arguments, pgarden)
					err := gardenWrapper(targetReader, targetWriter, configReader, ioStreams, arguments)
					checkError(err)
				}
				if pproject != "" {
					arguments := append(arguments, "project")
					arguments = append(arguments, pproject)
					err := projectWrapper(targetReader, targetWriter, configReader, ioStreams, arguments)
					checkError(err)
				}
				if pseed != "" {
					arguments := append(arguments, "seed")
					arguments = append(arguments, pseed)
					err := seedWrapper(targetReader, targetWriter, configReader, ioStreams, arguments)
					checkError(err)
				}
				if pshoot != "" {
					arguments := append(arguments, "shoot")
					arguments = append(arguments, pshoot)
					err := shootWrapper(targetReader, targetWriter, configReader, ioStreams, arguments)
					checkError(err)
				}

				if pnamespace != "" {
					err := namespaceWrapper(targetReader, targetWriter, pnamespace)
					checkError(err)
				}
				return nil
			}
			if len(args) < 1 && pgarden == "" && pproject == "" && pseed == "" && pshoot == "" && pnamespace == "" || len(args) > 5 {
				return errors.New("command must be in the format: target <project|garden|seed|shoot|namespace> NAME")
			}
			switch args[0] {
			case "garden":
				err := gardenWrapper(targetReader, targetWriter, configReader, ioStreams, args)
				if err != nil {
					return err
				}
			case "project":
				err := projectWrapper(targetReader, targetWriter, configReader, ioStreams, args)
				if err != nil {
					return err
				}
			case "seed":
				err := seedWrapper(targetReader, targetWriter, configReader, ioStreams, args)
				if err != nil {
					return err
				}
			case "shoot":
				err := shootWrapper(targetReader, targetWriter, configReader, ioStreams, args)
				if err != nil {
					return err
				}
			case "namespace":
				if len(args) != 2 || args[1] == "" {
					return errors.New("command must be in the format: target namespace NAME")
				}
				err := namespaceWrapper(targetReader, targetWriter, args[1])
				if err != nil {
					return err
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
					seeds := resolveNameSeed(target, args[0])
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
				var err error
				Client, err = clientToTarget("garden")
				checkError(err)
				clientset, err := target.GardenerClient()
				checkError(err)
				seedList := getSeeds(clientset)
				for _, seed := range seedList.Items {
					if args[0] == seed.Name {
						targetSeed(targetReader, targetWriter, args[0], true)
						os.Exit(0)
					}
				}
				gardenClientset, err := target.GardenerClient()
				checkError(err)
				projectList, err := gardenClientset.CoreV1beta1().Projects().List(metav1.ListOptions{})
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
				} else if len(shoots) == 1 {
					targetShoot(targetWriter, shoots[0])
				} else if len(shoots) > 1 {
					k8sClientToGarden, err := target.K8SClientToKind(TargetKindGarden)
					checkError(err)
					fmt.Fprintln(ioStreams.Out, "shoots:")
					for _, shoot := range shoots {
						projectName, err := getProjectNameByShootNamespace(k8sClientToGarden, shoot.Namespace)
						checkError(err)

						fmt.Fprintln(ioStreams.Out, "- project: "+projectName)
						fmt.Fprintln(ioStreams.Out, "  shoot: "+shoot.Name)
					}
				}
				if pnamespace != "" {
					err := namespaceWrapper(targetReader, targetWriter, pnamespace)
					if err != nil {
						checkError(err)
					}
				}

			}
			return nil
		},
		ValidArgs: []string{"project", "garden", "seed", "shoot", "namespace"},
	}

	cmd.PersistentFlags().StringVarP(&pgarden, "garden", "g", "", "garden name")
	cmd.PersistentFlags().StringVarP(&pproject, "project", "p", "", "project name")
	cmd.PersistentFlags().StringVarP(&pseed, "seed", "s", "", "seed name")
	cmd.PersistentFlags().StringVarP(&pshoot, "shoot", "t", "", "shoot name")
	cmd.PersistentFlags().StringVarP(&pnamespace, "namespace", "n", "", "namespace name")

	return cmd
}

// resolveNameProject resolves name to project
func resolveNameProject(target TargetInterface, name string) (matches []string) {
	gardenClientset, err := target.GardenerClient()
	checkError(err)
	if !strings.Contains(name, "*") {
		project, err := gardenClientset.CoreV1beta1().Projects().Get(name, metav1.GetOptions{})
		if err != nil {
			return []string{}
		}
		return []string{project.Name}
	}

	projectList, err := gardenClientset.CoreV1beta1().Projects().List(metav1.ListOptions{})
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
	err := targetWriter.WriteTarget(pathTarget, target)
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

	err := targetWriter.WriteTarget(pathTarget, target)
	checkError(err)
	fmt.Println("Garden:")
	fmt.Println("KUBECONFIG=" + getKubeConfigOfCurrentTarget())
}

// resolveNameSeed resolves name to seed
func resolveNameSeed(target TargetInterface, name string) (matches []string) {
	tmp := KUBECONFIG
	var err error
	Client, err = clientToTarget("garden")
	checkError(err)
	matcher := ""
	clientset, err := target.GardenerClient()
	checkError(err)
	seedList := getSeeds(clientset)
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
	var err error
	Client, err = clientToTarget("garden")
	checkError(err)
	target := targetReader.ReadTarget(pathTarget)
	gardenName := target.Stack()[0].Name
	gardenClientset, err := target.GardenerClient()
	checkError(err)
	seed, err := gardenClientset.CoreV1beta1().Seeds().Get(name, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Seed not found")
		os.Exit(2)
	}
	kubeSecret, err := Client.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
	checkError(err)
	pathSeed := filepath.Join(pathGardenHome, "cache", gardenName, "seeds", name)
	err = os.MkdirAll(pathSeed, os.ModePerm)
	checkError(err)
	err = ioutil.WriteFile(filepath.Join(pathSeed, "kubeconfig.yaml"), kubeSecret.Data["kubeconfig"], 0644)
	checkError(err)
	KUBECONFIG = filepath.Join(pathSeed, "kubeconfig.yaml")
	if !cachevar && cache {
		err = ioutil.WriteFile(filepath.Join(pathSeed, "kubeconfig.yaml"), kubeSecret.Data["kubeconfig"], 0644)
		checkError(err)
	}

	new := target.Stack()[:1]
	new = append(new, TargetMeta{
		Kind: TargetKindSeed,
		Name: name,
	})
	target.SetStack(new)

	err = targetWriter.WriteTarget(pathTarget, target)
	checkError(err)
	fmt.Println("Seed:")
	fmt.Println("KUBECONFIG=" + getKubeConfigOfCurrentTarget())
}

// resolveNameShoot resolves name to shoot
func resolveNameShoot(target TargetInterface, name string) []gardencorev1beta1.Shoot {
	gardenClientset, err := target.GardenerClient()
	checkError(err)

	isRegexName := true
	listOptions := metav1.ListOptions{}
	if !strings.Contains(name, "*") {
		isRegexName = false
		fieldSelector := fields.OneTermEqualSelector("metadata.name", name)
		listOptions.FieldSelector = fieldSelector.String()
	}

	var shootList *gardencorev1beta1.ShootList
	if len(target.Stack()) == 2 && target.Stack()[1].Kind == TargetKindProject {
		projectName := target.Stack()[1].Name
		project, err := gardenClientset.CoreV1beta1().Projects().Get(projectName, metav1.GetOptions{})
		checkError(err)

		projectNamespace := project.Spec.Namespace
		shootList, err = gardenClientset.CoreV1beta1().Shoots(*projectNamespace).List(listOptions)
		checkError(err)
	} else if len(target.Stack()) == 2 && target.Stack()[1].Kind == TargetKindSeed {
		shootList, err = gardenClientset.CoreV1beta1().Shoots("").List(listOptions)
		checkError(err)

		var filteredShoots []gardencorev1beta1.Shoot
		for _, shoot := range shootList.Items {
			if *shoot.Spec.SeedName == target.Stack()[1].Name {
				filteredShoots = append(filteredShoots, shoot)
			}
		}
		shootList.Items = filteredShoots
	} else {
		shootList, err = gardenClientset.CoreV1beta1().Shoots("").List(listOptions)
		checkError(err)
	}

	if isRegexName {
		var (
			matches []gardencorev1beta1.Shoot
			matcher string
		)
		for _, shoot := range shootList.Items {
			shootName := shoot.Name
			if strings.HasPrefix(name, "*") && strings.HasSuffix(name, "*") {
				matcher = strings.Replace(name, "*", "", 2)
				if strings.Contains(shootName, matcher) {
					matches = append(matches, shoot)
				}
			} else if strings.HasSuffix(name, "*") {
				matcher = strings.Replace(name, "*", "", 1)
				if strings.HasPrefix(shootName, matcher) {
					matches = append(matches, shoot)
				}
			} else if strings.HasPrefix(name, "*") {
				matcher = strings.Replace(name, "*", "", 1)
				if strings.HasSuffix(shootName, matcher) {
					matches = append(matches, shoot)
				}
			} else {
				if shootName == name {
					matches = append(matches, shoot)
				}
			}
		}

		return matches
	}

	return shootList.Items
}

// targetShoot targets shoot cluster with project as default value in stack
func targetShoot(targetWriter TargetWriter, shoot gardencorev1beta1.Shoot) {
	var target Target
	ReadTarget(pathTarget, &target)

	// Get and cache seed kubeconfig for future commands
	gardenName := target.Stack()[0].Name
	pathSeedCache := filepath.Join(pathGardenHome, "cache", gardenName, "seeds")
	pathProjectCache := filepath.Join(pathGardenHome, "cache", gardenName, "projects")

	gardenClientset, err := target.GardenerClient()
	checkError(err)
	seed, err := gardenClientset.CoreV1beta1().Seeds().Get(*shoot.Spec.SeedName, metav1.GetOptions{})
	checkError(err)
	gardenClient, err := target.K8SClientToKind(TargetKindGarden)
	checkError(err)
	seedKubeconfigSecret, err := gardenClient.CoreV1().Secrets(seed.Spec.SecretRef.Namespace).Get(seed.Spec.SecretRef.Name, metav1.GetOptions{})
	checkError(err)
	var seedCacheDir = filepath.Join(pathSeedCache, *shoot.Spec.SeedName)
	err = os.MkdirAll(seedCacheDir, os.ModePerm)
	checkError(err)
	var seedKubeconfigPath = filepath.Join(seedCacheDir, "kubeconfig.yaml")
	err = ioutil.WriteFile(seedKubeconfigPath, seedKubeconfigSecret.Data["kubeconfig"], 0644)
	checkError(err)

	// Get shoot kubeconfig
	var shootKubeconfigSecretName = fmt.Sprintf("%s.kubeconfig", shoot.Name)
	shootKubeconfigSecret, err := gardenClient.CoreV1().Secrets(shoot.Namespace).Get(shootKubeconfigSecretName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Kubeconfig not available, using empty one. Be aware only a limited number of cmds are available!")
	}

	k8sClientToGarden, err := target.K8SClientToKind(TargetKindGarden)
	checkError(err)
	projectName, err := getProjectNameByShootNamespace(k8sClientToGarden, shoot.Namespace)
	checkError(err)

	if len(target.Target) == 1 {
		target.Target = append(target.Target, TargetMeta{"project", projectName})
		target.Target = append(target.Target, TargetMeta{"shoot", shoot.Name})
	} else if len(target.Target) == 2 {
		drop(targetWriter)
		if target.Target[1].Kind == "seed" {
			target.Target[1].Kind = "seed"
			target.Target[1].Name = *shoot.Spec.SeedName
		} else if target.Target[1].Kind == "project" {
			target.Target[1].Kind = "project"
			target.Target[1].Name = projectName
		}
		target.Target = append(target.Target, TargetMeta{"shoot", shoot.Name})
	} else if len(target.Target) == 3 {
		drop(targetWriter)
		drop(targetWriter)
		if len(target.Target) > 2 && target.Target[1].Kind == "seed" {
			target.Target = target.Target[:len(target.Target)-2]
			target.Target = append(target.Target, TargetMeta{"seed", *shoot.Spec.SeedName})
			target.Target = append(target.Target, TargetMeta{"shoot", shoot.Name})
		} else if len(target.Target) > 2 && target.Target[1].Kind == "project" {
			target.Target = target.Target[:len(target.Target)-2]
			target.Target = append(target.Target, TargetMeta{"project", projectName})
			target.Target = append(target.Target, TargetMeta{"shoot", shoot.Name})
		}
	} else if len(target.Target) == 4 {
		drop(targetWriter)
		drop(targetWriter)
		drop(targetWriter)
		if len(target.Target) > 3 && target.Target[1].Kind == "seed" {
			target.Target = target.Target[:len(target.Target)-3]
			target.Target = append(target.Target, TargetMeta{"seed", *shoot.Spec.SeedName})
			target.Target = append(target.Target, TargetMeta{"shoot", shoot.Name})
		} else if len(target.Target) > 3 && target.Target[1].Kind == "project" {
			target.Target = target.Target[:len(target.Target)-3]
			target.Target = append(target.Target, TargetMeta{"project", projectName})
			target.Target = append(target.Target, TargetMeta{"shoot", shoot.Name})
		}
	}

	// Write target
	err = targetWriter.WriteTarget(pathTarget, &target)
	checkError(err)

	// Cache shoot kubeconfig
	var shootCacheDir string
	if target.Target[1].Kind == "seed" {
		shootCacheDir = filepath.Join(pathSeedCache, target.Target[1].Name, shoot.Name)
	} else if target.Target[1].Kind == "project" {
		shootCacheDir = filepath.Join(pathProjectCache, target.Target[1].Name, shoot.Name)
	}

	err = os.MkdirAll(shootCacheDir, os.ModePerm)
	checkError(err)
	var shootKubeconfigPath = filepath.Join(shootCacheDir, "kubeconfig.yaml")
	err = ioutil.WriteFile(shootKubeconfigPath, shootKubeconfigSecret.Data["kubeconfig"], 0644)
	checkError(err)

	KUBECONFIG = shootKubeconfigPath
	fmt.Println("Shoot:")
	fmt.Println("KUBECONFIG=" + KUBECONFIG)
}

func getProjectNameByShootNamespace(k8sClientToGarden kubernetes.Interface, shootNamespace string) (string, error) {
	namespace, err := k8sClientToGarden.CoreV1().Namespaces().Get(shootNamespace, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	labelValue, ok := namespace.Labels[ProjectName]
	if !ok {
		return "", fmt.Errorf("label %q on namespace %q not found", ProjectName, namespace.Name)
	}

	return labelValue, nil
}

// getSeedForProject
func getSeedForProject(shootName string) (seedName string) {
	var err error
	Client, err = clientToTarget("garden")
	checkError(err)
	gardenClientset, err := gardencoreclientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	shootList, err := gardenClientset.CoreV1beta1().Shoots("").List(metav1.ListOptions{})
	checkError(err)
	for _, item := range shootList.Items {
		if item.Name == shootName {
			seedName = *item.Spec.SeedName
		}
	}
	return seedName
}

// getKubeConfigOfClusterType return config of specified type
func getKubeConfigOfClusterType(clusterType TargetKind) (pathToKubeconfig string) {
	var target Target
	ReadTarget(pathTarget, &target)
	gardenName := target.Stack()[0].Name
	switch clusterType {
	case TargetKindGarden:
		if strings.Contains(getGardenKubeConfig(), "~") {
			pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(getGardenKubeConfig(), "~", "", 1)))
		} else {
			pathToKubeconfig = getGardenKubeConfig()
		}
	case TargetKindSeed:
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, "kubeconfig.yaml")
		} else {
			pathToKubeconfig = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", getSeedForProject(target.Target[2].Name), "kubeconfig.yaml")
		}
	case TargetKindShoot:
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", getSeedForProject(target.Target[2].Name), target.Target[2].Name, "kubeconfig.yaml")
		} else if target.Target[1].Kind == "project" {
			pathToKubeconfig = filepath.Join(pathGardenHome, "cache", gardenName, "projects", target.Target[1].Name, target.Target[2].Name, "kubeconfig.yaml")
		}
	}
	return pathToKubeconfig
}

// getKubeConfigOfCurrentTarget returns the path to the kubeconfig of current target
func getKubeConfigOfCurrentTarget() (pathToKubeconfig string) {
	var targetReal Target
	var target Target
	ReadTarget(pathTarget, &targetReal)

	if len(targetReal.Target) == 1 && targetReal.Stack()[0].Kind == "namespace" {
		fmt.Println("the target has only namespace, this is invalid, at least one garden needs to be targeted before using namespace")
		os.Exit(2)
	} else if len(targetReal.Target) > 1 && len(targetReal.Target) <= 4 {
		if targetReal.Stack()[len(targetReal.Target)-1].Kind == "namespace" {
			target.Target = targetReal.Target[:len(targetReal.Target)-1]
		} else {
			target.Target = targetReal.Target
		}
	} else if len(targetReal.Target) == 1 && targetReal.Stack()[0].Kind != "namespace" {
		target.Target = targetReal.Target
	} else {
		fmt.Println("length of target.Stack is illegal")
		os.Exit(2)
	}

	gardenName := target.Stack()[0].Name
	if len(target.Target) == 1 {
		if strings.Contains(getGardenKubeConfig(), "~") {
			pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(getGardenKubeConfig(), "~", "", 1)))
		} else {
			pathToKubeconfig = getGardenKubeConfig()
		}
	} else if (len(target.Target) == 2) && (target.Target[1].Kind != "project") {
		pathToKubeconfig = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, "kubeconfig.yaml")
	} else if (len(target.Target) == 2) && (target.Target[1].Kind == "project") {
		pathToKubeconfig = getGardenKubeConfigViaGardenName(target.Target[0].Name)
	} else if len(target.Target) == 3 {
		if target.Target[1].Kind == "seed" {
			pathToKubeconfig = filepath.Join(pathGardenHome, "cache", gardenName, "seeds", target.Target[1].Name, target.Target[2].Name, "kubeconfig.yaml")
		} else if target.Target[1].Kind == "project" {
			pathToKubeconfig = filepath.Join(pathGardenHome, "cache", gardenName, "projects", target.Target[1].Name, target.Target[2].Name, "kubeconfig.yaml")
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

// getGardenKubeConfigViaGardenName returns path to garden kubeconfig file via garden name
func getGardenKubeConfigViaGardenName(name string) (pathToGardenKubeConfig string) {
	pathToGardenKubeConfig = ""
	var gardenClusters GardenClusters
	yamlGardenConfig, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(yamlGardenConfig, &gardenClusters)
	checkError(err)
	for _, value := range gardenClusters.GardenClusters {
		if value.Name == name {
			pathToGardenKubeConfig = value.KubeConfig
		}
	}
	return pathToGardenKubeConfig
}

func gardenWrapper(targetReader TargetReader, targetWriter TargetWriter, configReader ConfigReader, ioStreams IOStreams, args []string) error {
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
	}
	return nil
}

func projectWrapper(targetReader TargetReader, targetWriter TargetWriter, configReader ConfigReader, ioStreams IOStreams, args []string) error {
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
	}
	return nil
}

func seedWrapper(targetReader TargetReader, targetWriter TargetWriter, configReader ConfigReader, ioStreams IOStreams, args []string) error {
	if len(args) != 2 {
		return errors.New("command must be in the format: target seed NAME")
	}
	target := targetReader.ReadTarget(pathTarget)
	if len(target.Stack()) < 1 {
		return errors.New("no garden cluster targeted")
	}
	seeds := resolveNameSeed(target, args[1])
	if len(seeds) == 0 {
		return fmt.Errorf("no match for %q", args[1])
	} else if len(seeds) == 1 {
		targetSeed(targetReader, targetWriter, seeds[0], true)
	} else if len(seeds) > 1 {
		fmt.Println("seeds:")
		for _, val := range seeds {
			fmt.Println("- seed: " + val)
		}
	}
	return nil
}

func shootWrapper(targetReader TargetReader, targetWriter TargetWriter, configReader ConfigReader, ioStreams IOStreams, args []string) error {
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
		k8sClientToGarden, err := target.K8SClientToKind(TargetKindGarden)
		checkError(err)
		fmt.Fprintln(ioStreams.Out, "shoots:")
		for _, shoot := range shoots {
			projectName, err := getProjectNameByShootNamespace(k8sClientToGarden, shoot.Namespace)
			checkError(err)

			fmt.Fprintln(ioStreams.Out, "- project: "+projectName)
			fmt.Fprintln(ioStreams.Out, "  shoot: "+shoot.Name)
		}
	}
	return nil
}

//set namespace for current kubectl ctx
func namespaceWrapper(targetReader TargetReader, targetWriter TargetWriter, kubectlNameSpace string) error {

	err := targetNamespace(targetWriter, kubectlNameSpace)
	if err != nil {
		return err
	}

	if kubectlNameSpace == "" {
		return errors.New("Namespace must be provided")
	}

	currentConfig := getKubeConfigOfCurrentTarget()

	out, err := ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+currentConfig+"; kubectl config current-context")
	if err != nil {
		fmt.Println(err)
	}
	currentConext := strings.TrimSuffix(string(out), "\n")
	fmt.Println("Namespace:")
	fmt.Printf("Set namespace to %s for current context %s \n", kubectlNameSpace, currentConext)
	out, err = ExecCmdReturnOutput("bash", "-c", "export KUBECONFIG="+currentConfig+"; kubectl config set-context "+currentConext+" --namespace="+kubectlNameSpace)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(out))

	return nil
}

//write current namespace to target
func targetNamespace(targetWriter TargetWriter, ns string) error {
	var target Target
	ReadTarget(pathTarget, &target)

	if len(target.Target) > 4 {
		return errors.New("the length is greater than 4 and illegal")
	}
	if len(target.Target) == 0 {
		return errors.New("the length is 0 and illegal. at least one garden needs to be targeted")
	}
	if len(target.Target) == 1 {
		if string(target.Target[0].Kind) != "garden" {
			return errors.New("if one element in target, this needs to be garden")
		}
		target.Target = append(target.Target, TargetMeta{"namespace", ns})
	}
	if len(target.Target) == 2 {
		if target.Target[1].Kind != "namespace" {
			target.Target = append(target.Target, TargetMeta{"namespace", ns})
		} else {
			target.Target = target.Target[:len(target.Target)-1]
			target.Target = append(target.Target, TargetMeta{"namespace", ns})
		}
	}
	if len(target.Target) == 3 {
		if target.Target[2].Kind != "namespace" {
			target.Target = append(target.Target, TargetMeta{"namespace", ns})
		} else {
			target.Target = target.Target[:len(target.Target)-1]
			target.Target = append(target.Target, TargetMeta{"namespace", ns})
		}
	}
	if len(target.Target) == 4 {
		target.Target = target.Target[:len(target.Target)-1]
		target.Target = append(target.Target, TargetMeta{"namespace", ns})
	}

	err := targetWriter.WriteTarget(pathTarget, &target)
	if err != nil {
		return err
	}
	return nil
}
