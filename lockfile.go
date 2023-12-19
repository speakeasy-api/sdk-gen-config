package config

import "github.com/google/uuid"

type LockFile struct {
	LockVersion          string                       `yaml:"lockVersion"`
	ID                   string                       `yaml:"id"`
	Management           Management                   `yaml:"management"`
	Features             map[string]map[string]string `yaml:"features,omitempty"`
	GeneratedFiles       []string                     `yaml:"generatedFiles,omitempty"`
	AdditionalProperties map[string]any               `yaml:",inline"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
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
