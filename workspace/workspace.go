package workspace

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	SpeakeasyFolder = ".speakeasy"
	GenFolder       = ".gen"
)

var GitIgnoreEntries = []string{"logs/", "temp/", "reports/"}

// EnsureSpeakeasyDir creates .speakeasy/ inside parentDir with a .gitignore containing GitIgnoreEntries.
func EnsureSpeakeasyDir(parentDir string) error {
	dir := filepath.Join(parentDir, SpeakeasyFolder)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", SpeakeasyFolder, err)
	}
	return ensureGitIgnore(dir)
}

func ensureGitIgnore(dir string) error {
	path := filepath.Join(dir, ".gitignore")

	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	content := string(existing)
	var toAdd []string
	for _, entry := range GitIgnoreEntries {
		if !containsLine(content, entry) {
			toAdd = append(toAdd, entry)
		}
	}

	if len(toAdd) == 0 {
		return nil
	}

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	content += strings.Join(toAdd, "\n") + "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func containsLine(content, line string) bool {
	for _, l := range strings.Split(content, "\n") {
		if strings.TrimSpace(l) == strings.TrimSpace(line) {
			return true
		}
	}
	return false
}

type FS interface {
	fs.StatFS
}

type FindWorkspaceOptions struct {
	FindFile     string // An optional file to find in the workspace
	AllowOutside bool   // Allow searching outside the workspace
	Recursive    bool   // Recursively search for the workspace in parent directories
	FS           FS     // An optional filesystem to use
}

type FindWorkspaceResult struct {
	Data []byte
	Path string
}

func FindWorkspace(workingDir string, opts FindWorkspaceOptions) (*FindWorkspaceResult, error) {
	path, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	for {
		// If we are not allowing outside the workspace start searching in the .speakeasy folder in this dir
		if !opts.AllowOutside && filepath.Base(path) != SpeakeasyFolder && filepath.Base(path) != GenFolder {
			path = filepath.Join(path, SpeakeasyFolder)
		}

		if opts.FindFile != "" {
			path = filepath.Join(path, opts.FindFile)
		}

		_, err := stat(path, opts.FS)
		if err != nil {
			notExists := func(err error) error {
				if opts.FindFile != "" {
					return fmt.Errorf("could not find %s: %w", opts.FindFile, err)
				}
				return err
			}

			if errors.Is(err, fs.ErrNotExist) {
				// If we are not searching recursively return not found
				if !opts.Recursive {
					return nil, notExists(err)
				}

				currentDir := path
				if opts.FindFile != "" {
					currentDir = filepath.Dir(path)
				}

				switch {
				case filepath.Base(currentDir) == SpeakeasyFolder:
					// Check gen dir next
					path = filepath.Join(filepath.Dir(currentDir), GenFolder)
				case filepath.Base(currentDir) == GenFolder:
					parentDir := filepath.Dir(filepath.Dir(currentDir))

					// If the current dir parent is the same as the parent dir we have likely hit the root
					if filepath.Dir(currentDir) == parentDir {
						// Check for the root of the filesystem or path
						// ie `.` for `./something`
						// or `/` for `/some/absolute/path` in linux
						// or `:\\` for `C:\\` in windows
						if parentDir == "." || parentDir == "/" || parentDir[1:] == ":\\" {
							return nil, notExists(err)
						}
					}

					// Go up a dir
					path = parentDir
				default:
					// Check speakeasy dir next
					path = filepath.Join(currentDir, SpeakeasyFolder)
				}
				continue
			}

			return nil, fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if opts.FindFile != "" {
			data, err := readFileFunc(path, opts.FS)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", path, err)
			}
			return &FindWorkspaceResult{
				Data: data,
				Path: path,
			}, nil
		}

		return &FindWorkspaceResult{
			Path: path,
		}, nil
	}
}

func FindWorkspaceTempDir(wd string, opts FindWorkspaceOptions) string {
	res, err := FindWorkspace(wd, opts)
	if err != nil {
		res = &FindWorkspaceResult{
			Path: filepath.Join(wd, SpeakeasyFolder),
		}
	}

	return filepath.Join(res.Path, "temp")
}

func stat(path string, fs FS) (fs.FileInfo, error) {
	if fs == nil {
		return os.Stat(path)
	}
	return fs.Stat(path)
}

func readFileFunc(path string, fs fs.FS) ([]byte, error) {
	if fs == nil {
		return os.ReadFile(path)
	}
	f, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}
