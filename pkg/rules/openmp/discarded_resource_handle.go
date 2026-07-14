package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DiscardedResourceHandle struct{}

func (DiscardedResourceHandle) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "discarded-resource-handle",
		Name:            "Discarded resource handle",
		Summary:         "Reports resource handles discarded before they can be released",
		Explanation:     "File, database, and database-result creators return handles that must be closed or freed. The rule reports direct standalone calls whose returned handle is immediately lost.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"resource", "handle", "database", "file"},
	}
}

func (DiscardedResourceHandle) Run(ctx *lint.Context) {
	ctx.Walk.IterKind(parser.KindExpressionStatement, func(statement *parser.Node) {
		if statement.HasError || ctx.Walk.Uncertain(statement) {
			return
		}
		expression := unwrapParentheses(statement.Field("expression"))
		if expression == nil || expression.Kind != parser.KindCallExpression {
			return
		}
		native, name, ok := calledNative(ctx, expression)
		if !ok || native.Release == "" {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("resource handle returned by %q is discarded; release it with %q", name, native.Release),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(expression),
		})
	})
}
