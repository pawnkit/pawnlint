package security

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/api"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/pkg/diagnostic"
	"github.com/pawnkit/pawnlint/pkg/lint"
	"github.com/pawnkit/pawnlint/pkg/project"
)

type TaintedDataToSink struct{}

type taintFunctionKey struct {
	file *project.File
	node *parser.Node
}

type taintFunction struct {
	key         taintFunctionKey
	declaration project.Declaration
	parameters  []*semantic.Symbol
	usable      bool
}

type taintCallKey struct {
	file *project.File
	node *parser.Node
}

type taintLabels map[string]struct{}

type taintEvent struct {
	node *parser.Node
	kind parser.Kind
}

type taintFact struct {
	labels taintLabels
	known  bool
}

type taintCallable struct {
	name       string
	parameters []api.Parameter
	buffers    []api.Buffer
	callees    []*taintFunction
	known      bool
}

type taintAsyncCall struct {
	callee *taintFunction
	offset int
}

type taintFinding struct {
	file    *project.File
	node    *parser.Node
	call    string
	sink    string
	sources taintLabels
}

type taintAnalyzer struct {
	ctx          *lint.Context
	unit         *project.Unit
	functions    map[taintFunctionKey]*taintFunction
	calls        map[taintCallKey][]*taintFunction
	asyncCalls   map[taintCallKey][]taintAsyncCall
	dynamicCalls map[taintCallKey][]taintAsyncCall
	inputs       map[taintFunctionKey]map[int]taintLabels
	returns      map[taintFunctionKey]taintLabels
	outputs      map[taintFunctionKey]map[int]taintLabels
	callers      map[taintFunctionKey]map[taintFunctionKey]struct{}
	queued       map[taintFunctionKey]bool
	queue        []taintFunctionKey
	findings     map[string]taintFinding
}

func (TaintedDataToSink) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "tainted-data-to-sink",
		Name:            "Tainted data to sink",
		Summary:         "Reports configured input reaching a configured sensitive sink",
		Explanation:     "Configured sources are traced through local expressions, known buffer writers, project parameters, return values, and scalar reference outputs. The rule reports flows into configured sinks and stops when resolution or transformation is uncertain.",
		Category:        diagnostic.CategorySecurity,
		DefaultSeverity: diagnostic.SeverityWarning,
		AnalysisLevel:   lint.ProjectAnalysis,
		Stability:       lint.StabilityPreview,
		DefaultEnabled:  false,
		Fixable:         false,
		Tags:            []string{"security", "taint", "input", "sink", "project"},
	}
}

func (TaintedDataToSink) Run(ctx *lint.Context) {
	if ctx.Project == nil || ctx.ProjectFile == nil || ctx.Project.CallGraph == nil || !ctx.ProjectFile.Provided {
		return
	}
	var unit *project.Unit
	for _, candidate := range ctx.Project.Units {
		if candidate.Root == ctx.ProjectFile {
			unit = candidate
			break
		}
	}
	if unit == nil {
		return
	}
	analyzer := newTaintAnalyzer(ctx, unit)
	analyzer.propagate()
	analyzer.collect()
	analyzer.report()
}

func newTaintAnalyzer(ctx *lint.Context, unit *project.Unit) *taintAnalyzer {
	analyzer := &taintAnalyzer{
		ctx:          ctx,
		unit:         unit,
		functions:    make(map[taintFunctionKey]*taintFunction),
		calls:        make(map[taintCallKey][]*taintFunction),
		asyncCalls:   make(map[taintCallKey][]taintAsyncCall),
		dynamicCalls: make(map[taintCallKey][]taintAsyncCall),
		inputs:       make(map[taintFunctionKey]map[int]taintLabels),
		returns:      make(map[taintFunctionKey]taintLabels),
		outputs:      make(map[taintFunctionKey]map[int]taintLabels),
		callers:      make(map[taintFunctionKey]map[taintFunctionKey]struct{}),
		queued:       make(map[taintFunctionKey]bool),
		findings:     make(map[string]taintFinding),
	}
	members := make(map[*project.File]struct{}, len(unit.Files))
	for _, file := range unit.Files {
		members[file] = struct{}{}
	}
	for _, declaration := range ctx.Project.CallGraph.Functions {
		if _, included := members[declaration.File]; !included {
			continue
		}
		node := declaration.PointerNode()
		key := taintFunctionKey{file: declaration.File, node: node}
		function := &taintFunction{key: key, declaration: declaration, usable: taintFunctionUsable(declaration.File, node)}
		function.parameters = taintParameterSymbols(declaration.File, node)
		analyzer.functions[key] = function
		analyzer.enqueue(key)
	}
	for _, call := range ctx.Project.CallGraph.Calls {
		if _, included := members[call.File]; !included {
			continue
		}
		callee := analyzer.functions[taintFunctionKey{file: call.Callee.File, node: call.Callee.PointerNode()}]
		if callee != nil {
			node := call.PointerNode()
			if call.Kind == project.CallDynamic {
				key := taintCallKey{file: call.File, node: node}
				analyzer.dynamicCalls[key] = append(analyzer.dynamicCalls[key], taintAsyncCall{callee: callee, offset: call.ArgumentOffset})
				continue
			}
			callKey := taintCallKey{file: call.File, node: node}
			analyzer.calls[callKey] = append(analyzer.calls[callKey], callee)
			callerNode := call.File.Walk.EnclosingFunction(node)
			callerKey := taintFunctionKey{file: call.File, node: callerNode}
			if analyzer.functions[callerKey] != nil {
				if analyzer.callers[callee.key] == nil {
					analyzer.callers[callee.key] = make(map[taintFunctionKey]struct{})
				}
				analyzer.callers[callee.key][callerKey] = struct{}{}
			}
		}
	}
	for _, call := range ctx.Project.CallGraph.AsyncCalls {
		if _, included := members[call.File]; !included || call.ArgumentOffset < 0 {
			continue
		}
		callee := analyzer.functions[taintFunctionKey{file: call.Callee.File, node: call.Callee.PointerNode()}]
		if callee != nil {
			key := taintCallKey{file: call.File, node: call.PointerNode()}
			analyzer.asyncCalls[key] = append(analyzer.asyncCalls[key], taintAsyncCall{callee: callee, offset: call.ArgumentOffset})
		}
	}
	for _, function := range analyzer.functions {
		if !function.usable || !walk.HasChildToken(function.key.node, token.KwPublic) {
			continue
		}
		callback, known := ctx.Callbacks()[function.declaration.Name]
		if !known {
			continue
		}
		for index, parameter := range callback.Parameters {
			if parameter.TaintSource != "" && index < len(function.parameters) {
				analyzer.addInput(function.key, index, taintLabels{parameter.TaintSource: {}})
			}
		}
	}
	return analyzer
}

func (analyzer *taintAnalyzer) propagate() {
	for len(analyzer.queue) != 0 {
		key := analyzer.queue[0]
		analyzer.queue = analyzer.queue[1:]
		analyzer.queued[key] = false
		if function := analyzer.functions[key]; function != nil && function.usable {
			analyzer.analyze(function, false)
		}
	}
}

func (analyzer *taintAnalyzer) collect() {
	for _, function := range analyzer.functions {
		if function.usable {
			analyzer.analyze(function, true)
		}
	}
}

func (analyzer *taintAnalyzer) analyze(function *taintFunction, collect bool) {
	environment := make(map[*semantic.Symbol]taintLabels)
	callResults := make(map[*parser.Node]taintFact)
	for index, labels := range analyzer.inputs[function.key] {
		if index < len(function.parameters) && function.parameters[index] != nil {
			environment[function.parameters[index]] = copyTaintLabels(labels)
		}
	}
	for index, parameter := range function.parameters {
		if parameter == nil {
			continue
		}
		if environment[parameter] == nil {
			environment[parameter] = make(taintLabels)
		}
		environment[parameter][taintParameterLabel(index)] = struct{}{}
	}
	events := taintEvents(function.key.file, function.key.node)
	for _, event := range events {
		switch event.kind {
		case parser.KindVariableDeclarator:
			analyzer.applyDeclarator(function, environment, event.node)
		case parser.KindAssignmentExpression:
			analyzer.applyAssignment(function, environment, event.node)
		case parser.KindUpdateExpression:
			analyzer.applyUpdate(function, environment, event.node)
		case parser.KindCallExpression:
			analyzer.applyCall(function, environment, callResults, event.node, collect)
		case parser.KindReturnStatement:
			analyzer.applyReturn(function, environment, callResults, event.node)
		}
	}
	for index, parameter := range function.parameters {
		if taintProjectParameterOutputKnown(function.key.file, parameter) {
			analyzer.addOutput(function.key, index, environment[parameter])
		}
	}
}

func (analyzer *taintAnalyzer) applyDeclarator(function *taintFunction, environment map[*semantic.Symbol]taintLabels, node *parser.Node) {
	symbol := taintDeclaredSymbol(function.key.file, node)
	if symbol == nil {
		return
	}
	initializer := node.Field("initializer")
	if initializer == nil || taintContainsCall(initializer) {
		delete(environment, symbol)
		return
	}
	labels, known := taintExpression(function.key.file, environment, initializer)
	if known {
		taintSet(environment, symbol, labels, false)
	} else {
		delete(environment, symbol)
	}
}

func (analyzer *taintAnalyzer) applyAssignment(function *taintFunction, environment map[*semantic.Symbol]taintLabels, node *parser.Node) {
	left := node.Field("left")
	symbol, direct := taintWrittenSymbol(function.key.file, left)
	if symbol == nil {
		return
	}
	conditional := taintConditional(function.key.file, function.key.node, node) || !direct || node.Tok.Kind != token.Assign
	right := node.Field("right")
	if right == nil || taintContainsCall(right) {
		if !conditional {
			delete(environment, symbol)
		}
		return
	}
	labels, known := taintExpression(function.key.file, environment, right)
	if !known {
		if !conditional {
			delete(environment, symbol)
		}
		return
	}
	taintSet(environment, symbol, labels, conditional)
}

func (analyzer *taintAnalyzer) applyUpdate(function *taintFunction, environment map[*semantic.Symbol]taintLabels, node *parser.Node) {
	symbol, _ := taintWrittenSymbol(function.key.file, node.Field("expression"))
	if symbol != nil && !taintConditional(function.key.file, function.key.node, node) {
		delete(environment, symbol)
	}
}

func (analyzer *taintAnalyzer) applyCall(function *taintFunction, environment map[*semantic.Symbol]taintLabels, callResults map[*parser.Node]taintFact, call *parser.Node, collect bool) {
	file := function.key.file
	if call.HasError || call.Tok.Origin != nil || file.Walk.Inactive(call) || file.Walk.Uncertain(call) {
		return
	}
	arguments := call.Field("arguments")
	if arguments == nil {
		return
	}
	callable := analyzer.callable(file, call)
	if !callable.known {
		callResults[call] = taintFact{}
		if !taintConditional(file, function.key.node, call) {
			for _, argument := range arguments.Children {
				for symbol := range taintExpressionSymbols(file, argument) {
					delete(environment, symbol)
				}
			}
		}
		return
	}
	facts := make([]taintLabels, len(arguments.Children))
	known := make([]bool, len(arguments.Children))
	for index, argument := range arguments.Children {
		facts[index], known[index] = taintExpressionWithCalls(file, environment, callResults, argument)
	}
	for index, parameter := range callable.parameters {
		if index >= len(arguments.Children) {
			break
		}
		labels := concreteTaintLabels(facts[index])
		if collect && parameter.TaintSink != "" && known[index] && len(labels) != 0 {
			analyzer.addFinding(file, arguments.Children[index], callable.name, parameter.TaintSink, labels)
		}
	}
	for _, callee := range callable.callees {
		for index, labels := range facts {
			concrete := concreteTaintLabels(labels)
			if known[index] && len(concrete) != 0 && index < len(callee.parameters) {
				analyzer.addInput(callee.key, index, concrete)
			}
		}
	}
	for _, target := range analyzer.asyncCalls[taintCallKey{file: file, node: call}] {
		for index := target.offset; index < len(facts); index++ {
			concrete := concreteTaintLabels(facts[index])
			parameter := index - target.offset
			if known[index] && len(concrete) != 0 && parameter < len(target.callee.parameters) {
				analyzer.addInput(target.callee.key, parameter, concrete)
			}
		}
	}
	for _, target := range analyzer.dynamicCalls[taintCallKey{file: file, node: call}] {
		for index := target.offset; index < len(facts); index++ {
			concrete := concreteTaintLabels(facts[index])
			parameter := index - target.offset
			if known[index] && len(concrete) != 0 && parameter < len(target.callee.parameters) {
				analyzer.addInput(target.callee.key, parameter, concrete)
			}
		}
	}
	result := make(taintLabels)
	for _, callee := range callable.callees {
		mergeTaintLabels(result, instantiateTaintLabels(analyzer.returns[callee.key], facts, known))
	}
	callResults[call] = taintFact{labels: result, known: true}
	if symbol, direct := taintCallDestination(file, call); symbol != nil {
		taintSet(environment, symbol, result, taintConditional(file, function.key.node, call) || !direct)
	}
	for _, buffer := range callable.buffers {
		destinationIndex := buffer.Parameter - 1
		if destinationIndex < 0 || destinationIndex >= len(arguments.Children) {
			continue
		}
		symbol, direct := taintWrittenSymbol(file, arguments.Children[destinationIndex])
		if symbol == nil {
			continue
		}
		labels := make(taintLabels)
		for index, fact := range facts {
			if index != destinationIndex && known[index] {
				mergeTaintLabels(labels, fact)
			}
		}
		taintSet(environment, symbol, labels, taintConditional(file, function.key.node, call) || !direct)
	}
	for index, parameter := range callable.parameters {
		if index >= len(arguments.Children) || parameter.TaintSource == "" {
			continue
		}
		symbol, direct := taintWrittenSymbol(file, arguments.Children[index])
		if symbol != nil {
			labels := taintLabels{parameter.TaintSource: {}}
			taintSet(environment, symbol, labels, taintConditional(file, function.key.node, call) || !direct)
		}
	}
	for index, parameter := range callable.parameters {
		if index >= len(arguments.Children) || !parameter.Output || parameter.TaintSource != "" || taintBufferParameter(callable.buffers, index+1) {
			continue
		}
		symbol, direct := taintWrittenSymbol(file, arguments.Children[index])
		if symbol != nil && direct && !taintConditional(file, function.key.node, call) {
			delete(environment, symbol)
		}
	}
	conditional := taintConditional(file, function.key.node, call)
	if len(callable.callees) != 0 && !conditional {
		for index := range arguments.Children {
			labels := make(taintLabels)
			mayWrite := false
			for _, callee := range callable.callees {
				if index >= len(callee.parameters) || !taintProjectParameterMayWrite(callee.key.file, callee.parameters[index]) {
					continue
				}
				mayWrite = true
				mergeTaintLabels(labels, instantiateTaintLabels(analyzer.outputs[callee.key][index], facts, known))
			}
			if !mayWrite {
				continue
			}
			symbol, direct := taintWrittenSymbol(file, arguments.Children[index])
			if symbol != nil && direct {
				taintSet(environment, symbol, labels, false)
			}
		}
	}
	for index, parameter := range callable.parameters {
		if index >= len(arguments.Children) || parameter.Const || parameter.Output || parameter.TaintSource != "" || taintBufferParameter(callable.buffers, index+1) || conditional {
			continue
		}
		if parameter.ArrayRank == 0 && !parameter.Reference {
			continue
		}
		symbol, direct := taintWrittenSymbol(file, arguments.Children[index])
		if symbol != nil && direct {
			delete(environment, symbol)
		}
	}
}

func (analyzer *taintAnalyzer) applyReturn(function *taintFunction, environment map[*semantic.Symbol]taintLabels, callResults map[*parser.Node]taintFact, node *parser.Node) {
	value := node.Field("value")
	if value == nil {
		return
	}
	labels, known := taintExpressionWithCalls(function.key.file, environment, callResults, value)
	if known {
		analyzer.addReturn(function.key, labels)
	}
}

func (analyzer *taintAnalyzer) callable(file *project.File, call *parser.Node) taintCallable {
	calleeNode := call.Field("function")
	if calleeNode == nil || calleeNode.Kind != parser.KindIdentifier {
		return taintCallable{}
	}
	name := file.Walk.Text(calleeNode)
	if callees := analyzer.calls[taintCallKey{file: file, node: call}]; len(callees) != 0 {
		callable := taintCallable{name: name, callees: callees, known: true}
		if contract, exists := analyzer.ctx.Functions()[name]; exists {
			callable.parameters = contract.Parameters
		}
		return callable
	}
	if declaration, resolved := analyzer.ctx.Project.Resolve(file, calleeNode); resolved {
		if !declaration.Valid() || !declaration.HasToken(token.KwNative) {
			return taintCallable{}
		}
	}
	if native, exists := analyzer.ctx.Natives()[name]; exists {
		return taintCallable{name: name, parameters: native.Parameters, buffers: native.Buffers, known: true}
	}
	if function, exists := analyzer.ctx.Functions()[name]; exists {
		return taintCallable{name: name, parameters: function.Parameters, known: true}
	}
	return taintCallable{}
}

func (analyzer *taintAnalyzer) addInput(key taintFunctionKey, index int, labels taintLabels) {
	if len(labels) == 0 {
		return
	}
	if analyzer.inputs[key] == nil {
		analyzer.inputs[key] = make(map[int]taintLabels)
	}
	if analyzer.inputs[key][index] == nil {
		analyzer.inputs[key][index] = make(taintLabels)
	}
	changed := mergeTaintLabels(analyzer.inputs[key][index], labels)
	if changed {
		analyzer.enqueue(key)
	}
}

func (analyzer *taintAnalyzer) addReturn(key taintFunctionKey, labels taintLabels) {
	labels = analyzer.summaryTaintLabels(key, labels)
	if len(labels) == 0 {
		return
	}
	if analyzer.returns[key] == nil {
		analyzer.returns[key] = make(taintLabels)
	}
	if mergeTaintLabels(analyzer.returns[key], labels) {
		analyzer.enqueueCallers(key)
	}
}

func (analyzer *taintAnalyzer) addOutput(key taintFunctionKey, index int, labels taintLabels) {
	labels = analyzer.summaryTaintLabels(key, labels)
	if len(labels) == 0 {
		return
	}
	if analyzer.outputs[key] == nil {
		analyzer.outputs[key] = make(map[int]taintLabels)
	}
	if analyzer.outputs[key][index] == nil {
		analyzer.outputs[key][index] = make(taintLabels)
	}
	if mergeTaintLabels(analyzer.outputs[key][index], labels) {
		analyzer.enqueueCallers(key)
	}
}

func (analyzer *taintAnalyzer) summaryTaintLabels(key taintFunctionKey, labels taintLabels) taintLabels {
	result := make(taintLabels)
	inputs := make(taintLabels)
	for _, current := range analyzer.inputs[key] {
		mergeTaintLabels(inputs, current)
	}
	for label := range labels {
		if taintParameterIndex(label) >= 0 {
			result[label] = struct{}{}
		} else if _, propagated := inputs[label]; !propagated {
			result[label] = struct{}{}
		}
	}
	return result
}

func (analyzer *taintAnalyzer) enqueueCallers(key taintFunctionKey) {
	for caller := range analyzer.callers[key] {
		analyzer.enqueue(caller)
	}
}

func (analyzer *taintAnalyzer) enqueue(key taintFunctionKey) {
	if analyzer.functions[key] == nil || analyzer.queued[key] {
		return
	}
	analyzer.queued[key] = true
	analyzer.queue = append(analyzer.queue, key)
}

func (analyzer *taintAnalyzer) addFinding(file *project.File, node *parser.Node, call, sink string, sources taintLabels) {
	key := file.Path + "\x00" + fmt.Sprint(node.Start) + "\x00" + sink
	if finding, exists := analyzer.findings[key]; exists {
		mergeTaintLabels(finding.sources, sources)
		analyzer.findings[key] = finding
		return
	}
	analyzer.findings[key] = taintFinding{file: file, node: node, call: call, sink: sink, sources: copyTaintLabels(sources)}
}

func (analyzer *taintAnalyzer) report() {
	findings := make([]taintFinding, 0, len(analyzer.findings))
	for _, finding := range analyzer.findings {
		findings = append(findings, finding)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].file.Path != findings[j].file.Path {
			return findings[i].file.Path < findings[j].file.Path
		}
		return findings[i].node.Start < findings[j].node.Start
	})
	for _, finding := range findings {
		sources := make([]string, 0, len(finding.sources))
		for source := range finding.sources {
			sources = append(sources, source)
		}
		sort.Strings(sources)
		quoted := make([]string, len(sources))
		for index, source := range sources {
			quoted[index] = fmt.Sprintf("%q", source)
		}
		analyzer.ctx.Report(diagnostic.Diagnostic{
			Message:     fmt.Sprintf("data from %s reaches %q sink parameter of %q", strings.Join(quoted, ", "), finding.sink, finding.call),
			Filename:    finding.file.Path,
			Range:       finding.file.Walk.Range(finding.node),
			Suggestions: []diagnostic.Suggestion{{Description: "validate or sanitize the value before this call"}},
		})
	}
}

func taintFunctionUsable(file *project.File, function *parser.Node) bool {
	if function == nil || function.HasError || file.Walk.Inactive(function) || file.Walk.Uncertain(function) {
		return false
	}
	for _, kind := range []parser.Kind{parser.KindMacroInvocation, parser.KindMacroInvocationBlock, parser.KindConditionalSplice} {
		for _, node := range file.Walk.OfKind(kind) {
			if taintInside(node, function) {
				return false
			}
		}
	}
	return true
}

func taintParameterSymbols(file *project.File, function *parser.Node) []*semantic.Symbol {
	list := function.Field("parameters")
	if list == nil {
		return nil
	}
	var result []*semantic.Symbol
	for _, parameter := range list.Children {
		if parameter.Kind != parser.KindParameter {
			continue
		}
		var found *semantic.Symbol
		for _, symbol := range file.Semantic.Symbols {
			if symbol.Kind == semantic.SymbolParameter && symbol.Decl == parameter && !symbol.Ambiguous {
				found = symbol
				break
			}
		}
		result = append(result, found)
	}
	return result
}

func taintEvents(file *project.File, function *parser.Node) []taintEvent {
	var events []taintEvent
	for _, kind := range []parser.Kind{parser.KindVariableDeclarator, parser.KindAssignmentExpression, parser.KindUpdateExpression, parser.KindCallExpression, parser.KindReturnStatement} {
		for _, node := range file.Walk.OfKind(kind) {
			if file.Walk.EnclosingFunction(node) == function && !file.Walk.Inactive(node) && !file.Walk.Uncertain(node) {
				events = append(events, taintEvent{node: node, kind: kind})
			}
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if taintInside(events[j].node, events[i].node) && events[i].node != events[j].node {
			return events[i].kind == parser.KindVariableDeclarator || events[i].kind == parser.KindAssignmentExpression || events[i].kind == parser.KindUpdateExpression
		}
		if taintInside(events[i].node, events[j].node) && events[i].node != events[j].node {
			return events[j].kind != parser.KindVariableDeclarator && events[j].kind != parser.KindAssignmentExpression && events[j].kind != parser.KindUpdateExpression
		}
		if events[i].node.Start != events[j].node.Start {
			return events[i].node.Start < events[j].node.Start
		}
		return events[i].node.End < events[j].node.End
	})
	return events
}

func taintCallDestination(file *project.File, call *parser.Node) (*semantic.Symbol, bool) {
	current := call
	for parent := file.Walk.Parent(current); parent != nil; parent = file.Walk.Parent(current) {
		switch parent.Kind {
		case parser.KindParenthesizedExpression, parser.KindTaggedExpression:
			if parent.Field("expression") != current {
				return nil, false
			}
			current = parent
		case parser.KindVariableDeclarator:
			if parent.Field("initializer") != current {
				return nil, false
			}
			return taintDeclaredSymbol(file, parent), true
		case parser.KindAssignmentExpression:
			if parent.Tok.Kind != token.Assign || parent.Field("right") != current {
				return nil, false
			}
			return taintWrittenSymbol(file, parent.Field("left"))
		default:
			return nil, false
		}
	}
	return nil, false
}

func taintDeclaredSymbol(file *project.File, node *parser.Node) *semantic.Symbol {
	for _, symbol := range file.Semantic.Symbols {
		if symbol.Decl == node && symbol.Kind == semantic.SymbolLocal && !symbol.Ambiguous {
			return symbol
		}
	}
	return nil
}

func taintWrittenSymbol(file *project.File, node *parser.Node) (*semantic.Symbol, bool) {
	for node != nil && node.Kind == parser.KindParenthesizedExpression {
		node = node.Field("expression")
	}
	direct := true
	identifier := node
	if node != nil && node.Kind == parser.KindSubscriptExpression {
		direct = false
		identifier = node.Field("array")
	}
	if identifier == nil || identifier.Kind != parser.KindIdentifier {
		return nil, false
	}
	symbol := file.Semantic.Resolve(identifier)
	if symbol == nil || symbol.Ambiguous || symbol.Kind != semantic.SymbolLocal && symbol.Kind != semantic.SymbolParameter {
		return nil, false
	}
	return symbol, direct
}

func taintExpression(file *project.File, environment map[*semantic.Symbol]taintLabels, node *parser.Node) (taintLabels, bool) {
	labels := make(taintLabels)
	known := true
	var visit func(*parser.Node)
	visit = func(current *parser.Node) {
		if current == nil || !known {
			return
		}
		if current.Kind == parser.KindCallExpression || current.HasError || current.Tok.Origin != nil || file.Walk.Inactive(current) || file.Walk.Uncertain(current) {
			known = false
			return
		}
		if current.Kind == parser.KindIdentifier {
			symbol := file.Semantic.Resolve(current)
			if symbol == nil {
				return
			}
			if symbol.Ambiguous || symbol.Kind == semantic.SymbolGlobal {
				known = false
				return
			}
			mergeTaintLabels(labels, environment[symbol])
		}
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	return labels, known
}

func taintExpressionWithCalls(file *project.File, environment map[*semantic.Symbol]taintLabels, calls map[*parser.Node]taintFact, node *parser.Node) (taintLabels, bool) {
	labels := make(taintLabels)
	known := true
	var visit func(*parser.Node)
	visit = func(current *parser.Node) {
		if current == nil || !known {
			return
		}
		if current.Kind == parser.KindCallExpression {
			fact, exists := calls[current]
			if !exists || !fact.known {
				known = false
				return
			}
			mergeTaintLabels(labels, fact.labels)
			return
		}
		if current.HasError || current.Tok.Origin != nil || file.Walk.Inactive(current) || file.Walk.Uncertain(current) {
			known = false
			return
		}
		if current.Kind == parser.KindIdentifier {
			symbol := file.Semantic.Resolve(current)
			if symbol == nil {
				return
			}
			if symbol.Ambiguous || symbol.Kind == semantic.SymbolGlobal {
				known = false
				return
			}
			mergeTaintLabels(labels, environment[symbol])
		}
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	return labels, known
}

func taintExpressionSymbols(file *project.File, node *parser.Node) map[*semantic.Symbol]struct{} {
	result := make(map[*semantic.Symbol]struct{})
	var visit func(*parser.Node)
	visit = func(current *parser.Node) {
		if current == nil {
			return
		}
		if current.Kind == parser.KindIdentifier {
			symbol := file.Semantic.Resolve(current)
			if symbol != nil && !symbol.Ambiguous && (symbol.Kind == semantic.SymbolLocal || symbol.Kind == semantic.SymbolParameter) {
				result[symbol] = struct{}{}
			}
		}
		for _, child := range current.Children {
			visit(child)
		}
	}
	visit(node)
	return result
}

func taintConditional(file *project.File, function, node *parser.Node) bool {
	for current := file.Walk.Parent(node); current != nil && current != function; current = file.Walk.Parent(current) {
		switch current.Kind {
		case parser.KindIfStatement, parser.KindWhileStatement, parser.KindDoWhileStatement, parser.KindForStatement,
			parser.KindSwitchStatement, parser.KindCaseClause, parser.KindDefaultClause, parser.KindTernaryExpression:
			return true
		case parser.KindBinaryExpression:
			if current.Tok.Kind == token.AndAnd || current.Tok.Kind == token.OrOr {
				return true
			}
		}
	}
	return false
}

func taintContainsCall(node *parser.Node) bool {
	if node == nil {
		return false
	}
	if node.Kind == parser.KindCallExpression {
		return true
	}
	for _, child := range node.Children {
		if taintContainsCall(child) {
			return true
		}
	}
	return false
}

func taintBufferParameter(buffers []api.Buffer, parameter int) bool {
	for _, buffer := range buffers {
		if buffer.Parameter == parameter {
			return true
		}
	}
	return false
}

func taintProjectParameterMayWrite(file *project.File, symbol *semantic.Symbol) bool {
	if symbol == nil || symbol.Decl == nil || walk.HasChildToken(symbol.Decl, token.KwConst) {
		return false
	}
	if file != nil && file.Parsed != nil && walk.ReferencesByAmpersand(file.Parsed.Tokens, symbol.Decl) {
		return true
	}
	for _, child := range symbol.Decl.Children {
		if child.Kind == parser.KindDimension {
			return true
		}
	}
	return false
}

func taintProjectParameterOutputKnown(file *project.File, symbol *semantic.Symbol) bool {
	if symbol == nil || symbol.Decl == nil || file == nil || file.Parsed == nil || !walk.ReferencesByAmpersand(file.Parsed.Tokens, symbol.Decl) {
		return false
	}
	for _, child := range symbol.Decl.Children {
		if child.Kind == parser.KindDimension {
			return false
		}
	}
	return true
}

func taintSet(environment map[*semantic.Symbol]taintLabels, symbol *semantic.Symbol, labels taintLabels, union bool) {
	if union {
		if environment[symbol] == nil {
			environment[symbol] = make(taintLabels)
		}
		mergeTaintLabels(environment[symbol], labels)
		return
	}
	if len(labels) == 0 {
		delete(environment, symbol)
		return
	}
	environment[symbol] = copyTaintLabels(labels)
}

func mergeTaintLabels(destination, source taintLabels) bool {
	changed := false
	for label := range source {
		if _, exists := destination[label]; !exists {
			destination[label] = struct{}{}
			changed = true
		}
	}
	return changed
}

func copyTaintLabels(source taintLabels) taintLabels {
	result := make(taintLabels, len(source))
	mergeTaintLabels(result, source)
	return result
}

func taintParameterLabel(index int) string {
	return "\x00parameter:" + strconv.Itoa(index)
}

func taintParameterIndex(label string) int {
	const prefix = "\x00parameter:"
	if !strings.HasPrefix(label, prefix) {
		return -1
	}
	index, err := strconv.Atoi(strings.TrimPrefix(label, prefix))
	if err != nil {
		return -1
	}
	return index
}

func concreteTaintLabels(labels taintLabels) taintLabels {
	result := make(taintLabels)
	for label := range labels {
		if taintParameterIndex(label) < 0 {
			result[label] = struct{}{}
		}
	}
	return result
}

func instantiateTaintLabels(summary taintLabels, arguments []taintLabels, known []bool) taintLabels {
	result := make(taintLabels)
	for label := range summary {
		index := taintParameterIndex(label)
		if index < 0 {
			result[label] = struct{}{}
			continue
		}
		if index < len(arguments) && index < len(known) && known[index] {
			mergeTaintLabels(result, arguments[index])
		}
	}
	return result
}

func taintInside(node, container *parser.Node) bool {
	return node != nil && container != nil && node.Start >= container.Start && node.End <= container.End
}
