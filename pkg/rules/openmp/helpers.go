package openmp

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

func suggestions(description string) []diagnostic.Suggestion {
	if description == "" {
		return nil
	}
	return []diagnostic.Suggestion{{Description: description}}
}

func calledNative(ctx *lint.Context, call *parser.Node) (api.Native, string, bool) {
	if call == nil || call.HasError || ctx.Walk.Uncertain(call) {
		return api.Native{}, "", false
	}
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return api.Native{}, "", false
	}
	name := ctx.Walk.Text(callee)
	if projectDefinesName(ctx, name) {
		return api.Native{}, "", false
	}
	native, known := ctx.Natives()[name]
	if !known {
		return api.Native{}, "", false
	}
	if symbol := ctx.Semantic.Resolve(callee); symbol != nil && !walk.HasChildToken(symbol.Decl, token.KwNative) {
		return api.Native{}, "", false
	}
	return native, name, true
}

func argumentCount(ctx *lint.Context, arguments *parser.Node) (int, bool) {
	if arguments == nil || arguments.HasError {
		return 0, false
	}
	for _, argument := range arguments.Children {
		if argument.HasError || !argumentExpression(argument.Kind) {
			return 0, false
		}
	}
	if len(arguments.Children) == 0 {
		return 0, true
	}
	count := 1
	for index := 1; index < len(arguments.Children); index++ {
		if tokenBetween(ctx.File.Parsed.Tokens, token.Comma, arguments.Children[index-1].End, arguments.Children[index].Start) {
			count++
		}
	}
	return count, true
}

func tokenBetween(tokens []token.Token, kind token.Kind, start, end int) bool {
	for _, current := range tokens {
		if current.Start.Offset < start {
			continue
		}
		if current.End.Offset > end {
			return false
		}
		if current.Kind == kind {
			return true
		}
	}
	return false
}

func argumentExpression(kind parser.Kind) bool {
	switch kind {
	case parser.KindIdentifier, parser.KindLiteral, parser.KindCallExpression,
		parser.KindSubscriptExpression, parser.KindTernaryExpression,
		parser.KindBinaryExpression, parser.KindUnaryExpression,
		parser.KindUpdateExpression, parser.KindAssignmentExpression,
		parser.KindSizeofExpression, parser.KindTagofExpression,
		parser.KindDefinedExpression, parser.KindTaggedExpression,
		parser.KindParenthesizedExpression, parser.KindArrayLiteral,
		parser.KindExpressionList, parser.KindStringizeExpression,
		parser.KindStringConcat:
		return true
	default:
		return false
	}
}

func hasNamedArgument(arguments *parser.Node) bool {
	for _, argument := range arguments.Children {
		if argument.Kind == parser.KindAssignmentExpression {
			left := argument.Field("left")
			if left != nil && left.Kind == parser.KindArgumentName {
				return true
			}
		}
	}
	return false
}

func unwrapParentheses(node *parser.Node) *parser.Node {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	return node
}

func argumentWord(count int) string {
	if count == 1 {
		return "argument"
	}
	return "arguments"
}

func providedVerb(count int) string {
	if count == 1 {
		return "was"
	}
	return "were"
}

func projectDeclaresName(ctx *lint.Context, name string) bool {
	return ctx.Project != nil && len(ctx.Project.Declarations[name]) != 0
}

func projectDefinesName(ctx *lint.Context, name string) bool {
	if definesName(ctx.Walk, name) {
		return true
	}
	if ctx.Project == nil {
		return false
	}
	for _, file := range ctx.Project.Files {
		if definesName(file.Walk, name) {
			return true
		}
	}
	return false
}

func definesName(tree *walk.Model, name string) bool {
	for _, node := range tree.OfKind(parser.KindDirectiveDefine) {
		if !tree.Inactive(node) && tree.Text(node.Field("name")) == name {
			return true
		}
	}
	return false
}
