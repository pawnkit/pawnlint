package correctness

import (
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DiscardedExpression struct{}

func (DiscardedExpression) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "discarded-expression",
		Name:            "Discarded expression",
		Summary:         "A standalone expression with no side effects does nothing",
		Explanation:     explanationDiscardedExpression,
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"expression", "dead-code"},
	}
}

const explanationDiscardedExpression = `A side-effect-free expression used as a statement does nothing:

` + "```pawn" + `
playerid + 1;
` + "```" + `

Calls, assignments, and updates are not reported. This rule has no fix because
the intended action is unknown.`

func (DiscardedExpression) Run(ctx *lint.Context) {
	m := ctx.Walk
	m.IterKind(parser.KindExpressionStatement, func(n *parser.Node) {
		expr := n.Field("expression")
		if expr == nil {
			return
		}
		if m.Uncertain(n) {
			return
		}
		if hasSideEffect(expr) {
			return
		}
		if isLikelyMacroOrLabel(ctx, expr) {
			return
		}
		if hasMacroPrefix(ctx, expr) {
			return
		}
		d := diagnostic.Diagnostic{
			RuleID:   "discarded-expression",
			Message:  "expression has no effect; its result is discarded",
			Filename: ctx.File.Path,
			Range:    m.Range(expr),
			Notes: []diagnostic.RelatedLocation{{
				Range:   m.Range(n),
				Message: "if this was meant to do work, assign it or call a function",
			}},
		}
		ctx.Report(d)
	})
}

func hasMacroPrefix(ctx *lint.Context, expr *parser.Node) bool {
	if ctx == nil || ctx.File == nil || ctx.File.Parsed == nil || expr == nil {
		return false
	}
	tokens := ctx.File.Parsed.Tokens
	index := sort.Search(len(tokens), func(i int) bool {
		return tokens[i].Start.Offset >= expr.Start
	})
	if index <= 0 || index >= len(tokens) {
		return false
	}
	current := &tokens[index]
	for previousIndex := index - 1; previousIndex >= 0; previousIndex-- {
		previous := &tokens[previousIndex]
		if previous.End.Line != current.Start.Line {
			break
		}
		if previous.Kind == token.ColonColon || previous.Kind == token.KwEnum {
			return true
		}
		if previous.Kind == token.Identifier && previous.Text(ctx.File.Source) == "stop" {
			return true
		}
		if previous.Kind == token.Semicolon || previous.Kind == token.LBrace || previous.Kind == token.RBrace {
			break
		}
	}
	return false
}

func hasSideEffect(n *parser.Node) bool {
	if n == nil {
		return false
	}
	switch n.Kind {
	case parser.KindCallExpression:
		return true
	case parser.KindAssignmentExpression:
		return true
	case parser.KindUpdateExpression:
		return true
	case parser.KindMacroInvocation, parser.KindMacroInvocationBlock:
		return true
	}
	if n.Tok.Kind == token.PlusPlus || n.Tok.Kind == token.MinusMinus {
		return true
	}
	for _, c := range n.Children {
		if hasSideEffect(c) {
			return true
		}
	}
	return false
}

func isLikelyMacroOrLabel(ctx *lint.Context, expr *parser.Node) bool {
	if expr == nil {
		return false
	}
	switch expr.Kind {
	case parser.KindIdentifier, parser.KindLiteral,
		parser.KindStringConcat, parser.KindStringizeExpression:
		return true
	}
	return false
}
