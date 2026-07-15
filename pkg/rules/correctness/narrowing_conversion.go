package correctness

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type NarrowingConversion struct{}

func (NarrowingConversion) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "narrowing-conversion",
		Name:            "Narrowing conversion",
		Summary:         "Reports values that may not fit in packed characters",
		Explanation:     "Assignments through packed-character selection store only values from 0 through 255. The rule reports definite constant and bounded ranges outside that range. Unknown values, ordinary cell subscripts, macros, and uncertain expressions are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"conversions", "packed", "characters", "ranges"},
	}
}

func (NarrowingConversion) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, assignment := range ctx.Walk.OfKind(parser.KindAssignmentExpression) {
		if assignment.Tok.Kind != token.Assign || narrowingConversionSkipped(ctx, assignment) {
			continue
		}
		left := assignment.Field("left")
		right := assignment.Field("right")
		if !packedCharacterSelection(left) {
			continue
		}
		valueRange, ok := expressionComparisonRange(ctx, right)
		if !ok || valueRange.minimum >= 0 && valueRange.maximum <= 255 {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  "value may not fit in an unsigned packed character",
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(right),
		})
	}
}

func packedCharacterSelection(node *parser.Node) bool {
	return node != nil && node.Kind == parser.KindSubscriptExpression && !node.HasError && node.Tok.Kind == token.LBrace
}

func narrowingConversionSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || node.HasError || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return true
	}
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return true
		}
	}
	return false
}
