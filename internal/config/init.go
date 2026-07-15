package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pawnkit/pawnlint/pkg/lint"
)

func InitConfigText(reg *lint.Registrar) string {
	var b strings.Builder
	b.WriteString("# pawnlint configuration. See docs/configuration.md for the full reference.\n\n")
	b.WriteString(`# Profile selects the default set of rules. Allowed:
#   recommended (default), strict, all.
profile = "recommended"

# Target dialect. Allowed: openmp (default), samp.
target = "openmp"

# Glob patterns of sources to lint.
include = [
 "gamemodes/**/*.pwn",
 "filterscripts/**/*.pwn",
 "includes/**/*.inc",
]

# Glob patterns to exclude.
exclude = [
 "vendor/**",
 "dependencies/**",
 "generated/**",
]

# Predefined preprocessor symbols.
defines = []

# Generated build contexts. See docs/configuration.md#builds.
#   [[builds]]
#   name = "main"
#   entry = "gamemodes/main.pwn"
#   files = ["gamemodes/**", "includes/**"]
#   include-paths = ["dependencies/library"]
#   defines = ["FEATURE"]

# Additional define sets to analyze and merge results from. Code guarded by
# #if defined NAME is only analyzed when NAME is known, so without a variant
# for each target the untested target's branch is silently skipped. See
# docs/configuration.md#variants.
#
# Example:
#   [[variants]]
#   name = "openmp"
#   defines = ["OPENMP"]
#
#   [[variants]]
#   name = "samp"
#   defines = ["SAMP"]

# Include search paths (mirrors compiler -i).
include-paths = []

# Additional API metadata JSON files.
api-metadata = []

# Existing findings baseline. Paths are relative to this configuration file.
# baseline = "pawnlint-baseline.json"

[lint]
# Treat warnings as errors when the failure threshold is reached.
warnings-as-errors = false
# Maximum diagnostics to emit (0 = unlimited).
max-diagnostics = 0

# Per-rule overrides. Each rule can be set to a severity string
# ("error", "warning", "info", "hint", "off") or to a table with a "severity"
# key plus rule-specific options. Unknown rule IDs are errors.
#
# Example:
#   [rules]
#   float-equality = "off"
#
#   [rules.cyclomatic-complexity]
#   severity = "warning"
#   maximum = 20
[rules]
`)
	metas := reg.Sorted()
	for _, m := range metas {
		enabled := "yes"
		if !m.DefaultEnabled {
			enabled = "no"
		}
		b.WriteString("# ")
		b.WriteString(m.ID)
		b.WriteString(" — ")
		b.WriteString(m.Summary)
		b.WriteString(" (default: ")
		b.WriteString(m.DefaultSeverity.String())
		b.WriteString(", enabled: ")
		b.WriteString(enabled)
		b.WriteString(")\n")
		b.WriteString("# ")
		b.WriteString(m.ID)
		b.WriteString(" = \"")
		b.WriteString(m.DefaultSeverity.String())
		b.WriteString("\"\n\n")
	}
	b.WriteString(`# Path-scoped rule overrides. Same shape as [rules], applied only to files
# whose project-relative path matches at least one glob in "paths". Later
# overrides win. See docs/configuration.md#overrides.
#
# Example:
#   [[overrides]]
#   paths = ["testdata/**", "generated/**"]
#   [overrides.rules]
#   unused-local = "off"
`)
	return b.String()
}

func ListRulesText(reg *lint.Registrar) string {
	metas := reg.All()
	sort.SliceStable(metas, func(i, j int) bool {
		if metas[i].Category != metas[j].Category {
			return metas[i].Category < metas[j].Category
		}
		return metas[i].ID < metas[j].ID
	})
	var b strings.Builder
	for _, m := range metas {
		b.WriteString(m.ID)
		b.WriteString("\t")
		b.WriteString(m.Category.String())
		b.WriteString("\t")
		b.WriteString(m.DefaultSeverity.String())
		b.WriteString("\t")
		if m.DefaultEnabled {
			b.WriteString("on")
		} else {
			b.WriteString("off")
		}
		b.WriteString("\t")
		b.WriteString(m.Summary)
		b.WriteString("\n")
	}
	return b.String()
}

func ExplainText(m lint.Metadata) string {
	var b strings.Builder
	b.WriteString(m.ID)
	b.WriteString(" — ")
	b.WriteString(m.Summary)
	b.WriteString("\n\nCategory:     ")
	b.WriteString(m.Category.String())
	b.WriteString("\nSeverity:     ")
	b.WriteString(m.DefaultSeverity.String())
	b.WriteString("\nAnalysis:     ")
	b.WriteString(analysisLevelString(m.AnalysisLevel))
	b.WriteString("\nStability:    ")
	b.WriteString(m.Stability.String())
	b.WriteString("\nDefault:      ")
	if m.DefaultEnabled {
		b.WriteString("enabled")
	} else {
		b.WriteString("disabled")
	}
	b.WriteString("\nFixable:      ")
	if m.Fixable {
		b.WriteString("yes")
	} else {
		b.WriteString("no")
	}
	if len(m.Tags) > 0 {
		b.WriteString("\nTags:         ")
		b.WriteString(strings.Join(m.Tags, ", "))
	}
	if len(m.Options) > 0 {
		b.WriteString("\nOptions:")
		for _, option := range m.Options {
			b.WriteString("\n  ")
			b.WriteString(option.Name)
			b.WriteString(" (")
			b.WriteString(option.Type.String())
			b.WriteString(", default ")
			b.WriteString(fmt.Sprint(option.Default))
			b.WriteString("): ")
			b.WriteString(option.Summary)
		}
	}
	b.WriteString("\n\n")
	b.WriteString(m.Explanation)
	return b.String()
}

func analysisLevelString(l lint.AnalysisLevel) string {
	switch l {
	case lint.SyntaxAnalysis:
		return "syntax"
	case lint.SemanticAnalysis:
		return "semantic"
	case lint.ControlFlowAnalysis:
		return "control-flow"
	case lint.ProjectAnalysis:
		return "project"
	default:
		return "unknown"
	}
}
