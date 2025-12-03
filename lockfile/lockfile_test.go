package lockfile_test

import (
	"testing"

	"github.com/speakeasy-api/sdk-gen-config/lockfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_NewStructure(t *testing.T) {
	// New lockfile YAML with 3-way merge fields
	newYAML := []byte(`
lockVersion: "2.0.0"
id: "test-uuid"
management: {}
persistentEdits:
  generation_id: "uuid-550e"
  pristine_commit_hash: "a1b2c3"
  pristine_tree_hash: "tree-e5f6"
trackedFiles:
  "pkg/models/user.go":
    id: "uuid-breadcrumb-123"
    last_write_checksum: "sha1:file-hash-789"
    pristine_git_object: "blob-123"
`)

	lf, err := lockfile.Load(newYAML)
	require.NoError(t, err)

	// Verify persistentEdits
	require.NotNil(t, lf.PersistentEdits)
	assert.Equal(t, "uuid-550e", lf.PersistentEdits.GenerationID)
	assert.Equal(t, "a1b2c3", lf.PersistentEdits.PristineCommitHash)
	assert.Equal(t, "tree-e5f6", lf.PersistentEdits.PristineTreeHash)

	// Verify tracked file
	tf, ok := lf.TrackedFiles.Get("pkg/models/user.go")
	require.True(t, ok)

	assert.Equal(t, "uuid-breadcrumb-123", tf.ID)
	assert.Equal(t, "sha1:file-hash-789", tf.LastWriteChecksum)
	assert.Equal(t, "blob-123", tf.PristineGitObject)
}

func TestLoad_OmitsEmptyPersistentEdits(t *testing.T) {
	// Lockfile YAML without persistentEdits section
	yamlWithoutPE := []byte(`
lockVersion: "2.0.0"
id: "test-uuid"
management: {}
trackedFiles:
  "src/file.go":
    id: "file-uuid"
    last_write_checksum: "sha1:hash"
`)

	lf, err := lockfile.Load(yamlWithoutPE)
	require.NoError(t, err)
	require.NotNil(t, lf)

	// Verify persistentEdits is nil when not present
	assert.Nil(t, lf.PersistentEdits, "PersistentEdits should be nil when not in YAML")
}

func TestNew_CreatesValidLockFile(t *testing.T) {
	lf := lockfile.New()
	require.NotNil(t, lf)

	assert.Equal(t, "2.0.0", lf.LockVersion)
	assert.NotEmpty(t, lf.ID)
	assert.NotNil(t, lf.TrackedFiles)
	assert.Equal(t, 0, lf.TrackedFiles.Len())
	assert.Nil(t, lf.PersistentEdits, "PersistentEdits should be nil in new lockfile")
}
