package openmp

import (
	"fmt"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type MisspelledCallback struct{}

func (MisspelledCallback) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "misspelled-callback",
		Name:            "Misspelled callback",
		Summary:         "Reports public functions one edit away from a target callback",
		Explanation:     "A one-character callback typo creates an ordinary public function that the server never calls. Only unique one-edit matches are reported.",
		Category:        diagnostic.CategorySuspicious,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SyntaxAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"callbacks", "openmp", "samp", "api"},
	}
}

func (MisspelledCallback) Run(ctx *lint.Context) {
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
		if !strings.HasPrefix(name, "On") {
			return
		}
		if _, known := callbacks[name]; known {
			return
		}
		match := ""
		for candidate := range callbacks {
			if editDistanceOne(name, candidate) {
				if match != "" {
					return
				}
				match = candidate
			}
		}
		if match == "" {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("public function %q looks like a misspelled callback; did you mean %q?", name, match),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(nameNode),
		})
	})
}

func editDistanceOne(left, right string) bool {
	if left == right || len(left)-len(right) > 1 || len(right)-len(left) > 1 {
		return false
	}
	if len(left) == len(right) {
		differences := 0
		for i := range len(left) {
			if left[i] != right[i] {
				differences++
			}
		}
		return differences == 1
	}
	if len(left) > len(right) {
		left, right = right, left
	}
	for i, j, skipped := 0, 0, false; i < len(left) && j < len(right); {
		if left[i] == right[j] {
			i++
			j++
			continue
		}
		if skipped {
			return false
		}
		skipped = true
		j++
	}
	return true
}
