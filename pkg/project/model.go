package project

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

type Source struct {
	Path    string
	Content []byte
}

type Options struct {
	WorkingDir      string
	IncludePaths    []string
	Defines         []string
	DefinesComplete bool
}

type File struct {
	Path      string
	Source    []byte
	Parsed    *parser.File
	Walk      *walk.Model
	Semantic  *semantic.Model
	Includes  []*Include
	Provided  bool
	canonical string
	instance  string
	defines   []string
	final     []string
	resolving bool
	resolved  bool
	complete  bool
}

type Include struct {
	Node       *parser.Node
	Path       string
	Resolved   *File
	Candidates []string
	Optional   bool
	Uncertain  bool
}

type Unit struct {
	Root    *File
	Files   []*File
	members map[*File]struct{}
}

type Declaration struct {
	Name   string
	Kind   semantic.SymbolKind
	File   *File
	Node   *parser.Node
	Symbol *semantic.Symbol
}

type DuplicateFunction struct {
	Name   string
	First  Declaration
	Second Declaration
	Owner  *File
}

type DuplicateGlobal struct {
	Name   string
	First  Declaration
	Second Declaration
	Owner  *File
}

type Reference struct {
	File *File
	Node *parser.Node
	Kind semantic.ReferenceKind
}

type Model struct {
	Files             []*File
	Units             []*Unit
	Declarations      map[string][]Declaration
	CallGraph         *CallGraph
	includeCycles     []IncludeCycle
	missingIncludes   []IncludeIssue
	ambiguousIncludes []IncludeIssue
	byCanonical       map[string]*File
	byContext         map[string]*File
	physical          map[string]*physicalFile
	references        map[string][]Reference
	resolved          map[*File]map[*parser.Node]Declaration
	ambiguous         map[*File]map[*parser.Node]bool
	options           Options
}

type physicalFile struct {
	source []byte
	parsed *parser.File
}

func Build(sources []Source, options Options) (*Model, error) {
	options.IncludePaths = append([]string(nil), options.IncludePaths...)
	options.Defines = append([]string(nil), options.Defines...)
	if options.WorkingDir == "" {
		options.WorkingDir = "."
	}
	workingDir, err := filepath.Abs(options.WorkingDir)
	if err != nil {
		return nil, err
	}
	options.WorkingDir = filepath.Clean(workingDir)
	for i, path := range options.IncludePaths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(options.WorkingDir, path)
		}
		options.IncludePaths[i] = filepath.Clean(path)
	}
	model := &Model{
		Declarations: make(map[string][]Declaration),
		byCanonical:  make(map[string]*File),
		byContext:    make(map[string]*File),
		physical:     make(map[string]*physicalFile),
		references:   make(map[string][]Reference),
		resolved:     make(map[*File]map[*parser.Node]Declaration),
		ambiguous:    make(map[*File]map[*parser.Node]bool),
		options:      options,
	}
	for _, source := range sources {
		file, err := model.addFile(source.Path, source.Content, true, options.Defines)
		if err != nil {
			return nil, err
		}
		if file.Path == "" {
			file.Path = source.Path
		}
	}
	for _, file := range append([]*File(nil), model.Files...) {
		if err := model.resolveFileIncludes(file); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(model.Files, func(i, j int) bool {
		if model.Files[i].canonical != model.Files[j].canonical {
			return model.Files[i].canonical < model.Files[j].canonical
		}
		return model.Files[i].instance < model.Files[j].instance
	})
	model.buildDeclarations()
	model.buildUnits()
	model.includeCycles = model.buildIncludeCycles()
	model.buildIncludeIssues()
	model.buildReferences()
	model.CallGraph = model.buildCallGraph()
	return model, nil
}

func (m *Model) File(path string) *File {
	if m == nil {
		return nil
	}
	canonical, err := canonicalPath(path, m.options.WorkingDir)
	if err != nil {
		return nil
	}
	return m.byCanonical[canonical]
}

func (m *Model) InProgram(file *File) bool {
	if m == nil || file == nil {
		return false
	}
	for _, unit := range m.Units {
		if !strings.EqualFold(filepath.Ext(unit.Root.Path), ".pwn") {
			continue
		}
		for _, included := range unit.Files {
			if included == file {
				return true
			}
		}
	}
	return false
}

func canonicalPath(path, workingDir string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(workingDir, path)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}
