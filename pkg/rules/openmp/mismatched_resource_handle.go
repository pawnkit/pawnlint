package openmp

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MismatchedResourceHandle struct{}

func (MismatchedResourceHandle) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "mismatched-resource-handle",
		Name:            "Mismatched resource handle",
		Summary:         "Reports handles passed to the wrong resource releaser",
		Explanation:     "File, database, and database-result handles have distinct release functions. The rule reports calls only when the argument has one definite incompatible resource tag.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"resource", "handle", "database", "file", "tag"},
	}
}

func (MismatchedResourceHandle) Run(ctx *lint.Context) {
	releasers := resourceReleasers(ctx)
	ctx.Walk.IterKind(parser.KindCallExpression, func(call *parser.Node) {
		_, name, ok := calledNative(ctx, call)
		expected, release := releasers[name]
		if !ok || !release {
			return
		}
		arguments := call.Field("arguments")
		if _, ok := argumentCount(ctx, arguments); !ok || hasNamedArgument(arguments) || len(arguments.Children) == 0 {
			return
		}
		argument := arguments.Children[0]
		actual, ok := resourceTag(ctx, argument)
		if !ok || actual == expected {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("%q releases %s handles, but this argument has tag %s", name, expected, actual),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(argument),
		})
	})
}

func resourceReleasers(ctx *lint.Context) map[string]string {
	natives := ctx.Natives()
	result := make(map[string]string)
	for _, creator := range natives {
		if creator.Release == "" {
			continue
		}
		releaser, ok := natives[creator.Release]
		if ok && len(releaser.Parameters) != 0 && releaser.Parameters[0].Tag != "" {
			result[creator.Release] = releaser.Parameters[0].Tag
		}
	}
	return result
}

func resourceTag(ctx *lint.Context, node *parser.Node) (string, bool) {
	if tags := ctx.Semantic.ExpressionTags(node); len(tags) == 1 {
		return tags[0], true
	}
	node = unwrapParentheses(node)
	if node == nil || node.Kind != parser.KindCallExpression {
		return "", false
	}
	native, _, ok := calledNative(ctx, node)
	if !ok || native.ReturnTag == "" {
		return "", false
	}
	return native.ReturnTag, true
}
