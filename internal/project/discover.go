package project

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type File struct {
	Path    string
	Content []byte
}

type Options struct {
	Roots []string

	Include []string

	Exclude []string

	WorkingDir string
}

func Discover(opts Options) ([]File, error) {
	var paths []string
	seen := map[string]bool{}
	add := func(p string) {
		ap, err := filepath.Abs(p)
		if err != nil {
			ap = p
		}
		if seen[ap] {
			return
		}
		seen[ap] = true
		paths = append(paths, p)
	}
	if len(opts.Roots) > 0 {
		for _, r := range opts.Roots {
			info, err := os.Stat(r)
			if err != nil {
				return nil, &PathError{Path: r, Err: err}
			}
			if info.IsDir() {
				if err := walkDir(r, opts, add); err != nil {
					return nil, &PathError{Path: r, Err: err}
				}
			} else {
				if isPawnFile(r) {
					add(r)
				}
			}
		}
	} else {
		for _, g := range opts.Include {
			base := opts.WorkingDir
			matches, err := globWalk(base, g, opts)
			if err != nil {
				return nil, err
			}
			for _, m := range matches {
				add(m)
			}
		}
	}
	sort.Strings(paths)
	out := make([]File, 0, len(paths))
	for _, p := range paths {
		content, err := os.ReadFile(p)
		if err != nil {
			return nil, &PathError{Path: p, Err: err}
		}
		out = append(out, File{Path: p, Content: content})
	}
	return out, nil
}

func walkDir(root string, opts Options, add func(string)) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isPawnFile(path) {
			return nil
		}
		rel := RelPath(opts.WorkingDir, path)
		if isExcludedRel(rel, opts) {
			return nil
		}
		add(path)
		return nil
	})
}

func RelPath(base, path string) string {
	if base == "" {
		return filepath.ToSlash(path)
	}
	r, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

func isExcludedRel(rel string, opts Options) bool {
	if len(opts.Exclude) == 0 {
		return false
	}
	for _, g := range opts.Exclude {
		if MatchGlob(g, rel) {
			return true
		}
		if MatchGlob(g, filepath.Base(rel)) {
			return true
		}
	}
	return false
}

func globWalk(base, pattern string, opts Options) ([]string, error) {
	var out []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(base, path)
		rel = filepath.ToSlash(rel)
		if !isPawnFile(path) {
			return nil
		}
		if isExcludedRel(rel, opts) {
			return nil
		}
		if MatchGlob(pattern, rel) {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}

func isPawnFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".pwn" || ext == ".inc"
}

type PathError struct {
	Path string
	Err  error
}

func (e *PathError) Error() string { return e.Path + ": " + e.Err.Error() }
func (e *PathError) Unwrap() error { return e.Err }
