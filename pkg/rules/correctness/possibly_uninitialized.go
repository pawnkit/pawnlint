package correctness

import (
	"fmt"
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type PossiblyUninitialized struct{}

func (PossiblyUninitialized) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "possibly-uninitialized",
		Name:            "Possibly uninitialized",
		Summary:         "Reports local variables read before an explicit assignment on every path",
		Explanation:     "Pawn zero-fills local cells, but the compiler still tracks whether a local received an explicit value. This rule reports reads that can occur before an initializer or assignment. Unknown and by-reference call arguments stop tracking conservatively, while API parameters marked as outputs establish assignment.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"control-flow", "initialization", "data-flow"},
	}
}

type assignmentEventKind uint8

const (
	eventReset assignmentEventKind = iota
	eventRead
	eventSet
)

type assignmentEvent struct {
	kind   assignmentEventKind
	offset int
	node   *parser.Node
}

func (PossiblyUninitialized) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	for _, function := range ctx.Flow.Functions {
		if function.Uncertain {
			continue
		}
		for _, symbol := range ctx.Semantic.Symbols {
			if !initializationCandidate(ctx, function, symbol) {
				continue
			}
			events, ok := initializationEvents(ctx, function, symbol)
			if !ok {
				continue
			}
			assignedIn := definiteAssignment(function, events)
			for block, blockEvents := range events {
				if !function.ReachableBlock(block) {
					continue
				}
				assigned := assignedIn[block]
				for _, event := range blockEvents {
					switch event.kind {
					case eventReset:
						assigned = false
					case eventSet:
						assigned = true
					case eventRead:
						if assigned {
							continue
						}
						ctx.Report(diagnostic.Diagnostic{
							Message:  fmt.Sprintf("%q may be read before an explicit assignment", symbol.Name),
							Filename: ctx.File.Path,
							Range:    ctx.Walk.Range(event.node),
						})
					}
				}
			}
		}
	}
}

func initializationCandidate(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol) bool {
	if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal || symbol.Function != function.Node || symbol.Decl == nil {
		return false
	}
	for _, child := range symbol.Decl.Children {
		if child.Kind == parser.KindDimension {
			return false
		}
	}
	declaration := ctx.Walk.Parent(symbol.Decl)
	if declaration == nil {
		return false
	}
	for _, child := range declaration.Children {
		if child.Tok.Kind == token.KwStatic {
			return false
		}
	}
	return function.Block(symbol.Decl) != nil
}

func initializationEvents(ctx *lint.Context, function *controlflow.Function, symbol *semantic.Symbol) (map[*controlflow.Block][]assignmentEvent, bool) {
	events := make(map[*controlflow.Block][]assignmentEvent)
	declarationBlock := function.Block(symbol.Decl)
	if declarationBlock == nil || declarationBlock.Node != nil && declarationBlock.Node.Kind == parser.KindForStatement {
		return nil, false
	}
	events[declarationBlock] = append(events[declarationBlock], assignmentEvent{kind: eventReset, offset: symbol.Decl.Start})
	if symbol.Decl.Field("initializer") != nil {
		events[declarationBlock] = append(events[declarationBlock], assignmentEvent{kind: eventSet, offset: symbol.Decl.End})
	}
	for _, reference := range ctx.Semantic.References(symbol) {
		block := function.Block(reference.Node)
		if block == nil || block.Node != nil && block.Node.Kind == parser.KindForStatement {
			return nil, false
		}
		switch reference.Kind {
		case semantic.ReferenceRead, semantic.ReferenceCall:
			effect, argument := callArgumentEffect(ctx, reference.Node)
			if argument {
				switch effect {
				case argumentEffectOutput:
					events[block] = append(events[block], assignmentEvent{kind: eventSet, offset: callEnd(ctx, reference.Node)})
					continue
				case argumentEffectUnknown:
					return nil, false
				}
			}
			events[block] = append(events[block], assignmentEvent{kind: eventRead, offset: reference.Node.Start, node: reference.Node})
		case semantic.ReferenceWrite:
			events[block] = append(events[block], assignmentEvent{kind: eventSet, offset: writeCompletion(ctx, reference.Node)})
		case semantic.ReferenceReadWrite:
			events[block] = append(events[block], assignmentEvent{kind: eventRead, offset: reference.Node.Start, node: reference.Node})
			events[block] = append(events[block], assignmentEvent{kind: eventSet, offset: writeCompletion(ctx, reference.Node)})
		}
	}
	for block := range events {
		sort.SliceStable(events[block], func(i, j int) bool {
			if events[block][i].offset != events[block][j].offset {
				return events[block][i].offset < events[block][j].offset
			}
			return events[block][i].kind < events[block][j].kind
		})
	}
	return events, true
}

type argumentEffect uint8

const (
	argumentEffectRead argumentEffect = iota
	argumentEffectOutput
	argumentEffectUnknown
)

func callArgumentEffect(ctx *lint.Context, node *parser.Node) (argumentEffect, bool) {
	arguments := ctx.Walk.Parent(node)
	if arguments == nil || arguments.Kind != parser.KindArgumentList {
		return argumentEffectRead, false
	}
	index := -1
	for current, argument := range arguments.Children {
		if argument == node {
			index = current
			break
		}
	}
	if index < 0 {
		return argumentEffectRead, true
	}
	call := ctx.Walk.Parent(arguments)
	if call == nil || call.Kind != parser.KindCallExpression {
		return argumentEffectUnknown, true
	}
	callee := call.Field("function")
	if callee == nil || callee.Kind != parser.KindIdentifier {
		return argumentEffectUnknown, true
	}
	name := ctx.Walk.Text(callee)
	if native, ok := ctx.Natives()[name]; ok {
		parameter, ok := nativeParameter(native.Parameters, index)
		if !ok {
			return argumentEffectUnknown, true
		}
		if parameter.Output {
			return argumentEffectOutput, true
		}
		if parameter.Reference {
			return argumentEffectUnknown, true
		}
		return argumentEffectRead, true
	}
	if symbol := ctx.Semantic.Resolve(callee); symbol != nil && !symbol.Ambiguous {
		parameter, ok := functionParameter(symbol.Decl, index)
		if !ok {
			return argumentEffectUnknown, true
		}
		if walk.ReferencesByAmpersand(ctx.File.Parsed.Tokens, parameter) {
			return argumentEffectUnknown, true
		}
		return argumentEffectRead, true
	}
	return argumentEffectUnknown, true
}

func nativeParameter(parameters []api.Parameter, index int) (api.Parameter, bool) {
	if index < len(parameters) {
		return parameters[index], true
	}
	if len(parameters) != 0 && parameters[len(parameters)-1].Variadic {
		return parameters[len(parameters)-1], true
	}
	return api.Parameter{}, false
}

func functionParameter(function *parser.Node, index int) (*parser.Node, bool) {
	if function == nil {
		return nil, false
	}
	parameters := function.Field("parameters")
	if parameters == nil || len(parameters.Children) == 0 {
		return nil, false
	}
	if index < len(parameters.Children) {
		return parameters.Children[index], true
	}
	last := parameters.Children[len(parameters.Children)-1]
	if last.Tok.Kind == token.Ellipsis || walk.HasChildToken(last, token.Ellipsis) {
		return last, true
	}
	return nil, false
}

func callEnd(ctx *lint.Context, node *parser.Node) int {
	for current := node; current != nil; current = ctx.Walk.Parent(current) {
		if current.Kind == parser.KindCallExpression {
			return current.End
		}
	}
	return node.End
}

func writeCompletion(ctx *lint.Context, node *parser.Node) int {
	parent := ctx.Walk.Parent(node)
	if parent == nil {
		return node.End
	}
	switch parent.Kind {
	case parser.KindAssignmentExpression, parser.KindUpdateExpression:
		return parent.End
	default:
		return node.End
	}
}

func definiteAssignment(function *controlflow.Function, events map[*controlflow.Block][]assignmentEvent) map[*controlflow.Block]bool {
	assignedIn := make(map[*controlflow.Block]bool, len(function.Blocks))
	assignedOut := make(map[*controlflow.Block]bool, len(function.Blocks))
	for _, block := range function.Blocks {
		if function.ReachableBlock(block) {
			assignedIn[block] = true
			assignedOut[block] = true
		}
	}
	changed := true
	for changed {
		changed = false
		for _, block := range function.Blocks {
			if !function.ReachableBlock(block) {
				continue
			}
			incoming := false
			if block != function.Entry {
				incoming = true
				hasPredecessor := false
				for _, predecessor := range block.Predecessors {
					if !function.ReachableBlock(predecessor) {
						continue
					}
					hasPredecessor = true
					incoming = incoming && assignedOut[predecessor]
				}
				if !hasPredecessor {
					incoming = false
				}
			}
			outgoing := applyAssignmentEvents(incoming, events[block])
			if assignedIn[block] != incoming || assignedOut[block] != outgoing {
				assignedIn[block] = incoming
				assignedOut[block] = outgoing
				changed = true
			}
		}
	}
	return assignedIn
}

func applyAssignmentEvents(assigned bool, events []assignmentEvent) bool {
	for _, event := range events {
		switch event.kind {
		case eventReset:
			assigned = false
		case eventSet:
			assigned = true
		}
	}
	return assigned
}
