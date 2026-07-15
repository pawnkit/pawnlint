// Package walk provides traversal and query helpers over the pawn-parser CST,
// including a cached index of nodes by Kind.
package walk

import (
	"sort"

	parser "github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/source"
)

type Model struct {
	File       *parser.File
	Path       string
	LineTable  *source.LineTable
	parents    map[*parser.Node]*parser.Node
	byKind     map[parser.Kind][]*parser.Node
	directives []*parser.Node
	branches   map[*parser.Node]branchState
	uncertain  map[*parser.Node]bool
	inactive   map[*parser.Node]bool
	defines    []string
	snapshots  []DefineSnapshot
	complete   bool
}

type DefineSnapshot struct {
	Offset  int
	Defines []string
}

type branchState uint8

var compilerDefines = []string{
	"true", "false", "EOS", "cellbits", "cellmax", "cellmin", "charbits",
	"charmin", "charmax", "ucharmax", "__Pawn", "__PawnBuild", "__line",
	"__compat", "debug",
}

const (
	branchActive branchState = iota
	branchInactive
	branchUncertain
)

func New(path string, pf *parser.File) *Model {
	return NewWithDefines(path, pf, nil)
}

func NewWithDefines(path string, pf *parser.File, defines []string) *Model {
	return NewWithDefineSnapshots(path, pf, defines, nil)
}

func NewWithDefineSnapshots(path string, pf *parser.File, defines []string, snapshots []DefineSnapshot) *Model {
	return NewWithDefineContext(path, pf, defines, snapshots, false)
}

func NewWithDefineContext(path string, pf *parser.File, defines []string, snapshots []DefineSnapshot, complete bool) *Model {
	var src []byte
	if pf != nil {
		src = pf.Source
	}
	m := &Model{
		File:      pf,
		Path:      path,
		LineTable: source.NewLineTable(src),
		parents:   make(map[*parser.Node]*parser.Node),
		byKind:    make(map[parser.Kind][]*parser.Node),
		branches:  make(map[*parser.Node]branchState),
		uncertain: make(map[*parser.Node]bool),
		inactive:  make(map[*parser.Node]bool),
		defines:   append(append([]string(nil), compilerDefines...), defines...),
		snapshots: cloneDefineSnapshots(snapshots),
		complete:  complete,
	}
	if pf != nil && pf.Root != nil {
		m.index(pf.Root, nil)
		m.indexConditionalStates()
		m.indexNodeStates()
	}
	return m
}

func cloneDefineSnapshots(snapshots []DefineSnapshot) []DefineSnapshot {
	cloned := make([]DefineSnapshot, len(snapshots))
	for i, snapshot := range snapshots {
		cloned[i] = DefineSnapshot{Offset: snapshot.Offset, Defines: append([]string(nil), snapshot.Defines...)}
	}
	sort.SliceStable(cloned, func(i, j int) bool { return cloned[i].Offset < cloned[j].Offset })
	return cloned
}

func (m *Model) index(n, parent *parser.Node) {
	if n == nil {
		return
	}
	m.parents[n] = parent
	m.byKind[n.Kind] = append(m.byKind[n.Kind], n)
	if n.Kind == parser.KindDirectiveDefine || n.Kind == parser.KindDirectiveUndef {
		m.directives = append(m.directives, n)
	}
	for _, c := range n.Children {
		m.index(c, n)
	}
}
