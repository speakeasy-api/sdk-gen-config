package config

const Version = "1.0.0"

type Management struct {
	DocChecksum       string `yaml:"docChecksum"`
	DocVersion        string `yaml:"docVersion"`
	SpeakeasyVersion  string `yaml:"speakeasyVersion"`
	GenerationVersion string `yaml:"generationVersion,omitempty"`
}

type Comments struct {
	Disabled                        bool `yaml:"disabled,omitempty"`
	OmitDescriptionIfSummaryPresent bool `yaml:"omitDescriptionIfSummaryPresent,omitempty"`
}

type Generation struct {
	BaseServerURL          string    `yaml:"baseServerUrl,omitempty"`
	Comments               *Comments `yaml:"comments,omitempty"`
	TelemetryEnabled       bool      `yaml:"telemetryEnabled"`
	SDKClassName           string    `yaml:"sdkClassName"`
	TagNamespacingDisabled bool      `yaml:"tagNamespacingDisabled,omitempty"`
	SingleTagPerOp         bool      `yaml:"singleTagPerOp"`
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
	ValidationRegex       *string `yaml:"validationRegex,omitempty" json:"validation_regex,omitempty"`
	ValidationMessage     *string `yaml:"validationMessage,omitempty" json:"validation_message,omitempty"`
}

type Config struct {
	ConfigVersion string                    `yaml:"configVersion"`
	Management    *Management               `yaml:"management,omitempty"`
	Generation    Generation                `yaml:"generation"`
	Languages     map[string]LanguageConfig `yaml:",inline"`
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
	Name string     `yaml:"name"`
	On   GenerateOn `yaml:"on"`
	Jobs Jobs       `yaml:"jobs"`
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
	Uses    string  `yaml:"uses"`
	With    With    `yaml:"with"`
	Secrets Secrets `yaml:"secrets"`
}

type With struct {
	SpeakeasyVersion     string `yaml:"speakeasy_version,omitempty"`
	OpenAPIDocLocation   string `yaml:"openapi_doc_location,omitempty"`
	OpenAPIDocAuthHeader string `yaml:"openapi_doc_auth_header,omitempty"`
	Languages            string `yaml:"languages,omitempty"`
	PublishPython        bool   `yaml:"publish_python,omitempty"`
	PublishTypescript    bool   `yaml:"publish_typescript,omitempty"`
	PublishJava          bool   `yaml:"publish_java,omitempty"`
	PublishPhp           bool   `yaml:"publish_php,omitempty"`
	CreateRelease        bool   `yaml:"create_release,omitempty"`
	Mode                 string `yaml:"mode,omitempty"`
	ForceInput           string `yaml:"force,omitempty"`
}

type Secrets struct {
	GithubAccessToken   string `yaml:"github_access_token"`
	SpeakeasyApiKey     string `yaml:"speakeasy_api_key"`
	OpenAPIDocAuthToken string `yaml:"openapi_doc_auth_token,omitempty"`
	PypiToken           string `yaml:"pypi_token,omitempty"`
	NpmToken            string `yaml:"npm_token,omitempty"`
	PackagistUsername   string `yaml:"packagist_username,omitempty"`
	PackagistToken      string `yaml:"packagist_token,omitempty"`
	OssrhUsername       string `yaml:"maven_username,omitempty"`
	OssrhPassword       string `yaml:"maven_password,omitempty"`
	JavaGPGSecretKey    string `yaml:"java_gpg_secret_key,omitempty"`
	JavaGPGPassphrase   string `yaml:"java_gpg_passphrase,omitempty"`
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
			SDKClassName:   "SDK",
			SingleTagPerOp: false,
		},
		Languages: map[string]LanguageConfig{},
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
