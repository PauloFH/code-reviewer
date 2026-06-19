package gitfetch

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const (
	maxFileBytes = 512 * 1024 // skip/limit files larger than this for reads/grep
	maxGrepHits  = 200
)

// localWorkspace is a read-only CodeWorkspace backed by a directory on disk.
type localWorkspace struct {
	root string
}

func newLocalWorkspace(root string) *localWorkspace {
	return &localWorkspace{root: root}
}

func (w *localWorkspace) Root() string { return w.root }

// resolve maps a workspace-relative path to an absolute path inside the root,
// rejecting anything that escapes the root.
func (w *localWorkspace) resolve(p string) (string, error) {
	clean := filepath.Clean("/" + filepath.FromSlash(p)) // anchor at root, kill ".."
	full := filepath.Join(w.root, clean)
	rel, err := filepath.Rel(w.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %q escapes workspace", p)
	}
	return full, nil
}

func (w *localWorkspace) ReadFile(path string) ([]byte, error) {
	full, err := w.resolve(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory", path)
	}
	if info.Size() > maxFileBytes {
		return nil, fmt.Errorf("%q too large (%d bytes)", path, info.Size())
	}
	return os.ReadFile(full)
}

func (w *localWorkspace) ListDir(path string) ([]domain.DirEntry, error) {
	full, err := w.resolve(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(full)
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
	walkErr := filepath.WalkDir(w.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(w.root, path)
		if pathGlob != "" {
			ok, _ := filepath.Match(pathGlob, rel)
			if !ok {
				return nil
			}
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxFileBytes {
			return nil
		}
		f, err := os.Open(path)
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
				hits = append(hits, domain.GrepHit{Path: rel, Line: line, Text: strings.TrimSpace(text)})
				if len(hits) >= maxGrepHits {
					return filepath.SkipAll
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

// compile-time check
var _ domain.CodeWorkspace = (*localWorkspace)(nil)
