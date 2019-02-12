# Gardenctl

![](https://github.com/gardener/gardenctl/blob/master/logo/logo_gardener_cli_large.png)

[![Go Report Card](https://goreportcard.com/badge/github.com/gardener/gardenctl)](https://goreportcard.com/report/github.com/gardener/gardenctl)

# What is gardenctl?

`gardenctl` is a command-line client for administrative purposes for the [Gardener](https://github.com/gardener/gardener). It facilitates the administration of one or many garden, seed and shoot clusters, e.g. to check for issues which occured in one of these clusters. Details about the concept behind the Gardener are described in the [Gardener wiki](https://github.com/gardener/documentation/wiki/Architecture).

# Installation

`gardenctl` is shipped for mac and linux in a binary format. 

1. Download the latest release:
```bash
curl -LO https://github.com/gardener/gardenctl/releases/download/$(curl -s https://raw.githubusercontent.com/gardener/gardenctl/master/LATEST)/gardenctl-darwin-amd64
```

To download a specific version, replace the `$(curl -s https://raw.githubusercontent.com/gardener/gardenctl/master/LATEST)` portion of the command with the specific version.

For example, to download version 0.7.0 on macOS, type:
```bash
curl -LO https://github.com/gardener/gardenctl/releases/download/0.7.0/gardenctl-darwin-amd64
```

2. Make the gardenctl binary executable.
```bash
chmod +x ./gardenctl-darwin-amd64
```

3. Move the binary in to your PATH.
```bash
sudo mv ./gardenctl-darwin-amd64 /usr/local/bin/gardenctl
```

# How to build it

If no binary builds are available for your platform or architecture, you can build it from source,`go get` it or build the docker image from Dockerfile. Please keep in mind to use an up to date version of [golang](https://golang.org/doc/devel/release.html). 

## Prerequisites

To build `gardenctl` from sources you need to have a running Golang environment with `dep` as dependency management system. Moreover, since `gardenctl` allows to execute `kubectl` as well as a running `kubectl` installation is recommended, but not required. Please check this [description](https://github.com/gardener/gardener/blob/master/docs/development/local_setup.md) for further details.

## Build gardenctl 

### From source

First, you need to create a target folder structure before cloning and building `gardenctl`.

```bash
mkdir -p ~/go/src/github.com/gardener
cd ~/go/src/github.com/gardener
git clone https://github.com/gardener/gardenctl.git
cd gardenctl
go build gardenctl.go
```

In case dependencies are missing, run `dep ensure` and build `gardenctl` again via `go build gardenctl.go`.

After the successful build you get the executable `gardenctl` in the the directory `~/go/src/github.com/gardener/gardenctl`. Next, make it available by moving the executable to e.g. `/usr/local/bin`.

```bash
sudo mv gardenctl /usr/local/bin
```

`gardenctl` supports auto completion. This recommended feature is bound to `gardenctl` or the alias `g`. To configure it you can run:

```bash
echo "gardenctl completion && source gardenctl_completion.sh && rm gardenctl_completion.sh" >> ~/.bashrc
source ~/.bashrc
```

### Via go tools
First install `gardenctl` via the `go get` command.
```go
go get github.com/gardener/gardenctl
```

It will locate the binary under `$GOPATH/bin/gardenctl`. To generate the auto completion and add it to your `~/.bashrc` file, run the following command:

```bash
echo "$GOPATH/bin/gardenctl completion && source gardenctl_completion.sh && rm gardenctl_completion.sh" >> ~/.bashrc
```
### Via Dockerfile

First clone the repository as described in the the build step "From source". As next step add the garden "config" file and "clusters" folder with the corresponding kubeconfig files for the garden cluster. Then build the container image via `docker build -t gardener/gardenctl:v1 .` in the cloned repository and run a shell in the image with `docker run -it gardener/gardenctl:v1 /bin/bash`.

# Configure gardenctl

`gardenctl` requires a configuration file. The default location is in `~/.garden/config`, but it can be overwritten with the environment variable `GARDENCONFIG`.

Here an example file:
``` yaml
githubURL: https://github.location.company.corp
gardenClusters:
- name: dev
  kubeConfig: ~/clusters/dev/kubeconfig.yaml
- name: prod
  kubeConfig: ~/clusters/prod/kubeconfig.yaml
```

The path to the kubeconfig files of a garden cluster can be relative by using the ~ (tilde) expansion or absolute.

`gardenctl` caches some information, e.g. the garden project names. The location of this cache is per default `$GARDENCTL_HOME/cache`. If `GARDENCTL_HOME` is not set, `~/.garden` is assumed.

`gardenctl` makes it easy to get additional information of your IaaS provider by using the secrets stored in the corresponding projects in the Gardener. To use this functionality, the CLIs of the IaaS providers need to be available. 

Please check the IaaS provider documentation for more details about their CLIs.
  - [aws](https://aws.amazon.com/cli/)
  - [az](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
  - [gcloud](https://cloud.google.com/sdk/downloads)
  - [openstack](https://pypi.python.org/pypi/python-openstackclient)

Moreover, `gardenctl` offers auto completion. To use it, the command
```bash
gardenctl completion
``` 
creates the file `gardenctl_completion.sh` which can then be sourced later on via 
```bash
source gardenctl_completion.sh
```
Please keep in mind that the auto completion is bound to `gardenctl` or the alias `g`.

# Use gardenctl

`gardenctl` requires the definition of a target, e.g. garden, project, seed or shoot. The following commands, e.g. `gardenctl ls shoots` usees the target definition as a context for getting the information. 

Targets represent a hierarchical structure of resources. On top, there is/are the garden/s. E.g. in case you setup a development and a production garden, you would have two entries in your `~/.garden/config`. Via `gardenctl ls gardens` you get a list of the available gardens. 

- `gardenctl get target`   
  Displays the current target
- `gardenctl target [garden|project|seed|shoot]`   
  Set the target e.g. to a garden. It is as well possible to set the target directly to a element deeper in the hierarchy, e.g. to a shoot.
- `gardenctl drop target`   
  Drop the deepest target. 

## Examples of basic usage:

- List all seed cluster  
`gardenctl ls seeds`
- List all projects with shoot cluster  
`gardenctl ls projects`
- Target a seed cluster  
`gardenctl target seed-gce-dev`
- Target a project  
`gardenctl target garden-vora`
- Open prometheus ui for a targeted shoot-cluster  
`gardenctl show prometheus`
- Execute an aws command on a targeted aws shoot cluster  
`gardenctl aws ec2 describe-instances` or   
`gardenctl aws ec2 describe-instances --no-cache` without locally caching credentials
- Target a shoot directly and get all kube-dns pods in kube-system namespace  
`gardenctl target myshoot`  
`gardenctl kubectl get pods -- -n kube-system | grep kube-dns`  
- List all cluster with an issue  
`gardenctl ls issues`
- Drop an element from target stack  
`gardenctl drop`
- Open a shell to a cluster node  
`gardenctl shell nodename`
- Show logs from elasticsearch  
`gardenctl logs etcd-main --elasticsearch`
- Show last 100 logs from elasticsearch from the last 2 hours  
`gardenctl logs etcd-main --elasticsearch --since=2h --tail=100`

## Advanced usage based on JsonQuery

The following examples are based on [jq](https://stedolan.github.io/jq/). The [Json Query Playground](https://jqplay.org/jq?q=.%5B%5D&j=%5B%5D) offers a convenient environment to test the queries.

Below a list of examples:

- List the project name, shoot name and the state for all projects with issues
```bash
gardenctl ls issues -o json | jq '.issues[] | { project: .project, shoot: .shoot, state: .status.lastOperation.state }'
```
- Print all issues of a single project e.g. `garden-myproject`
```bash
gardenctl ls issues -o json | jq '.issues[] | if (.project=="garden-myproject") then . else empty end' 
```
- Print all issues with error state "Error"
```bash
gardenctl ls issues -o json | jq '.issues[] | if (.status.lastOperation.state=="Error") then . else empty end'
```
- Print all issues with error state not equal "Succeded"
```bash
gardenctl ls issues -o json | jq '.issues[] | if (.status.lastOperation.state!="Succeeded") then . else empty end'
```
- Print `createdBy` information (typically email addresses) of all shoots
```bash
gardenctl k get shoots -- -n garden-core -o json | jq -r ".items[].metadata | {email: .annotations.\"garden.sapcloud.io/createdBy\", name: .name, namespace: .namespace}"
```

Here a few on cluster analysis:

- Which states are there and how many clusters are in this state?
```bash 
gardenctl ls issues -o json | jq '.issues | group_by( .status.lastOperation.state ) | .[] | {state:.[0].status.lastOperation.state, count:length}'
 ```

- Get all clusters in state `Failed`
```bash
gardenctl ls issues -o json | jq '.issues[] | if (.status.lastOperation.state=="Failed") then . else empty end'
```
