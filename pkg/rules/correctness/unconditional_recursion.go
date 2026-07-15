package correctness

import (
	"fmt"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type UnconditionalRecursion struct{}

func (UnconditionalRecursion) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "unconditional-recursion",
		Name:            "Unconditional recursion",
		Summary:         "Reports recursive cycles with no terminating path",
		Explanation:     "A recursive component cannot terminate when every reachable path in every member must call the component again. Base cases, conditional evaluation, non-recursive control-flow cycles, macros, unresolved calls, and uncertain functions suppress the diagnostic.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		DefaultEnabled:  true,
		Fixable:         false,
		Tags:            []string{"recursion", "calls", "control-flow", "project"},
	}
}

func (UnconditionalRecursion) Run(ctx *lint.Context) {
	if ctx.Project == nil || ctx.Project.CallGraph == nil {
		return
	}
	file := ctx.Project.File(ctx.File.Path)
	if !ctx.Project.InProgram(file) {
		return
	}
	flows := make(map[*project.File]*controlflow.Model)
	if file != nil {
		flows[file] = ctx.Flow
	}
	for _, component := range ctx.Project.CallGraph.RecursiveComponents() {
		members := make(map[*parser.Node]bool, len(component))
		names := make([]string, 0, len(component))
		for _, function := range component {
			members[function.Node] = true
			names = append(names, function.Name)
		}
		if !unconditionalComponent(ctx, component, members, flows) {
			continue
		}
		cycle := strings.Join(names, " -> ")
		for _, function := range component {
			if function.File != file || function.Symbol == nil {
				continue
			}
			message := fmt.Sprintf("function %q cannot return without calling recursive cycle %s", function.Name, cycle)
			if len(component) == 1 {
				message = fmt.Sprintf("function %q unconditionally calls itself", function.Name)
			}
			diagnosticValue := diagnostic.Diagnostic{
				Message:  message,
				Filename: ctx.File.Path,
				Range:    file.Walk.Range(function.Symbol.NameNode),
			}
			for _, call := range ctx.Project.CallGraph.Outgoing(function) {
				if !members[call.Callee.Node] || !unconditionalCall(call.File, call.Node) {
					continue
				}
				diagnosticValue.Notes = append(diagnosticValue.Notes, diagnostic.RelatedLocation{
					Range:   call.File.Walk.Range(call.Node.Field("function")),
					Message: fmt.Sprintf("recursive call to %q is here", call.Callee.Name),
				})
				break
			}
			ctx.Report(diagnosticValue)
		}
	}
}

func unconditionalComponent(ctx *lint.Context, component []project.Declaration, members map[*parser.Node]bool, flows map[*project.File]*controlflow.Model) bool {
	for _, declaration := range component {
		flow := flows[declaration.File]
		if flow == nil {
			flow = controlflow.Build(declaration.File.Walk, declaration.File.Semantic)
			flows[declaration.File] = flow
		}
		function := flow.Function(declaration.Node)
		if function == nil || function.Uncertain {
			return false
		}
		targets := make(map[*controlflow.Block]bool)
		for _, call := range ctx.Project.CallGraph.Outgoing(declaration) {
			if !members[call.Callee.Node] || !unconditionalCall(call.File, call.Node) {
				continue
			}
			block := function.Block(call.Node)
			if function.ReachableBlock(block) {
				targets[block] = true
			}
		}
		if len(targets) == 0 || !allPathsReachRecursiveCall(function, targets) {
			return false
		}
	}
	return true
}

func allPathsReachRecursiveCall(function *controlflow.Function, targets map[*controlflow.Block]bool) bool {
	state := make(map[*controlflow.Block]uint8)
	var visit func(*controlflow.Block) bool
	visit = func(block *controlflow.Block) bool {
		if block == nil || !function.ReachableBlock(block) {
			return true
		}
		if targets[block] {
			return true
		}
		if block == function.Exit || len(block.Successors) == 0 || state[block] == 1 {
			return false
		}
		if state[block] == 2 {
			return true
		}
		state[block] = 1
		for _, edge := range block.Successors {
			if !visit(edge.To) {
				return false
			}
		}
		state[block] = 2
		return true
	}
	return visit(function.Entry)
}

func unconditionalCall(file *project.File, call *parser.Node) bool {
	if file == nil || call == nil || call.HasError || call.Tok.Origin != nil || file.Walk.Inactive(call) || file.Walk.Uncertain(call) {
		return false
	}
	callee := call.Field("function")
	if callee == nil || callee.Tok.Origin != nil {
		return false
	}
	for index := range file.Parsed.Tokens {
		current := &file.Parsed.Tokens[index]
		if current.Start.Offset >= callee.Start && current.End.Offset <= call.End && current.Origin != nil {
			return false
		}
	}
	for current := call; current != nil; current = file.Walk.Parent(current) {
		parent := file.Walk.Parent(current)
		if parent == nil {
			break
		}
		switch parent.Kind {
		case parser.KindMacroBody, parser.KindMacroInvocation, parser.KindMacroInvocationBlock,
			parser.KindConditionalSplice, parser.KindSharedConditional, parser.KindConditionalFunction,
			parser.KindSizeofExpression, parser.KindTagofExpression, parser.KindDefinedExpression:
			return false
		case parser.KindTernaryExpression:
			if !unconditionalInside(call, parent.Field("condition")) {
				return false
			}
		case parser.KindBinaryExpression:
			if (parent.Tok.Kind == token.AndAnd || parent.Tok.Kind == token.OrOr) && unconditionalInside(call, parent.Field("right")) {
				return false
			}
		case parser.KindUnaryExpression:
			if !unconditionalBuiltinUnary(parent.Tok.Kind) {
				return false
			}
		case parser.KindFunctionDefinition:
			return true
		}
	}
	return true
}

func unconditionalBuiltinUnary(kind token.Kind) bool {
	switch kind {
	case token.Plus, token.Minus, token.Bang, token.Tilde, token.PlusPlus, token.MinusMinus:
		return true
	default:
		return false
	}
}

func unconditionalInside(node, container *parser.Node) bool {
	return node != nil && container != nil && node.Start >= container.Start && node.End <= container.End
}
