package performance

import (
	"fmt"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type LargeLocalArray struct{}

func (LargeLocalArray) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "large-local-array",
		Name:            "Large local array",
		Summary:         "Reports large automatic arrays allocated on the Pawn stack",
		Explanation:     "Automatic local arrays consume stack cells on every active call. The rule reports arrays whose constant total capacity reaches the configured threshold and skips static or unresolved dimensions.",
		Category:        diagnostic.CategoryPerformance,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"arrays", "stack", "memory", "performance"},
		Options: []lint.Option{{
			Name: "threshold", Summary: "Minimum local array capacity to report",
			Type: lint.OptionInteger, Default: int64(1024), Minimum: 1, HasMinimum: true,
		}},
	}
}

func (LargeLocalArray) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	threshold := configuredThreshold(ctx, "large-local-array", 1024)
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolLocal || symbol.Ambiguous || symbol.Decl.Field("array") == nil {
			continue
		}
		declaration := ctx.Walk.Parent(symbol.Decl)
		if walk.HasChildToken(declaration, token.KwStatic) {
			continue
		}
		capacity, ok := arrayCells(ctx, symbol.Decl)
		if !ok || capacity < threshold {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:     fmt.Sprintf("local array %q allocates %d stack cells", symbol.Name, capacity),
			Filename:    ctx.File.Path,
			Range:       ctx.Walk.Range(symbol.NameNode),
			Suggestions: []diagnostic.Suggestion{{Description: fmt.Sprintf("reduce the array below %d cells or move storage out of the automatic stack", threshold)}},
		})
	}
}

func arrayCells(ctx *lint.Context, declaration *parser.Node) (int64, bool) {
	capacity := int64(1)
	found := false
	for _, child := range declaration.Children {
		if child.Kind != parser.KindDimension {
			continue
		}
		sizeNode := child.Field("size")
		packed := child.Field("packed") != nil
		if sizeNode != nil && sizeNode.Kind == parser.KindUnaryExpression && sizeNode.Tok.Kind == token.Identifier && sizeNode.Tok.Text(ctx.File.Source) == "char" {
			sizeNode = sizeNode.Field("expression")
			packed = true
		}
		size, ok := ctx.Semantic.Eval(sizeNode)
		if !ok || size <= 0 {
			return 0, false
		}
		if packed {
			size = (size + 3) / 4
		}
		if capacity > 1<<31/size {
			return 0, false
		}
		capacity *= size
		found = true
	}
	return capacity, found
}

func configuredThreshold(ctx *lint.Context, rule string, fallback int64) int64 {
	config := ctx.PerRule[rule]
	if config == nil {
		return fallback
	}
	switch value := config["threshold"].(type) {
	case int:
		if value > 0 {
			return int64(value)
		}
	case int64:
		if value > 0 {
			return value
		}
	case float64:
		if value > 0 && value == float64(int64(value)) {
			return int64(value)
		}
	}
	return fallback
}
