package workflow

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type fileStatus int

const (
	fileStatusLocal fileStatus = iota
	fileStatusNotExists
	fileStatusRemote
	fileStatusRegistry
)

func getFileStatus(filePath string) fileStatus {
	if strings.Contains(filePath, "registry.speakeasyapi.dev") {
		return fileStatusRegistry
	}
	if _, err := os.Stat(SanitizeFilePath(filePath)); err == nil || !errors.Is(err, os.ErrNotExist) {
		return fileStatusLocal
	}

	if _, err := url.ParseRequestURI(filePath); err == nil {
		return fileStatusRemote
	}

	return fileStatusNotExists
}

func SanitizeFilePath(path string) string {
	sanitizedPath := path
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}

		sanitizedPath = filepath.Join(homeDir, path[2:])
		if absPath, err := filepath.Abs(sanitizedPath); err == nil {
			sanitizedPath = absPath
		}

		return sanitizedPath
	}

	return sanitizedPath
}
