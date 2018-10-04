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
	"strings"

	yaml "gopkg.in/yaml.v2"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
		os.MkdirAll(dirname, permission)
	}
}

// CreateFileIfNotExists creates an empty file if it does not exist
func CreateFileIfNotExists(filename string, permission os.FileMode) {
	exists, err := FileExists(filename)
	checkError(err)
	if !exists {
		ioutil.WriteFile(filename, []byte{}, permission)
	}
}

// ExecCmd executes a command within set environment
func ExecCmd(input []byte, cmd string, suppressedOutput bool, environment ...string) (err error) {
	var command *exec.Cmd
	parts := strings.Fields(cmd)
	head := parts[0]
	if len(parts) > 1 {
		parts = parts[1:len(parts)]
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
			w.Write([]byte(input))
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
func ReadTarget(pathTarget string, target *Target) {
	targetFile, err := ioutil.ReadFile(pathTarget)
	checkError(err)
	err = yaml.Unmarshal(targetFile, target)
	checkError(err)
}

func NewConfigFromBytes(kubeconfig string) *restclient.Config {
	kubecf, err := ioutil.ReadFile(kubeconfig)
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
