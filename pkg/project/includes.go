package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/preprocess"
	"github.com/pawnkit/pawnlint/internal/semantic"
	sourceinfo "github.com/pawnkit/pawnlint/internal/source"
	"github.com/pawnkit/pawnlint/internal/source/cst"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

const compactTargetThreshold = 512 << 10

func (m *Model) addFile(path string, source []byte, provided bool, defines *defineEnvironment, includeRoot string) (*File, error) {
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
		retainTrivia := m.options.Features == nil || m.options.Features.Has(FeatureTrivia) || bytes.Contains(source, []byte("pawnlint-"))
		if m.options.ReleaseIncludes && (!provided || len(source) >= compactTargetThreshold) {
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
			physical = &physicalFile{source: source, compact: compact, lineTable: sourceinfo.NewLineTable(source)}
			m.physical[canonical] = physical
		} else {
			discardTrivia := !retainTrivia
			var parsed *parser.File
			if m.options.ParseCache != nil {
				started := time.Now()
				var cached bool
				parsed, cached = m.options.ParseCache.parse(canonical, source, discardTrivia)
				if !cached && m.options.ObserveTiming != nil {
					m.observe(TimingEvent{Stage: TimingParse, Duration: time.Since(started)})
				}
			} else if m.options.ObserveTiming == nil {
				parsed = parser.ParseWithOptions(source, parser.ParseOptions{DiscardTrivia: discardTrivia})
			} else {
				started := time.Now()
				parsed = parser.ParseWithOptions(source, parser.ParseOptions{DiscardTrivia: discardTrivia})
				m.observe(TimingEvent{Stage: TimingParse, Duration: time.Since(started)})
			}
			physical = &physicalFile{source: source, parsed: parsed, lineTable: sourceinfo.NewLineTable(source), syntaxIndex: walk.NewIndex(parsed)}
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
	file := &File{Path: display, Source: physical.source, Parsed: parsed, CompactParsed: compact, Provided: provided, canonical: canonical, includeRoot: includeRoot, defines: defines, complete: m.options.DefinesComplete, sourceID: uint32(len(m.Files) + 1), syntaxIndex: physical.syntaxIndex}
	if parsed != nil {
		file.Walk = walk.NewWithContext(display, parsed, defines.walk, nil, m.options.DefinesComplete, physical.lineTable, physical.syntaxIndex)
		file.Syntax = cst.Pointer(file.Walk)
	} else {
		file.CompactWalk = walk.NewCompactWithContext(display, compact, defines.walk, nil, m.options.DefinesComplete, physical.lineTable)
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

func (m *Model) resolveFileIncludes(file *File) error {
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
	dirty := false
	defineCursor := file.Syntax.NewDefineCursor()
	for _, node := range nodes {
		if dirty {
			file.rebuildWalk(snapshots)
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
			dirty = true
		}
	}
	if dirty {
		file.rebuildWalk(snapshots)
		defineCursor = file.Syntax.NewDefineCursor()
	}
	file.final = m.internDefines(defineCursor.KnownDefinesViewAt(len(file.Source) + 1))
	if m.options.ObserveTiming == nil {
		if file.Parsed != nil {
			file.Semantic = semantic.Build(file.Parsed, file.Walk)
		} else {
			file.CompactSemantic = semantic.BuildCompact(file.CompactParsed, file.CompactWalk)
		}
	} else {
		started := time.Now()
		if file.Parsed != nil {
			file.Semantic = semantic.Build(file.Parsed, file.Walk)
		} else {
			file.CompactSemantic = semantic.BuildCompact(file.CompactParsed, file.CompactWalk)
		}
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
	path = filepath.FromSlash(strings.ReplaceAll(path, "\\", "/"))
	var bases []string
	if filepath.IsAbs(path) {
		bases = []string{""}
	} else {
		if quoted {
			bases = append(bases, from.includeRoot)
			bases = append(bases, filepath.Dir(from.canonical))
		}
		bases = append(bases, m.options.IncludePaths...)
		bases = append(bases, m.options.WorkingDir)
	}
	seen := make(map[string]struct{})
	var candidates []string
	for _, base := range bases {
		candidate := path
		if base != "" {
			candidate = filepath.Join(base, path)
		}
		for _, name := range includeCandidates(candidate) {
			canonical, err := canonicalPath(name, m.options.WorkingDir)
			if err != nil {
				continue
			}
			if _, tried := seen[canonical]; tried {
				continue
			}
			seen[canonical] = struct{}{}
			if m.physical[canonical] != nil {
				candidates = append(candidates, canonical)
				continue
			}
			info, err := os.Stat(canonical)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, nil, err
			}
			if !info.Mode().IsRegular() {
				continue
			}
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
		source, err = os.ReadFile(chosen)
		if err != nil {
			return nil, nil, err
		}
	}
	resolved, err := m.addFile(chosen, source, false, defines, from.includeRoot)
	return resolved, candidates, err
}

func (f *File) rebuildWalk(snapshots []walk.DefineSnapshot) {
	f.snapshots = append(f.snapshots[:0], snapshots...)
	if f.Parsed != nil {
		f.Walk = walk.NewWithSharedContext(f.Path, f.Parsed, f.defines.walk, f.snapshots, f.complete, f.Walk.LineTable, f.syntaxIndex)
		f.Syntax = cst.Pointer(f.Walk)
	} else {
		f.CompactWalk = walk.NewCompactWithSharedContext(f.Path, f.CompactParsed, f.defines.walk, f.snapshots, f.complete, f.CompactWalk.LineTable)
		f.Syntax = cst.Compact(f.CompactWalk)
	}
}

func (m *Model) internDefines(defines []string) *defineEnvironment {
	hash := defineEnvironmentHash(defines)
	for _, environment := range m.defineEnvironments[hash] {
		if sameDefines(environment.names, defines) {
			return environment
		}
	}
	m.nextEnvironmentID++
	names := append([]string(nil), defines...)
	environment := &defineEnvironment{id: m.nextEnvironmentID, names: names, walk: walk.NewDefineContext(names)}
	m.defineEnvironments[hash] = append(m.defineEnvironments[hash], environment)
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

func includeCandidates(path string) []string {
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".inc" || extension == ".pwn" {
		return []string{path}
	}
	return []string{path, path + ".inc"}
}
