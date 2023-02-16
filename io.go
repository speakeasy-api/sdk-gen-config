package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("could not find gen.yaml")

type Option func(*options)

type (
	ReadFileFunc           func(filename string) ([]byte, error)
	WriteFileFunc          func(filename string, data []byte, perm os.FileMode) error
	GetLanguageDefaultFunc func(string) (*LanguageConfig, error)
)

type options struct {
	readFileFunc           ReadFileFunc
	writeFileFunc          WriteFileFunc
	getLanguageDefaultFunc GetLanguageDefaultFunc
}

func WithFileSystemFuncs(rf ReadFileFunc, wf WriteFileFunc) Option {
	return func(o *options) {
		o.readFileFunc = rf
		o.writeFileFunc = wf
	}
}

func WithLanguageDefaultFunc(f GetLanguageDefaultFunc) Option {
	return func(o *options) {
		o.getLanguageDefaultFunc = f
	}
}

func Load(dir string, lang string, uf UpgradeFunc, opts ...Option) (*Config, error) {
	o := &options{
		readFileFunc:  os.ReadFile,
		writeFileFunc: os.WriteFile,
	}
	for _, opt := range opts {
		opt(o)
	}

	cfg, err := GetDefaultConfig(lang, o.getLanguageDefaultFunc)
	if err != nil {
		return nil, err
	}

	// Find existing config file
	data, err := findConfigFile(dir, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Create new config file if it doesn't exist
			data, err = write(dir, cfg, o)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Unmarshal config file and check version
	cfgMap := map[string]any{}
	if err := yaml.Unmarshal(data, &cfgMap); err != nil {
		return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
	}

	version := ""

	v, ok := cfgMap["configVersion"]
	if ok {
		version, ok = v.(string)
		if !ok {
			version = ""
		}
	}

	if version != Version {
		// Upgrade config file if version is different and write it
		cfgMap, err = upgrade(version, cfgMap, uf)
		if err != nil {
			return nil, err
		}

		data, err = write(dir, cfgMap, o)
		if err != nil {
			return nil, err
		}
	}

	// Okay finally able to unmarshal the config file into expected struct
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
	}

	return cfg, nil
}

func findConfigFile(dir string, o *options) ([]byte, error) {
	path := filepath.Join(dir, "gen.yaml")

	for {
		data, err := o.readFileFunc(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				currentDir := filepath.Dir(path)
				if currentDir == "." || currentDir == "/" {
					return nil, ErrNotFound
				}

				path = filepath.Join(filepath.Dir(filepath.Dir(path)), "gen.yaml")
				continue
			}

			return nil, fmt.Errorf("could not read gen.yaml: %w", err)
		}

		return data, nil
	}
}

func write(dir string, cfg any, o *options) ([]byte, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not marshal gen.yaml: %w", err)
	}

	if err := o.writeFileFunc(filepath.Join(dir, "gen.yaml"), data, os.ModePerm); err != nil {
		return nil, fmt.Errorf("could not write gen.yaml: %w", err)
	}

	return data, nil
}
