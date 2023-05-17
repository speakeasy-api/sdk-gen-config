package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_upgrade_Success(t *testing.T) {
	type args struct {
		currentVersion string
		cfg            map[string]any
	}
	tests := []struct {
		name string
		args args
		want map[string]any
	}{
		{
			name: "upgrade all fields to v1.0.0",
			args: args{
				currentVersion: "",
				cfg: map[string]any{
					"management": map[string]any{
						"openapi-checksum":  "123",
						"openapi-version":   "1.0.0",
						"speakeasy-version": "1.0.0",
					},
					"comments": map[string]any{
						"disabled":                        true,
						"omitdescriptionifsummarypresent": true,
					},
					"baseserverurl":          "http://localhost:8080",
					"sdkclassname":           "MySDK",
					"tagnamespacingdisabled": true,
					"go": map[string]any{
						"version":     "0.0.1",
						"packagename": "openapi",
					},
				},
			},
			want: map[string]any{
				"configVersion": version100,
				"management": map[string]any{
					"docChecksum":      "123",
					"docVersion":       "1.0.0",
					"speakeasyVersion": "1.0.0",
				},
				"generation": map[string]any{
					"baseServerUrl":          "http://localhost:8080",
					"sdkClassName":           "MySDK",
					"tagNamespacingDisabled": true,
					"comments": map[string]any{
						"disableComments":                 true,
						"omitDescriptionIfSummaryPresent": true,
					},
				},
				"go": map[string]any{
					"version":     "0.0.1",
					"packageName": "openapi",
				},
			},
		},
		{
			name: "upgrade only some fields to v1.0.0",
			args: args{
				currentVersion: "",
				cfg: map[string]any{
					"sdkclassname": "MySDK",
					"go": map[string]any{
						"version":     "0.0.1",
						"packagename": "openapi",
					},
				},
			},
			want: map[string]any{
				"configVersion": version100,
				"generation": map[string]any{
					"sdkClassName": "MySDK",
				},
				"go": map[string]any{
					"version":     "0.0.1",
					"packageName": "openapi",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgraded, err := upgrade(tt.args.currentVersion, tt.args.cfg, testUpdateLang)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, upgraded)
		})
	}
}

func testUpdateLang(lang, oldVersion, newVersion string, cfg map[string]any) (map[string]any, error) {
	if oldVersion == "" {
		switch lang {
		case "go":
			upgraded := map[string]any{
				"version":     "0.0.1",
				"packageName": "openapi",
			}

			version, ok := cfg["version"]
			if ok {
				upgraded["version"] = version
			}

			packageName, ok := cfg["packagename"]
			if ok {
				upgraded["packageName"] = packageName
			}

			return upgraded, nil
		}
	}

	return cfg, nil
}
