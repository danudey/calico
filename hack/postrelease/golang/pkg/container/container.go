// Package container contains universal functionality for container images
package container

import (
	"fmt"
	"strings"
)

// Image is an object to represent a Docker/OCI image with tag on a specific docker host
type Image struct {
	HostName string
	Name     string
	Tag      string
}

// FullPath will combine the HostName, Name, and Tag into a full URI for the container
func (i Image) FullPath() string {
	var registryPath string

	if strings.HasSuffix(i.HostName, "gcr.io") {
		registryPath = "projectcalico-org"
	} else {
		registryPath = "calico"
	}

	return fmt.Sprintf("%s/%s/%s", i.HostName, registryPath, i.Name)
}
