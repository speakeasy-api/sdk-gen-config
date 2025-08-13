package config

import (
	"github.com/google/uuid"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"go.yaml.in/yaml/v4"
)

type LockFile struct {
	LockVersion          string                       `yaml:"lockVersion"`
	ID                   string                       `yaml:"id"`
	Management           Management                   `yaml:"management"`
	Features             map[string]map[string]string `yaml:"features,omitempty"`
	GeneratedFiles       []string                     `yaml:"generatedFiles,omitempty"`
	Examples             Examples                     `yaml:"examples,omitempty"`
	ExamplesVersion      string                       `yaml:"examplesVersion,omitempty"`
	GeneratedTests       GeneratedTests               `yaml:"generatedTests,omitempty"`
	AdditionalProperties map[string]any               `yaml:",inline"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility

	ReleaseNotes string `yaml:"releaseNotes,omitempty"`
}

type Management struct {
	DocChecksum          string         `yaml:"docChecksum,omitempty"`
	DocVersion           string         `yaml:"docVersion,omitempty"`
	SpeakeasyVersion     string         `yaml:"speakeasyVersion,omitempty"`
	GenerationVersion    string         `yaml:"generationVersion,omitempty"`
	ReleaseVersion       string         `yaml:"releaseVersion,omitempty"`
	ConfigChecksum       string         `yaml:"configChecksum,omitempty"`
	RepoURL              string         `yaml:"repoURL,omitempty"`
	RepoSubDirectory     string         `yaml:"repoSubDirectory,omitempty"`
	InstallationURL      string         `yaml:"installationURL,omitempty"`
	Published            bool           `yaml:"published,omitempty"`
	AdditionalProperties map[string]any `yaml:",inline"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

type (
	Examples       = *sequencedmap.Map[string, *sequencedmap.Map[string, OperationExamples]]
	GeneratedTests = *sequencedmap.Map[string, string]
)

type OperationExamples struct {
	Parameters  *ParameterExamples                                              `yaml:"parameters,omitempty"`
	RequestBody *sequencedmap.Map[string, yaml.Node]                            `yaml:"requestBody,omitempty"`
	Responses   *sequencedmap.Map[string, *sequencedmap.Map[string, yaml.Node]] `yaml:"responses,omitempty"`
}

type ParameterExamples struct {
	Path   *sequencedmap.Map[string, yaml.Node] `yaml:"path,omitempty"`
	Query  *sequencedmap.Map[string, yaml.Node] `yaml:"query,omitempty"`
	Header *sequencedmap.Map[string, yaml.Node] `yaml:"header,omitempty"`
}

var getUUID = func() string {
	return uuid.NewString()
}

func NewLockFile() *LockFile {
	return &LockFile{
		LockVersion: v2,
		ID:          getUUID(),
		Features:    map[string]map[string]string{},
	}
}
