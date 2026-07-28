package project

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"runtime"
	"strings"
	"sync"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawn-parser/token"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
	"golang.org/x/sync/errgroup"
)

type ParseCache struct {
	mu      sync.RWMutex
	entries map[string]parseCacheEntry

	analysisMu sync.RWMutex
	indexes    map[string]indexCacheEntry
	walks      map[string]walkCacheEntry
	semantics  map[string]semanticCacheEntry
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

func definesCacheKey(defines []string) string {
	var b strings.Builder
	for _, define := range defines {
		b.WriteString(define)
		b.WriteByte('\x00')
	}
	return b.String()
}

func analysisCacheKey(path, definesKey, snapshotsKey string, complete bool) string {
	var b strings.Builder
	b.WriteString(path)
	b.WriteByte('\x00')
	if complete {
		b.WriteByte('1')
	} else {
		b.WriteByte('0')
	}
	b.WriteByte('\x00')
	b.WriteString(definesKey)
	b.WriteByte('\x00')
	b.WriteString(snapshotsKey)
	return b.String()
}

type defineSnapshotIdentity struct {
	offset int
	hash   [sha256.Size]byte
}

func defineSnapshotsCacheKey(snapshots []defineSnapshotIdentity) string {
	if len(snapshots) == 0 {
		return ""
	}
	hash := sha256.New()
	var offset [8]byte
	for _, snapshot := range snapshots {
		binary.LittleEndian.PutUint64(offset[:], uint64(snapshot.offset))
		_, _ = hash.Write(offset[:])
		_, _ = hash.Write(snapshot.hash[:])
	}
	return hex.EncodeToString(hash.Sum(nil))
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
	if c == nil {
		return parseSource(source, discardTrivia, tokens), false
	}
	hash := sha256.Sum256(source)
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

func (c *ParseCache) getIndex(path string, source []byte) *walk.Index {
	if c == nil {
		return nil
	}
	hash := sha256.Sum256(source)
	c.analysisMu.RLock()
	entry, ok := c.indexes[path]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.index
	}
	return nil
}

func (c *ParseCache) putIndex(path string, source []byte, index *walk.Index) {
	if c == nil {
		return
	}
	c.analysisMu.Lock()
	if c.indexes == nil {
		c.indexes = make(map[string]indexCacheEntry)
	}
	c.indexes[path] = indexCacheEntry{hash: sha256.Sum256(source), index: index}
	c.analysisMu.Unlock()
}

func (c *ParseCache) getWalk(path string, source []byte, definesKey string, complete bool, snapshotsKey string) *walk.Model {
	if c == nil {
		return nil
	}
	hash := sha256.Sum256(source)
	key := analysisCacheKey(path, definesKey, snapshotsKey, complete)
	c.analysisMu.RLock()
	entry, ok := c.walks[key]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.walk
	}
	return nil
}

func (c *ParseCache) putWalk(path string, source []byte, definesKey string, complete bool, snapshotsKey string, model *walk.Model) {
	if c == nil {
		return
	}
	key := analysisCacheKey(path, definesKey, snapshotsKey, complete)
	c.analysisMu.Lock()
	if c.walks == nil {
		c.walks = make(map[string]walkCacheEntry)
	}
	c.walks[key] = walkCacheEntry{hash: sha256.Sum256(source), walk: model}
	c.analysisMu.Unlock()
}

func (c *ParseCache) getSemantic(path string, source []byte, definesKey string, complete bool, snapshotsKey string) *semantic.Model {
	if c == nil {
		return nil
	}
	hash := sha256.Sum256(source)
	key := analysisCacheKey(path, definesKey, snapshotsKey, complete)
	c.analysisMu.RLock()
	entry, ok := c.semantics[key]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.semantic
	}
	return nil
}

func (c *ParseCache) putSemantic(path string, source []byte, definesKey string, complete bool, snapshotsKey string, model *semantic.Model) {
	if c == nil {
		return
	}
	key := analysisCacheKey(path, definesKey, snapshotsKey, complete)
	c.analysisMu.Lock()
	if c.semantics == nil {
		c.semantics = make(map[string]semanticCacheEntry)
	}
	c.semantics[key] = semanticCacheEntry{hash: sha256.Sum256(source), semantic: model}
	c.analysisMu.Unlock()
}
