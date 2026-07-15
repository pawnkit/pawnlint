package maintainability

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DeclarationOrder struct{}

var defaultDeclarationOrder = []string{"include", "define", "enum", "constant", "variable", "native", "forward", "function"}

func (DeclarationOrder) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "declaration-order",
		Name:            "Declaration order",
		Summary:         "Reports declarations outside the configured source order",
		Explanation:     "Top-level declaration groups follow the configured order. Omitted groups are ignored. Local variables can optionally be required before executable statements in each block. Inactive, uncertain, and malformed syntax is ignored.",
		Category:        diagnostic.CategoryStyle,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"declarations", "order", "style"},
		Options: []lint.Option{
			{Name: "order", Summary: "Top-level declaration group order", Type: lint.OptionStringList, Default: defaultDeclarationOrder, Choices: defaultDeclarationOrder, Validate: validateDeclarationOrder},
			{Name: "locals-before-statements", Summary: "Require local declarations before executable statements", Type: lint.OptionBoolean, Default: false},
		},
	}
}

func validateDeclarationOrder(value any) error {
	values, _ := value.([]string)
	if len(values) == 0 {
		return fmt.Errorf("must contain at least one entry")
	}
	seen := make(map[string]bool, len(values))
	for _, item := range values {
		if seen[item] {
			return fmt.Errorf("must not contain duplicate entries")
		}
		seen[item] = true
	}
	return nil
}

type declarationOrderOptions struct {
	ranks                  map[string]int
	localsBeforeStatements bool
}

func (DeclarationOrder) Run(ctx *lint.Context) {
	options := configuredDeclarationOrder(ctx)
	reportTopLevelDeclarationOrder(ctx, options.ranks)
	if options.localsBeforeStatements {
		reportLocalDeclarationOrder(ctx)
	}
}

func configuredDeclarationOrder(ctx *lint.Context) declarationOrderOptions {
	order := defaultDeclarationOrder
	locals := false
	if ctx.PerRule != nil && ctx.PerRule["declaration-order"] != nil {
		values := ctx.PerRule["declaration-order"]
		if configured, ok := values["order"].([]string); ok {
			order = configured
		}
		locals, _ = values["locals-before-statements"].(bool)
	}
	ranks := make(map[string]int, len(order))
	for index, category := range order {
		ranks[category] = index
	}
	return declarationOrderOptions{ranks: ranks, localsBeforeStatements: locals}
}

func reportTopLevelDeclarationOrder(ctx *lint.Context, ranks map[string]int) {
	maximumRank := -1
	maximumCategory := ""
	for _, node := range definiteOrderedChildren(ctx, ctx.Walk.Root()) {
		category := topLevelDeclarationCategory(node)
		rank, configured := ranks[category]
		if !configured {
			continue
		}
		if rank < maximumRank {
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("%s declarations must appear before %s declarations", declarationOrderLabel(category), declarationOrderLabel(maximumCategory)),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(declarationOrderNode(node)),
			})
			continue
		}
		if rank > maximumRank {
			maximumRank = rank
			maximumCategory = category
		}
	}
}

func definiteOrderedChildren(ctx *lint.Context, root *parser.Node) []*parser.Node {
	var result []*parser.Node
	var appendChildren func(*parser.Node)
	appendChildren = func(node *parser.Node) {
		if node == nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) || node.HasError {
			return
		}
		switch node.Kind {
		case parser.KindSourceFile, parser.KindBlock, parser.KindConditionalRegion, parser.KindConditionalBranch:
			for _, child := range node.Children {
				appendChildren(child)
			}
		default:
			result = append(result, node)
		}
	}
	appendChildren(root)
	return result
}

func topLevelDeclarationCategory(node *parser.Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind {
	case parser.KindDirectiveInclude, parser.KindDirectiveTryInclude:
		return "include"
	case parser.KindDirectiveDefine:
		return "define"
	case parser.KindEnumDeclaration:
		return "enum"
	case parser.KindVariableDeclaration:
		if walk.HasChildToken(node, token.KwConst) {
			return "constant"
		}
		return "variable"
	case parser.KindFunctionDefinition:
		return "function"
	case parser.KindFunctionDeclaration:
		switch {
		case walk.HasChildToken(node, token.KwNative):
			return "native"
		case walk.HasChildToken(node, token.KwForward):
			return "forward"
		default:
			return "function"
		}
	default:
		return ""
	}
}

func declarationOrderNode(node *parser.Node) *parser.Node {
	if node == nil {
		return nil
	}
	if name := node.Field("name"); name != nil {
		return name
	}
	if node.Kind == parser.KindVariableDeclaration {
		for _, child := range node.Children {
			if child.Kind == parser.KindVariableDeclarator {
				if name := child.Field("name"); name != nil {
					return name
				}
			}
		}
	}
	if path := node.Field("path"); path != nil {
		return path
	}
	return node
}

func declarationOrderLabel(category string) string {
	switch category {
	case "include":
		return "include"
	case "define":
		return "define"
	case "enum":
		return "enum"
	case "constant":
		return "constant"
	case "variable":
		return "global variable"
	case "native":
		return "native"
	case "forward":
		return "forward"
	default:
		return "function"
	}
}

func reportLocalDeclarationOrder(ctx *lint.Context) {
	for _, block := range ctx.Walk.OfKind(parser.KindBlock) {
		if ctx.Walk.Inactive(block) || ctx.Walk.Uncertain(block) || block.HasError {
			continue
		}
		seenStatement := false
		for _, node := range definiteOrderedChildren(ctx, block) {
			if node.Kind == parser.KindVariableDeclaration {
				if seenStatement {
					ctx.Report(diagnostic.Diagnostic{
						Message:  "local declarations must appear before executable statements in their block",
						Filename: ctx.File.Path,
						Range:    ctx.Walk.Range(declarationOrderNode(node)),
					})
				}
				continue
			}
			if walk.IsStatement(node) {
				seenStatement = true
			}
		}
	}
}
