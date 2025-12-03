package config

import (
	"testing"

	"github.com/speakeasy-api/sdk-gen-config/lockfile"
	"github.com/stretchr/testify/assert"
)

func Test_upgrade_Success(t *testing.T) {
	getUUID = func() string {
		return "123"
	}
	lockfile.GetUUID = getUUID

	type args struct {
		currentVersion string
		cfg            map[string]any
		lockFile       map[string]any
	}
	tests := []struct {
		name         string
		args         args
		wantCfg      map[string]any
		wantLockFile map[string]any
	}{
		{
			name: "upgrades all fields from the original version through v1.0.0 to v2.0.0",
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
			wantCfg: map[string]any{
				"configVersion": v2,
				"generation": map[string]any{
					"baseServerUrl": "http://localhost:8080",
					"sdkClassName":  "MySDK",
				},
				"go": map[string]any{
					"version":     "0.0.1",
					"packageName": "openapi",
				},
			},
			wantLockFile: map[string]any{
				"lockVersion": lockfile.LockV2,
				"id":          "123",
				"management": map[string]any{
					"docChecksum":      "123",
					"docVersion":       "1.0.0",
					"speakeasyVersion": "1.0.0",
					"releaseVersion":   "0.0.1",
				},
			},
		},
		{
			name: "upgrade only some fields from the original version through v1.0.0 to v2.0.0",
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
			wantCfg: map[string]any{
				"configVersion": v2,
				"generation": map[string]any{
					"sdkClassName": "MySDK",
				},
				"go": map[string]any{
					"version":     "0.0.1",
					"packageName": "openapi",
				},
			},
			wantLockFile: map[string]any{
				"lockVersion": lockfile.LockV2,
				"id":          "123",
				"management": map[string]any{
					"releaseVersion": "0.0.1",
				},
			},
		},
		{
			name: "upgrades from v1.0.0 to v2.0.0",
			args: args{
				currentVersion: v1,
				cfg: map[string]any{
					"configVersion": v1,
					"management": map[string]any{
						"docChecksum":      "123",
						"docVersion":       "1.0.0",
						"speakeasyVersion": "1.0.0",
					},
					"generation": map[string]any{
						"baseServerUrl":          "http://localhost:8080",
						"sdkClassName":           "MySDK",
						"tagNamespacingDisabled": true,
						"singleTagPerOp":         true,
						"comments": map[string]any{
							"disableComments":                 true,
							"omitDescriptionIfSummaryPresent": true,
						},
						"repoURL": "http://localhost:8080",
					},
					"features": map[string]map[string]string{
						"go": {
							"core":   "1.0.0",
							"errors": "1.0.0",
						},
					},
					"go": map[string]any{
						"version":          "0.0.1",
						"packageName":      "openapi",
						"published":        true,
						"installationURL":  "https://github.com/speakeasy-api/sdk",
						"repoSubDirectory": "./go",
					},
				},
			},
			wantCfg: map[string]any{
				"configVersion": v2,
				"generation": map[string]any{
					"baseServerUrl": "http://localhost:8080",
					"sdkClassName":  "MySDK",
				},
				"go": map[string]any{
					"version":     "0.0.1",
					"packageName": "openapi",
				},
			},
			wantLockFile: map[string]any{
				"lockVersion": lockfile.LockV2,
				"id":          "123",
				"management": map[string]any{
					"docChecksum":      "123",
					"docVersion":       "1.0.0",
					"speakeasyVersion": "1.0.0",
					"repoURL":          "http://localhost:8080",
					"published":        true,
					"installationURL":  "https://github.com/speakeasy-api/sdk",
					"repoSubDirectory": "./go",
					"releaseVersion":   "0.0.1",
				},
				"features": map[string]map[string]string{
					"go": {
						"core":   "1.0.0",
						"errors": "1.0.0",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgraded, lockFile, err := upgrade(tt.args.currentVersion, tt.args.cfg, tt.args.lockFile, testUpdateLang)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCfg, upgraded)
			assert.Equal(t, tt.wantLockFile, lockFile)
		})
	}
}

func testUpdateLang(lang, template, oldVersion, newVersion string, cfg map[string]any) (map[string]any, error) {
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
