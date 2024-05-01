package config

import (
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/mitchellh/mapstructure"
)

const (
	v1      = "1.0.0"
	v2      = "2.0.0"
	Version = v2

	GithubWritePermission = "write"

	// Constants to be used as keys in the config files
	Languages            = "languages"
	Mode                 = "mode"
	GithubAccessToken    = "github_access_token"
	SpeakeasyApiKey      = "speakeasy_api_key"
	SpeakeasyServerURL   = "speakeasy_server_url"
	OpenAPIDocAuthHeader = "openapi_doc_auth_header"
	OpenAPIDocAuthToken  = "openapi_doc_auth_token"
	OpenAPIDocs          = "openapi_docs"
)

type OptionalPropertyRenderingOption string

const (
	OptionalPropertyRenderingOptionAlways      OptionalPropertyRenderingOption = "always"
	OptionalPropertyRenderingOptionNever       OptionalPropertyRenderingOption = "never"
	OptionalPropertyRenderingOptionWithExample OptionalPropertyRenderingOption = "withExample"
)

type UsageSnippets struct {
	OptionalPropertyRendering OptionalPropertyRenderingOption `yaml:"optionalPropertyRendering"`
	AdditionalProperties      map[string]any                  `yaml:",inline"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

type Fixes struct {
	NameResolutionDec2023                bool           `yaml:"nameResolutionDec2023"`
	ParameterOrderingFeb2024             bool           `yaml:"parameterOrderingFeb2024"`
	RequestResponseComponentNamesFeb2024 bool           `yaml:"requestResponseComponentNamesFeb2024"`
	AdditionalProperties                 map[string]any `yaml:",inline"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

type Auth struct {
	OAuth2ClientCredentialsEnabled bool `yaml:"oAuth2ClientCredentialsEnabled"`
}

type Generation struct {
	DevContainers               *DevContainers `yaml:"devContainers,omitempty"`
	BaseServerURL               string         `yaml:"baseServerUrl,omitempty"`
	SDKClassName                string         `yaml:"sdkClassName,omitempty"`
	MaintainOpenAPIOrder        bool           `yaml:"maintainOpenAPIOrder,omitempty"`
	UsageSnippets               *UsageSnippets `yaml:"usageSnippets,omitempty"`
	UseClassNamesForArrayFields bool           `yaml:"useClassNamesForArrayFields,omitempty"`
	Fixes                       *Fixes         `yaml:"fixes,omitempty"`
	Auth                        *Auth          `yaml:"auth,omitempty"`

	AdditionalProperties map[string]any `yaml:",inline"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

type DevContainers struct {
	Enabled bool `yaml:"enabled"`
	// This can be a local path or a remote URL
	SchemaPath           string         `yaml:"schemaPath"`
	AdditionalProperties map[string]any `yaml:",inline"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

type LanguageConfig struct {
	Version string         `yaml:"version"`
	Cfg     map[string]any `yaml:",inline"`
}

type SDKGenConfigField struct {
	Name                  string  `yaml:"name" json:"name"`
	Required              bool    `yaml:"required" json:"required"`
	RequiredForPublishing *bool   `yaml:"requiredForPublishing,omitempty" json:"required_for_publishing,omitempty"`
	DefaultValue          *any    `yaml:"defaultValue,omitempty" json:"default_value,omitempty"`
	Description           *string `yaml:"description,omitempty" json:"description,omitempty"`
	Language              *string `yaml:"language,omitempty" json:"language,omitempty"`
	SecretName            *string `yaml:"secretName,omitempty" json:"secret_name,omitempty"`
	ValidationRegex       *string `yaml:"validationRegex,omitempty" json:"validation_regex,omitempty"`
	ValidationMessage     *string `yaml:"validationMessage,omitempty" json:"validation_message,omitempty"`
	TestValue             *any    `yaml:"testValue,omitempty" json:"test_value,omitempty"`
}

type Configuration struct {
	ConfigVersion string                    `yaml:"configVersion"`
	Generation    Generation                `yaml:"generation"`
	Languages     map[string]LanguageConfig `yaml:",inline"`
	New           map[string]bool           `yaml:"-"`
}

type PublishWorkflow struct {
	Name        string      `yaml:"name"`
	Permissions Permissions `yaml:"permissions,omitempty"`
	On          PublishOn   `yaml:"on"`
	Jobs        Jobs        `yaml:"jobs"`
}

type PublishOn struct {
	Push Push `yaml:"push"`
}

type Push struct {
	Branches []string `yaml:"branches"`
	Paths    []string `yaml:"paths"`
}

type GenerateWorkflow struct {
	Name        string      `yaml:"name"`
	Permissions Permissions `yaml:"permissions,omitempty"`
	On          GenerateOn  `yaml:"on"`
	Jobs        Jobs        `yaml:"jobs"`
}

type Permissions struct {
	Checks       string `yaml:"checks,omitempty"`
	Contents     string `yaml:"contents,omitempty"`
	PullRequests string `yaml:"pull-requests,omitempty"`
	Statuses     string `yaml:"statuses,omitempty"`
}

type GenerateOn struct {
	WorkflowDispatch WorkflowDispatch `yaml:"workflow_dispatch"`
	Schedule         []Schedule       `yaml:"schedule,omitempty"`
}

type Jobs struct {
	Generate Job `yaml:"generate,omitempty"`
	Publish  Job `yaml:"publish,omitempty"`
}

type Job struct {
	Uses    string            `yaml:"uses"`
	With    map[string]any    `yaml:"with,omitempty"`
	Secrets map[string]string `yaml:"secrets,omitempty"`
}

type WorkflowDispatch struct {
	Inputs Inputs `yaml:"inputs"`
}

type Schedule struct {
	Cron string `yaml:"cron"`
}

type Inputs struct {
	Force Force `yaml:"force"`
}

type Force struct {
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Default     bool   `yaml:"default"`
}

func GetDefaultConfig(newSDK bool, getLangDefaultFunc GetLanguageDefaultFunc, langs map[string]bool) (*Configuration, error) {
	defaults := GetGenerationDefaults(newSDK)

	fields := map[string]any{}
	for _, field := range defaults {
		if field.DefaultValue != nil {
			if strings.Contains(field.Name, ".") {
				parts := strings.Split(field.Name, ".")

				currMap := fields

				for i, part := range parts {
					if i == len(parts)-1 {
						currMap[part] = *field.DefaultValue
					} else {
						if _, ok := currMap[part]; !ok {
							currMap[part] = map[string]any{}
						}

						currMap = currMap[part].(map[string]any)
					}
				}
			} else {
				fields[field.Name] = *field.DefaultValue
			}
		}
	}

	var genConfig Generation

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &genConfig,
		TagName: "yaml",
	})
	if err != nil {
		return nil, err
	}

	if err := d.Decode(fields); err != nil {
		return nil, err
	}

	cfg := &Configuration{
		ConfigVersion: Version,
		Generation:    genConfig,
		Languages:     map[string]LanguageConfig{},
		New:           map[string]bool{},
	}

	for lang, new := range langs {
		langDefault := &LanguageConfig{
			Version: "0.0.1",
		}

		if getLangDefaultFunc != nil {
			var err error
			langDefault, err = getLangDefaultFunc(lang, new)
			if err != nil {
				return nil, err
			}
		}

		cfg.Languages[lang] = *langDefault
	}

	return cfg, nil
}

func GetGenerationDefaults(newSDK bool) []SDKGenConfigField {
	return []SDKGenConfigField{
		{
			Name:              "baseServerURL",
			Required:          false,
			DefaultValue:      ptr(""),
			Description:       pointer.To("The base URL of the server. This value will be used if global servers are not defined in the spec."),
			ValidationRegex:   pointer.To(`^(https?):\/\/([\w\-]+\.)+\w+(\/.*)?$`),
			ValidationMessage: pointer.To("Must be a valid server URL"),
		},
		{
			Name:              "sdkClassName",
			Required:          false,
			DefaultValue:      ptr("SDK"),
			Description:       pointer.To("Generated name of the root SDK class"),
			ValidationRegex:   pointer.To(`^[\w.\-]+$`),
			ValidationMessage: pointer.To("Letters, numbers, or .-_ only"),
		},
		{
			Name:         "maintainOpenAPIOrder",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.To("Maintains the order of things like parameters and fields when generating the SDK"),
		},
		{
			Name:         "usageSnippets.optionalPropertyRendering",
			Required:     false,
			DefaultValue: ptr(OptionalPropertyRenderingOptionWithExample),
			Description:  pointer.To("Controls how optional properties are rendered in usage snippets, by default they will be rendered when an example is present in the OpenAPI spec"),
		},
		{
			Name:         "useClassNamesForArrayFields",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.To("Use class names for array fields instead of the child's schema type"),
		},
		{
			Name:         "fixes.nameResolutionDec2023",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.To("Enables a number of breaking changes introduced in December 2023, that improve name resolution for inline schemas and reduce chances of name collisions"),
		},
		{
			Name:         "fixes.parameterOrderingFeb2024",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.To("Enables fixes to the ordering of parameters for an operation if they include multiple types of parameters (ie header, query, path) to match the order they are defined in the OpenAPI spec"),
		},
		{
			Name:         "fixes.requestResponseComponentNamesFeb2024",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.To("Enables fixes that will name inline schemas within request and response components with the component name of the parent if only one content type is defined"),
		},
		{
			Name:         "fixes.methodSignaturesApr2024",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.To("Enables fixes that will detect and mark optional request and security method arguments and order them according to optionality."),
		},
		{
			Name:         "auth.oAuth2ClientCredentialsEnabled",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.To("Enables support for OAuth2 client credentials grant type (Enterprise tier only)"),
		},
	}
}

func (c *Configuration) GetGenerationFieldsMap() (map[string]any, error) {
	fields := map[string]any{}

	// Yes the decoder can encode too :face_palm:
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &fields,
		TagName: "yaml",
	})
	if err != nil {
		return nil, err
	}

	if err := d.Decode(c.Generation); err != nil {
		return nil, err
	}

	return fields, nil
}

func ptr(a any) *any {
	return &a
}
