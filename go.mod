module github.com/gardener/gardenctl

go 1.12

require (
	cloud.google.com/go v0.37.4 // indirect
	contrib.go.opencensus.io/exporter/ocagent v0.4.12 // indirect
	github.com/Azure/go-autorest v11.7.1+incompatible // indirect
	github.com/badoux/checkmail v0.0.0-20181210160741-9661bd69e9ad
	github.com/gardener/gardener v0.0.0-20190805161523-629693b1994f
	github.com/ghodss/yaml v1.0.0
	github.com/golang/mock v1.2.0
	github.com/jmoiron/jsonq v0.0.0-20150511023944-e874b168d07e
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/spf13/cobra v0.0.5
	golang.org/x/lint v0.0.0-20190313153728-d0100b6bd8b3
	google.golang.org/api v0.3.2 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/kube-openapi v0.0.0-20190401085232-94e1e7b7574c // indirect
)

replace github.com/gardener/gardener => github.com/gardener/gardener v0.0.0-20190805161523-629693b1994f // 0.27.1
