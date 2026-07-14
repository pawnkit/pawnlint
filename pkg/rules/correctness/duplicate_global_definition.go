package correctness

import (
	"fmt"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DuplicateGlobalDefinition struct{}

func (DuplicateGlobalDefinition) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "duplicate-global-definition",
		Name:            "Duplicate global definition",
		Summary:         "Reports global variables defined more than once in one include graph",
		Explanation:     "A translation unit cannot contain multiple global variables with the same name. Separate entry-point files are checked independently.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"variables", "project", "includes"},
	}
}

func (DuplicateGlobalDefinition) Run(ctx *lint.Context) {
	if ctx.Project == nil {
		return
	}
	current := ctx.Project.File(ctx.File.Path)
	if current == nil {
		return
	}
	for _, duplicate := range ctx.Project.DuplicateGlobals() {
		if duplicate.Owner != current {
			continue
		}
		message := fmt.Sprintf("global variable %q is defined more than once in this include graph", duplicate.Name)
		finding := diagnostic.Diagnostic{
			Message:  message,
			Filename: duplicate.Second.File.Path,
			Range:    duplicate.Second.File.Walk.Range(duplicate.Second.Symbol.NameNode),
		}
		if duplicate.First.File == duplicate.Second.File {
			finding.Notes = []diagnostic.RelatedLocation{{
				Range:   duplicate.First.File.Walk.Range(duplicate.First.Symbol.NameNode),
				Message: "first definition is here",
			}}
		} else {
			finding.Message += "; another definition is in " + duplicate.First.File.Path
		}
		ctx.Report(finding)
	}
}
