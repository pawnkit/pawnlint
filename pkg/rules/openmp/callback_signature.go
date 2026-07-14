package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type CallbackSignature struct{}

func (CallbackSignature) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "callback-signature",
		Name:            "Callback signature",
		Summary:         "Reports public callbacks that do not match the target API",
		Explanation:     "Callback names and parameters are defined by the selected open.mp or SA-MP API. The rule checks public functions against checked-in API metadata.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"callbacks", "openmp", "samp", "api"},
	}
}

func (CallbackSignature) Run(ctx *lint.Context) {
	callbacks := ctx.Callbacks()
	ctx.Walk.IterKind(parser.KindFunctionDefinition, func(node *parser.Node) {
		if !walk.HasChildToken(node, token.KwPublic) || ctx.Walk.Uncertain(node) {
			return
		}
		nameNode := node.Field("name")
		if nameNode == nil {
			return
		}
		name := ctx.Walk.Text(nameNode)
		callback, ok := callbacks[name]
		if !ok {
			return
		}
		reason := callbackDifference(ctx, node, callback)
		if reason == "" {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("callback %q does not match the %s API: %s", name, targetName(ctx.Target), reason),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(nameNode),
		})
	})
}

func callbackDifference(ctx *lint.Context, node *parser.Node, callback api.Callback) string {
	if nodeTag(ctx, node) != callback.ReturnTag {
		return "return tag differs"
	}
	params := functionParameters(node)
	if len(params) != len(callback.Parameters) {
		return fmt.Sprintf("parameter count differs (%d expected, %d found)", len(callback.Parameters), len(params))
	}
	for i, expected := range callback.Parameters {
		actual := params[i]
		position := i + 1
		if ctx.Walk.Text(actual.Field("name")) != expected.Name {
			return fmt.Sprintf("parameter %d name differs", position)
		}
		if nodeTag(ctx, actual) != expected.Tag {
			return fmt.Sprintf("parameter %d tag differs", position)
		}
		if countKind(actual, parser.KindDimension) != expected.ArrayRank {
			return fmt.Sprintf("parameter %d array rank differs", position)
		}
		if walk.HasChildToken(actual, token.KwConst) != expected.Const {
			return fmt.Sprintf("parameter %d const qualifier differs", position)
		}
		if walk.ReferencesByAmpersand(ctx.File.Parsed.Tokens, actual) != expected.Reference {
			return fmt.Sprintf("parameter %d reference form differs", position)
		}
		if (actual.Field("name") == nil && actual.Tok.Kind == token.Ellipsis) != expected.Variadic {
			return fmt.Sprintf("parameter %d variadic form differs", position)
		}
	}
	return ""
}

func functionParameters(node *parser.Node) []*parser.Node {
	list := node.Field("parameters")
	if list == nil {
		return nil
	}
	var result []*parser.Node
	for _, child := range list.Children {
		if child.Kind == parser.KindParameter {
			result = append(result, child)
		}
	}
	return result
}

func nodeTag(ctx *lint.Context, node *parser.Node) string {
	tag := node.Field("tag")
	if tag == nil || len(tag.Children) != 1 {
		return ""
	}
	return ctx.Walk.Text(tag.Children[0])
}

func countKind(node *parser.Node, kind parser.Kind) int {
	count := 0
	for _, child := range node.Children {
		if child.Kind == kind {
			count++
		}
	}
	return count
}

func targetName(target string) string {
	if target == "samp" {
		return "SA-MP"
	}
	return "open.mp"
}
