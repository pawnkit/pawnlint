package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"sync"

	"github.com/pawnkit/pawnlint/pkg/diagnostic"
)

const Version = 1

type Source struct {
	Path    string
	Content []byte
}

type KeyInput struct {
	Context string
	Config  any
	API     any
	Sources []Source
}

type entry struct {
	Version     int                     `json:"version"`
	Key         string                  `json:"key"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

type sourceDigest struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

type keyDocument struct {
	Version       int            `json:"version"`
	Context       string         `json:"context"`
	Config        any            `json:"config"`
	API           any            `json:"api"`
	Sources       []sourceDigest `json:"sources"`
	ParserVersion string         `json:"parserVersion"`
	RuleVersion   string         `json:"ruleVersion"`
}

func Key(input KeyInput) (string, error) {
	sources := make([]sourceDigest, len(input.Sources))
	for index, source := range input.Sources {
		sum := sha256.Sum256(source.Content)
		sources[index] = sourceDigest{Path: filepath.Clean(source.Path), Hash: hex.EncodeToString(sum[:])}
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Path < sources[j].Path })
	document := keyDocument{
		Version:       Version,
		Context:       input.Context,
		Config:        input.Config,
		API:           input.API,
		Sources:       sources,
		ParserVersion: parserVersion(),
		RuleVersion:   ruleVersion(),
	}
	data, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("cache key: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func Slot(context string) string {
	sum := sha256.Sum256([]byte(context))
	return hex.EncodeToString(sum[:])
}

func Load(directory, slot, key string) ([]diagnostic.Diagnostic, bool) {
	path := filepath.Join(directory, slot+".json")
	input, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer func() { _ = input.Close() }()
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	var cached entry
	if err := decoder.Decode(&cached); err != nil {
		return nil, false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, false
	}
	if cached.Version != Version || cached.Key != key {
		return nil, false
	}
	return cached.Diagnostics, true
}

func Validate(diagnostics []diagnostic.Diagnostic, sources []Source) bool {
	lengths := make(map[string]int, len(sources))
	for _, source := range sources {
		lengths[filepath.Clean(source.Path)] = len(source.Content)
	}
	for _, finding := range diagnostics {
		length, exists := lengths[filepath.Clean(finding.Filename)]
		if !exists || finding.Severity < diagnostic.SeverityError || finding.Severity > diagnostic.SeverityHint || finding.Category > diagnostic.CategoryRestriction || !validRange(finding.Range.Start.Offset, finding.Range.End.Offset, length) {
			return false
		}
		for _, note := range finding.Notes {
			if !validRange(note.Range.Start.Offset, note.Range.End.Offset, length) {
				return false
			}
		}
		if finding.Fix != nil && !validEdits(finding.Fix.Edits, length) {
			return false
		}
		for _, suggestion := range finding.Suggestions {
			if !validEdits(suggestion.Edits, length) {
				return false
			}
		}
	}
	return true
}

func validEdits(edits []diagnostic.Edit, length int) bool {
	for _, edit := range edits {
		if !validRange(edit.Range.Start.Offset, edit.Range.End.Offset, length) {
			return false
		}
	}
	return true
}

func validRange(start, end, length int) bool {
	return start >= 0 && end >= start && end <= length
}

func Write(directory, slot, key string, diagnostics []diagnostic.Diagnostic) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	data, err := json.Marshal(entry{Version: Version, Key: key, Diagnostics: diagnostics})
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	data = append(data, '\n')
	temporary, err := os.CreateTemp(directory, ".pawnlint-cache-*")
	if err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	temporaryPath := temporary.Name()
	remove := true
	defer func() {
		_ = temporary.Close()
		if remove {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(directory, slot+".json")); err != nil {
		return fmt.Errorf("cache: %w", err)
	}
	remove = false
	return nil
}

var versions struct {
	sync.Once
	parser string
	rules  string
}

func parserVersion() string {
	loadVersions()
	return versions.parser
}

func ruleVersion() string {
	loadVersions()
	return versions.rules
}

func loadVersions() {
	versions.Do(func() {
		versions.parser = "unknown"
		versions.rules = executableHash()
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, dependency := range info.Deps {
				if dependency.Path == "github.com/pawnkit/pawn-parser" {
					versions.parser = dependency.Version + ":" + dependency.Sum
					if dependency.Replace != nil {
						versions.parser += ":" + dependency.Replace.Path + ":" + dependency.Replace.Version + ":" + dependency.Replace.Sum
					}
				}
			}
		}
	})
}

func executableHash() string {
	path, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	input, err := os.Open(path)
	if err != nil {
		return "unknown"
	}
	defer func() { _ = input.Close() }()
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(hash.Sum(nil))
}
