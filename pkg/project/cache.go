package project

import (
	"crypto/sha256"
	"strings"
	"sync"

	"github.com/pawnkit/pawn-parser"
	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
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

// indexCacheEntry, walkCacheEntry, and semanticCacheEntry reuse the
// directive index, CST walk, and semantic model built for a file whose
// content and active #define set have not changed since the last build.
// Each is a pure function of those inputs, so a cache hit is exact, not an
// approximation. walks and semantics are keyed by (path, defines, complete)
// together, not path alone: the same include is commonly resolved under
// several different #define environments within one project, and a
// path-only slot would just thrash between them.
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

func analysisCacheKey(path string, defines []string, complete bool) string {
	var b strings.Builder
	b.WriteString(path)
	b.WriteByte('\x00')
	if complete {
		b.WriteByte('1')
	} else {
		b.WriteByte('0')
	}
	b.WriteByte('\x00')
	for _, define := range defines {
		b.WriteString(define)
		b.WriteByte('\x00')
	}
	return b.String()
}

func NewParseCache() *ParseCache {
	return &ParseCache{entries: make(map[string]parseCacheEntry)}
}

func (c *ParseCache) parse(path string, source []byte, discardTrivia bool) (*parser.File, bool) {
	if c == nil {
		return parser.ParseWithOptions(source, parser.ParseOptions{DiscardTrivia: discardTrivia}), false
	}
	hash := sha256.Sum256(source)
	c.mu.RLock()
	entry := c.entries[path]
	c.mu.RUnlock()
	if entry.file != nil && entry.hash == hash && entry.discardTrivia == discardTrivia {
		return entry.file, true
	}
	parsed := parser.ParseWithOptions(source, parser.ParseOptions{DiscardTrivia: discardTrivia})
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

func (c *ParseCache) getWalk(path string, source []byte, defines []string, complete bool) *walk.Model {
	if c == nil {
		return nil
	}
	hash := sha256.Sum256(source)
	key := analysisCacheKey(path, defines, complete)
	c.analysisMu.RLock()
	entry, ok := c.walks[key]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.walk
	}
	return nil
}

func (c *ParseCache) putWalk(path string, source []byte, defines []string, complete bool, model *walk.Model) {
	if c == nil {
		return
	}
	key := analysisCacheKey(path, defines, complete)
	c.analysisMu.Lock()
	if c.walks == nil {
		c.walks = make(map[string]walkCacheEntry)
	}
	c.walks[key] = walkCacheEntry{hash: sha256.Sum256(source), walk: model}
	c.analysisMu.Unlock()
}

func (c *ParseCache) getSemantic(path string, source []byte, defines []string, complete bool) *semantic.Model {
	if c == nil {
		return nil
	}
	hash := sha256.Sum256(source)
	key := analysisCacheKey(path, defines, complete)
	c.analysisMu.RLock()
	entry, ok := c.semantics[key]
	c.analysisMu.RUnlock()
	if ok && entry.hash == hash {
		return entry.semantic
	}
	return nil
}

func (c *ParseCache) putSemantic(path string, source []byte, defines []string, complete bool, model *semantic.Model) {
	if c == nil {
		return
	}
	key := analysisCacheKey(path, defines, complete)
	c.analysisMu.Lock()
	if c.semantics == nil {
		c.semantics = make(map[string]semanticCacheEntry)
	}
	c.semantics[key] = semanticCacheEntry{hash: sha256.Sum256(source), semantic: model}
	c.analysisMu.Unlock()
}
