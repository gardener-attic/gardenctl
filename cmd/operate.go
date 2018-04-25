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
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	sapcloud "github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/jmoiron/jsonq"
	yaml "gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// removeOldEntryAWS removes old credential entry for gardenctl if existing
func removeOldEntryAWS(filePath, contains string) {
	input, err := ioutil.ReadFile(filePath)
	checkError(err)
	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		if strings.Contains(line, contains) {
			lines = append(lines[:i], lines[i+3:]...)
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(filePath, []byte(output), 0644)
	checkError(err)
}

// operate executes a command on specified cli with pulled credentials for target
func operate(provider, arguments string) {
	secretName, region := "", ""
	namespaceSecret := ""
	var target Target
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, &target)
	checkError(err)
	Client, err = clientToTarget("garden")
	k8sGardenClient, err := sapcloud.NewClientFromFile(*kubeconfig)
	checkError(err)
	gardenClientset, err := clientset.NewForConfig(k8sGardenClient.GetConfig())
	checkError(err)
	k8sGardenClient.SetGardenClientset(gardenClientset)
	shootList, err := k8sGardenClient.GetGardenClientset().GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	for _, shoot := range shootList.Items {
		if shoot.Name == target.Target[2].Name {
			secretName = shoot.Spec.Cloud.SecretBindingRef.Name
			region = shoot.Spec.Cloud.Region
			namespaceSecret = shoot.Namespace
		}
	}
	secret, err := Client.CoreV1().Secrets(namespaceSecret).Get((secretName), metav1.GetOptions{})
	checkError(err)
	switch provider {
	case "aws":
		accessKeyID := []byte(secret.Data["accessKeyID"])
		secretAccessKey := []byte(secret.Data["secretAccessKey"])
		if !cachevar {
			awsPathCredentials := ""
			awsPathConfig := ""
			if target.Target[1].Kind == "project" {
				createDir(pathGardenHome+"/cache/projects/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.aws", 0751)
				awsPathCredentials = "cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.aws/credentials"
				awsPathConfig = "cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.aws/config"
			} else if target.Target[1].Kind == "seed" {
				createDir(pathGardenHome+"/cache/seeds/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.aws", 0751)
				awsPathCredentials = "cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.aws/credentials"
				awsPathConfig = "cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.aws/config"
			}
			createFile(pathGardenHome+"/"+awsPathCredentials, 0644)
			createFile(pathGardenHome+"/"+awsPathConfig, 0644)
			removeOldEntryAWS(pathGardenHome+"/"+awsPathCredentials, "[gardenctl]")
			removeOldEntryAWS(pathGardenHome+"/"+awsPathConfig, "[profile gardenctl]")
			credentials := "[gardenctl]\n" + "aws_access_key_id=" + string(accessKeyID[:]) + "\n" + "aws_secret_access_key=" + string(secretAccessKey[:]) + "\n"
			originalCredentials, err := os.OpenFile(pathGardenHome+"/"+awsPathCredentials, os.O_APPEND|os.O_WRONLY, 0644)
			checkError(err)
			_, err = originalCredentials.WriteString(credentials)
			checkError(err)
			originalCredentials.Close()
			config := "[profile gardenctl]\n" + "region=" + region + "\n" + "output=text\n"
			originalConfig, err := os.OpenFile(pathGardenHome+"/"+awsPathConfig, os.O_APPEND|os.O_WRONLY, 0644)
			_, err = originalConfig.WriteString(config)
			originalConfig.Close()
			checkError(err)
		}
		err := execCmd(arguments, false, "AWS_ACCESS_KEY_ID="+string(accessKeyID[:]), "AWS_SECRET_ACCESS_KEY="+string(secretAccessKey[:]), "AWS_DEFAULT_REGION="+region, "AWS_DEFAULT_OUTPUT=text")
		if err != nil {
			os.Exit(2)
		}
	case "gcp":
		serviceaccount := []byte(secret.Data["serviceaccount.json"])
		data := map[string]interface{}{}
		var tmpAccount string
		if !cachevar {
			gcpPathCredentials := ""
			if target.Target[1].Kind == "project" {
				createDir(pathGardenHome+"/cache/projects/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.gcp", 0751)
				gcpPathCredentials = "cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.gcp/credentials"
			} else if target.Target[1].Kind == "seed" {
				createDir(pathGardenHome+"/cache/seeds/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.gcp", 0751)
				gcpPathCredentials = "cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.gcp/credentials"
			}
			createFile(pathGardenHome+"/"+gcpPathCredentials, 0644)
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, gcpPathCredentials), os.O_WRONLY, 0644)
			checkError(err)
			_, err = originalCredentials.WriteString(string(serviceaccount))
			originalCredentials.Close()
			checkError(err)
			tmpAccount = execCmdReturnOutput("gcloud config list account --format json")
			dec := json.NewDecoder(strings.NewReader(tmpAccount))
			dec.Decode(&data)
			jq := jsonq.NewQuery(data)
			tmpAccount, err = jq.String("core", "account")
			if err != nil {
				os.Exit(2)
			}
			err = execCmd("gcloud auth activate-service-account --key-file="+pathGardenHome+"/"+gcpPathCredentials, false)
			if err != nil {
				os.Exit(2)
			}
		}
		dec := json.NewDecoder(strings.NewReader(string([]byte(secret.Data["serviceaccount.json"]))))
		dec.Decode(&data)
		jq := jsonq.NewQuery(data)
		account, err := jq.String("client_email")
		if err != nil {
			os.Exit(2)
		}
		project, err := jq.String("project_id")
		if err != nil {
			os.Exit(2)
		}
		err = execCmd(arguments+" --account="+account+" --project="+project, false)
		if err != nil {
			os.Exit(2)
		}
		err = execCmd("gcloud config set account "+tmpAccount, false)
		if err != nil {
			os.Exit(2)
		}

	case "az":
		clientID := []byte(secret.Data["clientID"])
		clientSecret := []byte(secret.Data["clientSecret"])
		tenantID := []byte(secret.Data["tenantID"])
		if !cachevar {
			azurePathCredentials := ""
			if target.Target[1].Kind == "project" {
				createDir(pathGardenHome+"/cache/projects/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.azure", 0751)
				azurePathCredentials = "cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.azure/credentials"
			} else if target.Target[1].Kind == "seed" {
				createDir(pathGardenHome+"/cache/seeds/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.azure", 0751)
				azurePathCredentials = "cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.azure/credentials"
			}
			createFile(pathGardenHome+"/"+azurePathCredentials, 0644)
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, azurePathCredentials), os.O_WRONLY, 0644)
			checkError(err)
			credentials := "clientID: " + string(clientID[:]) + "\n" + "clientSecret: " + string(clientSecret[:]) + "\n" + "tenantID: " + string(tenantID[:]) + "\n"
			_, err = originalCredentials.WriteString(credentials)
			originalCredentials.Close()
			checkError(err)
		}
		err := execCmd("az login --service-principal -u "+string(clientID[:])+" -p "+string(clientSecret[:])+" --tenant "+string(tenantID[:]), true)
		if err != nil {
			os.Exit(2)
		}
		err = execCmd(arguments, false)
		if err != nil {
			os.Exit(2)
		}
	case "openstack":
		authUrl := []byte(secret.Data["authUrl"])
		domainName := []byte(secret.Data["domainName"])
		password := []byte(secret.Data["password"])
		tenantName := []byte(secret.Data["tenantName"])
		username := []byte(secret.Data["username"])
		if !cachevar {
			openstackPathCredentials := ""
			if target.Target[1].Kind == "project" {
				createDir(pathGardenHome+"/cache/projects/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.openstack", 0751)
				openstackPathCredentials = "cache/projects/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.openstack/credentials"
			} else if target.Target[1].Kind == "seed" {
				createDir(pathGardenHome+"/cache/seeds/"+target.Target[1].Name+"/"+target.Target[2].Name+"/.openstack", 0751)
				openstackPathCredentials = "cache/seeds/" + target.Target[1].Name + "/" + target.Target[2].Name + "/.openstack/credentials"
			}
			createFile(pathGardenHome+"/"+openstackPathCredentials, 0644)
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, openstackPathCredentials), os.O_WRONLY, 0644)
			checkError(err)
			credentials := "authUrl: " + string(authUrl[:]) + "\n" + "domainName: " + string(domainName[:]) + "\n" + "password: " + string(password[:]) + "\n" + "tenantName: " + string(tenantName[:]) + "\n" + "username: " + string(username[:]) + "\n"
			_, err = originalCredentials.WriteString(credentials)
			originalCredentials.Close()
			checkError(err)
		}
		err := execCmd(arguments, false, "OS_IDENTITY_API_VERSION=3", "OS_AUTH_VERSION=3", "OS_AUTH_STRATEGY=keystone", "OS_AUTH_URL="+string(authUrl[:]), "OS_TENANT_NAME="+string(tenantName[:]),
			"OS_PROJECT_DOMAIN_NAME="+string(domainName[:]), "OS_USER_DOMAIN_NAME="+string(domainName[:]), "OS_USERNAME="+string(username[:]), "OS_PASSWORD="+string(password[:]), "OS_REGION_NAME="+region)
		if err != nil {
			os.Exit(2)
		}
	}
}
