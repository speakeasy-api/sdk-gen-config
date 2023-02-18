package config

const Version = "1.0.0"

type Management struct {
	DocChecksum      string `yaml:"docChecksum"`
	DocVersion       string `yaml:"docVersion"`
	SpeakeasyVersion string `yaml:"speakeasyVersion"`
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
}

type LanguageConfig struct {
	Version string         `yaml:"version"`
	Cfg     map[string]any `yaml:",inline"`
}

type Config struct {
	ConfigVersion string                    `yaml:"configVersion"`
	Management    *Management               `yaml:"management,omitempty"`
	Generation    Generation                `yaml:"generation"`
	Languages     map[string]LanguageConfig `yaml:",inline"`
}

func GetDefaultConfig(getLangDefaultFunc GetLanguageDefaultFunc, langs ...string) (*Config, error) {
	cfg := &Config{
		ConfigVersion: Version,
		Generation: Generation{
			SDKClassName: "SDK",
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
