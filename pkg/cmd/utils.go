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
	"os/exec"
	"path/filepath"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// checkError checks if an error during execution occurred
func checkError(err error) {
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
}

// HomeDir returns homedirectory of user
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

// FileExists check if the directory exists
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// CreateDir creates a directory if it does no exist
func CreateDir(dirname string, permission os.FileMode) {
	exists, err := FileExists(dirname)
	checkError(err)
	if !exists {
		err := os.MkdirAll(dirname, permission)
		checkError(err)
	}
}

// CreateFileIfNotExists creates an empty file if it does not exist
func CreateFileIfNotExists(filename string, permission os.FileMode) {
	exists, err := FileExists(filename)
	checkError(err)
	if !exists {
		err = ioutil.WriteFile(filename, []byte{}, permission)
		checkError(err)
	}
}

// ExecCmd executes a command within set environment
func ExecCmd(input []byte, cmd string, suppressedOutput bool, environment ...string) (err error) {
	var command *exec.Cmd
	parts := strings.Fields(cmd)
	head := parts[0]
	if len(parts) > 1 {
		parts = parts[1:]
	} else {
		parts = nil
	}
	command = exec.Command(head, parts...)
	for index, env := range environment {
		if index == 0 {
			command.Env = append(os.Environ(),
				env,
			)
		} else {
			command.Env = append(command.Env,
				env,
			)
		}
	}
	var stdin = os.Stdin
	if input != nil {
		r, w, err := os.Pipe()
		if err != nil {
			return err
		}
		defer r.Close()
		go func() {
			_, err = w.Write([]byte(input))
			checkError(err)
			w.Close()
		}()
		stdin = r
	}
	if suppressedOutput {
		err = command.Run()
		if err != nil {
			os.Exit(2)
		}
	} else {
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = stdin
		err = command.Run()
		if err != nil {
			os.Exit(2)
		}
	}
	return nil
}

// ExecCmdReturnOutput execute cmd and return output
func ExecCmdReturnOutput(cmd string, args ...string) (output string, err error) {
	out, err := exec.Command(cmd, args...).Output()
	return strings.TrimSpace(string(out[:])), err
}

// ReadTarget file into Target
// DEPRECATED: Use `TargetReader` instead.
func ReadTarget(pathTarget string, target *Target) {
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, target)
	checkError(err)
}

// NewConfigFromBytes returns a client from the given kubeconfig path
func NewConfigFromBytes(kubeconfig string) *restclient.Config {
	kubecf, err := ioutil.ReadFile(kubeconfig)
	checkError(err)
	configObj, err := clientcmd.Load(kubecf)
	if err != nil {
		fmt.Println("Could not load config")
		os.Exit(2)
	}
	clientConfig := clientcmd.NewDefaultClientConfig(*configObj, &clientcmd.ConfigOverrides{})
	client, err := clientConfig.ClientConfig()
	checkError(err)
	return client
}

// ValidateClientConfig validates that the auth info of a given kubeconfig doesn't have unsupported fields.
func ValidateClientConfig(config clientcmdapi.Config) error {
	validFields := []string{"client-certificate-data", "client-key-data", "token", "username", "password"}

	for user, authInfo := range config.AuthInfos {
		switch {
		case authInfo.ClientCertificate != "":
			return fmt.Errorf("client certificate files are not supported (user %q), these are the valid fields: %+v", user, validFields)
		case authInfo.ClientKey != "":
			return fmt.Errorf("client key files are not supported (user %q), these are the valid fields: %+v", user, validFields)
		case authInfo.TokenFile != "":
			return fmt.Errorf("token files are not supported (user %q), these are the valid fields: %+v", user, validFields)
		case authInfo.Impersonate != "" || len(authInfo.ImpersonateGroups) > 0:
			return fmt.Errorf("impersonation is not supported, these are the valid fields: %+v", validFields)
		case authInfo.AuthProvider != nil && len(authInfo.AuthProvider.Config) > 0:
			return fmt.Errorf("auth provider configurations are not supported (user %q), these are the valid fields: %+v", user, validFields)
		case authInfo.Exec != nil:
			return fmt.Errorf("exec configurations are not supported (user %q), these are the valid fields: %+v", user, validFields)
		}
	}

	return nil
}

// FetchShootFromTarget fetches shoot object from given target
func FetchShootFromTarget(target TargetInterface) (*gardencorev1beta1.Shoot, error) {
	gardenClientset, err := target.GardenerClient()
	if err != nil {
		return nil, err
	}

	var shoot *gardencorev1beta1.Shoot
	if target.Stack()[1].Kind == TargetKindProject {
		project, err := gardenClientset.CoreV1beta1().Projects().Get(target.Stack()[1].Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

		shoot, err = gardenClientset.CoreV1beta1().Shoots(*project.Spec.Namespace).Get(target.Stack()[2].Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		shootList, err := gardenClientset.CoreV1beta1().Shoots(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for index, s := range shootList.Items {
			if s.Name == target.Stack()[2].Name && *s.Spec.SeedName == target.Stack()[1].Name {
				shoot = &shootList.Items[index]
				break
			}
		}
	}

	return shoot, nil
}

//TidyKubeconfigWithHomeDir check if kubeconfig path contains ~, replace ~ with user home dir
func TidyKubeconfigWithHomeDir(pathToKubeconfig string) string {
	if strings.Contains(pathToKubeconfig, "~") {
		pathToKubeconfig = filepath.Clean(filepath.Join(HomeDir(), strings.Replace(pathToKubeconfig, "~", "", 1)))
	}
	return pathToKubeconfig
}

//CheckShootIsTargeted check if current target has shoot targeted
func CheckShootIsTargeted(target TargetInterface) bool {
	if (len(target.Stack()) < 3) || (target.Stack()[len(target.Stack())-1].Kind == "namespace" && target.Stack()[len(target.Stack())-2].Kind != "shoot") {
		return false
	}
	return true
}
