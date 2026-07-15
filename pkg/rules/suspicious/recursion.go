package suspicious

import (
	"fmt"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RecursiveCall struct{}

func (RecursiveCall) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "recursive-call",
		Name:            "Recursive call",
		Summary:         "Reports direct and mutual recursion in the project call graph",
		Explanation:     "Recursive Pawn calls consume a fixed runtime stack and can overflow. The rule reports statically resolved direct and named-call cycles and skips ambiguous targets.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"calls", "recursion", "project"},
	}
}

func (RecursiveCall) Run(ctx *lint.Context) {
	if ctx.Project == nil || ctx.Project.CallGraph == nil {
		return
	}
	file := ctx.Project.File(ctx.File.Path)
	if !ctx.Project.InProgram(file) {
		return
	}
	for _, component := range ctx.Project.CallGraph.RecursiveComponents() {
		members := make(map[*parser.Node]bool, len(component))
		names := make([]string, 0, len(component))
		for _, function := range component {
			members[function.Node] = true
			names = append(names, function.Name)
		}
		cycle := strings.Join(names, " -> ")
		for _, function := range component {
			if function.File != file || function.Symbol == nil {
				continue
			}
			message := fmt.Sprintf("function %q participates in recursive call cycle %s", function.Name, cycle)
			if len(component) == 1 {
				message = fmt.Sprintf("function %q calls itself recursively", function.Name)
			}
			diagnosticValue := diagnostic.Diagnostic{
				Message:  message,
				Filename: ctx.File.Path,
				Range:    file.Walk.Range(function.Symbol.NameNode),
			}
			for _, call := range ctx.Project.CallGraph.Outgoing(function) {
				if call.File != file || !members[call.Callee.Node] {
					continue
				}
				diagnosticValue.Notes = append(diagnosticValue.Notes, diagnostic.RelatedLocation{
					Range:   file.Walk.Range(call.Node.Field("function")),
					Message: fmt.Sprintf("recursive call to %q is here", call.Callee.Name),
				})
				break
			}
			ctx.Report(diagnosticValue)
		}
	}
}
