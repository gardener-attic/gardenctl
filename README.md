# Gardenctl
Gardenctl is a command-line client for the Gardener. 

## Build:
- Clone github repo to `~/go/src/github.com/gardener/gardenctl` <br />
or adapt go import path in `gardenctl.go`
- Run `go build gardenctl.go`
- If a dependency is missing run `dep ensure`

## Prerequisites:
- A gardenctl configuration file must be provided. 
- Gardenctl is looking for the `GARDENCONFIG` environment variable and if it is not set, it looks for the config under the default path `~/.garden.config`
- Path to kubeconfig file of a garden cluster can be relative or absolute
- An example configuration is shown below
- Cache can be set via `GARDENCTL_HOME` environment variable and if it is not set, it uses `~/.garden` as default
- To use gardenctl aws, az, gcloud, openstack or kubectl integration, the corresponding tools needs to be installed


## Example Config:
``` yaml
gardenClusters:
- name: dev
  kubeConfig: ~/clusters/garden-dev/kubeconfig.yaml
- name: prod
  kubeConfig: /Users/d123456/clusters/garden-prod/kubeconfig.yaml
```

## Examples Usage:
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

## Adding bash-completion
- `g completion` will generate a bash-completion file ("gardenctl_completion.sh")  <br />
- To enable completion add for example `source ~/go/src/github.com/gardener/gardenctl/gardenctl_completion.sh` to `~/.bashrc`

