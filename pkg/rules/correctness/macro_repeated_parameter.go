package correctness

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MacroRepeatedParameter struct{}

func (MacroRepeatedParameter) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "macro-repeated-parameter",
		Name:            "Macro repeated parameter",
		Summary:         "Reports macro parameters evaluated more than once",
		Explanation:     "A function-like macro that evaluates one parameter more than once can repeat calls, assignments, and increments supplied by the caller. The rule checks fully parsed replacement lists and ignores unevaluated sizeof, tagof, and defined operands, opaque bodies, uncertain definitions, and malformed macros.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"macros", "evaluation", "side-effects"},
	}
}

func (MacroRepeatedParameter) Run(ctx *lint.Context) {
	for _, define := range ctx.Walk.OfKind(parser.KindDirectiveDefine) {
		if define.HasError || ctx.Walk.Inactive(define) || ctx.Walk.Uncertain(define) {
			continue
		}
		parameters := define.Field("parameters")
		value := define.Field("value")
		if parameters == nil || value == nil || value.Kind == parser.KindMacroBody || value.Kind == parser.KindRaw || value.HasError {
			continue
		}
		macroName := ctx.Walk.Text(define.Field("name"))
		for _, parameter := range parameters.Children {
			name := ctx.Walk.Text(parameter)
			if name == "" {
				continue
			}
			count := macroParameterEvaluations(ctx, value, name, false)
			if count < 2 {
				continue
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("parameter %q is evaluated %d times by macro %q", name, count, macroName),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(parameter),
			})
		}
	}
}

func macroParameterEvaluations(ctx *lint.Context, node *parser.Node, name string, unevaluated bool) int {
	if node == nil {
		return 0
	}
	switch node.Kind {
	case parser.KindSizeofExpression, parser.KindTagofExpression, parser.KindDefinedExpression:
		unevaluated = true
	}
	count := 0
	if !unevaluated && node.Kind == parser.KindIdentifier && ctx.Walk.Text(node) == name && !macroParameterKeywordOperand(ctx, node) {
		count++
	}
	for _, child := range node.Children {
		count += macroParameterEvaluations(ctx, child, name, unevaluated)
	}
	return count
}

func macroParameterKeywordOperand(ctx *lint.Context, node *parser.Node) bool {
	if ctx.File == nil || ctx.File.Parsed == nil {
		return false
	}
	for index := len(ctx.File.Parsed.Tokens) - 1; index >= 0; index-- {
		tok := ctx.File.Parsed.Tokens[index]
		if tok.End.Offset > node.Start {
			continue
		}
		switch tok.Kind {
		case token.KwSizeof, token.KwTagof, token.KwDefined:
			return true
		default:
			return false
		}
	}
	return false
}
