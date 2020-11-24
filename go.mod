module github.com/gardener/gardenctl

go 1.15

require (
	github.com/Masterminds/semver v1.5.0
	github.com/badoux/checkmail v0.0.0-20181210160741-9661bd69e9ad
	github.com/gardener/gardener v1.5.0
	github.com/gardener/gardener-extension-provider-openstack v1.3.1-0.20200327120628-280d268ce96f
	github.com/gardener/machine-controller-manager v0.27.0
	github.com/golang/mock v1.4.3
	github.com/jmoiron/jsonq v0.0.0-20150511023944-e874b168d07e
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/spf13/cobra v0.0.6
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f
	gopkg.in/yaml.v2 v2.2.8
	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/metrics v0.16.8
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190918155943-95b840bb6a1f // kubernetes-1.16.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655 // kubernetes-1.16.0
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90 // kubernetes-1.16.0
)
