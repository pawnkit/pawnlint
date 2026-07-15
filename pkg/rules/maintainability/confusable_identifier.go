package maintainability

import (
	"fmt"
	"strings"

	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type ConfusableIdentifier struct{}

func (ConfusableIdentifier) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "confusable-identifier",
		Name:            "Confusable identifier",
		Summary:         "Reports visible declarations with visually confusable names",
		Explanation:     "Pawn identifiers are ASCII. The rule reports declarations whose names differ but become identical after normalizing the visually ambiguous groups 0/O/o and 1/I/l. Only definite declarations visible in the same lexical context are compared.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"naming", "suspicious", "identifiers"},
	}
}

func (ConfusableIdentifier) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	definitions := namingDefinitions(ctx)
	seen := make(map[*semantic.Scope]map[string]*semantic.Symbol)
	for _, symbol := range ctx.Semantic.Symbols {
		if !namingSymbol(ctx, symbol, definitions) {
			continue
		}
		skeleton := confusableSkeleton(symbol.Name)
		if previous := visibleConfusable(seen, symbol.Scope, skeleton, symbol.Name); previous != nil {
			ctx.Report(diagnostic.Diagnostic{
				Message:  fmt.Sprintf("%s name %q is visually confusable with %q", namingKind(symbol.Kind), symbol.Name, previous.Name),
				Filename: ctx.File.Path,
				Range:    ctx.Walk.Range(symbol.NameNode),
				Notes: []diagnostic.RelatedLocation{{
					Range:   ctx.Walk.Range(previous.NameNode),
					Message: "conflicting declaration is here",
				}},
			})
		}
		if seen[symbol.Scope] == nil {
			seen[symbol.Scope] = make(map[string]*semantic.Symbol)
		}
		if seen[symbol.Scope][skeleton] == nil {
			seen[symbol.Scope][skeleton] = symbol
		}
	}
}

func visibleConfusable(seen map[*semantic.Scope]map[string]*semantic.Symbol, scope *semantic.Scope, skeleton, name string) *semantic.Symbol {
	for current := scope; current != nil; current = current.Parent {
		previous := seen[current][skeleton]
		if previous != nil && previous.Name != name {
			return previous
		}
	}
	return nil
}

func confusableSkeleton(name string) string {
	var result strings.Builder
	result.Grow(len(name))
	for _, char := range name {
		switch char {
		case '0', 'O', 'o':
			result.WriteByte('0')
		case '1', 'I', 'l':
			result.WriteByte('1')
		default:
			result.WriteRune(char)
		}
	}
	return result.String()
}
