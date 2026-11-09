package gitfetch

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const (
	maxFileBytes = 512 * 1024 // skip/limit files larger than this for reads/grep
	maxGrepHits  = 200
)

// localWorkspace is a read-only CodeWorkspace confined to a directory: every
// access goes through an os.Root, so ".." and symlinks committed by the MR
// cannot reach outside it, and .git (which may hold remote config) is hidden.
type localWorkspace struct {
	dir  string
	root *os.Root
	fsys fs.FS
}

func openLocalWorkspace(dir string) (*localWorkspace, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &localWorkspace{dir: dir, root: root, fsys: root.FS()}, nil
}

func (w *localWorkspace) Root() string { return w.dir }

func (w *localWorkspace) close() error { return w.root.Close() }

// name maps a workspace-relative path to an fs.FS name, anchoring it at the
// root so ".." cannot climb above it.
func (w *localWorkspace) name(p string) (string, error) {
	clean := strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(p, `\`, "/")), "/")
	if clean == "" {
		return ".", nil
	}
	if clean == ".git" || strings.HasPrefix(clean, ".git/") {
		return "", fmt.Errorf("%q is not accessible", p)
	}
	return clean, nil
}

func (w *localWorkspace) ReadFile(p string) ([]byte, error) {
	name, err := w.name(p)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(w.fsys, name)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory", p)
	}
	if info.Size() > maxFileBytes {
		return nil, fmt.Errorf("%q too large (%d bytes)", p, info.Size())
	}
	return fs.ReadFile(w.fsys, name)
}

func (w *localWorkspace) ListDir(p string) ([]domain.DirEntry, error) {
	name, err := w.name(p)
	if err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(w.fsys, name)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DirEntry, 0, len(entries))
	for _, e := range entries {
		if e.Name() == ".git" {
			continue
		}
		out = append(out, domain.DirEntry{Name: e.Name(), IsDir: e.IsDir()})
	}
	return out, nil
}

func (w *localWorkspace) Grep(pattern, pathGlob string) ([]domain.GrepHit, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid grep pattern: %w", err)
	}

	var hits []domain.GrepHit
	walkErr := fs.WalkDir(w.fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if pathGlob != "" {
			if ok, _ := path.Match(pathGlob, name); !ok {
				return nil
			}
		}
		info, err := fs.Stat(w.fsys, name)
		if err != nil || info.IsDir() || info.Size() > maxFileBytes {
			return nil
		}
		f, err := w.fsys.Open(name)
		if err != nil {
			return nil
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		line := 0
		for sc.Scan() {
			line++
			text := sc.Text()
			if re.MatchString(text) {
				hits = append(hits, domain.GrepHit{Path: name, Line: line, Text: strings.TrimSpace(text)})
				if len(hits) >= maxGrepHits {
					return fs.SkipAll
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return hits, nil
}

var _ domain.CodeWorkspace = (*localWorkspace)(nil)
