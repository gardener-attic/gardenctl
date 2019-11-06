module github.com/gardener/gardenctl

go 1.13

require (
	github.com/Masterminds/semver v1.4.2
	github.com/badoux/checkmail v0.0.0-20181210160741-9661bd69e9ad
	github.com/gardener/gardener v0.0.0-20191018063251-c1b318de841e
	github.com/gardener/gardener-extensions v0.0.0-20190906160200-5c329d46ae81
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.3.1
	github.com/jmoiron/jsonq v0.0.0-20150511023944-e874b168d07e
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/spf13/cobra v0.0.5
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	golang.org/x/net v0.0.0-20190813141303-74dc4d7220e7 // indirect
	gopkg.in/yaml.v2 v2.2.4
	k8s.io/api v0.0.0-20191004102349-159aefb8556b
	k8s.io/apimachinery v0.0.0-20191004074956-c5d2f014d689
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

// TODO: Fix for https://github.com/Azure/go-autorest/issues/449.
replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
