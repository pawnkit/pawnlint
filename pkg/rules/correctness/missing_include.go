package correctness

import (
	"fmt"
	"path/filepath"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MissingInclude struct{}

func (MissingInclude) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "missing-include",
		Name:            "Missing include",
		Summary:         "Reports required includes that cannot be resolved",
		Explanation:     "Required includes must resolve through the source directory and configured include paths. Optional #tryinclude directives and uncertain paths are skipped.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"includes", "project", "configuration"},
	}
}

func (MissingInclude) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	for _, issue := range ctx.Project.MissingIncludes() {
		if current == nil || issue.Owner != current {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("include %q could not be resolved", issue.Include.Path),
			Filename: issue.File.Path,
			Range:    issue.File.Walk.Range(issue.Include.Node.Field("path")),
		})
	}
}

type AmbiguousInclude struct{}

func (AmbiguousInclude) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "ambiguous-include",
		Name:            "Ambiguous include",
		Summary:         "Reports includes shadowed by another matching file",
		Explanation:     "The include resolver selects the first matching file. Multiple matches make the selected dependency sensitive to path order and local files.",
		Category:        diagnostic.CategoryPortability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"includes", "project", "configuration", "portability"},
	}
}

func (AmbiguousInclude) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	for _, issue := range ctx.Project.AmbiguousIncludes() {
		if current == nil || issue.Owner != current || len(issue.Include.Candidates) < 2 {
			continue
		}
		selected := includeCandidatePath(issue.File.Path, issue.Include.Candidates[0])
		shadowed := includeCandidatePath(issue.File.Path, issue.Include.Candidates[1])
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("include %q selects %q but also matches %q", issue.Include.Path, selected, shadowed),
			Filename: issue.File.Path,
			Range:    issue.File.Walk.Range(issue.Include.Node.Field("path")),
		})
	}
}

func includeCandidatePath(from, candidate string) string {
	fromAbsolute, err := filepath.Abs(from)
	if err == nil {
		if relative, relErr := filepath.Rel(filepath.Dir(fromAbsolute), candidate); relErr == nil {
			return filepath.ToSlash(relative)
		}
	}
	return filepath.ToSlash(candidate)
}
