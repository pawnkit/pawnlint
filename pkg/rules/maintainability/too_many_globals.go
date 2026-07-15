package maintainability

import (
	"fmt"
	"sort"

	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type TooManyGlobals struct{}

func (TooManyGlobals) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "too-many-globals",
		Name:            "Too many globals",
		Summary:         "Reports files with too many global variables",
		Explanation:     "Each global variable declarator counts separately. Constants, enum entries, locals, parameters, inactive declarations, and uncertain declarations are excluded by default. Constants can be included through configuration.",
		Category:        diagnostic.CategoryMaintainability,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"size", "globals", "state", "maintainability"},
		Options: []lint.Option{
			{Name: "maximum", Summary: "Maximum global variables per file", Type: lint.OptionInteger, Default: int64(50), Minimum: 1, Maximum: 1_000_000, HasMinimum: true, HasMaximum: true},
			{Name: "include-constants", Summary: "Include constant globals", Type: lint.OptionBoolean, Default: false},
		},
	}
}

func (TooManyGlobals) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	maximum, includeConstants := configuredTooManyGlobals(ctx)
	var globals []*semantic.Symbol
	for _, symbol := range ctx.Semantic.Symbols {
		if symbol.Kind != semantic.SymbolGlobal || symbol.Decl == nil || symbol.NameNode == nil || !includeConstants && symbol.Constant {
			continue
		}
		globals = append(globals, symbol)
	}
	if len(globals) <= maximum {
		return
	}
	sort.SliceStable(globals, func(left, right int) bool {
		return globals[left].NameNode.Start < globals[right].NameNode.Start
	})
	subject := "mutable global variables"
	if includeConstants {
		subject = "global variables"
	}
	ctx.Report(diagnostic.Diagnostic{
		Message:  fmt.Sprintf("file declares %d %s, exceeding the maximum of %d", len(globals), subject, maximum),
		Filename: ctx.File.Path,
		Range:    ctx.Walk.Range(globals[maximum].NameNode),
	})
}

func configuredTooManyGlobals(ctx *lint.Context) (int, bool) {
	maximum := 50
	includeConstants := false
	if ctx.PerRule == nil || ctx.PerRule["too-many-globals"] == nil {
		return maximum, includeConstants
	}
	values := ctx.PerRule["too-many-globals"]
	if value, ok := values["maximum"].(int64); ok && value > 0 {
		maximum = int(value)
	}
	includeConstants, _ = values["include-constants"].(bool)
	return maximum, includeConstants
}
