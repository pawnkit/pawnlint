package correctness

import (
	"fmt"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type UnsafeStringTermination struct{}

func (UnsafeStringTermination) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unsafe-string-termination",
		Name:            "Unsafe string termination",
		Summary:         "Reports raw copies used as strings without EOS termination",
		Explanation:     "A raw memcpy does not append EOS. The rule reports array destinations later passed to a known string parameter in the same function without an intervening EOS assignment or terminating string write. Unresolved, macro-derived, uncertain, malformed, and non-string uses are ignored.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.SemanticAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"strings", "termination", "buffers", "memcpy"},
	}
}

func (UnsafeStringTermination) Run(ctx *lint.Context) {
	if ctx.Semantic == nil {
		return
	}
	calls := ctx.Walk.OfKind(parser.KindCallExpression)
	assignments := ctx.Walk.OfKind(parser.KindAssignmentExpression)
	for _, copyCall := range calls {
		if !unsafeStringCallNamed(ctx, copyCall, "memcpy") || unsafeStringSkipped(ctx, copyCall) {
			continue
		}
		arguments := copyCall.Field("arguments")
		if arguments == nil || len(arguments.Children) < 4 || unitNodeHasError(arguments) {
			continue
		}
		destinationNode := arguments.Children[0]
		destination := unsafeStringRootSymbol(ctx, destinationNode)
		if destination == nil || dimensionCount(destination.Decl) == 0 {
			continue
		}
		function := ctx.Walk.EnclosingFunction(copyCall)
		block := unsafeStringBlock(ctx, copyCall)
		use := unsafeStringNextUse(ctx, calls, destination, function, block, copyCall.End)
		if use == nil || unsafeStringTerminated(ctx, calls, assignments, destination, function, block, copyCall.End, use.Start) {
			continue
		}
		ctx.Report(diagnostic.Diagnostic{
			Message:  fmt.Sprintf("raw copy into %q may be used as a string without EOS termination", destination.Name),
			Filename: ctx.File.Path,
			Range:    ctx.Walk.Range(copyCall),
		})
	}
}

func unsafeStringNextUse(ctx *lint.Context, calls []*parser.Node, destination *semantic.Symbol, function, block *parser.Node, after int) *parser.Node {
	for _, call := range calls {
		if call.Start <= after || ctx.Walk.EnclosingFunction(call) != function || unsafeStringBlock(ctx, call) != block || unsafeStringSkipped(ctx, call) {
			continue
		}
		arguments := call.Field("arguments")
		callee := call.Field("function")
		if arguments == nil || callee == nil || callee.Kind != parser.KindIdentifier {
			continue
		}
		native, known := ctx.Natives()[ctx.Walk.Text(callee)]
		if !known || !unsafeStringNativeCall(ctx, callee) {
			continue
		}
		if unsafeStringRequiresTerminatedDestination(ctx.Walk.Text(callee)) && len(arguments.Children) > 0 && unsafeStringRootSymbol(ctx, arguments.Children[0]) == destination {
			return call
		}
		for index, argument := range arguments.Children {
			if index >= len(native.Parameters) {
				break
			}
			parameter := native.Parameters[index]
			if parameter.Const && parameter.ArrayRank > 0 && unsafeStringRootSymbol(ctx, argument) == destination {
				return call
			}
		}
	}
	return nil
}

func unsafeStringTerminated(ctx *lint.Context, calls, assignments []*parser.Node, destination *semantic.Symbol, function, block *parser.Node, after, before int) bool {
	for _, assignment := range assignments {
		if assignment.Start <= after || assignment.Start >= before || assignment.Tok.Kind != token.Assign || ctx.Walk.EnclosingFunction(assignment) != function || unsafeStringBlock(ctx, assignment) != block || unsafeStringSkipped(ctx, assignment) {
			continue
		}
		left := assignment.Field("left")
		value := assignment.Field("right")
		if left == nil || left.Kind != parser.KindSubscriptExpression || unsafeStringRootSymbol(ctx, left) != destination {
			continue
		}
		if constant, known := ctx.Constant(value); known && constant == 0 {
			return true
		}
	}
	for _, call := range calls {
		if call.Start <= after || call.Start >= before || ctx.Walk.EnclosingFunction(call) != function || unsafeStringBlock(ctx, call) != block || unsafeStringSkipped(ctx, call) {
			continue
		}
		index, terminating := unsafeStringWriter(ctx, call)
		arguments := call.Field("arguments")
		if terminating && arguments != nil && index < len(arguments.Children) && unsafeStringRootSymbol(ctx, arguments.Children[index]) == destination {
			return true
		}
	}
	return false
}

func unsafeStringRequiresTerminatedDestination(name string) bool {
	switch name {
	case "strcat", "strdel", "strins":
		return true
	default:
		return false
	}
}

func unsafeStringWriter(ctx *lint.Context, call *parser.Node) (int, bool) {
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier || !unsafeStringNativeCall(ctx, callee) {
		return 0, false
	}
	switch ctx.Walk.Text(callee) {
	case "format", "Format", "strformat", "strmid", "strpack", "strunpack", "uudecode", "uuencode":
		return 0, true
	case "fread":
		return 1, true
	default:
		return 0, false
	}
}

func unsafeStringRootSymbol(ctx *lint.Context, node *parser.Node) *semantic.Symbol {
	for node != nil && (node.Kind == parser.KindParenthesizedExpression || node.Kind == parser.KindSubscriptExpression) {
		if node.Kind == parser.KindParenthesizedExpression {
			node = node.Field("expression")
		} else {
			node = node.Field("array")
		}
	}
	if node == nil || node.Kind != parser.KindIdentifier {
		return nil
	}
	symbol := ctx.Semantic.Resolve(node)
	if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal && symbol.Kind != semantic.SymbolGlobal {
		return nil
	}
	return symbol
}

func unsafeStringCallNamed(ctx *lint.Context, call *parser.Node, name string) bool {
	callee := call.Field("function")
	return callee != nil && callee.Kind == parser.KindIdentifier && ctx.Walk.Text(callee) == name && unsafeStringNativeCall(ctx, callee)
}

func unsafeStringNativeCall(ctx *lint.Context, callee *parser.Node) bool {
	name := ctx.Walk.Text(callee)
	if _, known := ctx.Natives()[name]; !known {
		return false
	}
	if ctx.Project != nil && ctx.ProjectFile != nil {
		if declaration, ok := ctx.Project.Resolve(ctx.ProjectFile, callee); ok {
			return declaration.Kind == semantic.SymbolFunction && declaration.Valid() && declaration.HasToken(token.KwNative)
		}
	}
	if symbol := ctx.Semantic.ResolveAsCallTarget(callee); symbol != nil {
		return symbol.Kind == semantic.SymbolFunction && symbol.Decl != nil && !symbol.Ambiguous && walk.HasChildToken(symbol.Decl, token.KwNative)
	}
	return true
}

func unsafeStringSkipped(ctx *lint.Context, node *parser.Node) bool {
	if node == nil || unitNodeHasError(node) || node.Tok.Origin != nil || ctx.Walk.Inactive(node) || ctx.Walk.Uncertain(node) {
		return true
	}
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindDirectiveDefine, parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice:
			return true
		}
	}
	return false
}

func unsafeStringBlock(ctx *lint.Context, node *parser.Node) *parser.Node {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindBlock {
			return current
		}
	}
	return nil
}
