package project

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
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
	maxDefineContexts       = 1024
)

type ParseCache struct {
	mu      sync.RWMutex
	entries map[string]parseCacheEntry

	analysisMu  sync.RWMutex
	indexes     map[string]indexCacheEntry
	walks       map[analysisCacheKey]walkCacheEntry
	semantics   map[analysisCacheKey]semanticCacheEntry
	defines     map[[sha256.Size]byte][]defineContextCacheEntry
	defineCount int
}

type parseCacheEntry struct {
	hash          [sha256.Size]byte
	discardTrivia bool
	file          *parser.File
}

// PreparedSource is immutable parser input for [ParseCache.PrepareContext].
type PreparedSource struct {
	Path          string
	Content       []byte
	Tokens        []token.Token
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
			c.parse(source.Path, source.Content, source.DiscardTrivia, source.Tokens)
			return groupCtx.Err()
		})
	}
	return group.Wait()
}

func (c *ParseCache) parse(path string, source []byte, discardTrivia bool, tokens []token.Token) (*parser.File, bool) {
	return c.parseHashed(path, source, sha256.Sum256(source), discardTrivia, tokens)
}

func (c *ParseCache) parseHashed(path string, source []byte, hash [sha256.Size]byte, discardTrivia bool, tokens []token.Token) (*parser.File, bool) {
	if c == nil {
		return parseSource(source, discardTrivia, tokens), false
	}
	c.mu.RLock()
	entry := c.entries[path]
	c.mu.RUnlock()
	if entry.file != nil && entry.hash == hash && entry.discardTrivia == discardTrivia {
		return entry.file, true
	}
	parsed := parseSource(source, discardTrivia, tokens)
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[string]parseCacheEntry)
	}
	if existing := c.entries[path]; existing.file != nil && existing.hash == hash && existing.discardTrivia == discardTrivia {
		parsed = existing.file
	} else {
		c.entries[path] = parseCacheEntry{hash: hash, discardTrivia: discardTrivia, file: parsed}
	}
	c.mu.Unlock()
	return parsed, false
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
	c.defines = nil
	c.defineCount = 0
}
