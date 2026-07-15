package correctness

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type NonCallableSymbol struct{}

func (NonCallableSymbol) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "non-callable-symbol",
		Name:            "Non-callable symbol",
		Summary:         "Reports calls whose callee resolves to a variable, not a function",
		Explanation:     explanationNonCallableSymbol,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"call", "shadowing", "semantic"},
	}
}

const explanationNonCallableSymbol = `A local, parameter, or global variable is not callable. The most common cause
is a variable whose name shadows a native or user function:

` + "```pawn" + `
new time;
printf("%d", time());  // time is now the local cell, not the time() native
` + "```" + `

The compiler rejects this with "invalid function call, not a valid address".
Rename the variable or the call to resolve the ambiguity.`

func (NonCallableSymbol) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindCallExpression, func(node *parser.Node) {
		if ctx.Walk.Uncertain(node) {
			return
		}
		callee := node.Field("function")
		if callee == nil || callee.Kind != parser.KindIdentifier {
			return
		}
		symbol := ctx.Semantic.ResolveAsCallTarget(callee)
		if symbol == nil || symbol.Ambiguous || symbol.Kind == semantic.SymbolFunction {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("%s %q is not callable, so calling it is a compile error", symbolKindWord(symbol.Kind), symbol.Name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(callee),
		})
	})
}

func symbolKindWord(kind semantic.SymbolKind) string {
	switch kind {
	case semantic.SymbolGlobal:
		return "global variable"
	case semantic.SymbolLocal:
		return "local variable"
	case semantic.SymbolParameter:
		return "parameter"
	case semantic.SymbolEnumEntry, semantic.SymbolEnumRoot:
		return "enum constant"
	case semantic.SymbolLabel:
		return "label"
	default:
		return "symbol"
	}
}
