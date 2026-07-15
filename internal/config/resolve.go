package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func Resolve(f File, sourcePath string, reg *lint.Registrar) (*Resolved, error) {
	r := &Resolved{
		Source:          f,
		SourcePath:      sourcePath,
		Enabled:         make(map[string]diagnostic.Severity),
		RuleConfig:      make(map[string]map[string]any),
		AllKnownRuleIDs: make(map[string]struct{}),
	}
	for _, id := range reg.IDs() {
		r.AllKnownRuleIDs[id] = struct{}{}
	}
	for _, alias := range reg.Aliases() {
		r.AllKnownRuleIDs[alias.Deprecated] = struct{}{}
	}

	profile := strings.TrimSpace(f.Profile)
	if profile == "" {
		profile = string(lint.ProfileRecommended)
	}
	if !lint.AllowedProfile(profile) {
		return nil, fmt.Errorf("config: unknown profile %q (allowed: %s)", profile, strings.Join(lint.AllProfiles(), ", "))
	}
	r.Profile = profile
	if f.Target == "" {
		f.Target = string(TargetOpenMP)
		r.Source.Target = f.Target
	}
	if !allowedTarget(f.Target) {
		return nil, fmt.Errorf("config: unknown target %q (allowed: openmp, samp)", f.Target)
	}
	r.Target = Target(f.Target)
	metadata, err := loadAPIMetadata(f.APIMetadata, sourcePath, f.Target)
	if err != nil {
		return nil, err
	}
	r.API = metadata
	if f.Lint.MaxDiagnostics < 0 {
		return nil, fmt.Errorf("config: lint.max-diagnostics must be non-negative")
	}
	seenVariant := make(map[string]struct{}, len(f.Variants))
	for _, v := range f.Variants {
		name := strings.TrimSpace(v.Name)
		if name == "" {
			return nil, fmt.Errorf("config: variants entries must have a non-empty name")
		}
		if _, dup := seenVariant[name]; dup {
			return nil, fmt.Errorf("config: duplicate variant name %q", name)
		}
		seenVariant[name] = struct{}{}
	}
	if len(f.Builds) > 0 && len(f.Variants) > 0 {
		return nil, fmt.Errorf("config: builds and variants cannot be configured together")
	}
	seenBuild := make(map[string]struct{}, len(f.Builds))
	for _, build := range f.Builds {
		name := strings.TrimSpace(build.Name)
		if name == "" {
			return nil, fmt.Errorf("config: builds entries must have a non-empty name")
		}
		if _, duplicate := seenBuild[name]; duplicate {
			return nil, fmt.Errorf("config: duplicate build name %q", name)
		}
		seenBuild[name] = struct{}{}
		if strings.TrimSpace(build.Entry) == "" {
			return nil, fmt.Errorf("config: build %q must have a non-empty entry", name)
		}
		if !allowedTarget(build.Target) {
			return nil, fmt.Errorf("config: build %q has unknown target %q (allowed: openmp, samp)", name, build.Target)
		}
	}

	enabled := reg.EnabledForProfile(lint.Profile(profile))

	delta, disabled, ruleConfig, migrations, errs := parseRuleTable(f.Rules, reg)
	r.RuleMigrations = appendRuleMigrations(r.RuleMigrations, migrations...)
	for id, sev := range delta {
		enabled[id] = sev
	}
	for id := range disabled {
		delete(enabled, id)
	}
	r.RuleConfig = ruleConfig

	resolvedOverrides := make([]ResolvedOverride, 0, len(f.Overrides))
	for i, ov := range f.Overrides {
		if len(ov.Paths) == 0 {
			errs = append(errs, fmt.Sprintf("config: overrides[%d] must have at least one path pattern", i))
			continue
		}
		if len(ov.Rules) == 0 {
			errs = append(errs, fmt.Sprintf("config: overrides[%d] must configure at least one rule", i))
			continue
		}
		ovEnabled, ovDisabled, ovRuleConfig, ovMigrations, ovErrs := parseRuleTable(ov.Rules, reg)
		r.RuleMigrations = appendRuleMigrations(r.RuleMigrations, ovMigrations...)
		errs = append(errs, ovErrs...)
		resolvedOverrides = append(resolvedOverrides, ResolvedOverride{
			Paths:      ov.Paths,
			Enabled:    ovEnabled,
			Disabled:   ovDisabled,
			RuleConfig: ovRuleConfig,
		})
	}
	r.Overrides = resolvedOverrides

	if len(errs) > 0 {
		sort.Strings(errs)
		return nil, errors.Join(stringsToErrors(errs)...)
	}
	r.Enabled = enabled
	return r, nil
}

func (r *Resolved) APIForTarget(target Target) (*api.Metadata, error) {
	return loadAPIMetadata(r.Source.APIMetadata, r.SourcePath, string(target))
}

func parseRuleTable(rulesTOML map[string]any, reg *lint.Registrar) (enabled map[string]diagnostic.Severity, disabled map[string]struct{}, ruleConfig map[string]map[string]any, migrations []RuleMigration, errs []string) {
	enabled = make(map[string]diagnostic.Severity)
	disabled = make(map[string]struct{})
	ruleConfig = make(map[string]map[string]any)
	ids := make([]string, 0, len(rulesTOML))
	for id := range rulesTOML {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	seen := make(map[string]string, len(ids))
	for _, configuredID := range ids {
		v := rulesTOML[configuredID]
		id, deprecated, ok := reg.ResolveID(configuredID)
		if !ok {
			errs = append(errs, fmt.Sprintf("config: unknown rule ID %q", configuredID))
			continue
		}
		if previous, duplicate := seen[id]; duplicate {
			errs = append(errs, fmt.Sprintf("config: rule %q is configured by both %q and %q", id, previous, configuredID))
			continue
		}
		seen[id] = configuredID
		if deprecated {
			migrations = append(migrations, RuleMigration{Deprecated: configuredID, Replacement: id})
		}
		meta, _ := reg.Lookup(id)
		switch tv := v.(type) {
		case string:
			sev, ok := diagnostic.ParseSeverity(tv)
			if !ok {
				errs = append(errs, fmt.Sprintf("config: rule %q: invalid severity %q", configuredID, tv))
				continue
			}
			if sev == diagnostic.SeverityOff {
				disabled[id] = struct{}{}
			} else {
				enabled[id] = sev
			}
		case map[string]any:
			cfg := cloneMap(tv)
			if sevRaw, ok := cfg["severity"]; ok {
				sevStr, _ := sevRaw.(string)
				sev, ok := diagnostic.ParseSeverity(sevStr)
				if !ok {
					errs = append(errs, fmt.Sprintf("config: rule %q: invalid severity %q", configuredID, sevStr))
				} else if sev == diagnostic.SeverityOff {
					disabled[id] = struct{}{}
				} else {
					enabled[id] = sev
				}
				delete(cfg, "severity")
			}
			errs = append(errs, validateRuleOptions(configuredID, cfg, meta.Options)...)
			ruleConfig[id] = cfg
		default:
			errs = append(errs, fmt.Sprintf("config: rule %q: value must be a severity string or a table", configuredID))
		}
	}
	return enabled, disabled, ruleConfig, migrations, errs
}

func appendRuleMigrations(existing []RuleMigration, additions ...RuleMigration) []RuleMigration {
	seen := make(map[string]struct{}, len(existing)+len(additions))
	for _, migration := range existing {
		seen[migration.Deprecated] = struct{}{}
	}
	for _, migration := range additions {
		if _, duplicate := seen[migration.Deprecated]; duplicate {
			continue
		}
		seen[migration.Deprecated] = struct{}{}
		existing = append(existing, migration)
	}
	sort.Slice(existing, func(i, j int) bool { return existing[i].Deprecated < existing[j].Deprecated })
	return existing
}

func validateRuleOptions(ruleID string, values map[string]any, options []lint.Option) []string {
	known := make(map[string]lint.Option, len(options))
	for _, option := range options {
		known[option.Name] = option
	}
	var errs []string
	for name, value := range values {
		option, ok := known[name]
		if !ok {
			errs = append(errs, fmt.Sprintf("config: rule %q: unknown option %q", ruleID, name))
			continue
		}
		normalized, err := lint.NormalizeOption(option, value)
		if err != nil {
			errs = append(errs, fmt.Sprintf("config: rule %q option %q: %v", ruleID, name, err))
			continue
		}
		values[name] = normalized
	}
	return errs
}

func loadAPIMetadata(paths []string, sourcePath, target string) (*api.Metadata, error) {
	base := "."
	if sourcePath != "" {
		base = filepath.Dir(sourcePath)
	}
	custom := make([]*api.Metadata, 0, len(paths))
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		metadata, err := api.Load(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("config: %w", err)
		}
		custom = append(custom, metadata)
	}
	metadata, err := api.Merge(target, custom...)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return metadata, nil
}

func stringsToErrors(ss []string) []error {
	out := make([]error, len(ss))
	for i, s := range ss {
		out[i] = errors.New(s)
	}
	return out
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
