package security

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// dockerHubHosts are the names under which Trivy Operator and image references address Docker Hub.
var dockerHubHosts = []string{"docker.io", "index.docker.io", "registry-1.docker.io"}

// Image is the image a report was produced for.
type Image struct {
	// Name is the normalised registry host and repository without tag or digest.
	Name   string
	Tag    string
	Digest string
}

// ImageOf extracts the image of a Trivy Operator report.
func ImageOf(report *unstructured.Unstructured) Image {
	server, _, _ := unstructured.NestedString(report.Object, "report", "registry", "server")
	repository, _, _ := unstructured.NestedString(report.Object, "report", "artifact", "repository")
	tag, _, _ := unstructured.NestedString(report.Object, "report", "artifact", "tag")
	digest, _, _ := unstructured.NestedString(report.Object, "report", "artifact", "digest")

	ref := repository
	if server != "" {
		ref = server + "/" + repository
	}

	// Filesystem scans of some images put the whole image reference into the tag.
	if strings.Contains(tag, "/") {
		tag = tagOf(tag)
	}

	return Image{Name: NormalizeImage(ref), Tag: tag, Digest: digest}
}

// NormalizeImage returns the lower-case registry host and repository of an image reference
// without tag and digest. Docker Hub is written without its host and without the library/ namespace,
// so "docker.io/library/nginx:1.27", "index.docker.io/library/nginx" and "nginx" are the same image.
func NormalizeImage(ref string) string {
	ref, _, _ = strings.Cut(ref, "@")

	if i := strings.LastIndex(ref, "/"); strings.Contains(ref[i+1:], ":") {
		ref = ref[:i+1+strings.Index(ref[i+1:], ":")]
	}

	ref = strings.ToLower(ref)

	for _, host := range dockerHubHosts {
		if rest, ok := strings.CutPrefix(ref, host+"/"); ok {
			return strings.TrimPrefix(rest, "library/")
		}
	}

	return ref
}

// tagOf returns the tag of an image reference, or an empty string when the reference has none.
// A colon before the last slash belongs to the registry port.
func tagOf(ref string) string {
	ref, _, _ = strings.Cut(ref, "@")
	last := ref[strings.LastIndex(ref, "/")+1:]

	_, tag, _ := strings.Cut(last, ":")

	return tag
}
