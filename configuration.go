package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/speakeasy-api/openapi/pointer"
	jsg "github.com/swaggest/jsonschema-go"
)

const (
	v1      = "1.0.0"
	v2      = "2.0.0"
	Version = v2

	GithubWritePermission = "write"

	// Constants to be used as keys in the config files
	Languages                        = "languages"
	Mode                             = "mode"
	GithubAccessToken                = "github_access_token"
	SpeakeasyApiKey                  = "speakeasy_api_key"
	SpeakeasyServerURL               = "speakeasy_server_url"
	OpenAPIDocAuthHeader             = "openapi_doc_auth_header"
	OpenAPIDocAuthToken              = "openapi_doc_auth_token"
	OpenAPIDocs                      = "openapi_docs"
	DefaultGithubTokenSecretName     = "GITHUB_TOKEN"
	DefaultSpeakeasyAPIKeySecretName = "SPEAKEASY_API_KEY"
)

type OptionalPropertyRenderingOption string

const (
	OptionalPropertyRenderingOptionAlways      OptionalPropertyRenderingOption = "always"
	OptionalPropertyRenderingOptionNever       OptionalPropertyRenderingOption = "never"
	OptionalPropertyRenderingOptionWithExample OptionalPropertyRenderingOption = "withExample"
)

type SDKInitStyle string

const (
	SDKInitStyleConstructor SDKInitStyle = "constructor"
	SDKInitStyleBuilder     SDKInitStyle = "builder"
)

// ServerIndex is a type that can be either a string (server ID) or an integer (server index)
type ServerIndex string

func (ServerIndex) PrepareJSONSchema(schema *jsg.Schema) error {
	// Set the type to an array of types [string, integer]
	schema.Type = &jsg.Type{
		SliceOfSimpleTypeValues: []jsg.SimpleType{jsg.String, jsg.Integer},
	}
	schema.WithDescription("Controls which server is shown in usage snippets. If unset, no server will be shown. If an integer, it will be used as the server index. Otherwise, it will look for a matching server ID.")
	return nil
}

type UsageSnippets struct {
	_                         struct{}                        `additionalProperties:"true" description:"Configuration for usage snippets"`
	OptionalPropertyRendering OptionalPropertyRenderingOption `yaml:"optionalPropertyRendering" enum:"always,never,withExample" description:"Controls how optional properties are rendered in usage snippets"`
	SDKInitStyle              SDKInitStyle                    `yaml:"sdkInitStyle" enum:"constructor,builder" description:"Controls how the SDK initialization is depicted in usage snippets"`
	ServerToShowInSnippets    ServerIndex                     `yaml:"serverToShowInSnippets,omitempty"` // If unset, no server will be shown, if an integer, use as server_idx, else look for a matching id
	AdditionalProperties      map[string]any                  `yaml:",inline" jsonschema:"-"`           // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

type Fixes struct {
	_                                    struct{}       `additionalProperties:"true" description:"Fixes applied to the SDK generation"`
	NameResolutionDec2023                bool           `yaml:"nameResolutionDec2023,omitempty" description:"Enables name resolution fixes from December 2023"`
	NameResolutionFeb2025                bool           `yaml:"nameResolutionFeb2025" description:"Enables name resolution fixes from February 2025"`
	ParameterOrderingFeb2024             bool           `yaml:"parameterOrderingFeb2024" description:"Enables parameter ordering fixes from February 2024"`
	RequestResponseComponentNamesFeb2024 bool           `yaml:"requestResponseComponentNamesFeb2024" description:"Enables request and response component naming fixes from February 2024"`
	SecurityFeb2025                      bool           `yaml:"securityFeb2025" description:"Enables fixes and refactoring for security that were introduced in February 2025"`
	SharedErrorComponentsApr2025         bool           `yaml:"sharedErrorComponentsApr2025" description:"Enables fixes that mean that when a component is used in both 2XX and 4XX responses, only the top level component will be duplicated to the errors scope as opposed to the entire component tree"`
	AdditionalProperties                 map[string]any `yaml:",inline" jsonschema:"-"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

func (f *Fixes) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawFixes Fixes // Prevents recursion by creating a type without the UnmarshalYAML method

	var tmp rawFixes
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	if tmp.NameResolutionFeb2025 {
		tmp.NameResolutionDec2023 = true
	}

	// Copy the temporary values back to the original struct
	*f = Fixes(tmp)
	return nil
}

type Auth struct {
	_                              struct{} `additionalProperties:"false" description:"Authentication configuration"`
	OAuth2ClientCredentialsEnabled bool     `yaml:"oAuth2ClientCredentialsEnabled" description:"Enables support for OAuth2 client credentials grant type"`
	OAuth2PasswordEnabled          bool     `yaml:"oAuth2PasswordEnabled" description:"Enables support for OAuth2 resource owner password credentials grant type"`
	HoistGlobalSecurity            bool     `yaml:"hoistGlobalSecurity" description:"Enables hoisting of operation-level security schemes to global level when no global security is defined"`
}

type Tests struct {
	_                          struct{}       `additionalProperties:"true" description:"Test generation configuration"`
	GenerateTests              bool           `yaml:"generateTests" description:"Enables generation of tests"`
	GenerateNewTests           bool           `yaml:"generateNewTests" description:"Enables generation of new tests for any new operations in the OpenAPI specification"`
	SkipResponseBodyAssertions bool           `yaml:"skipResponseBodyAssertions" description:"Skip asserting that the client got the same response bodies returned by the mock server"`
	AdditionalProperties       map[string]any `yaml:",inline" jsonschema:"-"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

// PersistentEditsEnabled controls whether user edits persist across regenerations
type PersistentEditsEnabled string

const (
	// PersistentEditsEnabledTrue enables persistent edits
	PersistentEditsEnabledTrue PersistentEditsEnabled = "true"
	// PersistentEditsEnabledNever disables persistent edits and prevents prompting
	PersistentEditsEnabledNever PersistentEditsEnabled = "never"
)

// PersistentEdits configures whether user edits to generated SDKs persist across regenerations
// When enabled, user changes are preserved via 3-way merge with Git tracking
type PersistentEdits struct {
	_ struct{} `additionalProperties:"true" description:"Configures whether user edits to generated SDKs persist across regenerations"`
	// Enabled controls preservation of user edits:
	// - "true": enables persistent edits
	// - omitted/empty: disabled but will prompt on dirty detection
	// - "never": disabled and will never prompt
	Enabled *PersistentEditsEnabled `yaml:"enabled,omitempty" enum:"true,never" description:"Enables preservation of user edits across SDK regenerations. Set to 'never' to disable prompts."`

	AdditionalProperties map[string]any `yaml:",inline" jsonschema:"-"` // Captures any additional properties
}

// IsEnabled returns true if persistent edits are enabled
func (p *PersistentEdits) IsEnabled() bool {
	return p != nil && p.Enabled != nil && *p.Enabled == PersistentEditsEnabledTrue
}

// IsNever returns true if persistent edits are explicitly set to never prompt
func (p *PersistentEdits) IsNever() bool {
	return p != nil && p.Enabled != nil && *p.Enabled == PersistentEditsEnabledNever
}

type AllOfMergeStrategy string

const (
	AllOfMergeStrategyDeepMerge    AllOfMergeStrategy = "deepMerge"
	AllOfMergeStrategyShallowMerge AllOfMergeStrategy = "shallowMerge"
)

type Schemas struct {
	_                  struct{}           `additionalProperties:"false" description:"Schema processing configuration"`
	AllOfMergeStrategy AllOfMergeStrategy `yaml:"allOfMergeStrategy" enum:"deepMerge,shallowMerge" description:"Controls how allOf schemas are merged"`
}

type Generation struct {
	_                           struct{}       `additionalProperties:"true" description:"Generation configuration"`
	DevContainers               *DevContainers `yaml:"devContainers,omitempty"`
	BaseServerURL               string         `yaml:"baseServerUrl,omitempty" description:"The base URL of the server. This value will be used if global servers are not defined in the spec."`
	SDKClassName                string         `yaml:"sdkClassName,omitempty" description:"Generated name of the root SDK class"`
	MaintainOpenAPIOrder        bool           `yaml:"maintainOpenAPIOrder,omitempty" description:"Maintains the order of parameters and fields in the OpenAPI specification"`
	DeduplicateErrors           bool           `yaml:"deduplicateErrors,omitempty" description:"Deduplicates errors that have the same schema"`
	UsageSnippets               *UsageSnippets `yaml:"usageSnippets,omitempty"`
	UseClassNamesForArrayFields bool           `yaml:"useClassNamesForArrayFields,omitempty" description:"Use class names for array fields instead of the child's schema type"`
	Fixes                       *Fixes         `yaml:"fixes,omitempty"`
	Auth                        *Auth          `yaml:"auth,omitempty"`
	SkipErrorSuffix             bool           `yaml:"skipErrorSuffix,omitempty" description:"Skips the automatic addition of an error suffix to error types"`
	InferSSEOverload            bool           `yaml:"inferSSEOverload,omitempty" description:"Generates an overload if generator detects that the request body field 'stream: true' is used for client intent to request 'text/event-stream' response"`
	SDKHooksConfigAccess        bool           `yaml:"sdkHooksConfigAccess,omitempty" description:"Enables access to the SDK configuration from hooks"`
	Schemas                     Schemas        `yaml:"schemas"`
	RequestBodyFieldName        string         `yaml:"requestBodyFieldName" description:"The name of the field to use for the request body in generated SDKs"`

	// Mock server generation configuration.
	MockServer *MockServer `yaml:"mockServer,omitempty"`

	// PersistentEdits configures whether user edits persist across regenerations
	PersistentEdits PersistentEdits `yaml:"persistentEdits"`
	Tests           Tests           `yaml:"tests,omitempty"`

	AdditionalProperties map[string]any `yaml:",inline" jsonschema:"-"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

type DevContainers struct {
	_       struct{} `additionalProperties:"true" description:"Dev container configuration"`
	Enabled bool     `yaml:"enabled" description:"Whether dev containers are enabled"`
	// This can be a local path or a remote URL
	SchemaPath           string         `yaml:"schemaPath" description:"Path to the schema file for the dev container"`
	AdditionalProperties map[string]any `yaml:",inline" jsonschema:"-"` // Captures any additional properties that are not explicitly defined for backwards/forwards compatibility
}

// Generation configuration for the inter-templated mockserver target for test generation.
type MockServer struct {
	_ struct{} `additionalProperties:"false" description:"Mock server generation configuration"`
	// Disables the code generation of the mockserver target.
	Disabled bool `yaml:"disabled" description:"Disables the code generation of the mock server target"`
}

type LanguageConfig struct {
	_       struct{}       `additionalProperties:"true" description:"Language-specific SDK configuration"`
	Version string         `yaml:"version" description:"SDK version"`
	Cfg     map[string]any `yaml:",inline" jsonschema:"-"`
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

// Ensure you update schema/gen.config.schema.json on changes
type Configuration struct {
	_             struct{}                  `title:"Gen YAML Configuration Schema" additionalProperties:"false"`
	ConfigVersion string                    `yaml:"configVersion" description:"The version of the configuration file" minLength:"1" required:"true"`
	Generation    Generation                `yaml:"generation" required:"true"`
	Languages     map[string]LanguageConfig `yaml:",inline" jsonschema:"-"`
	New           map[string]bool           `yaml:"-"`
}

type PublishWorkflow struct {
	Name        string      `yaml:"name"`
	Permissions Permissions `yaml:"permissions,omitempty"`
	On          PublishOn   `yaml:"on"`
	Jobs        Jobs        `yaml:"jobs"`
}

type PublishOn struct {
	Push             Push                   `yaml:"push"`
	WorkflowDispatch *WorkflowDispatchEmpty `yaml:"workflow_dispatch,omitempty"`
}

type TagOn struct {
	Push             Push                  `yaml:"push"`
	WorkflowDispatch WorkflowDispatchEmpty `yaml:"workflow_dispatch"`
}

type TestingOn struct {
	PullRequest      Push                    `yaml:"pull_request"`
	Push             *Push                   `yaml:"push,omitempty"`
	WorkflowDispatch WorkflowDispatchTesting `yaml:"workflow_dispatch"`
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

type TaggingWorkflow struct {
	Name        string      `yaml:"name"`
	Permissions Permissions `yaml:"permissions,omitempty"`
	On          TagOn       `yaml:"on"`
	Jobs        Jobs        `yaml:"jobs"`
}

type TestingWorkflow struct {
	Name        string      `yaml:"name"`
	Permissions Permissions `yaml:"permissions,omitempty"`
	On          TestingOn   `yaml:"on"`
	Jobs        Jobs        `yaml:"jobs"`
}

type Permissions struct {
	Checks       string `yaml:"checks,omitempty"`
	Contents     string `yaml:"contents,omitempty"`
	PullRequests string `yaml:"pull-requests,omitempty"`
	Statuses     string `yaml:"statuses,omitempty"`
	IDToken      string `yaml:"id-token,omitempty"`
}

type GenerateOn struct {
	WorkflowDispatch WorkflowDispatch `yaml:"workflow_dispatch"`
	Schedule         []Schedule       `yaml:"schedule,omitempty"`
	PullRequest      PullRequestOn    `yaml:"pull_request,omitempty"`
}

type PullRequestOn struct {
	Types []string `yaml:"types,omitempty"`
}

type Jobs struct {
	Generate Job `yaml:"generate,omitempty"`
	Publish  Job `yaml:"publish,omitempty"`
	Tag      Job `yaml:"tag,omitempty"`
	Test     Job `yaml:"test,omitempty"`
}

type Job struct {
	Uses    string            `yaml:"uses"`
	With    map[string]any    `yaml:"with,omitempty"`
	Secrets map[string]string `yaml:"secrets,omitempty"`
}

type WorkflowDispatch struct {
	Inputs Inputs `yaml:"inputs"`
}

type WorkflowDispatchTesting struct {
	Inputs InputsTesting `yaml:"inputs"`
}

type WorkflowDispatchEmpty struct{}

type Schedule struct {
	Cron string `yaml:"cron"`
}

type InputsTesting struct {
	Target Target `yaml:"target"`
}

type Inputs struct {
	Force               Force                `yaml:"force"`
	PushCodeSamplesOnly *PushCodeSamplesOnly `yaml:"push_code_samples_only,omitempty"`
	SetVersion          *SetVersion          `yaml:"set_version,omitempty"`
	Target              *Target              `yaml:"target,omitempty"`
}

type Force struct {
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Default     bool   `yaml:"default"`
}

type PushCodeSamplesOnly struct {
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Default     bool   `yaml:"default"`
}

type SetVersion struct {
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
}

type Target struct {
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
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
			Description:       pointer.From("The base URL of the server. This value will be used if global servers are not defined in the spec."),
			ValidationRegex:   pointer.From(`^(https?):\/\/([\w\-]+\.)+\w+(\/.*)?$`),
			ValidationMessage: pointer.From("Must be a valid server URL"),
		},
		{
			Name:              "sdkClassName",
			Required:          false,
			DefaultValue:      ptr("SDK"),
			Description:       pointer.From("Generated name of the root SDK class"),
			ValidationRegex:   pointer.From(`^[\w.\-]+$`),
			ValidationMessage: pointer.From("Letters, numbers, or .-_ only"),
		},
		{
			Name:         "maintainOpenAPIOrder",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Maintains the order of things like parameters and fields when generating the SDK"),
		},
		{
			Name:         "deduplicateErrors",
			Required:     false,
			DefaultValue: ptr(false),
			Description:  pointer.From("Deduplicates errors that have the same schema"),
		},
		{
			Name:         "inferSSEOverload",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Generates an overload if generator detects that the request body field `stream: true` is used for client intent to request `text/event-stream` response"),
		},
		{
			Name:         "skipErrorSuffix",
			Required:     false,
			DefaultValue: ptr(false),
			Description:  pointer.From("Skips the automatic addition of an error suffix to error types"),
		},
		{
			Name:         "usageSnippets.optionalPropertyRendering",
			Required:     false,
			DefaultValue: ptr(OptionalPropertyRenderingOptionWithExample),
			Description:  pointer.From("Controls how optional properties are rendered in usage snippets, by default they will be rendered when an example is present in the OpenAPI spec"),
		},
		{
			Name:         "usageSnippets.sdkInitStyle",
			Required:     false,
			DefaultValue: ptr(SDKInitStyleConstructor),
			Description:  pointer.From("Controls how the SDK initialization is depicted in usage snippets, by default it will use the constructor"),
		},
		{
			Name:         "usageSnippets.serverToShowInSnippets",
			Required:     false,
			DefaultValue: nil,
			Description:  pointer.From("Controls which server is shown in usage snippets. If unset, no server will be shown. If an integer, it will be used as the server index. Otherwise, it will look for a matching server ID."),
		},
		{
			Name:         "useClassNamesForArrayFields",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Use class names for array fields instead of the child's schema type"),
		},
		{
			Name:         "fixes.nameResolutionDec2023",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables a number of breaking changes introduced in December 2023, that improve name resolution for inline schemas and reduce chances of name collisions"),
		},
		{
			Name:         "fixes.nameResolutionFeb2025",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables a number of breaking changes introduced in February 2025, that improve name resolution for inline schemas and reduce chances of name collisions"),
		},
		{
			Name:         "fixes.parameterOrderingFeb2024",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables fixes to the ordering of parameters for an operation if they include multiple types of parameters (ie header, query, path) to match the order they are defined in the OpenAPI spec"),
		},
		{
			Name:         "fixes.requestResponseComponentNamesFeb2024",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables fixes that will name inline schemas within request and response components with the component name of the parent if only one content type is defined"),
		},
		{
			Name:         "fixes.methodSignaturesApr2024",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables fixes that will detect and mark optional request and security method arguments and order them according to optionality."),
		},
		{
			Name:         "auth.oAuth2ClientCredentialsEnabled",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables support for OAuth2 client credentials grant type (Enterprise tier only)"),
		},
		{
			Name:         "auth.oAuth2PasswordEnabled",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables support for OAuth2 resource owner password credentials grant type (Enterprise tier only)"),
		},
		{
			Name:         "tests.generateTests",
			Required:     false,
			DefaultValue: ptr(!newSDK),
			Description:  pointer.From("Enables generation of tests"),
		},
		{
			Name:         "tests.generateNewTests",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables generation of new tests for any new operations found in the OpenAPI spec"),
		},
		{
			Name:         "tests.skipResponseBodyAssertions",
			Required:     false,
			DefaultValue: ptr(false),
			Description:  pointer.From("Skips the generation of response body assertions in tests"),
		},
		{
			Name:         "fixes.securityFeb2025",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables fixes and refactoring for security that were introduced in February 2025"),
		},
		{
			Name:         "fixes.sharedErrorComponentsApr2025",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables fixes that mean that when a component is used in both 2XX and 4XX responses, only the top level component will be duplicated to the errors scope as opposed to the entire component tree"),
		},
		{
			Name:         "sdkHooksConfigAccess",
			Required:     false,
			DefaultValue: ptr(newSDK),
			Description:  pointer.From("Enables access to the SDK configuration from hooks"),
		},
		{
			Name:         "auth.hoistGlobalSecurity",
			Required:     false,
			DefaultValue: ptr(true),
			Description:  pointer.From("Enables hoisting of operation-level security schemes to global level when no global security is defined"),
		},
		{
			Name:         "schemas.allOfMergeStrategy",
			Required:     false,
			DefaultValue: ptr(AllOfMergeStrategyShallowMerge),
			Description:  pointer.From("Controls how allOf schemas are merged, by default they will be merged using a shallow merge"),
		},
		{
			Name:     "requestBodyFieldName",
			Required: false,
			DefaultValue: func() *any {
				if newSDK {
					return ptr("body")
				}
				return ptr("")
			}(),
			Description: pointer.From("The name of the field to use for the request body in generated SDKs"),
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

func DefaultGenerationFile() *GenerateWorkflow {
	secrets := make(map[string]string)
	secrets[GithubAccessToken] = FormatGithubSecretName(DefaultGithubTokenSecretName)
	secrets[SpeakeasyApiKey] = FormatGithubSecretName(DefaultSpeakeasyAPIKeySecretName)
	return &GenerateWorkflow{
		Name: "Generate",
		On: GenerateOn{
			WorkflowDispatch: WorkflowDispatch{
				Inputs: Inputs{
					Force: Force{
						Description: "Force generation of SDKs",
						Type:        "boolean",
						Default:     false,
					},
				},
			},
			Schedule: []Schedule{
				{
					Cron: "0 0 * * *",
				},
			},
		},
		Jobs: Jobs{
			Generate: Job{
				Uses: "speakeasy-api/sdk-generation-action/.github/workflows/workflow-executor.yaml@v15",
				With: map[string]any{
					"speakeasy_version": "latest",
					"force":             "${{ github.event.inputs.force }}",
					Mode:                "pr",
				},
				Secrets: secrets,
			},
		},
		Permissions: Permissions{
			Checks:       GithubWritePermission,
			Statuses:     GithubWritePermission,
			Contents:     GithubWritePermission,
			PullRequests: GithubWritePermission,
		},
	}
}

func FormatGithubSecretName(name string) string {
	return fmt.Sprintf("${{ secrets.%s }}", strings.ToUpper(FormatGithubSecret(name)))
}

func FormatGithubSecret(secret string) string {
	if secret != "" && secret[0] == '$' {
		secret = secret[1:]
	}
	return strings.ToLower(secret)
}
