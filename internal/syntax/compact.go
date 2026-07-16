package syntax

import (
	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
)

type NodeID uint32

const NoNode = ^NodeID(0)

type CompactTree struct {
	file    *parser.CompactFile
	parents []NodeID
	byKind  map[parser.Kind][]NodeID
}

func NewCompactTree(file *parser.CompactFile) *CompactTree {
	tree := &CompactTree{file: file, byKind: make(map[parser.Kind][]NodeID)}
	if file == nil {
		return tree
	}
	tree.parents = make([]NodeID, len(file.Tree.Nodes))
	for index := range tree.parents {
		tree.parents[index] = NoNode
	}
	for index, node := range file.Tree.Nodes {
		id := NodeID(index)
		tree.byKind[node.Kind] = append(tree.byKind[node.Kind], id)
		for _, child := range file.Tree.ChildIndices(uint32(index)) {
			tree.parents[child] = id
		}
	}
	return tree
}

func (t *CompactTree) File() *parser.CompactFile {
	if t == nil {
		return nil
	}
	return t.file
}

func (t *CompactTree) Root() NodeID {
	if t == nil || t.file == nil || len(t.file.Tree.Nodes) == 0 {
		return NoNode
	}
	return NodeID(t.file.Tree.Root)
}

func (t *CompactTree) Len() int {
	if t == nil || t.file == nil {
		return 0
	}
	return len(t.file.Tree.Nodes)
}

func (t *CompactTree) Source() []byte {
	if t == nil || t.file == nil {
		return nil
	}
	return t.file.Source
}

func (t *CompactTree) Valid(node NodeID) bool {
	return t != nil && t.file != nil && uint32(node) < uint32(len(t.file.Tree.Nodes))
}

func (t *CompactTree) Kind(node NodeID) parser.Kind {
	if !t.Valid(node) {
		return parser.KindInvalid
	}
	return t.file.Tree.Nodes[node].Kind
}

func (t *CompactTree) Start(node NodeID) int {
	if !t.Valid(node) {
		return 0
	}
	return int(t.file.Tree.Nodes[node].Start)
}

func (t *CompactTree) End(node NodeID) int {
	if !t.Valid(node) {
		return 0
	}
	return int(t.file.Tree.Nodes[node].End)
}

func (t *CompactTree) HasError(node NodeID) bool {
	return t.Valid(node) && t.file.Tree.Nodes[node].HasError
}

func (t *CompactTree) MissingSemi(node NodeID) bool {
	return t.Valid(node) && t.file.Tree.Nodes[node].MissingSemi
}

func (t *CompactTree) TokenKind(node NodeID) token.Kind {
	if !t.Valid(node) {
		return token.Invalid
	}
	return t.file.Tree.Nodes[node].TokenKind
}

func (t *CompactTree) TokenStart(node NodeID) int {
	if !t.Valid(node) {
		return 0
	}
	return int(t.file.Tree.Nodes[node].TokenStart)
}

func (t *CompactTree) TokenEnd(node NodeID) int {
	if !t.Valid(node) {
		return 0
	}
	return int(t.file.Tree.Nodes[node].TokenEnd)
}

func (t *CompactTree) Parent(node NodeID) NodeID {
	if !t.Valid(node) {
		return NoNode
	}
	return t.parents[node]
}

func (t *CompactTree) ChildCount(node NodeID) int {
	if !t.Valid(node) {
		return 0
	}
	return int(t.file.Tree.Nodes[node].ChildCount)
}

func (t *CompactTree) Child(node NodeID, index int) NodeID {
	if !t.Valid(node) || index < 0 || index >= t.ChildCount(node) {
		return NoNode
	}
	record := t.file.Tree.Nodes[node]
	return NodeID(t.file.Tree.Children[record.ChildStart+uint32(index)])
}

func (t *CompactTree) Field(node NodeID, name string) NodeID {
	if !t.Valid(node) {
		return NoNode
	}
	field, ok := t.file.Tree.Field(uint32(node), name)
	if !ok {
		return NoNode
	}
	return NodeID(field)
}

func (t *CompactTree) Text(node NodeID) string {
	if !t.Valid(node) {
		return ""
	}
	return t.file.Tree.Nodes[node].Text(t.file.Source)
}

func (t *CompactTree) TokenText(node NodeID) string {
	if !t.Valid(node) {
		return ""
	}
	record := t.file.Tree.Nodes[node]
	if record.TokenEnd > uint32(len(t.file.Source)) || record.TokenStart > record.TokenEnd {
		return ""
	}
	return string(t.file.Source[record.TokenStart:record.TokenEnd])
}

func (t *CompactTree) OfKind(kind parser.Kind) []NodeID {
	if t == nil {
		return nil
	}
	return t.byKind[kind]
}
