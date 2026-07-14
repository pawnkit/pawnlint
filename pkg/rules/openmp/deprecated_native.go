package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DeprecatedNative struct{}

func (DeprecatedNative) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "deprecated-native",
		Name:            "Deprecated native",
		Summary:         "Reports calls to natives deprecated by the selected API",
		Explanation:     "The checked-in open.mp API metadata records compiler deprecation messages from the official includes. The rule reports direct calls and includes the upstream replacement guidance.",
		Category:        diagnostic.CategoryOpenMP,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"native", "deprecated", "migration", "api"},
	}
}

func (DeprecatedNative) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		native, name, ok := calledNative(ctx, node)
		if !ok || native.Deprecated == "" {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:   fmt.Sprintf("native %q is deprecated", name),
			Filename:  ctx.File.Path,
			Range:     ctx.Walk.Range(node.Field("function")),
			Suggested: native.Deprecated,
		})
	})
}
