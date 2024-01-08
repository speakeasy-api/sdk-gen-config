package workflow

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"slices"
)

type Source struct {
	Inputs   []Document `yaml:"inputs"`
	Overlays []Document `yaml:"overlays,omitempty"`
	Output   *string    `yaml:"output,omitempty"`
}

type Document struct {
	Location string `yaml:"location"`
	Auth     *Auth  `yaml:",inline"`
}

type Auth struct {
	Header string `yaml:"authHeader,omitempty"`
	Secret string `yaml:"authSecret,omitempty"`
}

func (s Source) Validate() error {
	if len(s.Inputs) == 0 {
		return fmt.Errorf("no inputs found")
	}

	for i, input := range s.Inputs {
		if err := input.Validate(); err != nil {
			return fmt.Errorf("failed to validate input %d: %w", i, err)
		}
	}

	for i, overlay := range s.Overlays {
		if err := overlay.Validate(); err != nil {
			return fmt.Errorf("failed to validate overlay %d: %w", i, err)
		}
	}

	_, err := s.GetOutputLocation()
	if err != nil {
		return fmt.Errorf("failed to get output location: %w", err)
	}

	return nil
}

func (s Source) GetOutputLocation() (string, error) {
	// If we have an output location, we can just return that
	if s.Output != nil {
		output := *s.Output

		ext := filepath.Ext(output)
		if (len(s.Inputs) > 1 || len(s.Overlays) > 0) && !slices.Contains([]string{".yaml", ".yml"}, ext) {
			return "", fmt.Errorf("when using multiple inputs or overlays, output must be a yaml file")
		}

		if (len(s.Inputs) == 0 || (len(s.Inputs) == 1 && output != s.Inputs[0].Location)) && len(s.Overlays) == 0 {
			return "", fmt.Errorf("when using a single input, output should not be specified")
		}

		return output, nil
	}

	ext := ".yaml"

	// If we only have a single input, no overlays and its a local path, we can just use that
	if len(s.Inputs) == 1 && len(s.Overlays) == 0 {
		inputFile := s.Inputs[0].Location

		switch getFileStatus(inputFile) {
		case fileStatusLocal:
			return inputFile, nil
		case fileStatusNotExists:
			return "", fmt.Errorf("input file %s does not exist", inputFile)
		case fileStatusRemote:
			ext = filepath.Ext(inputFile)
			if ext == "" {
				ext = ".yaml"
			}
		}
	}

	// Otherwise output will go to a temp file
	return filepath.Join(GetTempDir(), fmt.Sprintf("output_%s%s", randStringBytes(10), ext)), nil
}

func GetTempDir() string {
	return filepath.Join(".speakeasy", "temp")
}

func (s Source) GetTempMergeLocation() string {
	return filepath.Join(GetTempDir(), fmt.Sprintf("merge_%s.yaml", randStringBytes(10)))
}

func (s Source) GetTempOverlayLocation() string {
	return filepath.Join(GetTempDir(), fmt.Sprintf("overlay_%s.yaml", randStringBytes(10)))
}

func (d Document) Validate() error {
	if d.Location == "" {
		return fmt.Errorf("location is required")
	}

	if d.Auth != nil {
		if getFileStatus(d.Location) != fileStatusRemote {
			return fmt.Errorf("auth is only supported for remote documents")
		}

		if err := validateSecret(d.Auth.Secret); err != nil {
			return fmt.Errorf("failed to validate authSecret: %w", err)
		}
	}

	return nil
}

func (d Document) IsRemote() bool {
	return getFileStatus(d.Location) == fileStatusRemote
}

func (d Document) GetTempDownloadPath(tempDir string) string {
	return filepath.Join(tempDir, fmt.Sprintf("downloaded_%s%s", randStringBytes(10), filepath.Ext(d.Location)))
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var randStringBytes = func(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
