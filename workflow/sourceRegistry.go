package workflow

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
)

// Parse the location to extract the namespace ID, namespace name, and reference
// The location should be in the format registry.speakeasyapi.dev/org/workspace/name[:tag|@sha256:digest]
func ParseSpeakeasyRegistryReference(location string) *SpeakeasyRegistryDocument {
	// Parse the location to extract the organization, workspace, namespace, and reference
	// Examples:
	// registry.speakeasyapi.dev/org/workspace/name (default reference: latest)
	// registry.speakeasyapi.dev/org/workspace/name@sha256:1234567890abcdef
	// registry.speakeasyapi.dev/org/workspace/name:tag

	// Assert it starts with the registry prefix
	if !strings.HasPrefix(location, "registry.speakeasyapi.dev/") {
		return nil
	}

	// Extract the organization, workspace, and namespace
	parts := strings.Split(strings.TrimPrefix(location, "registry.speakeasyapi.dev/"), "/")
	if len(parts) != 3 {
		return nil
	}

	organizationSlug := parts[0]
	workspaceSlug := parts[1]
	suffix := parts[2]

	reference := "latest"
	namespaceName := suffix

	// Check if the suffix contains a reference
	if strings.Contains(suffix, "@") {
		// Reference is a digest
		reference = suffix[strings.Index(suffix, "@")+1:]
		namespaceName = suffix[:strings.Index(suffix, "@")]
	} else if strings.Contains(suffix, ":") {
		// Reference is a tag
		reference = suffix[strings.Index(suffix, ":")+1:]
		namespaceName = suffix[:strings.Index(suffix, ":")]
	}

	return &SpeakeasyRegistryDocument{
		OrganizationSlug: organizationSlug,
		WorkspaceSlug:    workspaceSlug,
		NamespaceID:      organizationSlug + "/" + workspaceSlug + "/" + namespaceName,
		NamespaceName:    namespaceName,
		Reference:        reference,
	}
}

func (d Document) GetTempDownloadPath(tempDir string) string {
	return filepath.Join(tempDir, fmt.Sprintf("downloaded_%s%s", randStringBytes(10), filepath.Ext(d.Location.Resolve())))
}

func (d Document) GetTempRegistryDir(tempDir string) string {
	return filepath.Join(tempDir, fmt.Sprintf("registry_%s", randStringBytes(10)))
}

const baseRegistryURL = "registry.speakeasyapi.dev/"

func (p SourceRegistry) Validate() error {
	if p.Location == "" {
		return fmt.Errorf("location is required")
	}

	location := p.Location.String()
	// perfectly valid for someone to add http prefixes
	location = strings.TrimPrefix(location, "https://")
	location = strings.TrimPrefix(location, "http://")

	if !strings.HasPrefix(location, baseRegistryURL) {
		return fmt.Errorf("registry location must begin with %s", baseRegistryURL)
	}

	if strings.Count(p.Location.Namespace(), "/") != 2 {
		return fmt.Errorf("registry location should look like %s<org>/<workspace>/<image>", baseRegistryURL)
	}

	return nil
}

func (p *SourceRegistry) SetNamespace(namespace string) error {
	p.Location = SourceRegistryLocation("https://" + baseRegistryURL + namespace)
	return p.Validate()
}

func (p *SourceRegistry) ParseRegistryLocation() (string, string, string, string, error) {
	if err := p.Validate(); err != nil {
		return "", "", "", "", err
	}

	location := p.Location.String()
	// perfectly valid for someone to add http prefixes
	location = strings.TrimPrefix(location, "https://")
	location = strings.TrimPrefix(location, "http://")

	subParts := strings.Split(location, baseRegistryURL)
	components := strings.Split(strings.TrimSuffix(subParts[1], "/"), "/")
	namespace := components[2]
	tag := ""
	if shaSplit := strings.Split(components[2], "@sha256:"); len(shaSplit) == 2 {
		namespace = shaSplit[0]
		tag = "sha256:" + shaSplit[1]
	}

	if tagSplit := strings.Split(components[2], ":"); tag == "" && len(tagSplit) == 2 {
		namespace = tagSplit[0]
		tag = tagSplit[1]
	}

	return components[0], components[1], namespace, tag, nil
}

// @<org>/<workspace>/<namespace_name> => <org>/<workspace>/<namespace_name>
func (n SourceRegistryLocation) Namespace() string {
	location := string(n)
	// perfectly valid for someone to add http prefixes
	location = strings.TrimPrefix(location, "https://")
	location = strings.TrimPrefix(location, "http://")
	return strings.TrimPrefix(location, baseRegistryURL)
}

// @<org>/<workspace>/<namespace_name> => <namespace_name>
func (n SourceRegistryLocation) NamespaceName() string {
	return n.Namespace()[strings.LastIndex(n.Namespace(), "/")+1:]
}

func (n SourceRegistryLocation) String() string {
	return string(n)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var randStringBytes = func(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
