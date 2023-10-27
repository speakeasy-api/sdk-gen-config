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
	GetLanguageDefaultFunc func(string, bool) (*LanguageConfig, error)
	TransformerFunc        func(*Config) (*Config, error)
)

type options struct {
	readFileFunc           ReadFileFunc
	writeFileFunc          WriteFileFunc
	UpgradeFunc            UpgradeFunc
	getLanguageDefaultFunc GetLanguageDefaultFunc
	langs                  []string
	transformerFunc        TransformerFunc
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

func WithTransformerFunc(f TransformerFunc) Option {
	return func(o *options) {
		o.transformerFunc = f
	}
}

func Load(dir string, opts ...Option) (*Config, error) {
	o := applyOptions(opts)

	newConfig := false
	newForLang := map[string]bool{}

	// Find existing config file
	data, path, err := findConfigFile(dir, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			path = filepath.Join(dir, "gen.yaml")
			newConfig = true

			for _, lang := range o.langs {
				newForLang[lang] = true
			}
		} else {
			return nil, err
		}
	}

	if !newConfig {
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

			// Write back out to disk and update data
			data, err = write(path, cfgMap, o)
			if err != nil {
				return nil, err
			}
		}

		if cfgMap["features"] == nil && version != "" {
			for _, lang := range o.langs {
				newForLang[lang] = true
			}
		} else if features, ok := cfgMap["features"].(map[string]interface{}); ok {
			for _, lang := range o.langs {
				if _, ok := features[lang]; !ok {
					newForLang[lang] = true
				}
			}
		}
	}

	requiredDefaults := map[string]bool{}
	for _, lang := range o.langs {
		requiredDefaults[lang] = newForLang[lang]
	}

	defaultCfg, err := GetDefaultConfig(newConfig, o.getLanguageDefaultFunc, requiredDefaults)
	if err != nil {
		return nil, err
	}

	cfg, err := GetDefaultConfig(newConfig, o.getLanguageDefaultFunc, requiredDefaults)
	if err != nil {
		return nil, err
	}

	if newConfig {
		// Write new cfg
		data, err = write(path, cfg, o)
		if err != nil {
			return nil, err
		}
	}

	// Okay finally able to unmarshal the config file into expected struct
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
	}

	cfg.New = newForLang

	// Maps are overwritten by unmarshal, so we need to ensure that the defaults are set
	for lang, langCfg := range defaultCfg.Languages {
		if _, ok := cfg.Languages[lang]; !ok {
			cfg.Languages[lang] = langCfg
		}

		for k, v := range langCfg.Cfg {
			if cfg.Languages[lang].Cfg == nil {
				langCfg = cfg.Languages[lang]
				langCfg.Cfg = map[string]interface{}{}
				cfg.Languages[lang] = langCfg
			}

			if _, ok := cfg.Languages[lang].Cfg[k]; !ok {
				cfg.Languages[lang].Cfg[k] = v
			}
		}
	}

	if o.transformerFunc != nil {
		cfg, err = o.transformerFunc(cfg)
		if err != nil {
			return nil, err
		}
	}

	// Finally write it out to finalize any upgrades/defaults added
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
