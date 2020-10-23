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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"

	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencoreclientset "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
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
				pathToKubeconfig = TidyKubeconfigWithHomeDir(getGardenKubeConfig())
				kubeconfig = flag.String("kubeconfig", pathToKubeconfig, "(optional) absolute path to the kubeconfig file")
			}
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}
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

// getSeedNamespaceNameForShoot returns namespace name
func getSeedNamespaceNameForShoot(shootName string) (namespaceSeed string) {
	var target Target
	ReadTarget(pathTarget, &target)
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
	return shoot.Status.TechnicalID
}

//getProjectObject returns the Project for Shoot
func getProjectObject() (*v1beta1.Project, error) {
	var target Target
	ReadTarget(pathTarget, &target)
	Client, err := target.K8SClientToKind("garden")
	checkError(err)

	if isTargeted("project") {
		projectName, err := getTargetName("project")
		checkError(err)
		return Client.GardenerV1beta1().Projects().Get(projectName, metav1.GetOptions{})
	} else if isTargeted("seed", "shoot") {
		if getRole() == "user" {
			return nil, errors.New("can't determine project. Target project instead of seed")
		}
		seedName, err := getTargetName("seed")
		checkError(err)
		shootName, err := getTargetName("shoot")
		checkError(err)

		shootList, err := Client.GardenerV1beta1().Shoots(metav1.NamespaceAll).List(metav1.ListOptions{
			FieldSelector: fields.SelectorFromSet(
				fields.Set{
					core.ShootSeedName: seedName,
					"metadata.name":    shootName,
				}).String(),
		})
		checkError(err)
		if len(shootList.Items) == 0 {
			return nil, errors.New("No shoot found with name " + shootName + " that is running on the seed " + seedName)
		} else if len(shootList.Items) > 1 {
			return nil, errors.New("There are multiple shoots with the name " + shootName + " that are running on the seed " + seedName)
		}
		projectNamespace := shootList.Items[0].Namespace

		projectList, err := Client.GardenerV1beta1().Projects().List(metav1.ListOptions{})
		checkError(err)

		for _, p := range projectList.Items {
			if *p.Spec.Namespace == projectNamespace {
				return &p, nil
			}
		}
	}

	return nil, errors.New("can't determine project")
}

//getShootObject return shoot object and error
func getShootObject() (*v1beta1.Shoot, error) {
	var target Target
	ReadTarget(pathTarget, &target)
	gardenClientset, err := target.GardenerClient()
	checkError(err)
	project, err := getProjectObject()
	checkError(err)
	shootName, err := getTargetName("shoot")
	checkError(err)
	shoot, err := gardenClientset.CoreV1beta1().Shoots(*project.Spec.Namespace).Get(shootName, metav1.GetOptions{})
	checkError(err)
	return shoot, nil
}

//getShootObject return shoot object and error
func getSeedObject() (*v1beta1.Seed, error) {
	var target Target
	ReadTarget(pathTarget, &target)
	gardenClientset, err := target.GardenerClient()
	checkError(err)
	shoot, err := getShootObject()
	checkError(err)
	seed, err := gardenClientset.CoreV1beta1().Seeds().Get(*shoot.Spec.SeedName, metav1.GetOptions{})
	checkError(err)
	return seed, nil
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
	baseURL, err := url.Parse(githubURL)
	checkError(err)
	baseURL.Path += "/api/v3/users/"
	baseURL.Path += url.PathEscape(os.Getenv("USER"))
	resp, err := http.Get(baseURL.String())
	checkError(err)
	defer resp.Body.Close()
	userInfo, err := ioutil.ReadAll(resp.Body)
	checkError(err)
	var yamlOut map[string]interface{}
	err = yaml.Unmarshal(userInfo, &yamlOut)
	checkError(err)
	githubEmail := yamlOut["email"].(string)
	fmt.Printf("used GitHub email: %s\n", githubEmail)
	return githubEmail
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

func isIPv4(host string) bool {
	return net.ParseIP(host) != nil && net.ParseIP(host).To4() != nil
}

func getPublicIP() string {
	ipURL, err := url.Parse("https://api.ipify.org")
	checkError(err)
	params := url.Values{}
	params.Add("format", "text")
	ipURL.RawQuery = params.Encode()
	resp, err := http.Get(ipURL.String())
	checkError(err)
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	checkError(err)
	if net.ParseIP(string(ip)) == nil {
		fmt.Printf("IP not valid:" + string(ip))
		os.Exit(0)
	}
	return string(ip)
}

// get role either user or operator
func getRole() string {
	var target Target
	var role string
	ReadTarget(pathTarget, &target)
	clientset, err := target.K8SClientToKind("garden")
	checkError(err)
	ssar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:     "get",
				Resource: "secrets",
			},
		},
	}
	ssar, err = clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ssar)
	checkError(err)
	if ssar.Status.Allowed {
		role = "operator"
	} else {
		role = "user"
	}
	return role
}

/*
getTargetMapInfo retun garden,project,seed,shoot,shootTechnicalID to global targetInfo
Use `getFromTargetInfo()` instead
*/
func getTargetMapInfo() {
	if len(targetInfo) > 0 {
		return
	}
	var target Target
	ReadTarget(pathTarget, &target)
	for _, t := range target.Stack() {
		targetInfo[string(t.Kind)] = string(t.Name)
	}

	if isTargeted("shoot") {
		shoot, err := getShootObject()
		checkError(err)
		targetInfo["shootTechnicalID"] = shoot.Status.TechnicalID

		if targetInfo["seed"] == "" {
			targetInfo["seed"] = *shoot.Spec.SeedName
		}
	}

	if targetInfo["project"] == "" {
		projectObj, err := getProjectObject()
		checkError(err)
		targetInfo["project"] = projectObj.Name
	}
}

/*
lookup Target Kind ("kind value") return Target Name "name value" from target file ~/.garden/sessions/plantingSession/target
*/
func getTargetName(Kind string) (string, error) {
	//if you modify getTargetName Please also modify GetTargetNameTest,
	//as GetTargetName used for miscellaneous_test.go for teting purposes.
	var target Target
	ReadTarget(pathTarget, &target)
	for _, t := range target.Stack() {
		if string(t.Kind) == Kind {
			return string(t.Name), nil
		}
	}
	return "", errors.New("Kind:" + Kind + "not found from ~/.garden/sessions/plantingSession/target")
}

//GetTargetNameTest for getTargetName testing
func GetTargetNameTest(targetReader TargetReader, Kind string) (string, error) {
	target := targetReader.ReadTarget(pathTarget)
	for _, t := range target.Stack() {
		if string(t.Kind) == Kind {
			return string(t.Name), nil
		}
	}
	return "", errors.New("Kind:" + Kind + "not found from ~/.garden/sessions/plantingSession/target")
}

/*
check if target Kind is exist in target file ~/.garden/sessions/plantingSession/target
*/
func isTargeted(args ...string) bool {
	//if you modify isTargeted Please also modify IsTargetedTest,
	//as IsTargeted used for miscellaneous_test.go for teting purposes.
	var target Target
	ReadTarget(pathTarget, &target)

	//target stack is empty return true
	if len(args) == 0 && len(target.Stack()) == 0 {
		return true
	}

	targetMap := make(map[string]interface{})
	for _, t := range target.Stack() {
		targetMap[string(t.Kind)] = string(t.Name)
	}

	for _, t := range args {
		if _, ok := targetMap[t]; ok {
		} else {
			return false
		}
	}

	return true
}

//IsTargetedTest for isTargeted testing
func IsTargetedTest(targetReader TargetReader, args ...string) bool {
	target := targetReader.ReadTarget(pathTarget)

	//target stack is empty return true
	if len(args) == 0 && len(target.Stack()) == 0 {
		return true
	}

	targetMap := make(map[string]interface{})
	for _, t := range target.Stack() {
		targetMap[string(t.Kind)] = string(t.Name)
	}

	for _, t := range args {
		if _, ok := targetMap[t]; ok {
		} else {
			return false
		}
	}
	return true
}

//getFromTargetInfo validation value from global map targetInfo garden/project/shoot/seed/shootTechnicalID/....
func getFromTargetInfo(key string) string {
	getTargetMapInfo()
	value := targetInfo[key]
	if value == "" {
		log.Fatalf("value %s not found in targetInfo\n", key)
	}
	return value
}
