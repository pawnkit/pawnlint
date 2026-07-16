package controlflow

import (
	"sync"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

type EdgeKind uint8

const (
	EdgeNormal EdgeKind = iota
	EdgeBranch
	EdgeJump
	EdgeReturn
	EdgeFallthrough
)

type Edge struct {
	To   *Block
	Kind EdgeKind
}

type Block struct {
	ID           int
	Node         *parser.Node
	Successors   []Edge
	Predecessors []*Block
}

type Function struct {
	Node             *parser.Node
	Entry            *Block
	Exit             *Block
	Blocks           []*Block
	Uncertain        bool
	nodes            map[*parser.Node]*Block
	locations        map[*parser.Node]*Block
	reachable        map[*Block]bool
	fallthroughExits []*Block
	valueIn          map[*Block]map[*semantic.Symbol]int64
	valueEvents      map[*Block][]valueEvent
	valueOnce        sync.Once
	aliasIn          map[*Block]aliasState
	aliasEvents      map[*Block][]aliasEvent
	aliasIndexes     map[*semantic.Symbol]int
	aliasOnce        sync.Once
	flowNodes        *functionNodes
	flowSymbols      []*semantic.Symbol
}

type Model struct {
	Functions []*Function
	byNode    map[*parser.Node]*Function
	semantics *semantic.Model
	tree      *walk.Model
	options   Options
}

type CallEffects struct {
	Complete         bool
	MutatedArguments []int
}

type Options struct {
	ResolveCallEffects func(*parser.Node) (CallEffects, bool)
}

type functionNodes struct {
	assignments []*parser.Node
	updates     []*parser.Node
	calls       []*parser.Node
}

func Build(tree *walk.Model, semantics *semantic.Model) *Model {
	return BuildWithOptions(tree, semantics, Options{})
}

func BuildWithOptions(tree *walk.Model, semantics *semantic.Model, options Options) *Model {
	model := &Model{byNode: make(map[*parser.Node]*Function), semantics: semantics, tree: tree, options: options}
	if tree == nil {
		return model
	}
	nodes := make(map[*parser.Node]*functionNodes)
	symbols := make(map[*parser.Node][]*semantic.Symbol)
	for _, function := range tree.OfKind(parser.KindFunctionDefinition) {
		nodes[function] = &functionNodes{}
	}
	for _, node := range tree.OfKind(parser.KindAssignmentExpression) {
		if group := nodes[tree.EnclosingFunction(node)]; group != nil {
			group.assignments = append(group.assignments, node)
		}
	}
	for _, node := range tree.OfKind(parser.KindUpdateExpression) {
		if group := nodes[tree.EnclosingFunction(node)]; group != nil {
			group.updates = append(group.updates, node)
		}
	}
	for _, node := range tree.OfKind(parser.KindCallExpression) {
		if group := nodes[tree.EnclosingFunction(node)]; group != nil {
			group.calls = append(group.calls, node)
		}
	}
	if semantics != nil {
		for _, symbol := range semantics.Symbols {
			if symbol != nil && symbol.Function != nil {
				symbols[symbol.Function] = append(symbols[symbol.Function], symbol)
			}
		}
	}
	for _, node := range tree.OfKind(parser.KindFunctionDefinition) {
		if tree.Inactive(node) {
			continue
		}
		builder := newBuilder(tree, semantics, node)
		function := builder.build()
		function.flowNodes = nodes[node]
		function.flowSymbols = symbols[node]
		model.Functions = append(model.Functions, function)
		model.byNode[node] = function
	}
	return model
}

func (m *Model) Eval(node *parser.Node) (int64, bool) {
	if m == nil || m.semantics == nil {
		return 0, false
	}
	if value, ok := m.semantics.Eval(node); ok {
		return value, true
	}
	function := m.byNode[m.tree.EnclosingFunction(node)]
	if function == nil {
		return m.semantics.Eval(node)
	}
	function.valueOnce.Do(func() {
		buildValueFlow(function, m.tree, m.semantics, m.options, function.flowNodes, function.flowSymbols)
	})
	block := function.Block(node)
	if block == nil || function.Uncertain || !function.ReachableBlock(block) {
		return m.semantics.Eval(node)
	}
	state := copyValueState(function.valueIn[block])
	applyValueEvents(state, function.valueEvents[block], node.Start, m.semantics)
	return m.semantics.EvalWithValues(node, state)
}

func (m *Model) Function(node *parser.Node) *Function {
	if m == nil {
		return nil
	}
	return m.byNode[node]
}

func (f *Function) Reachable(node *parser.Node) bool {
	if f == nil || f.Uncertain {
		return true
	}
	block := f.nodes[node]
	return block == nil || f.reachable[block]
}

func (f *Function) Block(node *parser.Node) *Block {
	if f == nil {
		return nil
	}
	return f.locations[node]
}

func (f *Function) ReachableBlock(block *Block) bool {
	if f == nil || f.Uncertain {
		return true
	}
	return block != nil && f.reachable[block]
}

func (f *Function) CanFallThrough() bool {
	if f == nil || f.Uncertain {
		return true
	}
	for _, block := range f.fallthroughExits {
		if f.reachable[block] {
			return true
		}
	}
	return false
}

type pendingGoto struct {
	from *Block
	name string
}

type targets struct {
	breakTo    *Block
	continueTo *Block
}

type builder struct {
	tree      *walk.Model
	semantics *semantic.Model
	function  *Function
	labels    map[string]*Block
	gotos     []pendingGoto
	nextID    int
}

func newBuilder(tree *walk.Model, semantics *semantic.Model, node *parser.Node) *builder {
	function := &Function{Node: node, nodes: make(map[*parser.Node]*Block), locations: make(map[*parser.Node]*Block), reachable: make(map[*Block]bool)}
	b := &builder{tree: tree, semantics: semantics, function: function, labels: make(map[string]*Block)}
	function.Entry = b.block(nil)
	function.Exit = b.block(nil)
	return b
}

func (b *builder) build() *Function {
	body := b.function.Node.Field("body")
	if body == nil || b.function.Node.HasError || b.tree.Uncertain(b.function.Node) {
		b.function.Uncertain = true
		return b.function
	}
	exits := b.buildNode(body, []*Block{b.function.Entry}, targets{})
	for _, exit := range exits {
		b.connect(exit, b.function.Exit, EdgeFallthrough)
		b.function.fallthroughExits = append(b.function.fallthroughExits, exit)
	}
	for _, jump := range b.gotos {
		target := b.labels[jump.name]
		if target == nil {
			b.function.Uncertain = true
			continue
		}
		b.connect(jump.from, target, EdgeJump)
	}
	b.markReachable()
	return b.function
}

func (b *builder) block(node *parser.Node) *Block {
	block := &Block{ID: b.nextID, Node: node}
	b.nextID++
	b.function.Blocks = append(b.function.Blocks, block)
	if node != nil {
		b.function.nodes[node] = block
		b.locate(node, block)
	}
	return block
}

func (b *builder) locate(node *parser.Node, block *Block) {
	if node == nil {
		return
	}
	b.function.locations[node] = block
	for _, child := range node.Children {
		b.locate(child, block)
	}
}

func (b *builder) connect(from, to *Block, kind EdgeKind) {
	if from == nil || to == nil {
		return
	}
	for _, edge := range from.Successors {
		if edge.To == to && edge.Kind == kind {
			return
		}
	}
	from.Successors = append(from.Successors, Edge{To: to, Kind: kind})
	to.Predecessors = append(to.Predecessors, from)
}

func (b *builder) connectAll(from []*Block, to *Block, kind EdgeKind) {
	for _, block := range from {
		b.connect(block, to, kind)
	}
}

func (b *builder) buildNode(node *parser.Node, incoming []*Block, target targets) []*Block {
	if node == nil || b.tree.Inactive(node) {
		return incoming
	}
	if node.HasError || b.tree.Uncertain(node) {
		b.function.Uncertain = true
	}
	switch node.Kind {
	case parser.KindConditionalRegion, parser.KindConditionalBranch:
		return b.buildSequence(node.Children, incoming, target)
	case parser.KindSharedConditional, parser.KindConditionalFunction, parser.KindConditionalSplice,
		parser.KindMacroInvocationBlock:
		b.function.Uncertain = true
		return incoming
	}
	if !walk.IsStatement(node) {
		return incoming
	}
	block := b.block(node)
	b.connectAll(incoming, block, EdgeNormal)
	switch node.Kind {
	case parser.KindBlock:
		return b.buildSequence(node.Children, []*Block{block}, target)
	case parser.KindIfStatement:
		consequence := b.buildNode(node.Field("consequence"), []*Block{block}, target)
		alternativeNode := node.Field("alternative")
		if alternativeNode == nil {
			return append(consequence, block)
		}
		alternative := b.buildNode(alternativeNode, []*Block{block}, target)
		return append(consequence, alternative...)
	case parser.KindWhileStatement, parser.KindForStatement:
		after := b.block(nil)
		condition, known := b.loopCondition(node)
		bodyIncoming := []*Block{block}
		if known && !condition {
			bodyIncoming = nil
		}
		bodyExits := b.buildNode(node.Field("body"), bodyIncoming, targets{breakTo: after, continueTo: block})
		b.connectAll(bodyExits, block, EdgeJump)
		if !known || !condition {
			b.connect(block, after, EdgeBranch)
		}
		return []*Block{after}
	case parser.KindDoWhileStatement:
		after := b.block(nil)
		conditionBlock := b.block(nil)
		bodyNode := node.Field("body")
		bodyExits := b.buildNode(bodyNode, []*Block{block}, targets{breakTo: after, continueTo: conditionBlock})
		b.connectAll(bodyExits, conditionBlock, EdgeJump)
		b.locate(node.Field("condition"), conditionBlock)
		condition, known := b.condition(node.Field("condition"))
		if !known || condition {
			b.connect(conditionBlock, b.function.nodes[bodyNode], EdgeBranch)
		}
		if !known || !condition {
			b.connect(conditionBlock, after, EdgeBranch)
		}
		return []*Block{after}
	case parser.KindSwitchStatement:
		return b.buildSwitch(node, block, target)
	case parser.KindCaseClause, parser.KindDefaultClause:
		return b.buildNode(node.Field("body"), []*Block{block}, target)
	case parser.KindReturnStatement:
		b.connect(block, b.function.Exit, EdgeReturn)
		return nil
	case parser.KindGotoStatement:
		label := node.Field("label")
		if label == nil {
			b.function.Uncertain = true
			return []*Block{block}
		}
		b.gotos = append(b.gotos, pendingGoto{from: block, name: b.tree.Text(label)})
		return nil
	case parser.KindLabelStatement:
		label := node.Field("label")
		name := b.tree.Text(label)
		if name == "" || b.labels[name] != nil {
			b.function.Uncertain = true
		} else {
			b.labels[name] = block
		}
		return []*Block{block}
	case parser.KindBreakStatement:
		if target.breakTo == nil {
			b.function.Uncertain = true
			return []*Block{block}
		}
		b.connect(block, target.breakTo, EdgeJump)
		return nil
	case parser.KindContinueStatement:
		if target.continueTo == nil {
			b.function.Uncertain = true
			return []*Block{block}
		}
		b.connect(block, target.continueTo, EdgeJump)
		return nil
	default:
		return []*Block{block}
	}
}

func (b *builder) buildSequence(nodes []*parser.Node, incoming []*Block, target targets) []*Block {
	exits := incoming
	for _, node := range nodes {
		exits = b.buildNode(node, exits, target)
	}
	return exits
}

func (b *builder) buildSwitch(node *parser.Node, block *Block, outer targets) []*Block {
	after := b.block(nil)
	hasDefault := false
	hasClause := false
	for _, child := range node.Children {
		if child.Kind != parser.KindCaseClause && child.Kind != parser.KindDefaultClause {
			continue
		}
		hasClause = true
		hasDefault = hasDefault || child.Kind == parser.KindDefaultClause
		exits := b.buildNode(child, []*Block{block}, targets{breakTo: after, continueTo: outer.continueTo})
		b.connectAll(exits, after, EdgeBranch)
	}
	if !hasClause || !hasDefault {
		b.connect(block, after, EdgeBranch)
	}
	return []*Block{after}
}

func (b *builder) loopCondition(node *parser.Node) (bool, bool) {
	condition := node.Field("condition")
	if node.Kind == parser.KindForStatement && condition == nil {
		return true, true
	}
	return b.condition(condition)
}

func (b *builder) condition(node *parser.Node) (bool, bool) {
	if b.semantics == nil {
		return false, false
	}
	value, ok := b.semantics.Eval(node)
	return value != 0, ok
}

func (b *builder) markReachable() {
	queue := []*Block{b.function.Entry}
	b.function.reachable[b.function.Entry] = true
	for len(queue) != 0 {
		block := queue[0]
		queue = queue[1:]
		for _, edge := range block.Successors {
			if b.function.reachable[edge.To] {
				continue
			}
			b.function.reachable[edge.To] = true
			queue = append(queue, edge.To)
		}
	}
}
