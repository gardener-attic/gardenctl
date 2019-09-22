module github.com/gardener/gardenctl

go 1.13

require (
	github.com/badoux/checkmail v0.0.0-20181210160741-9661bd69e9ad
	github.com/gardener/gardener v0.0.0-20190921111132-e71a6bc4f613
	github.com/gardener/gardener-extensions v0.0.0-20190906160200-5c329d46ae81
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.3.1
	github.com/jmoiron/jsonq v0.0.0-20150511023944-e874b168d07e
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/spf13/cobra v0.0.5
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
)

// TODO: Fix for https://github.com/Azure/go-autorest/issues/449.
replace github.com/census-instrumentation/opencensus-proto v0.1.0-0.20181214143942-ba49f56771b8 => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
