package openmp

import (
	"fmt"
	"sort"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/controlflow"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
)

type RequiredCallOrder struct{}

func (RequiredCallOrder) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "required-call-order",
		Name:            "Required call order",
		Summary:         "Reports API calls missing a required earlier call",
		Explanation:     "API metadata can require other natives to be called earlier on every path through the same function. Calls in uncertain control flow, nested calls, and expressions without a definite evaluation order are skipped.",
		Category:        diagnostic.CategoryCorrectness,
		DefaultSeverity: diagnostic.SeverityError,
		AnalysisLevel:   lint.ControlFlowAnalysis,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"calls", "order", "api", "contracts", "control-flow"},
	}
}

type callOrderEvent struct {
	offset       int
	name         string
	requirements []string
	node         *parser.Node
}

func (RequiredCallOrder) Run(ctx *lint.Context) {
	if ctx.Flow == nil || ctx.Semantic == nil {
		return
	}
	for _, function := range ctx.Flow.Functions {
		if function.Uncertain {
			continue
		}
		events := callOrderEvents(ctx, function)
		requirements := callOrderRequirements(events)
		available := make(map[string]map[*controlflow.Block]bool, len(requirements))
		for requirement := range requirements {
			available[requirement] = definitePriorCalls(function, events, requirement)
		}
		for block, blockEvents := range events {
			if !function.ReachableBlock(block) {
				continue
			}
			state := make(map[string]bool, len(requirements))
			for requirement := range requirements {
				state[requirement] = available[requirement][block]
			}
			for _, event := range blockEvents {
				for _, requirement := range event.requirements {
					if state[requirement] {
						continue
					}
					ctx.Report(diagnostic.Diagnostic{
						Message:  fmt.Sprintf("%q requires an earlier call to %q in this function", event.name, requirement),
						Filename: ctx.File.Path,
						Range:    ctx.Walk.Range(event.node),
					})
				}
				state[event.name] = true
			}
		}
	}
}

func callOrderEvents(ctx *lint.Context, function *controlflow.Function) map[*controlflow.Block][]callOrderEvent {
	events := make(map[*controlflow.Block][]callOrderEvent)
	for _, call := range ctx.Walk.OfKind(parser.KindCallExpression) {
		if ctx.Walk.EnclosingFunction(call) != function.Node || !safeCallOrderPosition(ctx, call) {
			continue
		}
		native, name, ok := calledNative(ctx, call)
		if !ok {
			continue
		}
		block := function.Block(call)
		if block == nil {
			continue
		}
		events[block] = append(events[block], callOrderEvent{offset: call.Start, name: name, requirements: native.RequiresBefore, node: call})
	}
	for block := range events {
		sort.SliceStable(events[block], func(i, j int) bool {
			if events[block][i].offset != events[block][j].offset {
				return events[block][i].offset < events[block][j].offset
			}
			return events[block][i].name < events[block][j].name
		})
	}
	return events
}

func safeCallOrderPosition(ctx *lint.Context, call *parser.Node) bool {
	for parent := ctx.Walk.Parent(call); parent != nil; parent = ctx.Walk.Parent(parent) {
		switch parent.Kind {
		case parser.KindCallExpression, parser.KindArgumentList, parser.KindTernaryExpression,
			parser.KindBinaryExpression, parser.KindExpressionList, parser.KindForStatement:
			return false
		}
		if walk.IsStatement(parent) {
			return true
		}
	}
	return false
}

func callOrderRequirements(events map[*controlflow.Block][]callOrderEvent) map[string]struct{} {
	result := make(map[string]struct{})
	for _, blockEvents := range events {
		for _, event := range blockEvents {
			for _, requirement := range event.requirements {
				result[requirement] = struct{}{}
			}
		}
	}
	return result
}

func definitePriorCalls(function *controlflow.Function, events map[*controlflow.Block][]callOrderEvent, requirement string) map[*controlflow.Block]bool {
	calledIn := make(map[*controlflow.Block]bool, len(function.Blocks))
	calledOut := make(map[*controlflow.Block]bool, len(function.Blocks))
	for _, block := range function.Blocks {
		if function.ReachableBlock(block) {
			calledIn[block] = true
			calledOut[block] = true
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
					incoming = incoming && calledOut[predecessor]
				}
				if !hasPredecessor {
					incoming = false
				}
			}
			outgoing := incoming
			for _, event := range events[block] {
				outgoing = outgoing || event.name == requirement
			}
			if calledIn[block] != incoming || calledOut[block] != outgoing {
				calledIn[block] = incoming
				calledOut[block] = outgoing
				changed = true
			}
		}
	}
	return calledIn
}
