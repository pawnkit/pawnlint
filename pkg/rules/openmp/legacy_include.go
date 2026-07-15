package openmp

import (
	"fmt"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type LegacyInclude struct{}

func (LegacyInclude) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "legacy-include",
		Name:            "Legacy include",
		Summary:         "Reports official SA-MP wrapper includes when targeting open.mp",
		Explanation:     "The official open.mp compatibility wrappers emit compiler warnings and direct users to include open.mp instead. The rule reports only the exact wrapper names shipped by the pinned API revision.",
		Category:        diagnostic.CategoryOpenMP,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"include", "migration", "compatibility", "api"},
	}
}

func (LegacyInclude) Run(ctx *lint.Context) {
	if ctx.Target == "samp" {
		return
	}
	legacy := api.LegacyIncludes()
	for _, kind := range []parser.Kind{parser.KindDirectiveInclude, parser.KindDirectiveTryInclude} {
		ctx.Walk.IterKind(kind, func(node *parser.Node) {
			if ctx.Walk.Uncertain(node) || ctx.Walk.Inactive(node) {
				return
			}
			path := normalizedInclude(ctx.Walk.Text(node.Field("path")))
			replacement, ok := legacy[path]
			if !ok {
				return
			}
			ctx.Report(diagnostic.Diagnostic{
				Message:     fmt.Sprintf("legacy include %q is a compatibility wrapper", path),
				Filename:    ctx.File.Path,
				Range:       ctx.Walk.Range(node.Field("path")),
				Suggestions: []diagnostic.Suggestion{{Description: "include <" + replacement + "> directly"}},
			})
		})
	}
}

func normalizedInclude(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && (raw[0] == '<' && raw[len(raw)-1] == '>' || raw[0] == '"' && raw[len(raw)-1] == '"') {
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
	}
	return strings.TrimSuffix(raw, ".inc")
}
