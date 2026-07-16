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
	hash          [sha256.Size]byte
	discardTrivia bool
	file          *parser.File
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
