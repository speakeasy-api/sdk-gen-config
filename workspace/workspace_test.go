package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDir_CreatesFromScratch(t *testing.T) {
	parent := t.TempDir()

	if err := EnsureDir(parent); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	gitignorePath := filepath.Join(parent, SpeakeasyFolder, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	content := string(data)
	for _, entry := range GitIgnoreEntries {
		if !strings.Contains(content, entry) {
			t.Errorf("expected .gitignore to contain %q, got:\n%s", entry, content)
		}
	}
}

func TestEnsureDir_Idempotent(t *testing.T) {
	parent := t.TempDir()

	if err := EnsureDir(parent); err != nil {
		t.Fatalf("first EnsureDir failed: %v", err)
	}
	if err := EnsureDir(parent); err != nil {
		t.Fatalf("second EnsureDir failed: %v", err)
	}

	gitignorePath := filepath.Join(parent, SpeakeasyFolder, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	content := string(data)
	for _, entry := range GitIgnoreEntries {
		count := strings.Count(content, entry)
		if count != 1 {
			t.Errorf("expected exactly 1 occurrence of %q, found %d in:\n%s", entry, count, content)
		}
	}
}

func TestEnsureDir_MergesMissingEntries(t *testing.T) {
	parent := t.TempDir()
	speakeasyDir := filepath.Join(parent, SpeakeasyFolder)
	if err := os.MkdirAll(speakeasyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a .gitignore with only the first entry
	if err := os.WriteFile(filepath.Join(speakeasyDir, ".gitignore"), []byte(GitIgnoreEntries[0]+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureDir(parent); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(speakeasyDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	for _, entry := range GitIgnoreEntries {
		count := strings.Count(content, entry)
		if count != 1 {
			t.Errorf("expected exactly 1 occurrence of %q, found %d in:\n%s", entry, count, content)
		}
	}
}

func TestEnsureDir_PreservesCustomContent(t *testing.T) {
	parent := t.TempDir()
	speakeasyDir := filepath.Join(parent, SpeakeasyFolder)
	if err := os.MkdirAll(speakeasyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	custom := "# custom rules\n*.bak\n"
	if err := os.WriteFile(filepath.Join(speakeasyDir, ".gitignore"), []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureDir(parent); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(speakeasyDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, custom) {
		t.Errorf("expected content to start with custom rules, got:\n%s", content)
	}
	for _, entry := range GitIgnoreEntries {
		if !strings.Contains(content, entry) {
			t.Errorf("expected .gitignore to contain %q, got:\n%s", entry, content)
		}
	}
}

func TestEnsureDir_HandlesNoTrailingNewline(t *testing.T) {
	parent := t.TempDir()
	speakeasyDir := filepath.Join(parent, SpeakeasyFolder)
	if err := os.MkdirAll(speakeasyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write content without trailing newline
	if err := os.WriteFile(filepath.Join(speakeasyDir, ".gitignore"), []byte("*.bak"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureDir(parent); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(speakeasyDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Should not have "*.bakentry" — there must be a newline between existing content and new entries
	if strings.Contains(content, "*.bak"+GitIgnoreEntries[0]) {
		t.Errorf("missing newline between existing content and new entries:\n%s", content)
	}
	if !strings.Contains(content, "*.bak\n") {
		t.Errorf("expected existing content to be preserved with newline, got:\n%s", content)
	}
	for _, entry := range GitIgnoreEntries {
		if !strings.Contains(content, entry) {
			t.Errorf("expected .gitignore to contain %q, got:\n%s", entry, content)
		}
	}
}
