package project

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pawnkit/pawnlint/internal/controlflow"
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

func TestParseCacheReusesAndInvalidatesProjectModels(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.pwn")
	cache := NewParseCache()
	source := []Source{{Path: path, Content: []byte("main() {}\n")}}
	options := Options{WorkingDir: dir, ParseCache: cache, IncludeSources: source}
	first, err := Build(source, options)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(source, options)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatal("unchanged project model was not reused")
	}
	cache.InvalidateFiles()
	third, err := Build(source, options)
	if err != nil {
		t.Fatal(err)
	}
	if third == second {
		t.Fatal("invalidated project model was reused")
	}
	changed := []Source{{Path: path, Content: []byte("main() { return 1; }\n")}}
	fourth, err := Build(changed, options)
	if err != nil {
		t.Fatal(err)
	}
	if fourth == third {
		t.Fatal("changed project model was reused")
	}
}

func TestFileCachesDerivedFlowByContext(t *testing.T) {
	file := &File{}
	first, built := file.CachedFlow("first", func() *controlflow.Model {
		return &controlflow.Model{}
	})
	if !built || first == nil {
		t.Fatal("first flow was not built")
	}
	second, built := file.CachedFlow("first", func() *controlflow.Model {
		t.Fatal("same flow context was rebuilt")
		return nil
	})
	if built || second != first {
		t.Fatal("same flow context was not reused")
	}
	third, built := file.CachedFlow("second", func() *controlflow.Model {
		return &controlflow.Model{}
	})
	if !built || third == first {
		t.Fatal("changed flow context was not rebuilt")
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

func TestModelReusesRecentDefineEnvironment(t *testing.T) {
	model := &Model{defineEnvironments: make(map[uint64][]*defineEnvironment)}
	first := model.internDefines([]string{"FIRST"})
	if repeated := model.internDefines([]string{"FIRST"}); repeated != first {
		t.Fatal("recent define environment was not reused")
	}
	model.internDefines([]string{"SECOND"})
	if repeated := model.internDefines([]string{"FIRST"}); repeated != first {
		t.Fatal("interned define environment was not reused")
	}
}

func TestParseCacheInvalidatesFilesystemProbes(t *testing.T) {
	cache := NewParseCache()
	path := filepath.Join(t.TempDir(), "added.inc")
	if _, err := cache.stat(path); !os.IsNotExist(err) {
		t.Fatalf("first stat error = %v", err)
	}
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.stat(path); !os.IsNotExist(err) {
		t.Fatalf("cached stat error = %v", err)
	}
	cache.InvalidateFiles()
	if _, err := cache.stat(path); err != nil {
		t.Fatalf("stat after invalidation: %v", err)
	}
}

func TestParseCacheBoundsFilesystemProbes(t *testing.T) {
	cache := NewParseCache()
	cache.statErrors = make(map[string]error, maxFilesystemProbes)
	for index := 0; index < maxFilesystemProbes; index++ {
		cache.statErrors[strconv.Itoa(index)] = os.ErrNotExist
	}
	path := filepath.Join(t.TempDir(), "current.inc")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.stat(path); err != nil {
		t.Fatal(err)
	}
	if len(cache.stats) != 1 || len(cache.statErrors) != 0 {
		t.Fatalf("filesystem probes = %d successes, %d failures", len(cache.stats), len(cache.statErrors))
	}
}

func TestParseCacheInvalidatesIncludeResolutions(t *testing.T) {
	cache := NewParseCache()
	key := includeResolutionCacheKey{spec: "shared"}
	cache.putIncludeResolution(key, []string{"shared.inc"})
	cache.InvalidateFiles()
	if _, ok := cache.getIncludeResolution(key); ok {
		t.Fatal("include resolution survived invalidation")
	}
}

func TestParseCacheBoundsIncludeResolutions(t *testing.T) {
	cache := NewParseCache()
	cache.includes = make(map[includeResolutionCacheKey][]string, maxIncludeResolutions)
	for index := 0; index < maxIncludeResolutions; index++ {
		cache.includes[includeResolutionCacheKey{spec: strconv.Itoa(index)}] = nil
	}
	current := includeResolutionCacheKey{spec: "current"}
	cache.putIncludeResolution(current, []string{"current.inc"})
	if len(cache.includes) != 1 {
		t.Fatalf("include resolutions = %d, want 1", len(cache.includes))
	}
	if paths, ok := cache.getIncludeResolution(current); !ok || len(paths) != 1 {
		t.Fatalf("current include resolution = %#v, %t", paths, ok)
	}
}
