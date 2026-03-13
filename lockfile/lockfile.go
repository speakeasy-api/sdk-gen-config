package lockfile

import (
	"github.com/google/uuid"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"gopkg.in/yaml.v3"
)

const (
	LockV2 = "2.0.0"
)

type (
	Examples       = *sequencedmap.Map[string, *sequencedmap.Map[string, OperationExamples]]
	GeneratedTests = *sequencedmap.Map[string, string]
	TrackedFiles   = *sequencedmap.Map[string, TrackedFile]
)

type TrackedFile struct {
	// Identity (The "Breadcrumb")
	// UUID embedded in the file header. Allows detecting "Moves/Renames".
	ID string `yaml:"id,omitempty"`

	// The Dirty Check (Optimization)
	// The SHA-1 of the file content exactly as written to disk last time.
	// If Disk_SHA == LastWriteChecksum, we skip the merge (Fast Path).
	LastWriteChecksum string `yaml:"last_write_checksum,omitempty"`

	// The O(1) Lookup Key
	// Stores the Blob Hash of the file from the PREVIOUS run.
	// Only populated if persistentEdits is enabled.
	PristineGitObject string `yaml:"pristine_git_object,omitempty"`

	// Deleted indicates the user deleted this file from disk.
	// Set pre-generation by scanning disk vs lockfile entries.
	// When true, the generator should not regenerate this file.
	Deleted bool `yaml:"deleted,omitempty"`

	// MovedTo indicates the user moved/renamed this file to a new path.
	// Set pre-generation by scanning @generated-id headers on disk.
	// The generator should write to the new path instead of the original.
	MovedTo string `yaml:"moved_to,omitempty"`

	AdditionalProperties map[string]any `yaml:",inline"`
}

type PersistentEdits struct {
	// Maps to refs/speakeasy/gen/<UUID>
	GenerationID string `yaml:"generation_id,omitempty"`

	// Parent Commit (Links history for compression)
	PristineCommitHash string `yaml:"pristine_commit_hash,omitempty"`

	// Content Checksum (Enables No-Op/Determinism check)
	PristineTreeHash string `yaml:"pristine_tree_hash,omitempty"`
}

type LockFile struct {
	LockVersion          string                       `yaml:"lockVersion"`
	ID                   string                       `yaml:"id"`
	Management           Management                   `yaml:"management"`
	PersistentEdits      *PersistentEdits             `yaml:"persistentEdits,omitempty"`
	Features             map[string]map[string]string `yaml:"features,omitempty"`
	TrackedFiles         TrackedFiles                 `yaml:"trackedFiles,omitempty"`
	Examples             Examples                     `yaml:"examples,omitempty"`
	ExamplesVersion      string                       `yaml:"examplesVersion,omitempty"`
	GeneratedTests       GeneratedTests               `yaml:"generatedTests,omitempty"`
	AdditionalProperties map[string]any               `yaml:",inline"`

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
	AdditionalProperties map[string]any `yaml:",inline"`
}

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

var GetUUID = func() string {
	return uuid.NewString()
}

func New() *LockFile {
	return &LockFile{
		LockVersion:  LockV2,
		ID:           GetUUID(),
		Features:     map[string]map[string]string{},
		TrackedFiles: sequencedmap.New[string, TrackedFile](),
	}
}
