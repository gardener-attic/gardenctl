package cmd_test

import (
	"os"

	. "github.com/gardener/gardenctl/cmd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Miscellaneous", func() {
	var gardenClusters GardenClusters
	var target Target
	dumpPath := "/tmp"
	pathTarget := dumpPath + "/target2"
	pathGardenConfig := dumpPath + "/gardenconfig"
	file, err := os.OpenFile(pathGardenConfig, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	gardenConfig := `
gardenClusters:
- name: dev
  kubeConfig: /tmp/kubeconfig.yaml
- name: prod
  kubeConfig: /tmp/kubeconfig.yaml
`
	content := []byte(gardenConfig)
	file.Write(content)
	err = file.Close()
	if err != nil {
		panic(err)
	}
	file, err = os.OpenFile(pathTarget, os.O_WRONLY|os.O_CREATE, 0644)
	content = []byte("")
	if err != nil {
		panic(err)
	}
	file.Write(content)
	err = file.Close()
	if err != nil {
		panic(err)
	}
	Context("After calling GetGardenClusterKubeConfigFromConfig", func() {
		It("First Garden Cluster should be set as default target Name if no garden cluster is specified", func() {
			GetGardenClusterKubeConfigFromConfig(pathGardenConfig, pathTarget)
			ReadTarget(pathTarget, &target)
			Expect(target.Target[0].Name).To(Equal("dev"))
		})
	})
	Context("After calling GetGardenClusters", func() {
		It("GardenCluster Name should be dev ", func() {
			GetGardenClusters(pathGardenConfig, &gardenClusters)
			Expect(gardenClusters.GardenClusters[0].Name).To(Equal("dev"))
			Expect(gardenClusters.GardenClusters[1].Name).To(Equal("prod"))
		})
	})

	var _ = AfterSuite(func() {
		os.Remove(pathTarget)
		os.Remove(pathGardenConfig)
	})
})
