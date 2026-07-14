package correctness

import (
	"fmt"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DuplicateFunctionDefinition struct{}

func (DuplicateFunctionDefinition) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "duplicate-function-definition",
		Name:            "Duplicate function definition",
		Summary:         "Reports functions defined more than once in one include graph",
		Explanation:     "A translation unit cannot contain multiple definitions of the same function. Separate entry-point files are checked independently.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"functions", "project", "includes"},
	}
}

func (DuplicateFunctionDefinition) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	if current == nil {
		return
	}
	for _, duplicate := range ctx.Project.DuplicateFunctions() {
		if duplicate.Owner != current {
			continue
		}
		message := fmt.Sprintf("function %q is defined more than once in this include graph", duplicate.Name)
		finding := diagnostic.Diagnostic{
			Message:  message,
			Filename: duplicate.Second.File.Path,
			Range:    duplicate.Second.File.Walk.Range(duplicate.Second.Node.Field("name")),
		}
		if duplicate.First.File == duplicate.Second.File {
			finding.Notes = []diagnostic.RelatedLocation{{
				Range:   duplicate.First.File.Walk.Range(duplicate.First.Node.Field("name")),
				Message: "first definition is here",
			}}
		} else {
			finding.Message += "; another definition is in " + duplicate.First.File.Path
		}
		ctx.Report(finding)
	}
}
