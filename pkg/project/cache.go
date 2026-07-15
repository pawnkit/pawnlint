package project

import (
	"crypto/sha256"
	"sync"

	"github.com/pawnkit/pawn-parser"
)

type ParseCache struct {
	mu      sync.RWMutex
	entries map[string]parseCacheEntry
}

type parseCacheEntry struct {
	hash [sha256.Size]byte
	file *parser.File
}

func NewParseCache() *ParseCache {
	return &ParseCache{entries: make(map[string]parseCacheEntry)}
}

func (c *ParseCache) parse(path string, source []byte) (*parser.File, bool) {
	if c == nil {
		return parser.Parse(source), false
	}
	hash := sha256.Sum256(source)
	c.mu.RLock()
	entry := c.entries[path]
	c.mu.RUnlock()
	if entry.file != nil && entry.hash == hash {
		return entry.file, true
	}
	parsed := parser.Parse(source)
	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[string]parseCacheEntry)
	}
	if existing := c.entries[path]; existing.file != nil && existing.hash == hash {
		parsed = existing.file
	} else {
		c.entries[path] = parseCacheEntry{hash: hash, file: parsed}
	}
	c.mu.Unlock()
	return parsed, false
}
