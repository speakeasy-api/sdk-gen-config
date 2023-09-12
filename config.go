package config

const (
	Version               = "1.0.0"
	GithubWritePermission = "write"

	// Constants to be used as keys in the config files

	BaseServerURL                   = "baseServerUrl"
	SDKClassName                    = "sdkClassName"
	SingleTagPerOp                  = "singleTagPerOp"
	TagNamespacingDisabled          = "tagNamespacingDisabled"
	Languages                       = "languages"
	Mode                            = "mode"
	GithubAccessToken               = "github_access_token"
	SpeakeasyApiKey                 = "speakeasy_api_key"
	SpeakeasyServerURL              = "speakeasy_server_url"
	OpenAPIDocAuthHeader            = "openapi_doc_auth_header"
	OpenAPIDocAuthToken             = "openapi_doc_auth_token"
	OpenAPIDocs                     = "openapi_docs"
	OmitDescriptionIfSummaryPresent = "omitDescriptionIfSummaryPresent"
	DisableComments                 = "disableComments"
	ClientServerStatusCodesAsErrors = "clientServerStatusCodesAsErrors"
)

var CommentFields = []string{DisableComments, OmitDescriptionIfSummaryPresent}

type Management struct {
	DocChecksum       string `yaml:"docChecksum"`
	DocVersion        string `yaml:"docVersion"`
	SpeakeasyVersion  string `yaml:"speakeasyVersion"`
	GenerationVersion string `yaml:"generationVersion,omitempty"`
}

type Generation struct {
	CommentFields map[string]bool `yaml:"comments,omitempty"`
	DevContainers *DevContainers  `yaml:"devContainers,omitempty"`
	Fields        map[string]any  `yaml:",inline"`
}

type DevContainers struct {
	Enabled bool `yaml:"enabled"`
	// This can be a local path or a remote URL
	SchemaPath string `yaml:"schemaPath"`
}

type LanguageConfig struct {
	Version string         `yaml:"version"`
	Cfg     map[string]any `yaml:",inline"`
}

type SdkGenConfigField struct {
	Name                  string  `yaml:"name" json:"name"`
	Required              bool    `yaml:"required" json:"required"`
	RequiredForPublishing *bool   `yaml:"requiredForPublishing,omitempty" json:"required_for_publishing,omitempty"`
	DefaultValue          *any    `yaml:"defaultValue,omitempty" json:"default_value,omitempty"`
	Description           *string `yaml:"description,omitempty" json:"description,omitempty"`
	Language              *string `yaml:"language,omitempty" json:"language,omitempty"`
	SecretName            *string `yaml:"secretName,omitempty" json:"secret_name,omitempty"`
	ValidationRegex       *string `yaml:"validationRegex,omitempty" json:"validation_regex,omitempty"`
	ValidationMessage     *string `yaml:"validationMessage,omitempty" json:"validation_message,omitempty"`
}

type Config struct {
	ConfigVersion string                       `yaml:"configVersion"`
	Management    *Management                  `yaml:"management,omitempty"`
	Generation    Generation                   `yaml:"generation"`
	Languages     map[string]LanguageConfig    `yaml:",inline"`
	New           bool                         `yaml:"-"`
	Features      map[string]map[string]string `yaml:"features,omitempty"`
}

type PublishWorkflow struct {
	Name string    `yaml:"name"`
	On   PublishOn `yaml:"on"`
	Jobs Jobs      `yaml:"jobs"`
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
	Permissions Permissions `yaml:"permissions"`
	On          GenerateOn  `yaml:"on"`
	Jobs        Jobs        `yaml:"jobs"`
}

type Permissions struct {
	Checks       string `yaml:"checks"`
	Contents     string `yaml:"contents,omitempty"`
	PullRequests string `yaml:"pull-requests,omitempty"`
	Statuses     string `yaml:"statuses"`
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
	With    map[string]any    `yaml:"with"`
	Secrets map[string]string `yaml:"secrets"`
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

type SdkGenConfig struct {
	SdkGenLanguageConfig map[string][]SdkGenConfigField `json:"language_configs"`
	SdkGenCommonConfig   []SdkGenConfigField            `json:"common_config"`
}

func GetDefaultConfig(getLangDefaultFunc GetLanguageDefaultFunc, langs ...string) (*Config, error) {
	cfg := &Config{
		ConfigVersion: Version,
		Generation: Generation{
			Fields: map[string]any{
				SDKClassName:   "SDK",
				SingleTagPerOp: false,
			},
		},
		Languages: map[string]LanguageConfig{},
		Features:  map[string]map[string]string{},
	}

	for _, lang := range langs {
		langDefault := &LanguageConfig{
			Version: "0.0.1",
		}

		if getLangDefaultFunc != nil {
			var err error
			langDefault, err = getLangDefaultFunc(lang)
			if err != nil {
				return nil, err
			}
		}

		cfg.Languages[lang] = *langDefault
	}

	return cfg, nil
}
