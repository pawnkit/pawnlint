package maintainability

import (
	"fmt"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RedundantTag struct{}

func (RedundantTag) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "redundant-tag",
		Name:            "Redundant tag",
		Summary:         "Reports tag overrides that repeat an expression's known tag",
		Explanation:     "A tag override is redundant when its operand already has exactly the same tag. Unknown, union, ambiguous, macro-dependent, and malformed expressions are ignored.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         true,
		Tags:            []string{"tags", "expressions", "semantic"},
	}
}

func (RedundantTag) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	for _, node := range ctx.Walk.OfKind(parser.KindTaggedExpression) {
		if node.HasError || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) || redundantTagNested(ctx, node) {
			continue
		}
		tag := node.Field("tag")
		expression := node.Field("expression")
		if tag == nil || expression == nil || expression.HasError {
			continue
		}
		name := ctx.Walk.Text(tag)
		expressionTag, known := ctx.ExpressionTag(expression)
		if name == "" || !known || expressionTag != name {
			continue
		}
		diagnosticItem := diagnostic.Diagnostic{
			Message:  fmt.Sprintf("tag override %q is redundant", name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(tag),
		}
		if fix := redundantTagFix(ctx, tag, expression, name); fix != nil {
			diagnosticItem.Fix = fix
		}
		ctx.Report(diagnosticItem)
	}
}

func redundantTagNested(ctx *lint.Context, node *parser.Node) bool {
	parent := ctx.Walk.Parent(node)
	if parent == nil || parent.Kind != parser.KindTaggedExpression || parent.Field("expression") != node {
		return false
	}
	parentTag := parent.Field("tag")
	tag := node.Field("tag")
	return parentTag != nil && tag != nil && ctx.Walk.Text(parentTag) == ctx.Walk.Text(tag)
}

func redundantTagFix(ctx *lint.Context, tag, expression *parser.Node, name string) *diagnostic.Fix {
	if tag.Start < 0 || expression.Start < tag.End || expression.Start > len(ctx.File.Source) {
		return nil
	}
	prefix := string(ctx.File.Source[tag.Start:expression.Start])
	if !strings.HasPrefix(prefix, name) || strings.TrimSpace(strings.TrimPrefix(prefix, name)) != ":" {
		return nil
	}
	return &diagnostic.Fix{
		Description: "remove the redundant tag override",
		Edits: []diagnostic.Edit{{
			Range:   ctx.File.LineTable.Range(tag.Start, expression.Start),
			NewText: "",
		}},
	}
}
