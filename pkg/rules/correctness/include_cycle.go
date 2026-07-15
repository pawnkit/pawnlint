package correctness

import (
	"path/filepath"
	"strings"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type IncludeCycle struct{}

func (IncludeCycle) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "include-cycle",
		Name:            "Include cycle",
		Summary:         "Reports cycles in the resolved include graph",
		Explanation:     "Include cycles make compilation order and preprocessor state difficult to reason about. Unresolved, inactive, optional-missing, and uncertain includes are skipped.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"includes", "project", "dependencies"},
	}
}

func (IncludeCycle) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	if current == nil {
		return
	}
	for _, cycle := range ctx.Project.IncludeCycles() {
		if cycle.Owner != current || len(cycle.Edges) == 0 {
			continue
		}
		closing := cycle.Edges[len(cycle.Edges)-1]
		ctx.Report(diagnostic.Diagnostic{
			Message:  "include cycle: " + includeCyclePath(cycle),
			Filename: closing.From.Path,
			Range:    closing.From.Walk.Range(closing.Include.Node.Field("path")),
		})
	}
}

func includeCyclePath(cycle project.IncludeCycle) string {
	parts := make([]string, 0, len(cycle.Edges)+1)
	parts = append(parts, filepath.Base(cycle.Edges[0].From.Path))
	for _, edge := range cycle.Edges {
		parts = append(parts, filepath.Base(edge.To.Path))
	}
	return strings.Join(parts, " -> ")
}
