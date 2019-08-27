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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	clientset "github.com/gardener/gardener/pkg/client/garden/clientset/versioned"
	"github.com/jmoiron/jsonq"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// removeOldEntry removes old credential and config entry for gardenctl if existing
func removeOldEntry(filePath, contains string) {
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
	profile := ""
	var target Target
	ReadTarget(pathTarget, &target)
	Client, err = clientToTarget("garden")
	gardenClientset, err := clientset.NewForConfig(NewConfigFromBytes(*kubeconfig))
	checkError(err)
	shootList, err := gardenClientset.GardenV1beta1().Shoots("").List(metav1.ListOptions{})
	for _, shoot := range shootList.Items {
		if shoot.Name == target.Target[2].Name {
			secretBindingName := shoot.Spec.Cloud.SecretBindingRef.Name
			region = shoot.Spec.Cloud.Region
			namespaceSecretBinding := shoot.Namespace
			profile = shoot.Spec.Cloud.Profile
			secretBinding, err := gardenClientset.GardenV1beta1().SecretBindings(namespaceSecretBinding).Get((secretBindingName), metav1.GetOptions{})
			checkError(err)
			secretName = secretBinding.SecretRef.Name
			namespaceSecret = secretBinding.SecretRef.Namespace
		}
	}
	secret, err := Client.CoreV1().Secrets(namespaceSecret).Get((secretName), metav1.GetOptions{})
	checkError(err)

	gardenName := target.Stack()[0].Name
	projectsPath := filepath.Join("cache", gardenName, "projects", target.Target[1].Name, target.Target[2].Name)
	seedsPath := filepath.Join("cache", gardenName, "seeds", target.Target[1].Name, target.Target[2].Name)

	switch provider {
	case "aws":
		accessKeyID := []byte(secret.Data["accessKeyID"])
		secretAccessKey := []byte(secret.Data["secretAccessKey"])
		if !cachevar {
			awsPathCredentials := ""
			awsPathConfig := ""
			if target.Target[1].Kind == "project" {
				CreateDir(filepath.Join(pathGardenHome, projectsPath, ".aws"), 0751)
				awsPathCredentials = filepath.Join(projectsPath, ".aws", "credentials")
				awsPathConfig = filepath.Join(projectsPath, ".aws", "config")
			} else if target.Target[1].Kind == "seed" {
				CreateDir(filepath.Join(pathGardenHome, seedsPath, ".aws"), 0751)
				awsPathCredentials = filepath.Join(seedsPath, ".aws", "credentials")
				awsPathConfig = filepath.Join(seedsPath, ".aws", "config")
			}
			CreateFileIfNotExists(filepath.Join(pathGardenHome, awsPathCredentials), 0644)
			CreateFileIfNotExists(filepath.Join(pathGardenHome, awsPathConfig), 0644)
			removeOldEntry(filepath.Join(pathGardenHome, awsPathCredentials), "[gardenctl]")
			removeOldEntry(filepath.Join(pathGardenHome, awsPathConfig), "[profile gardenctl]")
			credentials := "[gardenctl]\n" + "aws_access_key_id=" + string(accessKeyID[:]) + "\n" + "aws_secret_access_key=" + string(secretAccessKey[:]) + "\n"
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, awsPathCredentials), os.O_APPEND|os.O_WRONLY, 0644)
			checkError(err)
			_, err = originalCredentials.WriteString(credentials)
			checkError(err)
			originalCredentials.Close()
			config := "[profile gardenctl]\n" + "region=" + region + "\n" + "output=text\n"
			originalConfig, err := os.OpenFile(filepath.Join(pathGardenHome, awsPathConfig), os.O_APPEND|os.O_WRONLY, 0644)
			_, err = originalConfig.WriteString(config)
			originalConfig.Close()
			checkError(err)
		}
		err := ExecCmd(nil, arguments, false, "AWS_ACCESS_KEY_ID="+string(accessKeyID[:]), "AWS_SECRET_ACCESS_KEY="+string(secretAccessKey[:]), "AWS_DEFAULT_REGION="+region, "AWS_DEFAULT_OUTPUT=text")
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
				CreateDir(filepath.Join(pathGardenHome, projectsPath, ".gcp"), 0751)
				gcpPathCredentials = filepath.Join(projectsPath, ".gcp", "credentials")
			} else if target.Target[1].Kind == "seed" {
				CreateDir(filepath.Join(pathGardenHome, seedsPath, ".gcp"), 0751)
				gcpPathCredentials = filepath.Join(seedsPath, ".gcp", "credentials")
			}
			CreateFileIfNotExists(filepath.Join(pathGardenHome, gcpPathCredentials), 0644)
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, gcpPathCredentials), os.O_WRONLY, 0644)
			checkError(err)
			_, err = originalCredentials.WriteString(string(serviceaccount))
			originalCredentials.Close()
			checkError(err)
			tmpAccount, err = ExecCmdReturnOutput("bash", "-c", "gcloud config list account --format json")
			if err != nil {
				fmt.Println("Cmd was unsuccessful")
				os.Exit(2)
			}
			dec := json.NewDecoder(strings.NewReader(tmpAccount))
			dec.Decode(&data)
			jq := jsonq.NewQuery(data)
			tmpAccount, err = jq.String("core", "account")
			if err != nil {
				os.Exit(2)
			}
			err = ExecCmd(nil, "gcloud auth activate-service-account --key-file="+filepath.Join(pathGardenHome, gcpPathCredentials), false)
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
		err = ExecCmd(nil, arguments+" --account="+account+" --project="+project, false)
		if err != nil {
			os.Exit(2)
		}
		err = ExecCmd(nil, "gcloud config set account "+tmpAccount, false)
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
				CreateDir(filepath.Join(pathGardenHome, projectsPath, ".azure"), 0751)
				azurePathCredentials = filepath.Join(projectsPath, ".azure", "credentials")
			} else if target.Target[1].Kind == "seed" {
				CreateDir(filepath.Join(pathGardenHome, seedsPath, ".azure"), 0751)
				azurePathCredentials = filepath.Join(seedsPath, ".azure", "credentials")
			}
			CreateFileIfNotExists(filepath.Join(pathGardenHome, azurePathCredentials), 0644)
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, azurePathCredentials), os.O_WRONLY, 0644)
			checkError(err)
			credentials := "clientID: " + string(clientID[:]) + "\n" + "clientSecret: " + string(clientSecret[:]) + "\n" + "tenantID: " + string(tenantID[:]) + "\n"
			_, err = originalCredentials.WriteString(credentials)
			originalCredentials.Close()
			checkError(err)
		}
		err := ExecCmd(nil, "az login --service-principal -u "+string(clientID[:])+" -p "+string(clientSecret[:])+" --tenant "+string(tenantID[:]), true)
		if err != nil {
			os.Exit(2)
		}
		err = ExecCmd(nil, arguments, false)
		if err != nil {
			os.Exit(2)
		}
	case "openstack":
		authURL := ""
		cloudProfileList, err := gardenClientset.GardenV1beta1().CloudProfiles().List(metav1.ListOptions{})
		for _, cp := range cloudProfileList.Items {
			if cp.Name == profile {
				authURL = cp.Spec.OpenStack.KeyStoneURL
			}
		}
		domainName := []byte(secret.Data["domainName"])
		password := []byte(secret.Data["password"])
		tenantName := []byte(secret.Data["tenantName"])
		username := []byte(secret.Data["username"])
		if !cachevar {
			openstackPathCredentials := ""
			if target.Target[1].Kind == "project" {
				CreateDir(filepath.Join(pathGardenHome, projectsPath, ".openstack"), 0751)
				openstackPathCredentials = filepath.Join(projectsPath, ".openstack", "credentials")
			} else if target.Target[1].Kind == "seed" {
				CreateDir(filepath.Join(pathGardenHome, seedsPath, ".openstack"), 0751)
				openstackPathCredentials = filepath.Join(seedsPath, ".openstack", "credentials")
			}
			CreateFileIfNotExists(filepath.Join(pathGardenHome, openstackPathCredentials), 0644)
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, openstackPathCredentials), os.O_WRONLY, 0644)
			checkError(err)
			credentials := "authURL: " + authURL + "\n" + "domainName: " + string(domainName[:]) + "\n" + "password: " + string(password[:]) + "\n" + "tenantName: " + string(tenantName[:]) + "\n" + "username: " + string(username[:]) + "\n"
			_, err = originalCredentials.WriteString(credentials)
			originalCredentials.Close()
			checkError(err)
		}
		err = ExecCmd(nil, arguments, false, "OS_IDENTITY_API_VERSION=3", "OS_AUTH_VERSION=3", "OS_AUTH_STRATEGY=keystone", "OS_AUTH_URL="+authURL, "OS_TENANT_NAME="+string(tenantName[:]),
			"OS_PROJECT_DOMAIN_NAME="+string(domainName[:]), "OS_USER_DOMAIN_NAME="+string(domainName[:]), "OS_USERNAME="+string(username[:]), "OS_PASSWORD="+string(password[:]), "OS_REGION_NAME="+region)
		if err != nil {
			os.Exit(2)
		}
	case "aliyun":
		accessKeyID := []byte(secret.Data["accessKeyID"])
		accessKeySecret := []byte(secret.Data["accessKeySecret"])
		if !cachevar {
			aliyunPathCredentials := ""
			aliyunPathConfig := ""
			if target.Target[1].Kind == "project" {
				CreateDir(filepath.Join(pathGardenHome, projectsPath, ".aliyun"), 0751)
				aliyunPathCredentials = filepath.Join(projectsPath, ".aliyun", "credentials")
				aliyunPathConfig = filepath.Join(projectsPath, ".aliyun", "config")
			} else if target.Target[1].Kind == "seed" {
				CreateDir(filepath.Join(pathGardenHome, seedsPath, ".aliyun"), 0751)
				aliyunPathCredentials = filepath.Join(seedsPath, ".aliyun", "credentials")
				aliyunPathConfig = filepath.Join(seedsPath, ".aliyun", "config")
			}
			CreateFileIfNotExists(filepath.Join(pathGardenHome, aliyunPathCredentials), 0644)
			CreateFileIfNotExists(filepath.Join(pathGardenHome, aliyunPathConfig), 0644)
			removeOldEntry(filepath.Join(pathGardenHome, aliyunPathCredentials), "[gardenctl]")
			removeOldEntry(filepath.Join(pathGardenHome, aliyunPathConfig), "[profile gardenctl]")
			credentials := "[gardenctl]\n" + "accessKeyId=" + string(accessKeyID[:]) + "\n" + "accessKeySecret=" + string(accessKeySecret[:]) + "\n"
			originalCredentials, err := os.OpenFile(filepath.Join(pathGardenHome, aliyunPathCredentials), os.O_APPEND|os.O_WRONLY, 0644)
			checkError(err)
			defer originalCredentials.Close()
			_, err = originalCredentials.WriteString(credentials)
			checkError(err)
			config := "[profile gardenctl]\n" + "region=" + region + "\n" + "output=json\n"
			originalConfig, err := os.OpenFile(filepath.Join(pathGardenHome, aliyunPathConfig), os.O_APPEND|os.O_WRONLY, 0644)
			checkError(err)
			defer originalConfig.Close()
			_, err = originalConfig.WriteString(config)
			checkError(err)
		}
		err = ExecCmd(nil, "aliyun configure set --access-key-id="+string(accessKeyID[:])+" --access-key-secret="+string(accessKeySecret[:])+" --region="+region, true)
		if err != nil {
			os.Exit(2)
		}
		err = ExecCmd(nil, arguments, false)
		if err != nil {
			os.Exit(2)
		}
	}
}
