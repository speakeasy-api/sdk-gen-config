package workflow

import (
	"errors"
	"net/url"
	"os"
)

type fileStatus int

const (
	fileStatusLocal fileStatus = iota
	fileStatusNotExists
	fileStatusRemote
)

func getFileStatus(filePath string) fileStatus {
	_, err := url.ParseRequestURI(filePath)
	if err != nil {
		if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
			return fileStatusNotExists
		}

		return fileStatusLocal
	}

	return fileStatusRemote
}
