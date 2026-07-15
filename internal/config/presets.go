package config

import (
	"fmt"
	"path/filepath"
)

func loadPresets(path string, stack []string) (File, error) {
	for _, active := range stack {
		if active == path {
			chain := append(append([]string(nil), stack...), path)
			return File{}, fmt.Errorf("config: preset cycle: %v", chain)
		}
	}
	file, err := loadFile(path)
	if err != nil {
		return File{}, err
	}
	stack = append(stack, path)
	merged := File{Rules: map[string]any{}}
	for _, presetName := range file.Presets {
		presetPath := presetName
		if !filepath.IsAbs(presetPath) {
			presetPath = filepath.Join(filepath.Dir(path), presetPath)
		}
		presetPath, err = filepath.Abs(presetPath)
		if err != nil {
			return File{}, fmt.Errorf("config: preset %q: %w", presetName, err)
		}
		presetPath = filepath.Clean(presetPath)
		preset, err := loadPresets(presetPath, stack)
		if err != nil {
			return File{}, err
		}
		if err := validatePreset(presetPath, preset); err != nil {
			return File{}, err
		}
		merged = mergePreset(merged, preset)
	}
	file.Presets = nil
	return mergePreset(merged, file), nil
}

func validatePreset(path string, file File) error {
	if file.Target != "" || file.Include != nil || file.Exclude != nil || file.Defines != nil || file.IncludePaths != nil || file.APIMetadata != nil || file.Baseline != "" || file.Cache != "" || file.Builds != nil || file.Variants != nil {
		return fmt.Errorf("config: preset %s may only contain presets, profile, lint, rules, and overrides", path)
	}
	return nil
}

func mergePreset(base, override File) File {
	result := base
	if override.Profile != "" {
		result.Profile = override.Profile
	}
	if override.presence.warningsAsErrors || override.Lint.WarningsAsErrors {
		result.Lint.WarningsAsErrors = override.Lint.WarningsAsErrors
		result.presence.warningsAsErrors = true
	}
	if override.presence.maxDiagnostics || override.Lint.MaxDiagnostics != 0 {
		result.Lint.MaxDiagnostics = override.Lint.MaxDiagnostics
		result.presence.maxDiagnostics = true
	}
	result.Rules = mergeRuleMaps(result.Rules, override.Rules)
	if override.Overrides != nil {
		result.Overrides = append([]Override(nil), override.Overrides...)
	}
	result.Target = override.Target
	result.Include = cloneStrings(override.Include)
	result.Exclude = cloneStrings(override.Exclude)
	result.Defines = cloneStrings(override.Defines)
	result.IncludePaths = cloneStrings(override.IncludePaths)
	result.APIMetadata = cloneStrings(override.APIMetadata)
	result.Baseline = override.Baseline
	result.Cache = override.Cache
	result.Builds = append([]Build(nil), override.Builds...)
	result.Variants = append([]Variant(nil), override.Variants...)
	return withDefaultRules(result)
}

func mergeRuleMaps(base, override map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(override))
	for key, value := range base {
		result[key] = clonePresetValue(value)
	}
	for key, value := range override {
		if current, ok := result[key].(map[string]any); ok {
			if next, ok := value.(map[string]any); ok {
				result[key] = mergeRuleMaps(current, next)
				continue
			}
		}
		result[key] = clonePresetValue(value)
	}
	return result
}

func clonePresetValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return mergeRuleMaps(nil, typed)
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = clonePresetValue(item)
		}
		return result
	case []string:
		return cloneStrings(typed)
	default:
		return value
	}
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}
