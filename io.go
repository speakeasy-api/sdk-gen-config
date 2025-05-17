package config

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	fs "github.com/speakeasy-api/sdk-gen-config/fs"
	"github.com/speakeasy-api/sdk-gen-config/workspace"
	"gopkg.in/yaml.v3"
)

const (
	configFile = "gen.yaml"
	lockFile   = "gen.lock"
)

type Config struct {
	Config     *Configuration
	ConfigPath string
	LockFile   *LockFile
}

type Option func(*options)

type (
	GetLanguageDefaultFunc func(string, bool) (*LanguageConfig, error)
	TransformerFunc        func(*Config) (*Config, error)
	ValidateFunc           func(Config) error
)

type options struct {
	FS                     fs.FS
	UpgradeFunc            UpgradeFunc
	getLanguageDefaultFunc GetLanguageDefaultFunc
	langs                  []string
	transformerFunc        TransformerFunc
	validateFunc           ValidateFunc
	dontWrite              bool
}

func WithFileSystem(fs fs.FS) Option {
	return func(o *options) {
		o.FS = fs
	}
}

func WithDontWrite() Option {
	return func(o *options) {
		o.dontWrite = true
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

func WithValidateFunc(f ValidateFunc) Option {
	return func(o *options) {
		o.validateFunc = f
	}
}

func FindConfigFile(dir string, fileSystem fs.FS) (*workspace.FindWorkspaceResult, error) {
	configRes, err := workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
		FindFile:     configFile,
		AllowOutside: true,
		Recursive:    true,
		FS:           fileSystem,
	})
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			configRes = &workspace.FindWorkspaceResult{
				Path: filepath.Join(dir, workspace.SpeakeasyFolder, configFile),
			}
		} else {
			return nil, err
		}
	}

	return configRes, nil
}

func Load(dir string, opts ...Option) (*Config, error) {
	o := applyOptions(opts)

	newConfig := false
	newSDK := false
	newForLang := map[string]bool{}

	// Find existing config file
	configRes, err := FindConfigFile(dir, o.FS)
	if err != nil {
		return nil, err
	}
	if configRes.Data == nil {
		newConfig = true
		newSDK = true

		for _, lang := range o.langs {
			newForLang[lang] = true
		}
	}

	// Make sure to use the same workspace dir type as the config file
	workspaceDir := filepath.Base(filepath.Dir(configRes.Path))
	if workspaceDir != workspace.SpeakeasyFolder && workspaceDir != workspace.GenFolder {
		workspaceDir = workspace.SpeakeasyFolder
	}

	newLockFile := false
	lockFileRes, err := workspace.FindWorkspace(filepath.Join(dir, workspaceDir), workspace.FindWorkspaceOptions{
		FindFile: lockFile,
		FS:       o.FS,
	})
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("could not read gen.lock: %w", err)
		}
		lockFileRes = &workspace.FindWorkspaceResult{
			Path: filepath.Join(dir, workspaceDir, lockFile),
		}
		newLockFile = true
	}

	if !newConfig {
		// Unmarshal config file and check version
		cfgMap := map[string]any{}
		if err := yaml.Unmarshal(configRes.Data, &cfgMap); err != nil {
			return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
		}

		var lockFileMap map[string]any
		lockFilePresent := false
		if lockFileRes.Data != nil {
			if err := yaml.Unmarshal(lockFileRes.Data, &lockFileMap); err != nil {
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

		// If we aren't upgrading we assume if we are missing a lock file then this is a new SDK
		if version == Version {
			newSDK = newSDK || newLockFile
		}

		if version != Version && o.UpgradeFunc != nil {
			// Upgrade config file if version is different and write it
			cfgMap, lockFileMap, err = upgrade(version, cfgMap, lockFileMap, o.UpgradeFunc)
			if err != nil {
				return nil, err
			}

			// Write back out to disk and update data
			configRes.Data, err = write(configRes.Path, cfgMap, o)
			if err != nil {
				return nil, err
			}

			if lockFileMap != nil {
				lockFileRes.Data, err = write(lockFileRes.Path, lockFileMap, o)
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

	defaultCfg, err := GetDefaultConfig(newSDK, o.getLanguageDefaultFunc, requiredDefaults)
	if err != nil {
		return nil, err
	}

	cfg, err := GetDefaultConfig(newSDK, o.getLanguageDefaultFunc, requiredDefaults)
	if err != nil {
		return nil, err
	}

	// If this is a totally new config, we need to write out to disk for following operations
	if newConfig && o.UpgradeFunc != nil {
		// Write new cfg
		configRes.Data, err = write(configRes.Path, cfg, o)
		if err != nil {
			return nil, err
		}
	}

	if lockFileRes.Data == nil && o.UpgradeFunc != nil {
		lockFile := NewLockFile()
		lockFileRes.Data, err = write(lockFileRes.Path, lockFile, o)
		if err != nil {
			return nil, err
		}
	}

	// Okay finally able to unmarshal the config file into expected struct
	if err := yaml.Unmarshal(configRes.Data, cfg); err != nil {
		return nil, fmt.Errorf("could not unmarshal gen.yaml: %w", err)
	}

	var lockFile LockFile
	if err := yaml.Unmarshal(lockFileRes.Data, &lockFile); err != nil {
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

	config := &Config{
		Config:     cfg,
		ConfigPath: configRes.Path,
		LockFile:   &lockFile,
	}

	if o.transformerFunc != nil {
		config, err = o.transformerFunc(config)
		if err != nil {
			return nil, err
		}
	}

	if o.UpgradeFunc != nil {
		// Finally write out the files to solidfy any defaults, upgrades or transformations
		if _, err := write(configRes.Path, config.Config, o); err != nil {
			return nil, err
		}
		if _, err := write(lockFileRes.Path, config.LockFile, o); err != nil {
			return nil, err
		}
	}

	if o.validateFunc != nil {
		if err := o.validateFunc(*config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func GetTemplateVersion(dir, target string, opts ...Option) (string, error) {
	o := applyOptions(opts)

	configRes, err := FindConfigFile(dir, o.FS)
	if err != nil {
		return "", err
	}
	if configRes.Data == nil {
		return "", nil
	}

	cfg := &Configuration{}
	if err := yaml.Unmarshal(configRes.Data, cfg); err != nil {
		return "", fmt.Errorf("could not unmarshal gen.yaml: %w", err)
	}

	if cfg.Languages == nil {
		return "", nil
	}

	langCfg, ok := cfg.Languages[target]
	if !ok {
		return "", nil
	}

	tv, ok := langCfg.Cfg["templateVersion"]
	if !ok {
		return "", nil
	}

	return tv.(string), nil
}

func SaveConfig(dir string, cfg *Configuration, opts ...Option) error {
	o := applyOptions(opts)

	configRes, err := FindConfigFile(dir, o.FS)
	if err != nil {
		return err
	}

	if _, err := write(configRes.Path, cfg, o); err != nil {
		return err
	}

	return nil
}

func SaveLockFile(dir string, lf *LockFile, opts ...Option) error {
	o := applyOptions(opts)

	lockFileRes, err := workspace.FindWorkspace(dir, workspace.FindWorkspaceOptions{
		FindFile: lockFile,
		FS:       o.FS,
	})
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		lockFileRes = &workspace.FindWorkspaceResult{
			Path: filepath.Join(dir, workspace.SpeakeasyFolder, lockFile),
		}
	}

	if _, err := write(lockFileRes.Path, lf, o); err != nil {
		return err
	}

	return nil
}

func GetConfigChecksum(dir string, opts ...Option) (string, error) {
	o := applyOptions(opts)

	configRes, err := FindConfigFile(dir, o.FS)
	if err != nil {
		return "", err
	}
	if configRes.Data == nil {
		return "", nil
	}

	hash := md5.Sum(configRes.Data)
	return hex.EncodeToString(hash[:]), nil
}

func write(path string, cfg any, o *options) ([]byte, error) {
	var b bytes.Buffer
	buf := bufio.NewWriter(&b)

	e := yaml.NewEncoder(buf)
	e.SetIndent(2)
	if err := e.Encode(cfg); err != nil {
		return nil, fmt.Errorf("could not marshal %s: %w", path, err)
	}

	if err := buf.Flush(); err != nil {
		return nil, fmt.Errorf("could not marshal %s: %w", path, err)
	}

	data := b.Bytes()

	if o.dontWrite {
		return data, nil
	}

	writeFileFunc := os.WriteFile
	if o.FS != nil {
		writeFileFunc = o.FS.WriteFile
	}

	if err := writeFileFunc(path, data, 0o666); err != nil {
		return nil, fmt.Errorf("could not write gen.yaml: %w", err)
	}

	return data, nil
}

func applyOptions(opts []Option) *options {
	o := &options{
		FS:    nil,
		langs: []string{},
	}
	for _, opt := range opts {
		opt(o)
	}

	return o
}
