package module

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/zhuangbiaowei/LocalAIStack/internal/i18n"
)

var validCategories = map[Category]struct{}{
	CategoryLanguage:    {},
	CategoryRuntime:     {},
	CategoryFramework:   {},
	CategoryService:     {},
	CategoryApplication: {},
	CategoryTool:        {},
	CategoryModel:       {},
}

func ValidateManifest(manifest Manifest) error {
	var errs []error
	if strings.TrimSpace(manifest.Name) == "" {
		errs = append(errs, i18n.Errorf("name is required"))
	}
	if _, ok := validCategories[manifest.Category]; !ok {
		errs = append(errs, i18n.Errorf("invalid category %q", manifest.Category))
	}
	if manifest.Version == "" {
		errs = append(errs, i18n.Errorf("version is required"))
	} else if _, err := ParseVersion(manifest.Version); err != nil {
		errs = append(errs, i18n.Errorf("version %q is invalid: %w", manifest.Version, err))
	}
	if strings.TrimSpace(manifest.Description) == "" {
		errs = append(errs, i18n.Errorf("description is required"))
	}
	if len(manifest.Runtime.Modes) == 0 {
		errs = append(errs, i18n.Errorf("runtime.modes must include at least one entry"))
	}
	for _, dep := range manifest.Dependencies.Modules {
		if _, _, err := ParseModuleDependency(dep); err != nil {
			errs = append(errs, i18n.Errorf("invalid module dependency %q: %w", dep, err))
		}
	}
	if manifest.Integrity.Checksum != "" {
		normalized := normalizeChecksum(manifest.Integrity.Checksum)
		if _, err := hex.DecodeString(normalized); err != nil || len(normalized) != 64 {
			errs = append(errs, i18n.Errorf("invalid integrity.checksum value"))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
