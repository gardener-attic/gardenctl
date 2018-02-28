# Gardenctl

![](https://github.com/gardener/gardenctl/blob/master/logo/logo_gardener_cli_large.png)

# What is gardenctl?
`gardenctl` is a command-line client for administrative purposes for the [Gardener](https://github.com/gardener/gardener). It facilitates the administration of even a big amount of Garden, Seed and Shoot clusters e.g. to check for any issue which occured in one of these systems. Details about the concept behind the Gardener are described in this [Gardener wiki](https://github.com/gardener/documentation/wiki/Architecture).



# How to build it
Currently, there are no binary builds available, so you need to build it from source. 

## Prerequisites

To build `gardenctl` from sources you need to have a running Golang environment with `dep` as dependency management system. Moreover, since `gardenctl` allows to execute `kubectl` as well as a running `kubectl` installation is recommended, but not required. Please check this [description](https://github.com/gardener/gardener/blob/master/docs/development/local_setup.md) for further details.

## Build gardenctl from source

First, you need to create a target folder structure before cloning and building `gardenctl`.

```bash
mkdir -p ~/go/src/github.com/gardener
cd ~/go/src/github.com/gardener
git clone https://github.com/gardener/gardenctl.git
cd gardenctl
go build gardenctl.go
```

In case dependencies are missing, run `dep ensure` and build `gardenctl` again via `go build gardenctl.go`.

After the successful build you get the executable `gardenctl` in the the directory `~/go/src/github.com/gardener`. Next, make it available by moving the executable to e.g. `/usr/local/bin`.

```bash
sudo mv gardenctl /usr/local/bin
```

`gardenctl` allows like `kubectl` command completion. This recommended feature is bound to the alias `g`. To configure it you could e.g. run
```bash
echo "gardenctl completion && source gardenctl_completion.sh && rm gardenctl_completion.sh" >> ~/.bashrc
echo "alias g=gardenctl" >> ~/.bashrc
source ~/.bashrc
```

## Configure gardenctl

`gardenctl` requires a configuration file, e.g. 
``` yaml
gardenClusters:
- name: dev
  kubeConfig: ~/clusters/garden-dev/kubeconfig.yaml
- name: prod
  kubeConfig: /Users/d123456/clusters/garden-prod/kubeconfig.yaml
```
The path to the kubeconfig file of a garden cluster can be relative by using the ~ (tilde) expansion or absolute.

The default location and name is of the `gardenctl` configuration file is  `~/.garden/config`. This default path and name can be overwritten with the environment variable `GARDENCONFIG`. `gardenctl` caches some information, e.g. the garden project names. The location of this cache is per default `$GARDENCTL_HOME/cache`. If `GARDENCTL_HOME` is not set, `~/.garden` is used.

`gardenctl` makes it easy to get additional information of your IaaS provider by using the secrets stored in the corresponding projects in the Gardener. To use this functionality, the command line interfaces of the IaaS provider need to be available. 

Please check the IaaS provider documentation for more details about their clis.
  - [aws](https://aws.amazon.com/cli/)
  - [az](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
  - [gcloud](https://cloud.google.com/sdk/downloads)
  - [openstack](https://pypi.python.org/pypi/python-openstackclient)


Moreover, `gardenctl` offers auto completion. To use it, the command
```bash
g completion
``` 
creates the file `gardenctl_completion.sh` which could then be sourced later on via 
```bash
source gardenctl_completion.sh
```
Please keep in mind that the auto completion is bound to the alias `g`. Therefore make sure that you add the alias definition into your shell startup file, e.g. for bash add `alias gardenctl=g` to `~/.bashrc`. 

## Use gardenctl

`gardenctl` requires the definition of a target, e.g. garden, project, seed or shoot. The following commands, e.g. `gardenctl ls shoots` use the target definition as a context for getting the information. 

Targets represents a hierarchical structure of the resources. On top, there is/are the garden/s. E.g. in case you setup a development and a production garden, you would have two entries in your `~/.garden/config`. Via `g ls gardens` you get a list of the available gardens. 

- `g get target`   
  Displays the current target
- `g target [garden|project|seed|shoot]`   
  Set the target e.g. to a garden. It is as well possible to set the target directly to a element deeper in the hierarchy, e.g. to a shoot.
- `g drop target`   
  Drop the deepest target. 

## Examples of basic usage:
- List all seed cluster <br />
`g ls seeds`
- List all projects with shoot cluster <br />
`g ls projects`
- Target a seed cluster <br />
`g target seed-gce-dev`
- Target a project <br />
`g target garden-vora`
- Open prometheus ui for a targeted shoot-cluster <br />
`g show prometheus`
- Execute an aws command on a targeted aws shoot cluster <br />
`g aws ec2 describe-instances` or <br />
`g aws ec2 describe-instances --no-cache` without locally caching credentials
- Target a shoot directly and get all kube-dns pods in kube-system namespace <br />
`g target myshoot`<br />
`g kubectl get pods -- -n kube-system | grep kube-dns`<br />
- List all cluster with an issue <br />
`g ls issues`
- Drop an element from target stack <br />
`g drop`

## Advanced usage based on JsonQuery

The following examples are based on [jq](https://stedolan.github.io/jq/). The [Json Query Playground](https://jqplay.org/jq?q=.%5B%5D&j=%5B%5D) offers a convenient environment to test the queries.

Below a list of examples. 

- From the project name and its state for all projects with issues
```bash
g ls issues -o json | jq '.issues[] | { project: .project, state: .status.lastOperation.state }'
```
- Print all issues of a single project e.g. `garden-myproject`
```bash
g ls issues -o json | jq '.issues[] | if (.project=="garden-myproject") then . else empty end' 
```
- Print all issues with error state "Error"
```bash
g ls issues -o json | jq '.issues[] | if (.status.lastOperation.state=="Error") then . else empty end'
```
- Print all issues with error state not equal "Succeded"
```bash
g ls issues -o json | jq '.issues[] | if (.status.lastOperation.state!="Succeeded") then . else empty end'
```
- Print `createdBy` information (typically email addresses) of all shoots
```bash
g k get shoots -- -n garden-core -o json | jq -r ".items[].metadata | {email: .annotations.\"garden.sapcloud.io/createdBy\", name: .name, namespace: .namespace}"
```

Cluster analysis

- Which states are there and how many clusters are in this state?
```bash 
 g ls issues -o json | jq '.issues | group_by( .status.lastOperation.state ) | .[] | {state:.[0].status.lastOperation.state, count:length}'
 ```

- Get all clusters in state `Failed`
```bash
 g ls issues -o json | jq '.issues[] | if (.status.lastOperation.state=="Failed") then . else empty end'
```

