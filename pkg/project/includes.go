package project

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/preprocess"
	"github.com/pawnkit/pawnlint/internal/semantic"
	sourceinfo "github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"github.com/pawnkit/pawnlint/internal/syntax"
)

const compactTargetThreshold = 512 << 10

func (m *Model) addFile(path string, source []byte, provided bool, defines *defineEnvironment, includeRoot string) (*File, error) {
	if err := m.ctx.Err(); err != nil {
		return nil, err
	}
	canonical, err := canonicalPath(path, m.options.WorkingDir)
	if err != nil {
		return nil, err
	}
	if includeRoot == "" {
		includeRoot = filepath.Dir(canonical)
	}
	instance := fileContextKey{canonical: canonical, environment: defines.id, includeRoot: includeRoot}
	if existing := m.byContext[instance]; existing != nil {
		existing.Provided = existing.Provided || provided
		if provided {
			m.byCanonical[canonical] = existing
		}
		return existing, nil
	}
	physical := m.physical[canonical]
	if physical == nil {
		sourceHash := sha256.Sum256(source)
		retainTrivia := m.options.Features == nil || m.options.Features.Has(FeatureTrivia) || bytes.Contains(source, []byte("pawnlint-"))
		if provided && m.options.RootParsed != nil && bytes.Equal(m.options.RootParsed.Source, source) {
			parsed := m.options.RootParsed
			syntaxIndex := m.options.ParseCache.getIndex(canonical, sourceHash)
			if syntaxIndex == nil {
				syntaxIndex = walk.NewIndex(parsed)
				m.options.ParseCache.putIndex(canonical, sourceHash, syntaxIndex)
			}
			physical = &physicalFile{source: source, hash: sourceHash, parsed: parsed, lineTable: sourceinfo.NewLineTable(source), syntaxIndex: syntaxIndex}
			m.physical[canonical] = physical
		} else if m.options.ReleaseIncludes && (!provided || len(source) >= compactTargetThreshold) {
			started := time.Now()
			var compact *parser.CompactFile
			switch {
			case retainTrivia:
				compact = parser.ParseWithProfile(source, parser.ProfileLossless)
			case m.options.Features.Has(FeatureRuntimeCalls):
				compact = parser.ParseCompact(source, parser.ParseOptions{DiscardTrivia: true})
			default:
				compact = parser.ParseWithProfile(source, parser.ProfileAnalysis)
			}
			if m.options.ObserveTiming != nil {
				m.observe(TimingEvent{Stage: TimingParse, Duration: time.Since(started)})
			}
			physical = &physicalFile{
				source: source, hash: sourceHash, compact: compact,
				compactTree: syntax.NewCompactTree(compact),
				lineTable:   sourceinfo.NewLineTable(source),
			}
			m.physical[canonical] = physical
		} else {
			discardTrivia := !retainTrivia
			var rootTokens []token.Token
			if provided && !m.rootTokensUsed && m.options.RootTokens != nil {
				rootTokens = m.options.RootTokens
				m.rootTokensUsed = true
			}
			var parsed *parser.File
			if m.options.ParseCache != nil {
				started := time.Now()
				var cached bool
				parsed, cached = m.options.ParseCache.parseHashed(canonical, source, sourceHash, discardTrivia, rootTokens)
				if !cached && m.options.ObserveTiming != nil {
					m.observe(TimingEvent{Stage: TimingParse, Duration: time.Since(started)})
				}
			} else if m.options.ObserveTiming == nil {
				parsed = parseSource(source, discardTrivia, rootTokens)
			} else {
				started := time.Now()
				parsed = parseSource(source, discardTrivia, rootTokens)
				m.observe(TimingEvent{Stage: TimingParse, Duration: time.Since(started)})
			}
			syntaxIndex := m.options.ParseCache.getIndex(canonical, sourceHash)
			if syntaxIndex == nil {
				syntaxIndex = walk.NewIndex(parsed)
				m.options.ParseCache.putIndex(canonical, sourceHash, syntaxIndex)
			}
			physical = &physicalFile{source: source, hash: sourceHash, parsed: parsed, lineTable: sourceinfo.NewLineTable(source), syntaxIndex: syntaxIndex}
			m.physical[canonical] = physical
		}
	}
	parsed := physical.parsed
	compact := physical.compact
	if parsed == nil && compact == nil {
		return nil, fmt.Errorf("project: parse %s", path)
	}
	display := path
	if !provided {
		display = canonical
	}
	file := &File{Path: display, Source: physical.source, Parsed: parsed, CompactParsed: compact, Provided: provided, canonical: canonical, includeRoot: includeRoot, defines: defines, complete: m.options.DefinesComplete, sourceID: uint32(len(m.Files) + 1), sourceHash: physical.hash, syntaxIndex: physical.syntaxIndex}
	if parsed != nil {
		walkModel := m.options.ParseCache.getWalk(canonical, physical.hash, defines.definesKey, m.options.DefinesComplete, [sha256.Size]byte{})
		if walkModel == nil {
			walkModel = walk.NewWithContext(display, parsed, defines.walk, nil, m.options.DefinesComplete, physical.lineTable, physical.syntaxIndex)
			m.options.ParseCache.putWalk(canonical, physical.hash, defines.definesKey, m.options.DefinesComplete, [sha256.Size]byte{}, walkModel)
		}
		file.Walk = walkModel
		file.Syntax = cst.Pointer(file.Walk)
	} else {
		file.CompactWalk = walk.NewCompactWithTreeContext(display, physical.compactTree, defines.walk, nil, m.options.DefinesComplete, physical.lineTable)
		file.Syntax = cst.Compact(file.CompactWalk)
	}
	m.Files = append(m.Files, file)
	m.sourceFiles[file.sourceID] = file
	m.byContext[instance] = file
	if m.byCanonical[canonical] == nil || provided {
		m.byCanonical[canonical] = file
	}
	return file, nil
}

func parseSource(source []byte, discardTrivia bool, tokens []token.Token) *parser.File {
	if tokens != nil {
		return parser.ParseTokensWithOptions(source, tokens, parser.ParseOptions{DiscardTrivia: discardTrivia})
	}
	return parser.ParseWithOptions(source, parser.ParseOptions{DiscardTrivia: discardTrivia})
}

func (m *Model) resolveFileIncludes(file *File) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	if file == nil || file.resolved || file.resolving {
		return nil
	}
	file.resolving = true
	defer func() { file.resolving = false }()
	var nodes []cst.Node
	for _, kind := range []parser.Kind{parser.KindDirectiveInclude, parser.KindDirectiveTryInclude} {
		nodes = append(nodes, file.Syntax.OfKind(kind)...)
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].Start() < nodes[j].Start() })
	var snapshots []walk.DefineSnapshot
	var snapshotIdentities []defineSnapshotIdentity
	dirty := false
	defineCursor := file.Syntax.NewDefineCursor()
	for _, node := range nodes {
		if err := m.ctx.Err(); err != nil {
			return err
		}
		if dirty {
			file.rebuildWalk(snapshots, defineSnapshotsCacheKey(snapshotIdentities), m.options.ParseCache)
			defineCursor = file.Syntax.NewDefineCursor()
			dirty = false
		}
		if file.Syntax.Inactive(node) {
			continue
		}
		rawPath := strings.TrimSpace(node.Field("path").Text())
		path := includePath(rawPath)
		include := &Include{Node: node.Pointer(), Path: path, Optional: node.Kind() == parser.KindDirectiveTryInclude, Uncertain: file.Syntax.Uncertain(node), syntax: node}
		file.Includes = append(file.Includes, include)
		if path == "" || include.Uncertain {
			continue
		}
		defines := m.internDefines(defineCursor.KnownDefinesViewAt(node.Start()))
		resolved, candidates, err := m.resolveInclude(file, path, strings.HasPrefix(rawPath, `"`), defines)
		if err != nil {
			return err
		}
		include.Resolved = resolved
		include.Candidates = candidates
		if resolved == nil {
			continue
		}
		if err := m.resolveFileIncludes(resolved); err != nil {
			return err
		}
		if resolved.final != nil && len(resolved.final.names) > 0 && defines != resolved.final {
			snapshots = append(snapshots, walk.DefineSnapshot{Offset: node.End(), Defines: resolved.final.names})
			snapshotIdentities = append(snapshotIdentities, defineSnapshotIdentity{offset: node.End(), hash: resolved.final.cacheHash})
			dirty = true
		}
	}
	if dirty {
		file.rebuildWalk(snapshots, defineSnapshotsCacheKey(snapshotIdentities), m.options.ParseCache)
		defineCursor = file.Syntax.NewDefineCursor()
	}
	if err := m.ctx.Err(); err != nil {
		return err
	}
	file.final = m.internDefines(defineCursor.KnownDefinesViewAt(len(file.Source) + 1))
	if file.Parsed != nil {
		if cached := m.options.ParseCache.getSemantic(file.canonical, file.sourceHash, file.defines.definesKey, file.complete, file.snapshotsKey); cached != nil {
			file.Semantic = cached
		} else {
			started := time.Now()
			file.Semantic = semantic.Build(file.Parsed, file.Walk)
			if m.options.ObserveTiming != nil {
				m.observe(TimingEvent{Stage: TimingSemantic, Duration: time.Since(started)})
			}
			m.options.ParseCache.putSemantic(file.canonical, file.sourceHash, file.defines.definesKey, file.complete, file.snapshotsKey, file.Semantic)
		}
	} else if m.options.ObserveTiming == nil {
		file.CompactSemantic = semantic.BuildCompact(file.CompactParsed, file.CompactWalk)
	} else {
		started := time.Now()
		file.CompactSemantic = semantic.BuildCompact(file.CompactParsed, file.CompactWalk)
		m.observe(TimingEvent{Stage: TimingSemantic, Duration: time.Since(started)})
	}
	if m.options.Features != nil && !m.options.Features.Has(FeatureRuntimeCalls) {
		file.resolved = true
		return nil
	}
	started := time.Now()
	imports := make(map[int]*preprocess.State)
	for _, include := range file.Includes {
		if include.Resolved != nil && !include.Uncertain && include.Resolved.expansionState != nil {
			imports[include.Start()] = include.Resolved.expansionState
		}
	}
	if file.CompactParsed != nil {
		expanded, expansionState := preprocess.ExpandCompactSyntaxWithState(file.CompactParsed, file.CompactWalk, file.sourceID, nil, imports)
		file.expansionState = expansionState
		file.ExpansionComplete = expanded.Complete
		for _, include := range file.Includes {
			if include.Uncertain || include.Resolved != nil && !include.Resolved.ExpansionComplete {
				file.ExpansionComplete = false
			}
		}
		parsed := expanded.Parsed
		if parsed == nil {
			parsed = file.CompactParsed
		}
		tree := file.CompactWalk
		if expanded.Changed {
			tree = walk.NewCompactWithDefineContext(file.Path, parsed, file.defines.names, nil, file.complete)
		}
		m.captureCompactRuntimeCalls(file, parsed, tree)
		if m.options.ObserveTiming != nil {
			m.observe(TimingEvent{Stage: TimingPreprocess, Duration: time.Since(started)})
		}
		file.resolved = true
		return nil
	}
	if m.options.ReleaseExpanded {
		expanded, expansionState := preprocess.ExpandCompactWithState(file.Parsed, file.Walk, file.sourceID, nil, imports)
		file.expansionState = expansionState
		file.ExpansionComplete = expanded.Complete
		for _, include := range file.Includes {
			if include.Uncertain || include.Resolved != nil && !include.Resolved.ExpansionComplete {
				file.ExpansionComplete = false
			}
		}
		if !expanded.Changed {
			file.ExpandedSource = file.Source
			file.ExpandedParsed = file.Parsed
			file.ExpandedWalk = file.Walk
			file.ExpandedSemantic = file.Semantic
			m.captureRuntimeCalls(file)
		} else {
			tree := walk.NewCompactWithDefineContext(file.Path, expanded.Parsed, file.defines.names, nil, file.complete)
			m.captureCompactRuntimeCalls(file, expanded.Parsed, tree)
		}
		file.ExpandedSource = nil
		file.ExpandedParsed = nil
		file.ExpandedWalk = nil
		file.ExpandedSemantic = nil
		if m.options.ObserveTiming != nil {
			m.observe(TimingEvent{Stage: TimingPreprocess, Duration: time.Since(started)})
		}
		file.resolved = true
		return nil
	}
	expanded, expansionState := preprocess.ExpandWithState(file.Parsed, file.Walk, file.sourceID, nil, imports)
	file.expansionState = expansionState
	file.ExpansionComplete = expanded.Complete
	for _, include := range file.Includes {
		if include.Uncertain || include.Resolved != nil && !include.Resolved.ExpansionComplete {
			file.ExpansionComplete = false
		}
	}
	if !expanded.Changed {
		file.ExpandedSource = file.Source
		file.ExpandedParsed = file.Parsed
		file.ExpandedWalk = file.Walk
		file.ExpandedSemantic = file.Semantic
	} else {
		file.ExpandedSource = expanded.Source
		file.ExpandedParsed = expanded.Parsed
		file.ExpandedWalk = walk.NewWithContext(file.Path, expanded.Parsed, file.defines.walk, nil, file.complete, nil, nil)
		if !m.options.ReleaseExpanded {
			file.ExpandedSemantic = semantic.Build(expanded.Parsed, file.ExpandedWalk)
		}
	}
	m.captureRuntimeCalls(file)
	if m.options.ObserveTiming != nil {
		m.observe(TimingEvent{Stage: TimingPreprocess, Duration: time.Since(started)})
	}
	file.resolved = true
	return nil
}

func (m *Model) observe(event TimingEvent) {
	if m.options.ObserveTiming != nil {
		m.options.ObserveTiming(event)
	}
}

func (m *Model) resolveInclude(from *File, path string, quoted bool, defines *defineEnvironment) (*File, []string, error) {
	resolvedPaths := m.includeResolver.ResolveAll(from.canonical, path, quoted, from.includeRoot)
	candidates := make([]string, 0, len(resolvedPaths))
	for _, resolvedPath := range resolvedPaths {
		canonical, err := canonicalPath(filepath.FromSlash(resolvedPath), m.options.WorkingDir)
		if err == nil {
			candidates = append(candidates, canonical)
		}
	}
	if len(candidates) == 0 {
		return nil, nil, nil
	}
	chosen := candidates[0]
	if existing := m.byContext[fileContextKey{canonical: chosen, environment: defines.id, includeRoot: from.includeRoot}]; existing != nil {
		return existing, candidates, nil
	}
	var source []byte
	if physical := m.physical[chosen]; physical != nil {
		source = physical.source
	} else {
		var err error
		source, err = m.includeResolver.ReadFile(chosen)
		if err != nil {
			return nil, nil, err
		}
	}
	resolved, err := m.addFile(chosen, source, false, defines, from.includeRoot)
	return resolved, candidates, err
}

func (f *File) rebuildWalk(snapshots []walk.DefineSnapshot, snapshotsKey [sha256.Size]byte, cache *ParseCache) {
	f.snapshots = append(f.snapshots[:0], snapshots...)
	f.snapshotsKey = snapshotsKey
	if f.Parsed != nil {
		lineTable := f.Walk.LineTable
		cached := cache.getWalk(f.canonical, f.sourceHash, f.defines.definesKey, f.complete, f.snapshotsKey)
		if cached == nil {
			cached = walk.NewWithSharedContext(f.Path, f.Parsed, f.defines.walk, f.snapshots, f.complete, lineTable, f.syntaxIndex)
			cache.putWalk(f.canonical, f.sourceHash, f.defines.definesKey, f.complete, f.snapshotsKey, cached)
		}
		f.Walk = cached
		f.Syntax = cst.Pointer(f.Walk)
	} else {
		f.CompactWalk = walk.NewCompactWithTreeContext(f.Path, f.CompactWalk.Tree, f.defines.walk, f.snapshots, f.complete, f.CompactWalk.LineTable)
		f.Syntax = cst.Compact(f.CompactWalk)
	}
}

func (m *Model) internDefines(defines []string) *defineEnvironment {
	if m.lastEnvironment != nil && sameDefines(m.lastEnvironment.names, defines) {
		return m.lastEnvironment
	}
	hash := defineEnvironmentHash(defines)
	for _, environment := range m.defineEnvironments[hash] {
		if sameDefines(environment.names, defines) {
			m.lastEnvironment = environment
			return environment
		}
	}
	m.nextEnvironmentID++
	names := append([]string(nil), defines...)
	definesKey := definesCacheKey(names)
	environment := &defineEnvironment{
		id: m.nextEnvironmentID, names: names, definesKey: definesKey, walk: m.options.ParseCache.defineContext(names, definesKey),
		cacheHash: definesKey,
	}
	m.defineEnvironments[hash] = append(m.defineEnvironments[hash], environment)
	m.lastEnvironment = environment
	return environment
}

func defineEnvironmentHash(defines []string) uint64 {
	const offset = uint64(14695981039346656037)
	const prime = uint64(1099511628211)
	hash := offset
	for _, define := range defines {
		for index := 0; index < len(define); index++ {
			hash ^= uint64(define[index])
			hash *= prime
		}
		hash ^= 0
		hash *= prime
	}
	return hash
}

func (m *Model) orderDefineEnvironments() {
	environments := make([]*defineEnvironment, 0, m.nextEnvironmentID)
	for _, bucket := range m.defineEnvironments {
		environments = append(environments, bucket...)
	}
	sort.Slice(environments, func(i, j int) bool {
		return compareDefines(environments[i].names, environments[j].names) < 0
	})
	for index, environment := range environments {
		environment.order = uint32(index)
	}
}

func normalizeDefines(defines []string) []string {
	seen := make(map[string]struct{}, len(defines))
	normalized := make([]string, 0, len(defines))
	for _, define := range defines {
		if define == "" {
			continue
		}
		if _, exists := seen[define]; exists {
			continue
		}
		seen[define] = struct{}{}
		normalized = append(normalized, define)
	}
	sort.Strings(normalized)
	return normalized
}

func sameDefines(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func compareDefines(left, right []string) int {
	for index := 0; index < len(left) && index < len(right); index++ {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

func includePath(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 && raw[0] == '<' && raw[len(raw)-1] == '>' {
		return strings.TrimSpace(raw[1 : len(raw)-1])
	}
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		if value, err := strconv.Unquote(raw); err == nil {
			return value
		}
		return raw[1 : len(raw)-1]
	}
	return raw
}
