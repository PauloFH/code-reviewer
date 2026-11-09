package ai

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const maxDiffBytes = 80_000

// maxInlineComments caps the inline comments of one review, after validation.
const maxInlineComments = 20

var irrelevantPatterns = []string{
	"package-lock.json",
	"pnpm-lock.yaml",
	"yarn.lock",
	"go.sum",
	"Cargo.lock",
	"composer.lock",
	"dist/",
	"build/",
	"target/",
	"node_modules/",
	"vendor/",
	".min.js",
	".min.css",
	".map",
	".pb.go",
	"_generated",
}

type diffLineKind int

const (
	lineOther      diffLineKind = iota // outside any hunk (diff --git, index, blank, ...)
	lineFileHeader                     // "--- old" / "+++ new" outside a hunk
	lineHunkHeader
	lineContext
	lineAdded
	lineRemoved
	lineNoNewline // "\ No newline at end of file": advances no counter
)

type diffLine struct {
	text  string
	kind  diffLineKind
	path  string // current file; the new-side path once "+++" is seen ("" for /dev/null)
	oldNo int
	newNo int
}

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// walkDiff is the single diff line model: annotation, filtering and inline
// comment validation all read the diff through it. Hunk bodies are delimited
// by the line counts in the "@@" header, so a removed line starting with "-- "
// (shown as "--- x") or an added "++ x" is never mistaken for a file header.
func walkDiff(diff string, fn func(diffLine)) {
	var path string
	var oldNo, newNo, oldLeft, newLeft int
	for _, text := range strings.Split(diff, "\n") {
		l := diffLine{text: text, kind: lineOther}
		inHunk := oldLeft > 0 || newLeft > 0
		switch {
		case strings.HasPrefix(text, "@@ "):
			l.kind = lineHunkHeader
			oldNo, newNo, oldLeft, newLeft = parseHunkHeader(text)
		case inHunk && strings.HasPrefix(text, `\`):
			l.kind = lineNoNewline
		case inHunk && strings.HasPrefix(text, "+"):
			l.kind, l.newNo = lineAdded, newNo
			newNo++
			newLeft--
		case inHunk && strings.HasPrefix(text, "-"):
			l.kind, l.oldNo = lineRemoved, oldNo
			oldNo++
			oldLeft--
		case inHunk && (text == "" || strings.HasPrefix(text, " ")):
			l.kind, l.oldNo, l.newNo = lineContext, oldNo, newNo
			oldNo++
			newNo++
			oldLeft--
			newLeft--
		case inHunk:
			oldLeft, newLeft = 0, 0 // not a hunk body line: the hunk is over
		}
		if l.kind == lineOther {
			if p, ok := strings.CutPrefix(text, "--- "); ok {
				l.kind, path = lineFileHeader, diffPath(p)
			} else if p, ok := strings.CutPrefix(text, "+++ "); ok {
				l.kind = lineFileHeader
				if p := diffPath(p); p != "" {
					path = p
				}
			} else if strings.HasPrefix(text, "diff --git ") {
				path = ""
			}
		}
		l.path = path
		fn(l)
	}
}

// parseHunkHeader returns the starting line numbers and the line counts of a
// "@@ -a,b +c,d @@" header (an omitted count is 1). A malformed header yields
// zero counts, i.e. no hunk body.
func parseHunkHeader(header string) (oldStart, newStart, oldCount, newCount int) {
	m := hunkHeaderPattern.FindStringSubmatch(header)
	if m == nil {
		return 0, 0, 0, 0
	}
	num := func(s string) int {
		if s == "" {
			return 1
		}
		n, _ := strconv.Atoi(s)
		return n
	}
	return num(m[1]), num(m[3]), num(m[2]), num(m[4])
}

// diffPath normalises a header or comment path: strips "a/" / "b/", maps /dev/null to "".
func diffPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "/dev/null" {
		return ""
	}
	return strings.TrimPrefix(strings.TrimPrefix(p, "a/"), "b/")
}

// reviewDiff is the exact diff text the model sees, before annotation:
// irrelevant files removed, then truncated to maxDiffBytes.
func reviewDiff(raw string) string {
	diff := filterDiff(raw)
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes] + "\n\n[diff truncado por exceder o limite de tamanho]"
	}
	return diff
}

// filterDiff drops the files that are noise for a review (lockfiles, generated
// code, build artefacts).
func filterDiff(raw string) string {
	var b strings.Builder
	skip := false
	walkDiff(raw, func(l diffLine) {
		if l.kind == lineFileHeader {
			skip = isIrrelevant(l.path)
		}
		if !skip {
			b.WriteString(l.text)
			b.WriteByte('\n')
		}
	})
	return strings.TrimSpace(b.String())
}

func isIrrelevant(path string) bool {
	base := filepath.Base(path)
	for _, pat := range irrelevantPatterns {
		switch {
		case strings.HasSuffix(pat, "/"):
			if strings.Contains(path, pat) {
				return true
			}
		case strings.HasPrefix(pat, "."):
			if strings.HasSuffix(base, pat) {
				return true
			}
		default:
			if base == pat || strings.Contains(path, pat) {
				return true
			}
		}
	}
	return false
}

// annotateDiffLineNumbers prefixes hunk lines with [new:N]/[old:N] markers so the
// model can cite exact new-side lines in inline comments.
func annotateDiffLineNumbers(diff string) string {
	if diff == "" {
		return ""
	}
	var b strings.Builder
	walkDiff(diff, func(l diffLine) {
		switch l.kind {
		case lineAdded:
			fmt.Fprintf(&b, "+[new:%d] %s\n", l.newNo, l.text[1:])
		case lineRemoved:
			fmt.Fprintf(&b, "-[old:%d] %s\n", l.oldNo, l.text[1:])
		case lineContext:
			fmt.Fprintf(&b, " [old:%d new:%d]%s\n", l.oldNo, l.newNo, strings.TrimPrefix(l.text, " "))
		default:
			b.WriteString(l.text)
			b.WriteByte('\n')
		}
	})
	return strings.TrimSpace(b.String())
}

// validateInlineComments keeps comments that point at an added line of diff
// (the exact text the model saw), normalising paths, dropping duplicates and
// capping at maxInlineComments.
func validateInlineComments(comments []domain.InlineComment, diff string) []domain.InlineComment {
	if len(comments) == 0 {
		return nil
	}
	added := map[string]map[int]bool{}
	walkDiff(diff, func(l diffLine) {
		if l.kind == lineAdded && l.path != "" {
			if added[l.path] == nil {
				added[l.path] = map[int]bool{}
			}
			added[l.path][l.newNo] = true
		}
	})

	var out []domain.InlineComment
	seen := map[domain.InlineComment]bool{}
	for _, c := range comments {
		c.Path = diffPath(c.Path)
		c.Body = strings.TrimSpace(c.Body)
		if c.Body == "" || !added[c.Path][c.Line] || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
		if len(out) == maxInlineComments {
			break
		}
	}
	return out
}
