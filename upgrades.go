package config

import (
	"errors"
	"fmt"

	"github.com/speakeasy-api/sdk-gen-config/lockfile"
)

var ErrFailedUpgrade = errors.New("failed to upgrade config")

type UpgradeFunc func(target, template, oldVersion, newVersion string, cfg map[string]any) (map[string]any, error)

func upgrade(currentVersion string, cfg map[string]any, lockFile map[string]any, uf UpgradeFunc) (map[string]any, map[string]any, error) {
	if currentVersion == "" {
		var err error
		currentVersion, cfg, err = upgradeToV100(cfg, uf)
		if err != nil {
			return nil, nil, err
		}
	}

	if currentVersion == v1 {
		var err error
		currentVersion, cfg, lockFile, err = upgradeToV200(cfg, uf)
		if err != nil {
			return nil, nil, err
		}
	}

	// Put upgrade logic for future versions here, also upgrade incrementally between versions

	if currentVersion != Version {
		return nil, nil, ErrFailedUpgrade
	}

	return cfg, lockFile, nil
}

func upgradeToV100(cfg map[string]any, uf UpgradeFunc) (string, map[string]any, error) {
	generation := map[string]any{}
	upgraded := map[string]any{
		"configVersion": v1,
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
			"disableComments":                 comments["disabled"],
			"omitDescriptionIfSummaryPresent": comments["omitdescriptionifsummarypresent"],
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

		langCfg, err := uf(lang, lang, "", v1, langCfg)
		if err != nil {
			return "", nil, err
		}

		upgraded[lang] = langCfg
	}

	return v1, upgraded, nil
}

func upgradeToV200(cfg map[string]any, uf UpgradeFunc) (string, map[string]any, map[string]any, error) {
	upgradedConfig := map[string]any{
		"configVersion": v2,
	}

	newLockFile := map[string]any{
		"lockVersion": lockfile.LockV2,
		"id":          lockfile.GetUUID(),
	}

	delete(cfg, "configVersion")

	management := map[string]any{}
	if mgmt, ok := cfg["management"]; ok {
		management, ok = mgmt.(map[string]any)
		if !ok {
			return "", nil, nil, fmt.Errorf("%w: management is not a map", ErrFailedUpgrade)
		}
		delete(cfg, "management")
	}

	if features, ok := cfg["features"]; ok {
		newLockFile["features"] = features
		delete(cfg, "features")
	}

	if generation, ok := cfg["generation"]; ok {
		genMap, ok := generation.(map[string]any)
		if !ok {
			return "", nil, nil, fmt.Errorf("%w: generation is not a map", ErrFailedUpgrade)
		}

		if repoURL, ok := genMap["repoURL"]; ok {
			management["repoURL"] = repoURL
			delete(genMap, "repoURL")
		}

		delete(genMap, "comments")
		delete(genMap, "singleTagPerOp")
		delete(genMap, "tagNamespacingDisabled")

		upgradedConfig["generation"] = genMap
		delete(cfg, "generation")
	}

	// Remaining keys are language configs
	for lang, langCfgAny := range cfg {
		langCfg, ok := langCfgAny.(map[string]any)
		if !ok {
			return "", nil, nil, fmt.Errorf("%w: %s is not a map", ErrFailedUpgrade, lang)
		}

		if published, ok := langCfg["published"]; ok {
			management["published"] = published
			delete(langCfg, "published")
		}

		if repoSubDirectory, ok := langCfg["repoSubDirectory"]; ok {
			management["repoSubDirectory"] = repoSubDirectory
			delete(langCfg, "repoSubDirectory")
		}

		if installationURL, ok := langCfg["installationURL"]; ok {
			management["installationURL"] = installationURL
			delete(langCfg, "installationURL")
		}

		if version, ok := langCfg["version"]; ok {
			management["releaseVersion"] = version
		}

		langCfg, err := uf(lang, lang, v1, v2, langCfg)
		if err != nil {
			return "", nil, nil, err
		}

		upgradedConfig[lang] = langCfg
	}

	if len(management) > 0 {
		newLockFile["management"] = management
	}

	return v2, upgradedConfig, newLockFile, nil
}
