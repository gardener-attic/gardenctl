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
	"log"
	"os"
	"os/exec"
	"strings"

	openstackinstall "github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack/install"
	openstackv1alpha1 "github.com/gardener/gardener-extension-provider-openstack/pkg/apis/openstack/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/jmoiron/jsonq"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// operate executes a command on specified cli with pulled credentials for target
func operate(provider, arguments string) string {
	secretName, region, namespaceSecret, profile := "", "", "", ""
	var target Target
	ReadTarget(pathTarget, &target)
	var err error
	var secret *v1.Secret
	Client, err = clientToTarget("garden")
	checkError(err)

	gardenClientset, err := target.GardenerClient()
	checkError(err)
	var shoot *gardencorev1beta1.Shoot
	if target.Stack()[1].Kind == "project" {
		project, err := gardenClientset.CoreV1beta1().Projects().Get(target.Stack()[1].Name, metav1.GetOptions{})
		checkError(err)
		shoot, err = gardenClientset.CoreV1beta1().Shoots(*project.Spec.Namespace).Get(target.Stack()[2].Name, metav1.GetOptions{})
		checkError(err)
	} else if target.Stack()[1].Kind == "seed" {
		shootList, err := gardenClientset.CoreV1beta1().Shoots("").List(metav1.ListOptions{})
		checkError(err)
		var filteredShoots []gardencorev1beta1.Shoot
		for _, s := range shootList.Items {
			if s.Name == target.Stack()[2].Name && *s.Spec.SeedName == target.Stack()[1].Name {
				filteredShoots = append(filteredShoots, s)
			}
		}
		if len(filteredShoots) > 1 {
			fmt.Println("There's more than one shoot with same name " + target.Stack()[2].Name + " in seed " + target.Stack()[1].Name)
			fmt.Println("Please target the shoot using project")
			os.Exit(2)
		}
		shoot = &(filteredShoots[0])
	}

	secretBindingName := shoot.Spec.SecretBindingName
	region = shoot.Spec.Region
	namespaceSecretBinding := shoot.Namespace
	profile = shoot.Spec.CloudProfileName

	secretBinding, err := gardenClientset.CoreV1beta1().SecretBindings(namespaceSecretBinding).Get((secretBindingName), metav1.GetOptions{})
	checkError(err)

	secretName = secretBinding.SecretRef.Name
	namespaceSecret = secretBinding.SecretRef.Namespace

	secret, err = Client.CoreV1().Secrets(namespaceSecret).Get((secretName), metav1.GetOptions{})
	checkError(err)

	var out []byte
	switch provider {
	case "aws":
		accessKeyID := []byte(secret.Data["accessKeyID"])
		secretAccessKey := []byte(secret.Data["secretAccessKey"])
		args := strings.Fields(arguments)
		cmd := exec.Command("aws", args...)
		newEnv := append(os.Environ(), "AWS_ACCESS_KEY_ID="+string(accessKeyID[:]), "AWS_SECRET_ACCESS_KEY="+string(secretAccessKey[:]), "AWS_DEFAULT_REGION="+region, "AWS_DEFAULT_OUTPUT=text")
		cmd.Env = newEnv
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("AWS CLI failed with %s\n%s\n", out, err)
		}

	case "gcp":
		serviceaccount := []byte(secret.Data["serviceaccount.json"])
		data := map[string]interface{}{}
		var tmpAccount string

		tmpFile, err := ioutil.TempFile(os.TempDir(), "tmpFile-")
		checkError(err)
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.Write(serviceaccount)
		checkError(err)
		err = tmpFile.Close()
		checkError(err)
		tmpAccount, err = ExecCmdReturnOutput("bash", "-c", "gcloud config list account --format json")
		if err != nil {
			fmt.Println("Cmd was unsuccessful")
			os.Exit(2)
		}
		dec := json.NewDecoder(strings.NewReader(tmpAccount))
		err = dec.Decode(&data)
		checkError(err)
		jq := jsonq.NewQuery(data)
		tmpAccount, err = jq.String("core", "account")
		if err != nil {
			os.Exit(2)
		}
		err = ExecCmd(nil, "gcloud auth activate-service-account --key-file="+tmpFile.Name(), false)
		if err != nil {
			os.Exit(2)
		}
		dec = json.NewDecoder(strings.NewReader(string([]byte(secret.Data["serviceaccount.json"]))))
		err = dec.Decode(&data)
		checkError(err)
		jq = jsonq.NewQuery(data)
		account, err := jq.String("client_email")
		if err != nil {
			os.Exit(2)
		}
		project, err := jq.String("project_id")
		if err != nil {
			os.Exit(2)
		}

		arguments := arguments + " --account=" + account + " --project=" + project
		args := strings.Fields(arguments)
		cmd := exec.Command("gcloud", args...)
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("gcloud CLI failed with %s\n%s\n", out, err)
		}

		err = ExecCmd(nil, "gcloud config set account "+tmpAccount, false)
		if err != nil {
			os.Exit(2)
		}

	case "az":
		clientID := []byte(secret.Data["clientID"])
		clientSecret := []byte(secret.Data["clientSecret"])
		tenantID := []byte(secret.Data["tenantID"])
		subscriptionID := []byte(secret.Data["subscriptionID"])

		err := ExecCmd(nil, "az login --service-principal -u "+string(clientID[:])+" -p "+string(clientSecret[:])+" --tenant "+string(tenantID[:]), true)
		if err != nil {
			os.Exit(2)
		}

		arguments := arguments + " --subscription " + string(subscriptionID[:])
		args := strings.Fields(arguments)
		cmd := exec.Command("az", args...)
		out, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("az CLI failed with %s\n%s\n", out, err)
		}

	case "openstack":
		authURL := ""
		cloudProfileList, err := gardenClientset.CoreV1beta1().CloudProfiles().List(metav1.ListOptions{})
		checkError(err)
		for _, cp := range cloudProfileList.Items {
			if cp.Name == profile {
				cloudProfileConfig, err := getOpenstackCloudProfileConfig(&cp)
				checkError(err)
				authURL, err = getKeyStoneURL(cloudProfileConfig, region)
				checkError(err)
			}
		}
		domainName := []byte(secret.Data["domainName"])
		password := []byte(secret.Data["password"])
		tenantName := []byte(secret.Data["tenantName"])
		username := []byte(secret.Data["username"])

		args := strings.Fields(arguments)
		cmd := exec.Command("openstack", args...)
		newEnv := append(os.Environ(), "OS_IDENTITY_API_VERSION=3", "OS_AUTH_VERSION=3", "OS_AUTH_STRATEGY=keystone", "OS_AUTH_URL="+authURL, "OS_TENANT_NAME="+string(tenantName[:]), "OS_PROJECT_DOMAIN_NAME="+string(domainName[:]), "OS_USER_DOMAIN_NAME="+string(domainName[:]), "OS_USERNAME="+string(username[:]), "OS_PASSWORD="+string(password[:]), "OS_REGION_NAME="+region)
		cmd.Env = newEnv
		out, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
			log.Fatalf("Openstack CLI failed with %s\n%s\n", out, err)
		}

	case "aliyun":
		accessKeyID := []byte(secret.Data["accessKeyID"])
		accessKeySecret := []byte(secret.Data["accessKeySecret"])
		err = ExecCmd(nil, "aliyun configure set --access-key-id="+string(accessKeyID[:])+" --access-key-secret="+string(accessKeySecret[:])+" --region="+region, true)
		if err != nil {
			os.Exit(2)
		}
		err = ExecCmd(nil, arguments, false)
		if err != nil {
			os.Exit(2)
		}
	}
	return (strings.TrimSpace(string(out[:])))
}

func getOpenstackCloudProfileConfig(in *gardencorev1beta1.CloudProfile) (*openstackv1alpha1.CloudProfileConfig, error) {
	if in.Spec.ProviderConfig == nil {
		return nil, fmt.Errorf("cannot fetch providerConfig of core.gardener.cloud/v1alpha1.CloudProfile %s", in.Name)
	}

	extensionsScheme := runtime.NewScheme()
	err := openstackinstall.AddToScheme(extensionsScheme)
	checkError(err)
	decoder := serializer.NewCodecFactory(extensionsScheme).UniversalDecoder()

	out := &openstackv1alpha1.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: openstackv1alpha1.SchemeGroupVersion.String(),
			Kind:       "CloudProfileConfig",
		},
	}

	switch {
	case in.Spec.ProviderConfig.Object != nil:
		var ok bool
		out, ok = in.Spec.ProviderConfig.Object.(*openstackv1alpha1.CloudProfileConfig)
		if !ok {
			return nil, fmt.Errorf("cannot cast providerConfig of core.gardener.cloud/v1beta1.CloudProfile %s", in.Name)
		}
	case in.Spec.ProviderConfig.Raw != nil:
		if _, _, err := decoder.Decode(in.Spec.ProviderConfig.Raw, nil, out); err != nil {
			return nil, fmt.Errorf("cannot decode providerConfig of core.gardener.cloud/v1beta1.CloudProfile %s", in.Name)
		}
	}

	return out, nil
}

func getKeyStoneURL(config *openstackv1alpha1.CloudProfileConfig, region string) (string, error) {
	if config.KeyStoneURL != "" {
		return config.KeyStoneURL, nil
	}

	for _, keyStoneURL := range config.KeyStoneURLs {
		if keyStoneURL.Region == region {
			return keyStoneURL.URL, nil
		}
	}

	return "", fmt.Errorf("cannot find KeyStone URL for region %q in CloudProfileConfig", region)
}
