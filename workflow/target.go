package workflow

import (
	"fmt"
	"path/filepath"
	"slices"
)

// Ensure your update schema/workflow.schema.json on changes
type Target struct {
	Target      string       `yaml:"target"`
	Source      string       `yaml:"source"`
	Output      *string      `yaml:"output,omitempty"`
	Publishing  *Publishing  `yaml:"publish,omitempty"`
	CodeSamples *CodeSamples `yaml:"codeSamples,omitempty"`
}

type Publishing struct {
	NPM       *NPM       `yaml:"npm,omitempty"`
	PyPi      *PyPi      `yaml:"pypi,omitempty"`
	Packagist *Packagist `yaml:"packagist,omitempty"`
	Java      *Java      `yaml:"java,omitempty"`
	RubyGems  *RubyGems  `yaml:"rubygems,omitempty"`
	Nuget     *Nuget     `yaml:"nuget,omitempty"`
	Terraform *Terraform `yaml:"terraform,omitempty"`
}

type CodeSamples struct {
	Output        string                    `yaml:"output,omitempty"`
	Registry      *SourceRegistry           `yaml:"registry,omitempty"`
	Style         *string                   `yaml:"style,omitempty"`         // Oneof "standard", "readme" (default: standard) (see codesamples.go)
	LangOverride  *string                   `yaml:"langOverride,omitempty"`  // The value to use for the "lang" field of each codeSample (default: auto-detect)
	LabelOverride *CodeSamplesLabelOverride `yaml:"labelOverride,omitempty"` // The value to use for the "label" field of each codeSample (default: operationId)
	Blocking      *bool                     `yaml:"blocking,omitempty"`      // Default: true. If false, code samples failures will not consider the workflow as failed
}

type CodeSamplesLabelOverride struct {
	FixedValue *string `yaml:"fixedValue,omitempty"`
	Omit       *bool   `yaml:"omit,omitempty"`
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
	OSSRHUsername     string `yaml:"ossrhUsername"`
	OSSHRPassword     string `yaml:"ossrhPassword"`
	GPGSecretKey      string `yaml:"gpgSecretKey"`
	GPGPassPhrase     string `yaml:"gpgPassPhrase"`
	UseSonatypeLegacy bool   `yaml:"useSonatypeLegacy,omitempty"`
}

type RubyGems struct {
	Token string `yaml:"token"`
}

type Nuget struct {
	APIKey string `yaml:"apiKey"`
}

type Terraform struct {
	GPGPrivateKey string `yaml:"gpgPrivateKey"`
	GPGPassPhrase string `yaml:"gpgPassPhrase"`
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

	if t.CodeSamples != nil {
		// Output only needed if registry location is unset
		if t.CodeSamples.Registry == nil {
			ext := filepath.Ext(t.CodeSamples.Output)
			if !slices.Contains([]string{".yaml", ".yml"}, ext) {
				return fmt.Errorf("failed to validate target: code samples output must be a yaml file")
			}
		}

		if t.CodeSamples.Style != nil {
			if !slices.Contains([]string{"standard", "readme"}, *t.CodeSamples.Style) {
				return fmt.Errorf("failed to validate target: code samples style must be one of 'standard', 'readme'")
			}
		}

		if t.CodeSamples.LabelOverride != nil {
			if t.CodeSamples.LabelOverride.FixedValue != nil && t.CodeSamples.LabelOverride.Omit != nil {
				return fmt.Errorf("failed to validate target: code samples labelOverride cannot be both fixedValue and omit")
			}
			if t.CodeSamples.LabelOverride.FixedValue == nil && t.CodeSamples.LabelOverride.Omit == nil {
				return fmt.Errorf("failed to validate target: code samples labelOverride must be either fixedValue or omit")
			}
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
	case "terraform":
		if p.Terraform != nil {
			if err := validateSecret(p.Terraform.GPGPrivateKey); err != nil {
				return fmt.Errorf("failed to validate terraform gpgPrivateKey: %w", err)
			}

			if err := validateSecret(p.Terraform.GPGPassPhrase); err != nil {
				return fmt.Errorf("failed to validate terraform gpgPassPhrase: %w", err)
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
	case "terraform":
		if p.Terraform != nil && p.Terraform.GPGPrivateKey != "" && p.Terraform.GPGPassPhrase != "" {
			return true
		}
	}

	return false
}
