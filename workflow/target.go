package workflow

import (
	"fmt"
	"slices"
)

type Target struct {
	Target     string      `yaml:"target"`
	Source     string      `yaml:"source"`
	Output     *string     `yaml:"output,omitempty"`
	Publishing *Publishing `yaml:"publish,omitempty"`
}

type Publishing struct {
	NPM       *NPM       `yaml:"npm,omitempty"`
	PyPi      *PyPi      `yaml:"pypi,omitempty"`
	Packagist *Packagist `yaml:"packagist,omitempty"`
	Java      *Java      `yaml:"java,omitempty"`
	RubyGems  *RubyGems  `yaml:"rubygems,omitempty"`
	Nuget     *Nuget     `yaml:"nuget,omitempty"`
}

type NPM struct {
	Token string `yaml:"token"`
}

type PyPi struct {
	Token string `yaml:"token"`
}

type Packagist struct {
	Username string `yaml:"username"`
	Token    string `yaml:"token"`
}

type Java struct {
	OSSRHUsername string `yaml:"ossrhUsername"`
	OSSHRPassword string `yaml:"ossrhPassword"`
	GPGSecretKey  string `yaml:"gpgSecretKey"`
	GPGPassPhrase string `yaml:"gpgPassPhrase"`
}

type RubyGems struct {
	Token string `yaml:"token"`
}

type Nuget struct {
	APIKey string `yaml:"apiKey"`
}

func (t Target) Validate(supportedLangs []string, sources map[string]Source) error {
	if t.Target == "" {
		return fmt.Errorf("target is required")
	}
	if !slices.Contains(supportedLangs, t.Target) {
		return fmt.Errorf("target %s is not supported", t.Target)
	}

	if t.Source == "" {
		return fmt.Errorf("source is required")
	}

	source, ok := sources[t.Source]
	if ok {
		if err := source.Validate(); err != nil {
			return fmt.Errorf("failed to validate source %s: %w", t.Source, err)
		}
	} else {
		switch getFileStatus(t.Source) {
		case fileStatusNotExists:
			return fmt.Errorf("source %s does not exist", t.Source)
		}
	}

	if t.Publishing != nil {
		if err := t.Publishing.Validate(t.Target); err != nil {
			return fmt.Errorf("failed to validate publish: %w", err)
		}
	}

	return nil
}

func (t Target) IsPublished() bool {
	return t.Publishing != nil && t.Publishing.IsPublished(t.Target)
}

func (p Publishing) Validate(target string) error {
	switch target {
	case "typescript":
		if p.NPM != nil && p.NPM.Token != "" {
			if err := validateSecret(p.NPM.Token); err != nil {
				return fmt.Errorf("failed to validate npm token: %w", err)
			}
		}
	case "python":
		if p.PyPi != nil && p.PyPi.Token != "" {
			if err := validateSecret(p.PyPi.Token); err != nil {
				return fmt.Errorf("failed to validate pypi token: %w", err)
			}
		}
	case "php":
		if p.Packagist != nil {
			if p.Packagist.Username == "" || p.Packagist.Token == "" {
				return fmt.Errorf("packagist username and token must be provided")
			}

			if err := validateSecret(p.Packagist.Token); err != nil {
				return fmt.Errorf("failed to validate packagist token: %w", err)
			}
		}
	case "java":
		if p.Java != nil {
			if p.Java.OSSRHUsername == "" || p.Java.OSSHRPassword == "" || p.Java.GPGSecretKey == "" || p.Java.GPGPassPhrase == "" {
				return fmt.Errorf("java publishing requires ossrhUsername, ossrhPassword, gpgSecretKey, and gpgPassPhrase")
			}

			if err := validateSecret(p.Java.OSSHRPassword); err != nil {
				return fmt.Errorf("failed to validate ossrhPassword: %w", err)
			}

			if err := validateSecret(p.Java.GPGSecretKey); err != nil {
				return fmt.Errorf("failed to validate gpgSecretKey: %w", err)
			}

			if err := validateSecret(p.Java.GPGPassPhrase); err != nil {
				return fmt.Errorf("failed to validate gpgPassPhrase: %w", err)
			}
		}
	case "ruby":
		if p.RubyGems != nil && p.RubyGems.Token != "" {
			if err := validateSecret(p.RubyGems.Token); err != nil {
				return fmt.Errorf("failed to validate rubygems token: %w", err)
			}
		}
	case "csharp":
		if p.Nuget != nil && p.Nuget.APIKey != "" {
			if err := validateSecret(p.Nuget.APIKey); err != nil {
				return fmt.Errorf("failed to validate nuget api key: %w", err)
			}
		}
	}

	return nil
}

func (p Publishing) IsPublished(target string) bool {
	switch target {
	case "typescript":
		if p.NPM != nil && p.NPM.Token != "" {
			return true
		}
	case "python":
		if p.PyPi != nil && p.PyPi.Token != "" {
			return true
		}
	case "php":
		if p.Packagist != nil {
			if p.Packagist.Username != "" && p.Packagist.Token != "" {
				return true
			}
		}
	case "java":
		if p.Java != nil {
			if p.Java.OSSRHUsername != "" && p.Java.OSSHRPassword != "" && p.Java.GPGSecretKey != "" && p.Java.GPGPassPhrase != "" {
				return true
			}
		}
	case "ruby":
		if p.RubyGems != nil && p.RubyGems.Token != "" {
			return true
		}
	case "csharp":
		if p.Nuget != nil && p.Nuget.APIKey != "" {
			return true
		}
	}

	return false
}
