package correctness

import (
	"fmt"
	"strings"

	discovery "github.com/pawnkit/pawnlint/internal/project"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ForbiddenInclude struct{}

func (ForbiddenInclude) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "forbidden-include",
		Name:            "Forbidden include",
		Summary:         "Reports includes denied by project policy",
		Explanation:     "Configured glob patterns can prohibit dependencies by their requested include path. Inactive and uncertain directives are skipped.",
		Category:        diagnostic.CategoryRestriction,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"includes", "project", "policy"},
		Options: []lint.Option{{
			Name:    "patterns",
			Summary: "Include path glob patterns to prohibit",
			Type:    lint.OptionStringList,
			Default: []string{},
		}},
	}
}

type UnusedInclude struct{}

func (UnusedInclude) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unused-include",
		Name:            "Unused include",
		Summary:         "Reports includes with no contribution to a complete build",
		Explanation:     "An include is reported only when its declarations are unused and removal has no known macro, directive, unresolved, public, native, forward, or shared dependency effect. A complete configured build context is required.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"includes", "unused", "project", "dependencies"},
	}
}

func (UnusedInclude) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	for _, issue := range ctx.Project.UnusedIncludes() {
		if current == nil || issue.Owner != current {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("include %q contributes no used declarations", issue.Include.Path),
			Filename: issue.File.Path,
			Range:    issue.File.Walk.Range(issue.Include.Node.Field("path")),
		})
	}
}

func (ForbiddenInclude) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	patterns := ruleStringList(ctx, "forbidden-include", "patterns")
	if len(patterns) == 0 {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	for _, issue := range ctx.Project.Includes() {
		if current == nil || issue.Owner != current {
			continue
		}
		path := strings.ReplaceAll(issue.Include.Path, "\\", "/")
		pattern := matchingIncludePattern(patterns, path)
		if pattern == "" {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("include %q is forbidden by pattern %q", issue.Include.Path, pattern),
			Filename: issue.File.Path,
			Range:    issue.File.Walk.Range(issue.Include.Node.Field("path")),
		})
	}
}

func ruleStringList(ctx *lint.Context, rule, option string) []string {
	if ctx == nil || ctx.PerRule == nil || ctx.PerRule[rule] == nil {
		return nil
	}
	values, _ := ctx.PerRule[rule][option].([]string)
	return values
}

func matchingIncludePattern(patterns []string, include string) string {
	for _, pattern := range patterns {
		if discovery.MatchGlob(pattern, include) {
			return pattern
		}
	}
	return ""
}

type IncludeLayering struct{}

func (IncludeLayering) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "include-layering",
		Name:            "Include layering",
		Summary:         "Reports dependencies outside a source layer's allowlist",
		Explanation:     "Configured include path globs define the dependencies allowed for matching files. Use path overrides to assign a different allowlist to each source layer.",
		Category:        diagnostic.CategoryRestriction,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"includes", "architecture", "project", "policy"},
		Options: []lint.Option{{
			Name:    "allow",
			Summary: "Include path glob patterns allowed in the layer",
			Type:    lint.OptionStringList,
			Default: []string{},
		}},
	}
}

func (IncludeLayering) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	allowed := ruleStringList(ctx, "include-layering", "allow")
	if len(allowed) == 0 {
		return
	}
	file := ctx.Project.File(ctx.File.Path)
	if file == nil {
		return
	}
	for _, include := range file.Includes {
		path := strings.ReplaceAll(include.Path, "\\", "/")
		if matchingIncludePattern(allowed, path) != "" {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message: fmt.Sprintf("include %q is outside this layer's allowed dependencies", include.Path),
			Range:   file.Walk.Range(include.Node.Field("path")),
		})
	}
}
