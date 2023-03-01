package config

import (
	"bufio"
	"bytes"
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
	UpgradeFunc            UpgradeFunc
	getLanguageDefaultFunc GetLanguageDefaultFunc
	langs                  []string
}

func WithFileSystemFuncs(rf ReadFileFunc, wf WriteFileFunc) Option {
	return func(o *options) {
		o.readFileFunc = rf
		o.writeFileFunc = wf
	}
}

func WithUpgradeFunc(f UpgradeFunc) Option {
	return func(o *options) {
		o.UpgradeFunc = f
	}
}

func WithLanguageDefaultFunc(f GetLanguageDefaultFunc) Option {
	return func(o *options) {
		o.getLanguageDefaultFunc = f
	}
}

func WithLanguages(langs ...string) Option {
	return func(o *options) {
		o.langs = langs
	}
}

func Load(dir string, opts ...Option) (*Config, error) {
	o := applyOptions(opts)

	defaultCfg, err := GetDefaultConfig(o.getLanguageDefaultFunc, o.langs...)
	if err != nil {
		return nil, err
	}

	cfg, err := GetDefaultConfig(o.getLanguageDefaultFunc, o.langs...)
	if err != nil {
		return nil, err
	}

	// Find existing config file
	data, path, err := findConfigFile(dir, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			path = filepath.Join(dir, "gen.yaml")

			// Create new config file if it doesn't exist

			// Special case backwards compatibility defaults
			cfg.Generation.SDKFlattening = true // default to true for new projects

			data, err = write(path, cfg, o)
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

	if version != Version && o.UpgradeFunc != nil {
		// Upgrade config file if version is different and write it
		cfgMap, err = upgrade(version, cfgMap, o.UpgradeFunc)
		if err != nil {
			return nil, err
		}

		data, err = write(path, cfgMap, o)
		if err != nil {
			return nil, err
		}
	}

	// Okay finally able to unmarshal the config file into expected struct
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
	}

	// Maps are overwritten by unmarshal, so we need to ensure that the defaults are set
	for lang, langCfg := range defaultCfg.Languages {
		if _, ok := cfg.Languages[lang]; !ok {
			cfg.Languages[lang] = langCfg
		}

		for k, v := range langCfg.Cfg {
			if _, ok := cfg.Languages[lang].Cfg[k]; !ok {
				cfg.Languages[lang].Cfg[k] = v
			}
		}
	}

	// And write it again to ensure it's in the correct format and contains all defaults
	if _, err := write(path, cfg, o); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(dir string, cfg *Config, opts ...Option) error {
	o := applyOptions(opts)

	_, path, err := findConfigFile(dir, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			path = filepath.Join(dir, "gen.yaml")
		} else {
			return err
		}
	}

	if _, err := write(path, cfg, o); err != nil {
		return err
	}

	return nil
}

func findConfigFile(dir string, o *options) ([]byte, string, error) {
	path := filepath.Join(dir, "gen.yaml")

	for {
		data, err := o.readFileFunc(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				currentDir := filepath.Dir(path)
				if currentDir == "." || currentDir == "/" {
					return nil, "", ErrNotFound
				}

				path = filepath.Join(filepath.Dir(filepath.Dir(path)), "gen.yaml")
				continue
			}

			return nil, "", fmt.Errorf("could not read gen.yaml: %w", err)
		}

		return data, path, nil
	}
}

func write(path string, cfg any, o *options) ([]byte, error) {
	var b bytes.Buffer
	buf := bufio.NewWriter(&b)

	e := yaml.NewEncoder(buf)
	e.SetIndent(2)
	if err := e.Encode(cfg); err != nil {
		return nil, fmt.Errorf("could not marshal gen.yaml: %w", err)
	}

	if err := buf.Flush(); err != nil {
		return nil, fmt.Errorf("could not marshal gen.yaml: %w", err)
	}

	data := b.Bytes()

	if err := o.writeFileFunc(path, data, os.ModePerm); err != nil {
		return nil, fmt.Errorf("could not write gen.yaml: %w", err)
	}

	return data, nil
}

func applyOptions(opts []Option) *options {
	o := &options{
		readFileFunc:  os.ReadFile,
		writeFileFunc: os.WriteFile,
		langs:         []string{},
	}
	for _, opt := range opts {
		opt(o)
	}

	return o
}
