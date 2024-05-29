package workflow

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type FileStatus int

const (
	fileStatusLocal FileStatus = iota
	fileStatusNotExists
	fileStatusRemote
	fileStatusRegistry
)

func GetFileStatus(filePath string) FileStatus {
	if strings.Contains(filePath, "registry.speakeasyapi.dev") {
		return fileStatusRegistry
	}
	if _, err := os.Stat(SanitizeFilePath(filePath)); err == nil || !errors.Is(err, os.ErrNotExist) {
		println("STAT ERROR: ", err.Error())
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
