package project

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"io/fs"
	"os"
	"runtime"
	"sync"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"golang.org/x/sync/errgroup"
)

const (
	maxAnalysisCacheEntries = 4096
	maxModelCacheEntries    = 64
	maxDefineContexts       = 1024
	maxFilesystemProbes     = 16384
	maxIncludeResolutions   = 16384
)

type ParseCache struct {
	mu      sync.RWMutex
	entries map[string]parseCacheEntry

	analysisMu  sync.RWMutex
	indexes     map[string]indexCacheEntry
	walks       map[analysisCacheKey]walkCacheEntry
	semantics   map[analysisCacheKey]semanticCacheEntry
	models      map[[sha256.Size]byte]*Model
	defines     map[[sha256.Size]byte][]defineContextCacheEntry
	defineCount int

	resolutionMu sync.RWMutex
	stats        map[string]fs.FileInfo
	statErrors   map[string]error
	includes     map[includeResolutionCacheKey][]string
}

type parseCacheEntry struct {
	hash          [sha256.Size]byte
	discardTrivia bool
	allowTrivia   bool
	file          *parser.File
}

// PreparedSource is immutable parser input for [ParseCache.PrepareContext].
type PreparedSource struct {
	Path          string
	Content       []byte
	Tokens        []token.Token
	CompactSyntax *parser.CompactFile
	DiscardTrivia bool
}

type indexCacheEntry struct {
	hash  [sha256.Size]byte
	index *walk.Index
}

type walkCacheEntry struct {
	hash [sha256.Size]byte
	walk *walk.Model
}

type semanticCacheEntry struct {
	hash     [sha256.Size]byte
	semantic *semantic.Model
}

type defineContextCacheEntry struct {
	names   []string
	context *walk.DefineContext
}

type includeResolutionCacheKey struct {
	context [sha256.Size]byte
	from    string
	spec    string
	root    string
	quoted  bool
}

func definesCacheKey(defines []string) [sha256.Size]byte {
	hash := sha256.New()
	for _, define := range defines {
		_, _ = hash.Write([]byte(define))
		_, _ = hash.Write([]byte{0})
	}
	var key [sha256.Size]byte
	hash.Sum(key[:0])
	return key
}

type analysisCacheKey struct {
	path      string
	defines   [sha256.Size]byte
	snapshots [sha256.Size]byte
	complete  bool
}

type defineSnapshotIdentity struct {
	offset int
	hash   [sha256.Size]byte
}

func defineSnapshotsCacheKey(snapshots []defineSnapshotIdentity) [sha256.Size]byte {
	if len(snapshots) == 0 {
		return [sha256.Size]byte{}
	}
	hash := sha256.New()
	var offset [8]byte
	for _, snapshot := range snapshots {
		binary.LittleEndian.PutUint64(offset[:], uint64(snapshot.offset))
		_, _ = hash.Write(offset[:])
		_, _ = hash.Write(snapshot.hash[:])
	}
	var key [sha256.Size]byte
	hash.Sum(key[:0])
	return key
}

func NewParseCache() *ParseCache {
	return &ParseCache{entries: make(map[string]parseCacheEntry)}
}

// InvalidateFiles clears cached filesystem probes.
func (c *ParseCache) InvalidateFiles() {
	if c == nil {
		return
	}
	c.resolutionMu.Lock()
	c.stats = nil
	c.statErrors = nil
	c.includes = nil
	c.resolutionMu.Unlock()
	c.analysisMu.Lock()
	c.models = nil
	c.analysisMu.Unlock()
}

func (c *ParseCache) stat(path string) (fs.FileInfo, error) {
	if c == nil {
		return os.Stat(path)
	}
	c.resolutionMu.RLock()
	info, found := c.stats[path]
	err, failed := c.statErrors[path]
	c.resolutionMu.RUnlock()
	if found {
		return info, nil
	}
	if failed {
		return nil, err
	}
	info, err = os.Stat(path)
	c.resolutionMu.Lock()
	if existing, ok := c.stats[path]; ok {
		info, err = existing, nil
	} else if existing, ok := c.statErrors[path]; ok {
		info, err = nil, existing
	} else {
		if len(c.stats)+len(c.statErrors) >= maxFilesystemProbes {
			c.stats = nil
			c.statErrors = nil
		}
		if err == nil {
			if c.stats == nil {
				c.stats = make(map[string]fs.FileInfo)
			}
			c.stats[path] = info
		} else {
			if c.statErrors == nil {
				c.statErrors = make(map[string]error)
			}
			c.statErrors[path] = err
		}
	}
	c.resolutionMu.Unlock()
	return info, err
}

func (c *ParseCache) getIncludeResolution(key includeResolutionCacheKey) ([]string, bool) {
	if c == nil {
		return nil, false
	}
	c.resolutionMu.RLock()
	paths, ok := c.includes[key]
	c.resolutionMu.RUnlock()
	return paths, ok
}

func (c *ParseCache) putIncludeResolution(key includeResolutionCacheKey, paths []string) {
	if c == nil {
		return
	}
	c.resolutionMu.Lock()
	if _, exists := c.includes[key]; !exists && len(c.includes) >= maxIncludeResolutions {
		c.includes = nil
	}
	if c.includes == nil {
		c.includes = make(map[includeResolutionCacheKey][]string)
	}
	c.includes[key] = paths
	c.resolutionMu.Unlock()
}

// PrepareContext parses sources concurrently into the cache.
func (c *ParseCache) PrepareContext(ctx context.Context, sources []PreparedSource) error {
	if c == nil || len(sources) == 0 {
		return ctx.Err()
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(max(1, runtime.GOMAXPROCS(0)))
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if _, ok := seen[source.Path]; ok {
			continue
		}
		seen[source.Path] = struct{}{}
		source := source
		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}
			c.prepare(source)
			return groupCtx.Err()
		})
	}
	return group.Wait()
}

func (c *ParseCache) prepare(source PreparedSource) {
	hash := sha256.Sum256(source.Content)
	if source.CompactSyntax == nil || !bytes.Equal(source.CompactSyntax.Source, source.Content) {
		c.parseHashed(source.Path, source.Content, hash, source.DiscardTrivia, source.Tokens)
		return
	}
	c.mu.RLock()
	entry := c.entries[source.Path]
	c.mu.RUnlock()
	if entry.matches(hash, source.DiscardTrivia) {
		return
	}
	parsed := source.CompactSyntax.ExpandTokensWithOptions(
		source.Tokens,
		parser.ParseOptions{DiscardTrivia: source.DiscardTrivia},
	)
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[string]parseCacheEntry)
	}
	if existing := c.entries[source.Path]; !existing.matches(hash, source.DiscardTrivia) {
		c.entries[source.Path] = parseCacheEntry{
			hash: hash, discardTrivia: source.DiscardTrivia, allowTrivia: true, file: parsed,
		}
	}
	c.mu.Unlock()
}

func (c *ParseCache) parseHashed(path string, source []byte, hash [sha256.Size]byte, discardTrivia bool, tokens []token.Token) (*parser.File, bool) {
	if c == nil {
		return parseSource(source, discardTrivia, tokens), false
	}
	c.mu.RLock()
	entry := c.entries[path]
	c.mu.RUnlock()
	if entry.matches(hash, discardTrivia) {
		return entry.file, true
	}
	parsed := parseSource(source, discardTrivia, tokens)
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[string]parseCacheEntry)
	}
	if existing := c.entries[path]; existing.matches(hash, discardTrivia) {
		parsed = existing.file
	} else {
		c.entries[path] = parseCacheEntry{hash: hash, discardTrivia: discardTrivia, file: parsed}
	}
	c.mu.Unlock()
	return parsed, false
}

func (e parseCacheEntry) matches(hash [sha256.Size]byte, discardTrivia bool) bool {
	return e.file != nil && e.hash == hash &&
		(e.discardTrivia == discardTrivia || e.allowTrivia && !e.discardTrivia)
}

func (c *ParseCache) getIndex(path string, hash [sha256.Size]byte) *walk.Index {
	if c == nil {
		return nil
	}
	c.analysisMu.RLock()
	entry, ok := c.indexes[path]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.index
	}
	return nil
}

func (c *ParseCache) putIndex(path string, hash [sha256.Size]byte, index *walk.Index) {
	if c == nil {
		return
	}
	c.analysisMu.Lock()
	if c.indexes == nil {
		c.indexes = make(map[string]indexCacheEntry)
	}
	c.indexes[path] = indexCacheEntry{hash: hash, index: index}
	c.analysisMu.Unlock()
}

func (c *ParseCache) getWalk(path string, hash, definesKey [sha256.Size]byte, complete bool, snapshotsKey [sha256.Size]byte) *walk.Model {
	if c == nil {
		return nil
	}
	key := analysisCacheKey{path: path, defines: definesKey, snapshots: snapshotsKey, complete: complete}
	c.analysisMu.RLock()
	entry, ok := c.walks[key]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.walk
	}
	return nil
}

func (c *ParseCache) putWalk(path string, hash, definesKey [sha256.Size]byte, complete bool, snapshotsKey [sha256.Size]byte, model *walk.Model) {
	if c == nil {
		return
	}
	key := analysisCacheKey{path: path, defines: definesKey, snapshots: snapshotsKey, complete: complete}
	c.analysisMu.Lock()
	if _, exists := c.walks[key]; !exists && len(c.walks)+len(c.semantics) >= maxAnalysisCacheEntries {
		c.resetAnalysisLocked()
	}
	if c.walks == nil {
		c.walks = make(map[analysisCacheKey]walkCacheEntry)
	}
	c.walks[key] = walkCacheEntry{hash: hash, walk: model}
	c.analysisMu.Unlock()
}

func (c *ParseCache) getSemantic(path string, hash, definesKey [sha256.Size]byte, complete bool, snapshotsKey [sha256.Size]byte) *semantic.Model {
	if c == nil {
		return nil
	}
	key := analysisCacheKey{path: path, defines: definesKey, snapshots: snapshotsKey, complete: complete}
	c.analysisMu.RLock()
	entry, ok := c.semantics[key]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.semantic
	}
	return nil
}

func (c *ParseCache) putSemantic(path string, hash, definesKey [sha256.Size]byte, complete bool, snapshotsKey [sha256.Size]byte, model *semantic.Model) {
	if c == nil {
		return
	}
	key := analysisCacheKey{path: path, defines: definesKey, snapshots: snapshotsKey, complete: complete}
	c.analysisMu.Lock()
	if _, exists := c.semantics[key]; !exists && len(c.walks)+len(c.semantics) >= maxAnalysisCacheEntries {
		c.resetAnalysisLocked()
	}
	if c.semantics == nil {
		c.semantics = make(map[analysisCacheKey]semanticCacheEntry)
	}
	c.semantics[key] = semanticCacheEntry{hash: hash, semantic: model}
	c.analysisMu.Unlock()
}

func (c *ParseCache) getModel(key [sha256.Size]byte) *Model {
	if c == nil {
		return nil
	}
	c.analysisMu.RLock()
	model := c.models[key]
	c.analysisMu.RUnlock()
	return model
}

func (c *ParseCache) putModel(key [sha256.Size]byte, model *Model) {
	if c == nil || model == nil {
		return
	}
	c.analysisMu.Lock()
	if c.models == nil || len(c.models) >= maxModelCacheEntries {
		c.models = make(map[[sha256.Size]byte]*Model)
	}
	c.models[key] = model
	c.analysisMu.Unlock()
}

func (c *ParseCache) defineContext(names []string, key [sha256.Size]byte) *walk.DefineContext {
	if c == nil {
		return walk.NewDefineContext(names)
	}
	c.analysisMu.RLock()
	for _, entry := range c.defines[key] {
		if sameDefines(entry.names, names) {
			c.analysisMu.RUnlock()
			return entry.context
		}
	}
	c.analysisMu.RUnlock()

	context := walk.NewDefineContext(names)
	c.analysisMu.Lock()
	for _, entry := range c.defines[key] {
		if sameDefines(entry.names, names) {
			c.analysisMu.Unlock()
			return entry.context
		}
	}
	if c.defineCount >= maxDefineContexts {
		c.resetAnalysisLocked()
	}
	if c.defines == nil {
		c.defines = make(map[[sha256.Size]byte][]defineContextCacheEntry)
	}
	c.defines[key] = append(c.defines[key], defineContextCacheEntry{
		names: append([]string(nil), names...), context: context,
	})
	c.defineCount++
	c.analysisMu.Unlock()
	return context
}

func (c *ParseCache) resetAnalysisLocked() {
	c.walks = nil
	c.semantics = nil
	c.models = nil
	c.defines = nil
	c.defineCount = 0
}
