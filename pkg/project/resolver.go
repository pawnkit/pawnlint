package project

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pawnkit/pawn-project/fsx"
	"github.com/pawnkit/pawn-project/include"
	"github.com/pawnkit/pawn-project/pathutil"
)

type includeResolution struct {
	fsys      fsx.FS
	files     map[string][]byte
	roots     []string
	resolvers map[string]*include.Resolver
}

func newIncludeResolver(sources []Source, options Options) *includeResolution {
	files := make(map[string][]byte, len(options.IncludeSources)+len(sources))
	for _, group := range [][]Source{options.IncludeSources, sources} {
		for _, source := range group {
			path, err := canonicalPath(source.Path, options.WorkingDir)
			if err == nil {
				slashPath := pathutil.ToSlash(path)
				files[slashPath] = source.Content
			}
		}
	}
	roots := make([]string, 0, len(options.IncludePaths)+1)
	for _, root := range options.IncludePaths {
		roots = append(roots, pathutil.ToSlash(root))
	}
	roots = append(roots, pathutil.ToSlash(options.WorkingDir))

	return &includeResolution{
		fsys:      &sourceFS{files: files, stats: make(map[string]fs.FileInfo), statErrors: make(map[string]error)},
		files:     files,
		roots:     roots,
		resolvers: make(map[string]*include.Resolver),
	}
}

func (r *includeResolution) ResolveAll(fromFile, spec string, quoted bool, includeRoot string) []string {
	root := pathutil.ToSlash(includeRoot)
	resolver := r.resolvers[root]
	if resolver == nil {
		resolver = include.NewWithQuotedRoots(r.fsys, r.roots, []string{root})
		r.resolvers[root] = resolver
	}
	return resolver.ResolveAll(pathutil.ToSlash(fromFile), spec, quoted)
}

func (r *includeResolution) ReadFile(path string) ([]byte, error) {
	path = pathutil.ToSlash(path)
	if content, ok := r.files[path]; ok {
		return content, nil
	}
	path = pathutil.Clean(path)
	if content, ok := r.files[path]; ok {
		return content, nil
	}
	return os.ReadFile(filepath.FromSlash(path)) //nolint:gosec // Resolution constrains the path.
}

type sourceFS struct {
	files      map[string][]byte
	stats      map[string]fs.FileInfo
	statErrors map[string]error
}

func (s *sourceFS) Stat(path string) (fs.FileInfo, error) {
	if content, ok := s.files[path]; ok {
		return sourceFileInfo{name: filepath.Base(path), size: int64(len(content))}, nil
	}
	if info, ok := s.stats[path]; ok {
		return info, nil
	}
	if err, ok := s.statErrors[path]; ok {
		return nil, err
	}
	cleaned := pathutil.Clean(path)
	if cleaned != path {
		return s.Stat(cleaned)
	}
	info, err := os.Stat(path)
	if err != nil {
		s.statErrors[path] = err
		return nil, err
	}
	s.stats[path] = info
	return info, nil
}

func (s *sourceFS) ReadFile(path string) ([]byte, error) {
	if content, ok := s.files[path]; ok {
		return append([]byte(nil), content...), nil
	}
	path = pathutil.Clean(path)
	if content, ok := s.files[path]; ok {
		return append([]byte(nil), content...), nil
	}
	return os.ReadFile(path) //nolint:gosec // Include roots constrain resolver paths.
}

func (s *sourceFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

type sourceFileInfo struct {
	name string
	size int64
}

func (i sourceFileInfo) Name() string     { return i.name }
func (i sourceFileInfo) Size() int64      { return i.size }
func (sourceFileInfo) Mode() fs.FileMode  { return 0o644 }
func (sourceFileInfo) ModTime() time.Time { return time.Time{} }
func (sourceFileInfo) IsDir() bool        { return false }
func (sourceFileInfo) Sys() any           { return nil }

var _ fsx.FS = (*sourceFS)(nil)
