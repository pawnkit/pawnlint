package config

import (
	"github.com/pawnkit/pawnlint/internal/api"
	discovery "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type Target string

const (
	TargetOpenMP Target = "openmp"
	TargetSAMP   Target = "samp"
)

func allowedTarget(t string) bool {
	switch Target(t) {
	case TargetOpenMP, TargetSAMP, "":
		return true
	default:
		return false
	}
}

type LintSection struct {
	WarningsAsErrors bool `toml:"warnings-as-errors" json:"warnings-as-errors" yaml:"warnings-as-errors"`
	MaxDiagnostics   int  `toml:"max-diagnostics" json:"max-diagnostics" yaml:"max-diagnostics"`
}

type File struct {
	Presets       []string       `toml:"presets" json:"presets" yaml:"presets"`
	Profile       string         `toml:"profile" json:"profile" yaml:"profile"`
	Target        string         `toml:"target" json:"target" yaml:"target"`
	Include       []string       `toml:"include" json:"include" yaml:"include"`
	Exclude       []string       `toml:"exclude" json:"exclude" yaml:"exclude"`
	Defines       []string       `toml:"defines" json:"defines" yaml:"defines"`
	IncludePaths  []string       `toml:"include-paths" json:"include-paths" yaml:"include-paths"`
	APIMetadata   []string       `toml:"api-metadata" json:"api-metadata" yaml:"api-metadata"`
	Baseline      string         `toml:"baseline" json:"baseline" yaml:"baseline"`
	Cache         string         `toml:"cache" json:"cache" yaml:"cache"`
	Lint          LintSection    `toml:"lint" json:"lint" yaml:"lint"`
	Rules         map[string]any `toml:"rules" json:"rules" yaml:"rules"`
	Builds        []Build        `toml:"builds" json:"builds" yaml:"builds"`
	Variants      []Variant      `toml:"variants" json:"variants" yaml:"variants"`
	Overrides     []Override     `toml:"overrides" json:"overrides" yaml:"overrides"`
	ExternalRules []ExternalRule `toml:"external-rules" json:"external-rules" yaml:"external-rules"`
	presence      filePresence
}

type ExternalRule struct {
	Name          string         `toml:"name" json:"name" yaml:"name"`
	Command       string         `toml:"command" json:"command" yaml:"command"`
	Arguments     []string       `toml:"arguments" json:"arguments" yaml:"arguments"`
	TimeoutMS     int            `toml:"timeout-ms" json:"timeout-ms" yaml:"timeout-ms"`
	Configuration map[string]any `toml:"configuration" json:"configuration" yaml:"configuration"`
}

type filePresence struct {
	warningsAsErrors bool
	maxDiagnostics   bool
}

type Build struct {
	Name             string   `toml:"name" json:"name" yaml:"name"`
	Entry            string   `toml:"entry" json:"entry" yaml:"entry"`
	WorkingDirectory string   `toml:"working-directory" json:"working-directory" yaml:"working-directory"`
	Files            []string `toml:"files" json:"files" yaml:"files"`
	Exclude          []string `toml:"exclude" json:"exclude" yaml:"exclude"`
	IncludePaths     []string `toml:"include-paths" json:"include-paths" yaml:"include-paths"`
	Defines          []string `toml:"defines" json:"defines" yaml:"defines"`
	Target           string   `toml:"target" json:"target" yaml:"target"`
}

type Variant struct {
	Name    string   `toml:"name" json:"name" yaml:"name"`
	Defines []string `toml:"defines" json:"defines" yaml:"defines"`
}

type Override struct {
	Paths []string       `toml:"paths" json:"paths" yaml:"paths"`
	Rules map[string]any `toml:"rules" json:"rules" yaml:"rules"`
}

type ResolvedOverride struct {
	Paths      []string
	Enabled    map[string]diagnostic.Severity
	Disabled   map[string]struct{}
	RuleConfig map[string]map[string]any
}

type Resolved struct {
	Source          File
	SourcePath      string
	Profile         string
	Target          Target
	Enabled         map[string]diagnostic.Severity
	RuleConfig      map[string]map[string]any
	API             *api.Metadata
	AllKnownRuleIDs map[string]struct{}
	Overrides       []ResolvedOverride
	RuleMigrations  []RuleMigration
}

type RuleMigration struct {
	Deprecated  string
	Replacement string
}

func (r *Resolved) IsEnabled(ruleID string) bool {
	s, ok := r.Enabled[ruleID]
	return ok && s != diagnostic.SeverityOff
}

func (r *Resolved) SeverityFor(ruleID string, reg *lint.Registrar) diagnostic.Severity {
	if s, ok := r.Enabled[ruleID]; ok {
		return s
	}
	if m, ok := reg.Lookup(ruleID); ok {
		return m.DefaultSeverity
	}
	return diagnostic.SeverityOff
}

func (r *Resolved) EnabledForPath(path string) map[string]diagnostic.Severity {
	if len(r.Overrides) == 0 {
		return r.Enabled
	}
	merged := make(map[string]diagnostic.Severity, len(r.Enabled))
	for id, sev := range r.Enabled {
		merged[id] = sev
	}
	for _, ov := range r.Overrides {
		if !matchesAnyGlob(ov.Paths, path) {
			continue
		}
		for id, sev := range ov.Enabled {
			merged[id] = sev
		}
		for id := range ov.Disabled {
			delete(merged, id)
		}
	}
	return merged
}

func (r *Resolved) RuleConfigForPath(path string) map[string]map[string]any {
	if len(r.Overrides) == 0 {
		return r.RuleConfig
	}
	merged := make(map[string]map[string]any, len(r.RuleConfig))
	for id, cfg := range r.RuleConfig {
		merged[id] = cfg
	}
	for _, ov := range r.Overrides {
		if !matchesAnyGlob(ov.Paths, path) {
			continue
		}
		for id, cfg := range ov.RuleConfig {
			merged[id] = cfg
		}
	}
	return merged
}

func matchesAnyGlob(patterns []string, path string) bool {
	for _, p := range patterns {
		if discovery.MatchGlob(p, path) {
			return true
		}
	}
	return false
}

func Defaults() File {
	return File{
		Profile: string(lint.ProfileRecommended),
		Target:  string(TargetOpenMP),
		Include: []string{
			"gamemodes/**/*.pwn",
			"filterscripts/**/*.pwn",
			"includes/**/*.inc",
		},
		Exclude: []string{
			"vendor/**",
			"dependencies/**",
			"generated/**",
		},
		Defines:      []string{},
		IncludePaths: []string{},
		APIMetadata:  []string{},
		Lint: LintSection{
			WarningsAsErrors: false,
			MaxDiagnostics:   0,
		},
		Rules: map[string]any{},
	}
}
