package workflow

import (
	"fmt"
	"path/filepath"
	"slices"
)

// Ensure you update schema/workflow.schema.json on changes
type Target struct {
	_           struct{}     `additionalProperties:"false"`
	Target      string       `yaml:"target" enum:"csharp,go,java,mcp-typescript,php,python,ruby,swift,terraform,typescript,unity,postman" required:"true"`
	Source      string       `yaml:"source" required:"true"`
	Output      *string      `yaml:"output,omitempty"`
	Publishing  *Publishing  `yaml:"publish,omitempty"`
	CodeSamples *CodeSamples `yaml:"codeSamples,omitempty"`

	// Configuration for target testing. By default, target testing is disabled.
	Testing *Testing `yaml:"testing,omitempty"`

	// Configuration for Gram deployment. By default, deployment is disabled.
	Deployment *Deployment `yaml:"deployment,omitempty"`
}

type Deployment struct {
	_       struct{} `additionalProperties:"false" description:"Gram deployment configuration for MCP servers. Presence of this block enables deployment."`
	Project string   `yaml:"project,omitempty" description:"Gram project name. Defaults to Gram's authenticated org context."`
}

// Configuration for target testing, such as `go test` for Go targets.
type Testing struct {
	_ struct{} `additionalProperties:"false" description:"Target testing configuration. By default, targets are not tested as part of the workflow."`
	// When enabled, the target will be tested as part of the workflow.
	Enabled *bool `yaml:"enabled,omitempty" description:"Defaults to false. If true, the target will be tested as part of the workflow."`

	// Configuration for mockserver handling during testing. By default, the
	// mockserver is enabled.
	MockServer *MockServer `yaml:"mockServer,omitempty" description:"Mock API server configuration for testing. By default and if generated, the mock API server is started before testing and used."`
}

// Configuration for mockserver handling during testing.
type MockServer struct {
	_ struct{} `additionalProperties:"false"`
	// When enabled, the mockserver will be started during testing.
	Enabled *bool `yaml:"enabled,omitempty" description:"Defaults to true. If false, the mock API server will not be started."`
}

type Publishing struct {
	_         struct{}   `additionalProperties:"false" description:"The publishing configuration. See https://www.speakeasy.com/docs/workflow-reference/publishing-reference"`
	NPM       *NPM       `yaml:"npm,omitempty" description:"NPM (Typescript) publishing configuration."`
	PyPi      *PyPi      `yaml:"pypi,omitempty" description:"PyPI (Python)publishing configuration."`
	Packagist *Packagist `yaml:"packagist,omitempty" description:"Packagist (PHP) publishing configuration."`
	Java      *Java      `yaml:"java,omitempty" description:"Maven (Java) publishing configuration."`
	RubyGems  *RubyGems  `yaml:"rubygems,omitempty" description:"Rubygems (Ruby) publishing configuration."`
	Nuget     *Nuget     `yaml:"nuget,omitempty" description:"NuGet (C#) publishing configuration."`
	Terraform *Terraform `yaml:"terraform,omitempty"`
}

type CodeSamples struct {
	_             struct{}                  `additionalProperties:"false" description:"Code samples configuration. See https://www.speakeasy.com/guides/openapi/x-codesamples"`
	Output        string                    `yaml:"output,omitempty" description:"The output file name"`
	Registry      *SourceRegistry           `yaml:"registry,omitempty" description:"The output registry location."`
	Style         *string                   `yaml:"style,omitempty" description:"Optional style for the code sample, one of 'standard' or 'readme'. Default is 'standard'."`     // Oneof "standard", "readme" (default: standard) (see codesamples.go)
	LangOverride  *string                   `yaml:"langOverride,omitempty" description:"Optional language override for the code sample. Default behavior is to auto-detect."`    // The value to use for the "lang" field of each codeSample (default: auto-detect)
	LabelOverride *CodeSamplesLabelOverride `yaml:"labelOverride,omitempty" description:"Optional label override for the code sample. Default is to use the operationId."`       // The value to use for the "label" field of each codeSample (default: operationId)
	Blocking      *bool                     `yaml:"blocking,omitempty" description:"Defaults to true. If false, code samples failures will not consider the workflow as failed"` // Default: true. If false, code samples failures will not consider the workflow as failed
	Disabled      *bool                     `yaml:"disabled,omitempty" description:"Optional flag to disable code samples."`                                                     // Default: false. If true, code samples will not be generated
}

type CodeSamplesLabelOverride struct {
	FixedValue *string `yaml:"fixedValue,omitempty" description:"Optional fixed value for the label."`
	Omit       *bool   `yaml:"omit,omitempty" description:"Optional flag to omit the label."`
}

var SupportedLanguagesUsageSnippets = []string{
	"go",
	"typescript",
	"python",
	"java",
	"php",
	"swift",
	"ruby",
	"csharp",
	"unity",
}

type NPM struct {
	_     struct{} `additionalProperties:"false"`
	Token string   `yaml:"token" required:"true"`
}

type PyPi struct {
	_     struct{} `additionalProperties:"false"`
	Token string   `yaml:"token" required:"true"`
}

type Packagist struct {
	_        struct{} `additionalProperties:"false"`
	Username string   `yaml:"username" required:"true"`
	Token    string   `yaml:"token" required:"true"`
}

type Java struct {
	_                 struct{} `additionalProperties:"false"`
	OSSRHUsername     string   `yaml:"ossrhUsername" required:"true"`
	OSSHRPassword     string   `yaml:"ossrhPassword" required:"true"`
	GPGSecretKey      string   `yaml:"gpgSecretKey" required:"true"`
	GPGPassPhrase     string   `yaml:"gpgPassPhrase" required:"true"`
	UseSonatypeLegacy bool     `yaml:"useSonatypeLegacy,omitempty" required:"true"`
}

type RubyGems struct {
	_     struct{} `additionalProperties:"false"`
	Token string   `yaml:"token" required:"true"`
}

type Nuget struct {
	_      struct{} `additionalProperties:"false"`
	APIKey string   `yaml:"apiKey" required:"true"`
}

type Terraform struct {
	GPGPrivateKey string `yaml:"gpgPrivateKey" required:"true"`
	GPGPassPhrase string `yaml:"gpgPassPhrase" required:"true"`
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

	if t.CodeSamplesEnabled() {
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
	case "mcp-typescript", "typescript":
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
	case "mcp-typescript", "typescript":
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

func (c CodeSamples) Enabled() bool {
	return c.Disabled == nil || !*c.Disabled
}

func (t Target) CodeSamplesEnabled() bool {
	return t.CodeSamples != nil && t.CodeSamples.Enabled()
}

func (t Target) DeploymentEnabled() bool {
	return t.Deployment != nil
}
