package project

import (
	"bytes"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/preprocess"
	"github.com/pawnkit/pawnlint/internal/semantic"
	sourceinfo "github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/cst"
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
	ReleaseExpanded bool
	ReleaseIncludes bool
	Features        *Features
	ParseCache      *ParseCache
	ObserveTiming   func(TimingEvent)
}

type TimingStage string

const (
	TimingParse      TimingStage = "parse"
	TimingPreprocess TimingStage = "preprocess"
	TimingSemantic   TimingStage = "semantic"
)

type TimingEvent struct {
	Stage    TimingStage
	Duration time.Duration
}

type File struct {
	Path              string
	Source            []byte
	Parsed            *parser.File
	CompactParsed     *parser.CompactFile
	Walk              *walk.Model
	CompactWalk       *walk.CompactModel
	Syntax            *cst.Model
	Semantic          *semantic.Model
	CompactSemantic   *semantic.CompactModel
	ExpandedSource    []byte
	ExpandedParsed    *parser.File
	ExpandedWalk      *walk.Model
	ExpandedSemantic  *semantic.Model
	ExpansionComplete bool
	Includes          []*Include
	Provided          bool
	canonical         string
	defines           *defineEnvironment
	final             *defineEnvironment
	resolving         bool
	resolved          bool
	complete          bool
	sourceID          uint32
	syntaxIndex       *walk.Index
	expansionState    *preprocess.State
	runtimeCalls      []runtimeCallFact
	expansionOrigins  map[*parser.Node][]expansionOriginFact
	snapshots         []walk.DefineSnapshot
	pointerOnce       sync.Once
	pointerNodes      map[nodeLocation]*parser.Node
	compactNodeMu     sync.Mutex
	compactNodes      map[parser.Kind]map[nodeLocation]cst.Node
}

type Include struct {
	Node       *parser.Node
	Path       string
	Resolved   *File
	Candidates []string
	Optional   bool
	Uncertain  bool
	syntax     cst.Node
}

type Unit struct {
	Root    *File
	Files   []*File
	members map[*File]struct{}
}

type Declaration struct {
	Name          string
	Kind          semantic.SymbolKind
	File          *File
	Node          *parser.Node
	Symbol        *semantic.Symbol
	syntax        cst.Node
	compactSymbol *semantic.CompactSymbol
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
	File   *File
	Node   *parser.Node
	Kind   semantic.ReferenceKind
	syntax cst.Node
}

type Model struct {
	Files              []*File
	Units              []*Unit
	Declarations       map[string][]Declaration
	CallGraph          *CallGraph
	includeCycles      []IncludeCycle
	duplicateFunctions []DuplicateFunction
	duplicateGlobals   []DuplicateGlobal
	includeDirectives  []IncludeIssue
	missingIncludes    []IncludeIssue
	ambiguousIncludes  []IncludeIssue
	duplicateIncludes  []IncludeIssue
	unusedIncludes     []IncludeIssue
	symbolConflicts    []SymbolConflict
	byCanonical        map[string]*File
	byContext          map[fileContextKey]*File
	defineEnvironments map[uint64][]*defineEnvironment
	nextEnvironmentID  uint32
	physical           map[string]*physicalFile
	references         map[declarationID][]Reference
	resolved           map[*File]map[cst.Node]Declaration
	ambiguous          map[*File]map[cst.Node]bool
	effects            map[declarationID]FunctionEffects
	functionVariantMap map[functionVariantKey][]Declaration
	functionVariantsMu sync.RWMutex
	definedNames       map[string]struct{}
	sourceFiles        map[uint32]*File
	options            Options
}

type physicalFile struct {
	source      []byte
	parsed      *parser.File
	compact     *parser.CompactFile
	lineTable   *sourceinfo.LineTable
	syntaxIndex *walk.Index
}

type nodeLocation struct {
	kind       parser.Kind
	start, end int
}

type fileContextKey struct {
	canonical   string
	environment uint32
}

type defineEnvironment struct {
	id    uint32
	order uint32
	names []string
	walk  *walk.DefineContext
}

func Build(sources []Source, options Options) (*Model, error) {
	options.IncludePaths = append([]string(nil), options.IncludePaths...)
	options.Defines = normalizeDefines(options.Defines)
	features := AllFeatures()
	if options.Features != nil {
		features = options.Features.withDependencies()
		options.Features = &features
	}
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
		Declarations:       make(map[string][]Declaration),
		byCanonical:        make(map[string]*File),
		byContext:          make(map[fileContextKey]*File),
		defineEnvironments: make(map[uint64][]*defineEnvironment),
		physical:           make(map[string]*physicalFile),
		references:         make(map[declarationID][]Reference),
		resolved:           make(map[*File]map[cst.Node]Declaration),
		ambiguous:          make(map[*File]map[cst.Node]bool),
		functionVariantMap: make(map[functionVariantKey][]Declaration),
		definedNames:       make(map[string]struct{}),
		sourceFiles:        make(map[uint32]*File),
		options:            options,
	}
	rootEnvironment := model.internDefines(options.Defines)
	for _, source := range sources {
		file, err := model.addFile(source.Path, source.Content, true, rootEnvironment)
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
	for _, file := range model.Files {
		file.expansionState = nil
	}
	model.orderDefineEnvironments()
	if features.Has(FeatureDefinedNames) {
		model.buildDefinedNames()
	}
	sort.SliceStable(model.Files, func(i, j int) bool {
		if model.Files[i].canonical != model.Files[j].canonical {
			return model.Files[i].canonical < model.Files[j].canonical
		}
		return model.Files[i].defines.order < model.Files[j].defines.order
	})
	needsDeclarations := features.Has(FeatureReferences) || features.Has(FeatureDuplicates) || features.Has(FeatureConflicts)
	needsUnits := needsDeclarations || features.Has(FeatureCallGraph)
	if needsDeclarations {
		model.buildDeclarations()
	}
	if needsUnits {
		model.buildUnits()
	}
	if features.Has(FeatureDuplicates) {
		model.duplicateFunctions = model.buildDuplicateFunctions()
		model.duplicateGlobals = model.buildDuplicateGlobals()
	}
	if features.Has(FeatureConflicts) {
		model.symbolConflicts = model.buildConflictingIncludeSymbols()
	}
	if features.Has(FeatureIncludeCycles) {
		model.includeCycles = model.buildIncludeCycles()
	}
	if features.Has(FeatureIncludeIssues) {
		model.buildIncludeIssues()
	}
	if features.Has(FeatureReferences) {
		model.buildReferences()
	}
	if features.Has(FeatureUnusedIncludes) {
		model.unusedIncludes = model.buildUnusedIncludes()
	}
	if features.Has(FeatureCallGraph) {
		model.CallGraph = model.buildCallGraph()
	}
	if features.Has(FeatureFunctionEffects) {
		model.buildFunctionEffects()
	}
	if options.ReleaseIncludes {
		model.ReleaseIncludeTokens(nil)
	}
	return model, nil
}

func (m *Model) ReleaseIncludeTokens(files []*File) {
	retained := make(map[*parser.File]struct{})
	for _, file := range m.Files {
		if file.Provided && file.Parsed != nil {
			retained[file.Parsed] = struct{}{}
		}
	}
	for _, file := range files {
		if file != nil && file.Parsed != nil {
			retained[file.Parsed] = struct{}{}
		}
	}
	for _, physical := range m.physical {
		if physical == nil {
			continue
		}
		if physical.compact != nil {
			physical.compact.Tokens = nil
			physical.compact.Origins = nil
			physical.compact.MacroNames = nil
			if !bytes.Contains(physical.source, []byte("pawnlint-")) {
				physical.compact.Trivia = nil
			}
			continue
		}
		if physical.parsed == nil {
			continue
		}
		if bytes.Contains(physical.source, []byte("pawnlint-")) {
			continue
		}
		if _, keep := retained[physical.parsed]; !keep {
			physical.parsed.Tokens = nil
		}
	}
}

func (m *Model) DefinesName(name string) bool {
	if m == nil {
		return false
	}
	_, ok := m.definedNames[name]
	return ok
}

func (m *Model) buildDefinedNames() {
	for _, file := range m.Files {
		for _, node := range file.Syntax.OfKind(parser.KindDirectiveDefine) {
			if file.Syntax.Inactive(node) {
				continue
			}
			name := node.Field("name").Text()
			if name != "" {
				m.definedNames[name] = struct{}{}
			}
		}
	}
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
