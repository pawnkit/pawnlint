package project

import (
	"crypto/sha256"
	"testing"

	"github.com/pawnkit/pawnlint/internal/semantic"
	"github.com/pawnkit/pawnlint/internal/source/walk"
)

func TestParseCacheBoundsDefineContexts(t *testing.T) {
	cache := NewParseCache()
	cache.walks = map[analysisCacheKey]walkCacheEntry{{}: {walk: &walk.Model{}}}
	cache.semantics = map[analysisCacheKey]semanticCacheEntry{{complete: true}: {semantic: &semantic.Model{}}}
	cache.defines = make(map[[sha256.Size]byte][]defineContextCacheEntry)
	cache.defineCount = maxDefineContexts

	cache.defineContext([]string{"CURRENT"}, definesCacheKey([]string{"CURRENT"}))

	if cache.defineCount != 1 {
		t.Fatalf("define contexts = %d, want 1", cache.defineCount)
	}
	if len(cache.walks) != 0 || len(cache.semantics) != 0 {
		t.Fatal("analysis entries survived define-context eviction")
	}
}

func TestParseCacheBoundsAnalysisVariants(t *testing.T) {
	cache := NewParseCache()
	cache.walks = make(map[analysisCacheKey]walkCacheEntry, maxAnalysisCacheEntries)
	for index := 0; index < maxAnalysisCacheEntries; index++ {
		cache.walks[analysisCacheKey{path: string(rune(index))}] = walkCacheEntry{}
	}
	cache.defines = map[[sha256.Size]byte][]defineContextCacheEntry{{}: nil}
	cache.defineCount = 1

	cache.putSemantic("current.pwn", [sha256.Size]byte{}, [sha256.Size]byte{}, true, [sha256.Size]byte{}, &semantic.Model{})

	if len(cache.walks) != 0 || len(cache.semantics) != 1 {
		t.Fatalf("cache sizes = walks %d, semantics %d", len(cache.walks), len(cache.semantics))
	}
	if cache.defineCount != 0 {
		t.Fatalf("define contexts = %d, want 0", cache.defineCount)
	}
}
