// Copyright 2018 The Gardener Authors.
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
)

// checkError checks if an error during execution occured
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

// fileExists check if the directory exists
func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// createDir creates a directory if it does no exist
func createDir(dirname string, permission os.FileMode) {
	exists, err := fileExists(dirname)
	checkError(err)
	if !exists {
		os.MkdirAll(dirname, permission)
	}
}

// createFile creates an empty file if it does no exist
func createFile(filename string, permission os.FileMode) {
	exists, err := fileExists(filename)
	checkError(err)
	if !exists {
		ioutil.WriteFile(filename, []byte{}, permission)
	}
}

// execCmd executes a command within set environment
func execCmd(cmd string, suppressedOutput bool, environment ...string) (err error) {
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
	if suppressedOutput {
		err = command.Run()
		if err != nil {
			return err
		}
	} else {
		val, err := command.Output()
		if err != nil {
			ee, ok := err.(*exec.ExitError)
			fmt.Println(string(ee.Stderr))
			if !ok {
				return err
			}
		}
		fmt.Println(string(val))
	}
	return nil
}

// execCmdReturnOutput executes a command within set environment and returns output
func execCmdReturnOutput(cmd string, environment ...string) (output string) {
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
	out, err := command.Output()
	checkError(err)
	return string(out[:])
}

// getKubeconfig returns path to kubeconfig
func getKubeconfig() (pathToKubeconfig string) {
	env := os.Environ()
	pathToKubeconfig = ""
	for _, val := range env {
		if strings.Contains(val, "KUBECONFIG=") {
			pathToKubeconfig = strings.Trim(val, "KUBECONFIG=")
		}
	}
	return pathToKubeconfig
}
