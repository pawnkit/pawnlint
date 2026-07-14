package openmp

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type InvalidSentinelComparison struct{}

func (InvalidSentinelComparison) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "invalid-sentinel-comparison",
		Name:            "Invalid sentinel comparison",
		Summary:         "Reports a native's result compared against the wrong INVALID_* constant",
		Explanation:     explanationInvalidSentinelComparison,
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"native", "constant", "semantic", "api"},
	}
}

const explanationInvalidSentinelComparison = `open.mp and SA-MP use different sentinel values for different ID types
(` + "`INVALID_PLAYER_ID`" + `, ` + "`INVALID_VEHICLE_ID`" + `, ` + "`INVALID_ACTOR_ID`" + `,
` + "`INVALID_OBJECT_ID`" + `), and mixing them up is a common copy-paste mistake:

` + "```pawn" + `
new vehicleid = GetPlayerVehicleID(playerid);
if (vehicleid == INVALID_PLAYER_ID)
` + "```" + `

This always evaluates false for a vehicle ID, since a vehicle's invalid
sentinel is ` + "`INVALID_VEHICLE_ID`" + `, not ` + "`INVALID_PLAYER_ID`" + `. The rule checks a
curated set of well-known ID-returning natives against the sentinel constant
they should be compared with, and only reports when the compared name is
another known sentinel, not an unresolved or project-defined identifier.`

var invalidSentinelForNative = map[string]string{
	"GetPlayerVehicleID":           "INVALID_VEHICLE_ID",
	"CreateVehicle":                "INVALID_VEHICLE_ID",
	"GetVehicleTrailer":            "INVALID_VEHICLE_ID",
	"GetPlayerCameraTargetVehicle": "INVALID_VEHICLE_ID",
	"CreateActor":                  "INVALID_ACTOR_ID",
	"GetPlayerCameraTargetActor":   "INVALID_ACTOR_ID",
	"GetPlayerTargetActor":         "INVALID_ACTOR_ID",
	"CreateObject":                 "INVALID_OBJECT_ID",
	"CreatePlayerObject":           "INVALID_OBJECT_ID",
	"GetPlayerCameraTargetObject":  "INVALID_OBJECT_ID",
	"GetPlayerCameraTargetPlayer":  "INVALID_PLAYER_ID",
	"GetPlayerTargetPlayer":        "INVALID_PLAYER_ID",
}

var knownInvalidSentinels = map[string]bool{
	"INVALID_VEHICLE_ID": true,
	"INVALID_PLAYER_ID":  true,
	"INVALID_ACTOR_ID":   true,
	"INVALID_OBJECT_ID":  true,
}

func (InvalidSentinelComparison) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	ctx.Walk.IterKind(parser.KindBinaryExpression, func(node *parser.Node) {
		if node.Tok.Kind != token.Eq && node.Tok.Kind != token.NotEq || ctx.Walk.Uncertain(node) {
			return
		}
		left := unwrapParentheses(node.Field("left"))
		right := unwrapParentheses(node.Field("right"))
		name, expected, sentinel, ok := sentinelComparison(ctx, left, right)
		if !ok {
			name, expected, sentinel, ok = sentinelComparison(ctx, right, left)
		}
		if !ok {
			return
		}
		actual := ctx.Walk.Text(sentinel)
		if actual == expected || !knownInvalidSentinels[actual] || ctx.Semantic.Resolve(sentinel) != nil {
			return
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("%q returns a value invalidated by %q, not %q", name, expected, actual),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(node),
		})
	})
}

func sentinelComparison(ctx *lint.Context, operand, other *parser.Node) (name, expected string, sentinel *parser.Node, ok bool) {
	if other == nil || other.Kind != parser.KindIdentifier || other.HasError {
		return "", "", nil, false
	}
	name, expected, ok = invalidSentinelCall(ctx, callBehindOperand(ctx, operand))
	if !ok {
		return "", "", nil, false
	}
	return name, expected, other, true
}

func callBehindOperand(ctx *lint.Context, node *parser.Node) *parser.Node {
	if node == nil {
		return nil
	}
	if node.Kind == parser.KindCallExpression {
		return node
	}
	if node.Kind != parser.KindIdentifier {
		return nil
	}
	symbol := ctx.Semantic.Resolve(node)
	if symbol == nil || symbol.Ambiguous || symbol.Decl == nil {
		return nil
	}
	if symbol.Kind != semantic.SymbolLocal && symbol.Kind != semantic.SymbolGlobal {
		return nil
	}
	return unwrapParentheses(symbol.Decl.Field("initializer"))
}

func invalidSentinelCall(ctx *lint.Context, call *parser.Node) (name, expected string, ok bool) {
	if call == nil || call.HasError || call.Kind != parser.KindCallExpression {
		return "", "", false
	}
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || callee.HasError {
		return "", "", false
	}
	name = ctx.Walk.Text(callee)
	expected, known := invalidSentinelForNative[name]
	if !known || ctx.Semantic.Resolve(callee) != nil {
		return "", "", false
	}
	return name, expected, true
}
