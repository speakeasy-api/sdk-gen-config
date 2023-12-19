package config

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var ErrNotFound = errors.New("could not find gen.yaml")

type Config struct {
	Config   *Configuration
	LockFile *LockFile
}

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
	configData, configPath, err := findConfigFile(dir, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			configPath = filepath.Join(dir, ".speakeasy", "gen.yaml")
			newConfig = true

			for _, lang := range o.langs {
				newForLang[lang] = true
			}
		} else {
			return nil, err
		}
	}

	lockFilePath := filepath.Join(dir, ".speakeasy", "gen.lock")
	lockFileData, err := o.readFileFunc(lockFilePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("could not read gen.lock: %w", err)
	}

	if !newConfig {
		// Unmarshal config file and check version
		cfgMap := map[string]any{}
		if err := yaml.Unmarshal(configData, &cfgMap); err != nil {
			return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
		}

		var lockFileMap map[string]any
		lockFilePresent := false
		if lockFileData != nil {
			if err := yaml.Unmarshal(lockFileData, &lockFileMap); err != nil {
				return nil, fmt.Errorf("could not unmarshal gen.lock: %w", err)
			}
			lockFilePresent = true
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
			cfgMap, lockFileMap, err = upgrade(version, cfgMap, lockFileMap, o.UpgradeFunc)
			if err != nil {
				return nil, err
			}

			// Write back out to disk and update data
			configData, err = write(configPath, cfgMap, o)
			if err != nil {
				return nil, err
			}

			if lockFileMap != nil {
				lockFileData, err = write(lockFilePath, lockFileMap, o)
				if err != nil {
					return nil, err
				}
			}
		}

		if lockFileMap != nil {
			if lockFileMap["features"] == nil && version != "" {
				for _, lang := range o.langs {
					newForLang[lang] = true
				}
			} else if features, ok := lockFileMap["features"].(map[string]interface{}); ok {
				for _, lang := range o.langs {
					if _, ok := features[lang]; !ok {
						newForLang[lang] = true
					}
				}
			}
		} else if !lockFilePresent {
			for _, lang := range o.langs {
				newForLang[lang] = true
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

	// We only write the config files out if upgrading is enabled otherwise we just want to read the new values
	if newConfig && o.UpgradeFunc != nil {
		// Write new cfg
		configData, err = write(configPath, cfg, o)
		if err != nil {
			return nil, err
		}
	}

	if lockFileData == nil && o.UpgradeFunc != nil {
		lockFile := NewLockFile()
		lockFileData, err = write(lockFilePath, lockFile, o)
		if err != nil {
			return nil, err
		}
	}

	// Okay finally able to unmarshal the config file into expected struct
	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
	}

	var lockFile LockFile
	if err := yaml.Unmarshal(lockFileData, &lockFile); err != nil {
		return nil, fmt.Errorf("could not unmarshal gen.lock: %w", err)
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

	if lockFile.Features == nil {
		lockFile.Features = make(map[string]map[string]string)
	}
	if lockFile.Management == nil {
		lockFile.Management = &Management{}
	}

	config := &Config{
		Config:   cfg,
		LockFile: &lockFile,
	}

	if o.transformerFunc != nil {
		config, err = o.transformerFunc(config)
		if err != nil {
			return nil, err
		}
	}

	if o.UpgradeFunc != nil {
		// Finally write out the files to solidfy any defaults, upgrades or transformations
		if _, err := write(configPath, config.Config, o); err != nil {
			return nil, err
		}
		if _, err := write(lockFilePath, config.LockFile, o); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func SaveConfig(dir string, cfg *Configuration, opts ...Option) error {
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

func SaveLockFile(dir string, lockFile *LockFile, opts ...Option) error {
	o := applyOptions(opts)

	if _, err := write(filepath.Join(dir, ".speakeasy", "gen.lock"), lockFile, o); err != nil {
		return err
	}

	return nil
}

func GetConfigChecksum(dir string, opts ...Option) (string, error) {
	o := applyOptions(opts)

	data, _, err := findConfigFile(dir, o)
	if err != nil {
		return "", err
	}

	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:]), nil
}

func findConfigFile(dir string, o *options) ([]byte, string, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	path := filepath.Join(absPath, ".speakeasy", "gen.yaml")

	for {
		data, err := o.readFileFunc(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				currentDir := filepath.Dir(path)
				// Check for the root of the filesystem or path
				// ie `.` for `./something`
				// or `/` for `/some/absolute/path` in linux
				// or `:\\` for `C:\\` in windows
				if currentDir == "." || currentDir == "/" || currentDir[1:] == ":\\" {
					return nil, "", ErrNotFound
				}
				parentDir := filepath.Dir(currentDir)
				if filepath.Base(currentDir) != ".speakeasy" {
					// Check the speakeasy dir in the parent dir first
					parentDir = filepath.Join(parentDir, ".speakeasy")
				}

				path = filepath.Join(parentDir, "gen.yaml")
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
		writeFileFunc: writeFile,
		langs:         []string{},
	}
	for _, opt := range opts {
		opt(o)
	}

	return o
}

func writeFile(filename string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(filename, data, perm)
}
