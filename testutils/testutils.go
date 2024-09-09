package testutils

import (
	"os"
	"path/filepath"
	"testing"
)

func CreateTempFile(t *testing.T, dir string, fileName, contents string) {
	t.Helper()

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if contents != "" {
		tmpFile := filepath.Join(dir, fileName)
		if err := os.WriteFile(tmpFile, []byte(contents), os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}
}

func ReadTestFile(t *testing.T, file string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}
