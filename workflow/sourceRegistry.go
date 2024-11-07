package workflow

import (
	"fmt"
	"math/rand"
	"path"
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

	// perfectly valid for someone to add http prefixes
	location = strings.TrimPrefix(location, "https://")
	location = strings.TrimPrefix(location, "http://")

	// Assert it starts with the registry prefix
	if !strings.HasPrefix(location, "registry.speakeasyapi.dev/") {
		return nil
	}

	// Extract the organization, workspace, and namespace
	parts := strings.Split(strings.TrimPrefix(location, "registry.speakeasyapi.dev/"), "/")
	if len(parts) != 3 {
		return nil
	}

	document := &SpeakeasyRegistryDocument{
		OrganizationSlug: parts[0],
		WorkspaceSlug:    parts[1],
	}
	suffix := parts[2]

	document.SetNamespaceName(suffix)
	document.Reference = "latest"

	// Check if the suffix contains a reference
	if strings.Contains(suffix, "@") {
		// Reference is a digest
		document.SetNamespaceName(suffix[:strings.Index(suffix, "@")])
		document.Reference = suffix[strings.Index(suffix, "@")+1:]
	} else if strings.Contains(suffix, ":") {
		// Reference is a tag
		document.SetNamespaceName(suffix[:strings.Index(suffix, ":")])
		document.Reference = suffix[strings.Index(suffix, ":")+1:]
	}

	return document
}

func (s *SpeakeasyRegistryDocument) SetNamespaceName(namespaceName string) {
	s.NamespaceName = namespaceName
	s.NamespaceID = s.OrganizationSlug + "/" + s.WorkspaceSlug + "/" + s.NamespaceName
}

func (s *SpeakeasyRegistryDocument) MakeURL(includeReference bool) SourceRegistryLocation {
	if includeReference && s.Reference != "" {
		separator := ':'
		if strings.Contains(s.Reference, "sha256:") {
			separator = '@'
		}
		url := path.Join(baseRegistryURL, fmt.Sprintf("%s%c%s", s.NamespaceID, separator, s.Reference))
		return SourceRegistryLocation(url)
	} else {
		url := path.Join(baseRegistryURL, s.NamespaceID)
		return SourceRegistryLocation(url)
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
	p.Location = SourceRegistryLocation(baseRegistryURL + namespace)
	return p.Validate()
}

func (s SourceRegistryLocation) Parse() *SpeakeasyRegistryDocument {
	return ParseSpeakeasyRegistryReference(string(s))
}

// @<org>/<workspace>/<namespace_name> => <org>/<workspace>/<namespace_name>
func (s SourceRegistryLocation) Namespace() string {
	if parsed := s.Parse(); parsed == nil {
		return ""
	} else {
		return parsed.NamespaceID
	}
}

// @<org>/<workspace>/<namespace_name> => <namespace_name>
func (s SourceRegistryLocation) NamespaceName() string {
	if parsed := s.Parse(); parsed == nil {
		return ""
	} else {
		return parsed.NamespaceName
	}
}

func (s SourceRegistryLocation) String() string {
	return string(s)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var randStringBytes = func(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
