package config

import (
	"errors"
	"fmt"
)

var ErrFailedUpgrade = errors.New("failed to upgrade config")

const (
	version100 = "1.0.0"
)

type UpgradeFunc func(lang, oldVersion, newVersion string, cfg map[string]any) (map[string]any, error)

func upgrade(currentVersion string, cfg map[string]any, uf UpgradeFunc) (map[string]any, error) {
	if currentVersion == "" {
		var err error
		currentVersion, cfg, err = upgradeToV100(cfg, uf)
		if err != nil {
			return nil, err
		}
	}

	// Put upgrade logic for future versions here, also upgrade incrementally between versions

	if currentVersion != Version {
		return nil, ErrFailedUpgrade
	}

	return cfg, nil
}

func upgradeToV100(cfg map[string]any, uf UpgradeFunc) (string, map[string]any, error) {
	generation := map[string]any{}
	upgraded := map[string]any{
		"configVersion": version100,
		"generation":    generation,
	}

	if mgmtAny, ok := cfg["management"]; ok {
		mgmt, ok := mgmtAny.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("%w: management is not a map", ErrFailedUpgrade)
		}

		upgraded["management"] = map[string]any{
			"docChecksum":      mgmt["openapi-checksum"],
			"docVersion":       mgmt["openapi-version"],
			"speakeasyVersion": mgmt["speakeasy-version"],
		}
		delete(cfg, "management")
	}

	if commentsAny, ok := cfg["comments"]; ok {
		comments, ok := commentsAny.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("%w: comments is not a map", ErrFailedUpgrade)
		}

		generation["comments"] = map[string]any{
			DisableComments:                 comments["disabled"],
			OmitDescriptionIfSummaryPresent: comments["omitdescriptionifsummarypresent"],
		}
		delete(cfg, "comments")
	}

	baseServerURL, ok := cfg["baseserverurl"]
	if ok {
		generation["baseServerUrl"] = baseServerURL
		delete(cfg, "baseserverurl")
	}

	sdkClassName, ok := cfg["sdkclassname"]
	if ok {
		generation["sdkClassName"] = sdkClassName
		delete(cfg, "sdkclassname")
	}

	tagNamespacingDisabled, ok := cfg["tagnamespacingdisabled"]
	if ok {
		generation["tagNamespacingDisabled"] = tagNamespacingDisabled
		delete(cfg, "tagnamespacingdisabled")
	}

	// Remaining keys are language configs
	for lang, langCfgAny := range cfg {
		langCfg, ok := langCfgAny.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("%w: %s is not a map", ErrFailedUpgrade, lang)
		}

		langCfg, err := uf(lang, "", version100, langCfg)
		if err != nil {
			return "", nil, err
		}

		upgraded[lang] = langCfg
	}

	return version100, upgraded, nil
}
