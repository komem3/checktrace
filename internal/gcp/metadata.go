package gcp

import (
	"cloud.google.com/go/compute/metadata"
)

func Project(local string) (string, error) {
	if metadata.OnGCE() {
		return metadata.ProjectID()
	}
	return local, nil
}
