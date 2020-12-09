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
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardenerlogger "github.com/gardener/gardener/pkg/logger"
	slices "github.com/srfrog/slices"
	yaml "gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	red   = "\033[1;31m%s\033[0m"
	green = "\033[1;32m%s\033[0m"
)

// checkError checks if an error during execution occurred
func checkError(err error) {
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			log.Println(string(exiterr.Stderr))
		}

		if debugSwitch {
			_, fn, line, _ := runtime.Caller(1)
			log.Fatalf("[error] %s:%d \n %v", fn, line, err)
		} else {
			log.Fatalf(err.Error())
		}
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

//ExecCmdSaveOutputFile save command output to file
func ExecCmdSaveOutputFile(input []byte, cmd string, fileName string, environment ...string) (err error) {
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
	if _, err := os.Stat(fileName); err == nil {
		e := os.Remove(fileName)
		checkError(e)
	}
	outfile, err := os.Create(fileName)
	checkError(err)
	defer outfile.Close()
	command.Stdout = outfile
	command.Stderr = os.Stderr
	command.Stdin = stdin
	err = command.Run()
	if err != nil {
		os.Exit(2)
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
	pathOfKubeconfig := getKubeConfigOfCurrentTarget()
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
			fmt.Printf(green, "Kubeconfig under path "+pathOfKubeconfig+" contains auth provider configurations that could contain malicious code. Please only continue if you have verified it to be uncritical\n")
			return nil
			// 	return fmt.Errorf("auth provider configurations are not supported (user %q), these are the valid fields: %+v", user, validFields)
		case authInfo.Exec != nil:
			fmt.Printf(green, "Kubeconfig under path "+pathOfKubeconfig+" contains exec configurations that could contain malicious code. Please only continue if you have verified it to be uncritical\n")
			return nil
			// 	return fmt.Errorf("exec configurations are not supported (user %q), these are the valid fields: %+v", user, validFields)
		}
	}

	return nil
}

func md5sum(path string) string {
	md5 := md5.New()
	data, err := ioutil.ReadFile(path)
	checkError(err)
	_, err = md5.Write([]byte(data))
	checkError(err)
	return hex.EncodeToString(md5.Sum(nil))
}

func kubeConfigMd5sumInit(input bool, gardenConfig *GardenConfig) {
	if input {
		rewriteGardenKubeConfig()
	} else {
		os.Exit(0)
	}
}

func hashCheck(value string, gardenConfig *GardenConfig) bool {
	for _, items := range gardenConfig.GardenClusters {
		if items.TrustedKubeConfigMd5 == value {
			return true
		}
	}
	return false
}

//gardenKubeConfigHashCheck hash check
func gardenKubeConfigHashCheck() bool {
	var gardenConfig *GardenConfig
	pathOfKubeconfig := getKubeConfigOfClusterType("garden")
	md5sum := md5sum(pathOfKubeconfig)

	tempFile, err := ioutil.ReadFile(pathGardenConfig)
	checkError(err)
	err = yaml.Unmarshal(tempFile, &gardenConfig)
	checkError(err)

	switch gardenConfig.GardenClusters[0].TrustedKubeConfigMd5 {
	case "":
		for _, items := range gardenConfig.GardenClusters {
			data, err := ioutil.ReadFile(items.KubeConfig)
			checkError(err)
			clientConfig, err := clientcmd.NewClientConfigFromBytes(data)
			checkError(err)
			rawConfig, err := clientConfig.RawConfig()
			checkError(err)
			if err := ValidateClientConfig(rawConfig); err != nil {
				checkError(err)
			}
		}

		text := askForConfirmation()
		kubeConfigMd5sumInit(text, gardenConfig)
		return true

	default:
		if hashCheck(md5sum, gardenConfig) {
			return true
		}
	}
	fmt.Printf(red, "The Kubeconfig under path "+pathOfKubeconfig+" is difference compare with last time, Please check !!! \n")
	text := askForConfirmation()
	kubeConfigMd5sumInit(text, gardenConfig)
	return false
}

func askForConfirmation() bool {
	fmt.Printf(red, "Do you wants to trust this kubeconfig Y/N: ")
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	text = strings.ToLower(strings.TrimSpace(text))[0:1]
	if !(text == "y" || text == "n") {
		fmt.Println("Please Yes or No only")
		os.Exit(0)
	} else if text == "y" {
		return true
	}
	return false
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

//GardenctlDebugLog only outputs debug msg when gardencl -d or gardenctl --debug is specified
func GardenctlDebugLog(logMsg string) {
	if debugSwitch {
		var logger = gardenerlogger.NewLogger("debug")
		logger.Debugf(logMsg)
	}
}

//GardenctlInfoLog outputs information msg at all time
func GardenctlInfoLog(logMsg string) {
	var logger = gardenerlogger.NewLogger("info")
	logger.Infof(logMsg)
}

//CheckToolInstalled checks whether cliName is installed on local machine
func CheckToolInstalled(cliName string) bool {
	_, err := exec.LookPath(cliName)
	if err != nil {
		fmt.Println(cliName + " is not installed on your system")
		return false
	}
	return true
}

//PrintoutObject print object in yaml or json format. Pass os.Stdout if desired
func PrintoutObject(objectToPrint interface{}, writer io.Writer, outputFormat string) error {
	if outputFormat == "yaml" {
		yaml, err := yaml.Marshal(objectToPrint)
		if err != nil {
			return err
		}
		fmt.Fprint(writer, string(yaml))
	} else if outputFormat == "json" {
		json, err := json.MarshalIndent(objectToPrint, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprint(writer, string(json))
	} else {
		return errors.New("output format not supported: '" + outputFormat + "'")
	}
	return nil
}

//CheckIPPortReachable check whether IP with port is reachable within certain period of time
func CheckIPPortReachable(ip string, port string) error {
	attemptCount := 0
	for attemptCount < 12 {
		timeout := time.Second * 10
		conn, _ := net.DialTimeout("tcp", net.JoinHostPort(ip, port), timeout)
		if conn != nil {
			defer conn.Close()
			fmt.Printf("IP %s port %s is reachable\n", ip, port)
			return nil
		}
		fmt.Println("waiting for 10 seconds to retry")
		time.Sleep(time.Second * 10)
		attemptCount++
	}
	return fmt.Errorf("IP %s port %s is not reachable", ip, port)
}

//rewriteGardenKubeConfig with md5sum and comments
func rewriteGardenKubeConfig() {
	backupfile := pathGardenConfig + ".bak"
	backup(pathGardenConfig, backupfile)

	readerfile, err := os.OpenFile(backupfile, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	defer readerfile.Close()

	writefile, err := os.OpenFile(pathGardenConfig, os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal("Error: ", err)
	}
	defer writefile.Close()
	reader := bufio.NewReader(readerfile)
	writer := bufio.NewWriter(writefile)

	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		_, err = writer.WriteString(string(line) + "\n")
		if err != nil {
			log.Fatal("Error: ", err)
		}

		slc := strings.Split(string(line), " ")
		var tempLine string
		if slices.ContainsPrefix(slc, "kubeConfig:") {
			if slices.ContainsPrefix(slc, "#") {
				tempLine = "#  TrustedKubeConfigMd5: " + md5sum(slc[3])
			} else {
				tempLine = "  TrustedKubeConfigMd5: " + md5sum(slc[3])
			}

			_, err = writer.WriteString(tempLine + "\n")
			if err != nil {
				log.Fatal("Error: ", err)
			}
		}
	}
	writer.Flush()
}

//back up file
func backup(source string, destination string) {
	input, err := ioutil.ReadFile(source)
	if err != nil {
		fmt.Println("Error loading", source)
		log.Fatal(err)
	}

	err = ioutil.WriteFile(destination, input, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
