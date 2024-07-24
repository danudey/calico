package calico_oss_test

import (
	"calico_postrelease/pkg/container"
	"calico_postrelease/pkg/openstack"
	"calico_postrelease/pkg/registry"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-github/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var release_tag = "v3.27.4"
var operator_release_version = "v1.32.4"

var release_hosts = []string{
	"docker.io",
	"quay.io",
	"gcr.io",
	"us.gcr.io",
	"asia.gcr.io",
	"eu.gcr.io",
}

var expected_arches = []string{"amd64", "arm64", "s390x", "ppc64le"}

var expected_images = []string{
	"node",
	"ctl",
	"apiserver",
	"typha",
	"cni",
	"kube-controllers",
	// "upgrade",
	"flannel-migration-controller",
	"dikastes",
	// "pilot-webhook",
	"pod2daemon-flexvol",
	"csi",
}

var expected_assets = []string{
	"calico-windows-%s.zip",
	"calicoctl-darwin-amd64",
	"calicoctl-darwin-arm64",
	"calicoctl-linux-amd64",
	"calicoctl-linux-arm64",
	"calicoctl-linux-ppc64le",
	"calicoctl-linux-s390x",
	"calicoctl-windows-amd64.exe",
	"install-calico-windows.ps1",
	"metadata.yaml",
	"ocp.tgz",
	"release-%s.tgz",
	"SHA256SUMS",
	"tigera-operator-%s.tgz",
}

// Commenting these out because we don't currently tag the
// individual windows images. Should we?
// var expected_windows_tags = []string{
// 	"windows-ltsc2022",
// 	"windows-1809",
// }

var expected_windows_images = []string{
	"cni-windows",
	"node-windows",
}

func TestGolang(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Calico OSS Postrelease Test Suite")
}

var regCheck registry.RegistryChecker
var err error
var imagesToTest []container.Image

var _ = BeforeSuite(func() {
	regCheck, err = registry.New()
	if err != nil {
		fmt.Println("I guess something failed")
		panic(err)
	}
	imagesToTest = make([]container.Image, 400)

	imagesToTest = append(imagesToTest, container.Image{
		Name:     "operator",
		Tag:      operator_release_version,
		HostName: "quay.io",
	})
})

var _ = AfterSuite(func() {
	// if cacheHits, found := regCheck.Cache.Get("CacheHit"); found {
	// 	fmt.Printf("Cache hits: %v\n", cacheHits)
	// }
	// if cacheMiss, found := regCheck.Cache.Get("CacheMiss"); found {
	// 	fmt.Printf("Cache misses: %v\n", cacheMiss)
	// }
})

var _ = Describe(
	"Validate published image",
	Label("docker"),
	func() {
		for _, host_name := range release_hosts {
			Context(fmt.Sprintf("at registry %s", host_name), Label(host_name), func() {

				for _, image_name := range expected_images {
					Context(fmt.Sprintf("image %s", image_name), Label(image_name), func() {
						var containerImage = container.Image{
							Name:     image_name,
							Tag:      release_tag,
							HostName: host_name,
						}
						It("should exist", func() {
							Expect(regCheck.CheckImageTagExists(containerImage)).NotTo(HaveOccurred())
						})
					})
				}
				for _, image_name := range expected_images {
					Describe(fmt.Sprintf("image %s", image_name),
						Label("image_name"),
						func() {
							for _, arch_name := range expected_arches {
								image_name := image_name
								host_name := host_name
								arch_name := arch_name
								var containerImage = container.Image{
									Name:     image_name,
									Tag:      fmt.Sprintf("%s-%s", release_tag, arch_name),
									HostName: host_name,
								}
								It(fmt.Sprintf("Should have %s", arch_name), Label(arch_name), func() {
									Expect(regCheck.CheckImageTagExists(containerImage)).NotTo(HaveOccurred())
								})
							}
						})
				}

				for _, image_name := range expected_windows_images {
					Context(fmt.Sprintf("arch-specific image %s", image_name), func() {
						var containerImage = container.Image{
							Name:     image_name,
							Tag:      release_tag,
							HostName: host_name,
						}
						It("should exist", func() {
							Expect(regCheck.CheckImageTagExists(containerImage)).NotTo(HaveOccurred())
						})
					})

				}

			})
		}

	})

var _ = Describe(
	"Validate Github publishing",
	Label("github"),
	func() {
		ghClient := github.NewClient(nil)

		Context(fmt.Sprintf("release %s", release_tag), func() {
			release, _, err := ghClient.Repositories.GetReleaseByTag(context.Background(), "projectcalico", "calico", release_tag)
			It("should be pubished", func() {
				if err != nil {
					Fail("Error!")
				}
			})
			Context("should contain asset", func() {
				available_names := make([]string, 0)

				for _, asset := range release.Assets {
					available_names = append(available_names, asset.GetName())
				}
				for _, desired_name := range expected_assets {
					if strings.Contains(desired_name, "%s") {
						desired_name = fmt.Sprintf(desired_name, release_tag)
					}
					It(desired_name, Label("asset"), func() {
						if !slices.Contains(available_names, desired_name) {
							Fail(fmt.Sprintf("missing asset %s in release", desired_name))
						}
					})
				}

			})
		})
	})

var _ = Describe(
	"Validate Openstack publishing",
	Label("openstack"),
	func() {
		var packageList = openstack.GetPackages()
		Context("check openstack files", func() {
			for _, packageObj := range packageList {
				It(
					fmt.Sprintf("should have published %s %s for %s", packageObj.Component, packageObj.Version, packageObj.OSVersion),
					Label(packageObj.Component),
					func() {
						resp, err := packageObj.Get()
						if err != nil {
							Fail("Failed to fetch package")
						}
						if resp.StatusCode != 200 {
							Fail(fmt.Sprintf("Caught unexpected HTTP status code %v", resp.StatusCode))
						}

					},
				)
			}
		})
	})
