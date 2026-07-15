package security

import (
	"fmt"
	"sort"
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

type taintCallable struct {
	name       string
	parameters []api.Parameter
	buffers    []api.Buffer
	callee     *taintFunction
	known      bool
}

type taintFinding struct {
	file    *project.File
	node    *parser.Node
	call    string
	sink    string
	sources taintLabels
}

type taintAnalyzer struct {
	ctx       *lint.Context
	unit      *project.Unit
	functions map[taintFunctionKey]*taintFunction
	calls     map[taintCallKey]*taintFunction
	inputs    map[taintFunctionKey]map[int]taintLabels
	queued    map[taintFunctionKey]bool
	queue     []taintFunctionKey
	findings  map[string]taintFinding
}

func (TaintedDataToSink) Metadata() lint.Metadata {
	return lint.Metadata{
		ID:              "tainted-data-to-sink",
		Name:            "Tainted data to sink",
		Summary:         "Reports configured input reaching a configured sensitive sink",
		Explanation:     "Configured callback inputs and callable output parameters are traced through direct local assignments, known buffer writers, and project function parameters. A diagnostic is reported when that data reaches a configured SQL, command, file, format, or custom sink. Unknown calls, unsupported transformations, ambiguous resolution, macros, and uncertain functions terminate the proof.",
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
		ctx:       ctx,
		unit:      unit,
		functions: make(map[taintFunctionKey]*taintFunction),
		calls:     make(map[taintCallKey]*taintFunction),
		inputs:    make(map[taintFunctionKey]map[int]taintLabels),
		queued:    make(map[taintFunctionKey]bool),
		findings:  make(map[string]taintFinding),
	}
	members := make(map[*project.File]struct{}, len(unit.Files))
	for _, file := range unit.Files {
		members[file] = struct{}{}
	}
	for _, declaration := range ctx.Project.CallGraph.Functions {
		if _, included := members[declaration.File]; !included {
			continue
		}
		key := taintFunctionKey{file: declaration.File, node: declaration.Node}
		function := &taintFunction{key: key, declaration: declaration, usable: taintFunctionUsable(declaration.File, declaration.Node)}
		function.parameters = taintParameterSymbols(declaration.File, declaration.Node)
		analyzer.functions[key] = function
		analyzer.enqueue(key)
	}
	for _, call := range ctx.Project.CallGraph.Calls {
		if _, included := members[call.File]; !included {
			continue
		}
		callee := analyzer.functions[taintFunctionKey{file: call.Callee.File, node: call.Callee.Node}]
		if callee != nil {
			analyzer.calls[taintCallKey{file: call.File, node: call.Node}] = callee
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
	for index, labels := range analyzer.inputs[function.key] {
		if index < len(function.parameters) && function.parameters[index] != nil {
			environment[function.parameters[index]] = copyTaintLabels(labels)
		}
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
			analyzer.applyCall(function, environment, event.node, collect)
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

func (analyzer *taintAnalyzer) applyCall(function *taintFunction, environment map[*semantic.Symbol]taintLabels, call *parser.Node, collect bool) {
	file := function.key.file
	if call.HasError || call.Tok.Origin != nil || file.Walk.Inactive(call) || file.Walk.Uncertain(call) || taintNestedCall(file, call) {
		return
	}
	arguments := call.Field("arguments")
	if arguments == nil {
		return
	}
	callable := analyzer.callable(file, call)
	if !callable.known {
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
		facts[index], known[index] = taintExpression(file, environment, argument)
	}
	for index, parameter := range callable.parameters {
		if index >= len(arguments.Children) {
			break
		}
		if collect && parameter.TaintSink != "" && known[index] && len(facts[index]) != 0 {
			analyzer.addFinding(file, arguments.Children[index], callable.name, parameter.TaintSink, facts[index])
		}
	}
	if callable.callee != nil {
		for index, labels := range facts {
			if known[index] && len(labels) != 0 && index < len(callable.callee.parameters) {
				analyzer.addInput(callable.callee.key, index, labels)
			}
		}
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
	if callable.callee != nil {
		for index, parameter := range callable.callee.parameters {
			if index >= len(arguments.Children) || !taintProjectParameterMayWrite(callable.callee.key.file, parameter) || conditional {
				continue
			}
			symbol, direct := taintWrittenSymbol(file, arguments.Children[index])
			if symbol != nil && direct {
				delete(environment, symbol)
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

func (analyzer *taintAnalyzer) callable(file *project.File, call *parser.Node) taintCallable {
	calleeNode := call.Field("function")
	if calleeNode == nil || calleeNode.Kind != parser.KindIdentifier {
		return taintCallable{}
	}
	name := file.Walk.Text(calleeNode)
	if callee := analyzer.calls[taintCallKey{file: file, node: call}]; callee != nil {
		callable := taintCallable{name: name, callee: callee, known: true}
		if contract, exists := analyzer.ctx.Functions()[name]; exists {
			callable.parameters = contract.Parameters
		}
		return callable
	}
	if declaration, resolved := analyzer.ctx.Project.Resolve(file, calleeNode); resolved {
		if declaration.Node == nil || !walk.HasChildToken(declaration.Node, token.KwNative) {
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
	for _, kind := range []parser.Kind{parser.KindVariableDeclarator, parser.KindAssignmentExpression, parser.KindUpdateExpression, parser.KindCallExpression} {
		for _, node := range file.Walk.OfKind(kind) {
			if file.Walk.EnclosingFunction(node) == function && !file.Walk.Inactive(node) && !file.Walk.Uncertain(node) {
				events = append(events, taintEvent{node: node, kind: kind})
			}
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].node.Start != events[j].node.Start {
			return events[i].node.Start < events[j].node.Start
		}
		return events[i].node.End < events[j].node.End
	})
	return events
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

func taintNestedCall(file *project.File, call *parser.Node) bool {
	for current := file.Walk.Parent(call); current != nil; current = file.Walk.Parent(current) {
		if current.Kind == parser.KindCallExpression {
			return true
		}
		if walk.IsStatement(current) {
			return false
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

func taintInside(node, container *parser.Node) bool {
	return node != nil && container != nil && node.Start >= container.Start && node.End <= container.End
}
