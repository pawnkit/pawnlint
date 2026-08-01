package suspicious

import (
	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type DuplicateCondition struct{}

func (DuplicateCondition) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "duplicate-condition",
		Name:            "Duplicate condition",
		Summary:         "Reports repeated pure conditions in an if and else-if chain",
		Explanation:     "A repeated pure condition in an else-if chain can never become true after the first copy was false. Calls and other expressions with side effects are skipped.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		Requirements:    lint.NeedSyntax | lint.NeedLocalSymbols | lint.NeedNames | lint.NeedTags,
		Scope:           lint.ScopeFile,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"conditions", "branches", "semantic"},
	}
}

func (DuplicateCondition) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	symbolIDs := make(map[*semantic.Symbol]uint64, len(ctx.Semantic.Symbols))
	for index, symbol := range ctx.Semantic.Symbols {
		symbolIDs[symbol] = uint64(index + 1)
	}
	ctx.Walk.IterKind(parser.KindIfStatement, func(node *parser.Node) {
		parent := ctx.Walk.Parent(node)
		if parent != nil && parent.Kind == parser.KindIfStatement && parent.Field("alternative") == node {
			return
		}
		var chain []*parser.Node
		for current := node; current != nil && current.Kind == parser.KindIfStatement; {
			chain = append(chain, current)
			next := current.Field("alternative")
			if next == nil || next.Kind != parser.KindIfStatement {
				break
			}
			current = next
		}
		if len(chain) < 2 {
			return
		}
		seen := make(map[uint64][]*parser.Node)
		pure := make(map[*parser.Node]bool)
		for _, current := range chain {
			condition := current.Field("condition")
			key, certain := duplicateConditionKey(ctx, condition, symbolIDs)
			if certain {
				for _, first := range seen[key] {
					if !duplicateConditionPure(ctx, pure, first) ||
						!duplicateConditionPure(ctx, pure, condition) {
						continue
					}
					if !ctx.Semantic.Equivalent(first, condition) {
						continue
					}
					ctx.Report(diagnostic.Diagnostic{
						Message:  "condition duplicates an earlier branch",
						Filename: ctx.File.Path,
						Range:    ctx.Walk.Range(condition),
						Notes: []diagnostic.RelatedLocation{{
							Range:   ctx.Walk.Range(first),
							Message: "first condition is here",
						}},
					})
					break
				}
				seen[key] = append(seen[key], condition)
			}
		}
	})
}

func duplicateConditionPure(ctx *lint.Context, cache map[*parser.Node]bool, node *parser.Node) bool {
	if pure, ok := cache[node]; ok {
		return pure
	}
	pure := ctx.Pure(node)
	cache[node] = pure
	return pure
}

func duplicateConditionKey(
	ctx *lint.Context,
	node *parser.Node,
	symbolIDs map[*semantic.Symbol]uint64,
) (uint64, bool) {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	if node == nil || node.HasError || ctx.Walk.Uncertain(node) {
		return 0, false
	}
	hash := uint64(1469598103934665603)
	hash = duplicateConditionMix(hash, uint64(node.Kind))
	hash = duplicateConditionMix(hash, uint64(node.Tok.Kind))
	switch node.Kind {
	case parser.KindIdentifier:
		symbol := ctx.Semantic.Resolve(node)
		id := symbolIDs[symbol]
		if id == 0 {
			return 0, false
		}
		return duplicateConditionMix(hash, id), true
	case parser.KindLiteral:
		for _, value := range ctx.Walk.TokenText(node.Tok) {
			hash = duplicateConditionMix(hash, uint64(value))
		}
		return hash, true
	}
	hash = duplicateConditionMix(hash, uint64(len(node.Children)))
	for _, child := range node.Children {
		childHash, certain := duplicateConditionKey(ctx, child, symbolIDs)
		if !certain {
			return 0, false
		}
		hash = duplicateConditionMix(hash, childHash)
	}
	return hash, true
}

func duplicateConditionMix(hash, value uint64) uint64 {
	return (hash ^ value) * 1099511628211
}
