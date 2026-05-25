package usecase

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/paulofh/code-reviewer/internal/domain"
)

const maxInlineCommentsPerReview = 20

var inlineHunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func filterInlineComments(comments []domain.InlineComment, diff string) []domain.InlineComment {
	if len(comments) == 0 || diff == "" {
		return nil
	}

	changed := changedNewLines(diff)
	filtered := make([]domain.InlineComment, 0, len(comments))
	seen := make(map[string]struct{})

	for _, comment := range comments {
		comment.Path = normalizeInlinePath(comment.Path)
		comment.Body = strings.TrimSpace(comment.Body)
		if comment.Path == "" || comment.Line <= 0 || comment.Body == "" {
			continue
		}
		lines := changed[comment.Path]
		if _, ok := lines[comment.Line]; !ok {
			continue
		}
		key := comment.Path + ":" + strconv.Itoa(comment.Line) + ":" + comment.Body
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, comment)
		if len(filtered) >= maxInlineCommentsPerReview {
			break
		}
	}

	return filtered
}

func changedNewLines(diff string) map[string]map[int]struct{} {
	changed := make(map[string]map[int]struct{})
	var currentPath string
	var oldLine, newLine int
	inHunk := false

	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			currentPath = normalizeInlinePath(strings.TrimSpace(strings.TrimPrefix(line, "+++ ")))
			inHunk = false
			continue
		case strings.HasPrefix(line, "@@ "):
			oldLine, newLine = parseInlineHunkStart(line)
			inHunk = oldLine > 0 || newLine > 0
			continue
		}

		if !inHunk || currentPath == "" || strings.HasPrefix(line, `\ No newline`) {
			continue
		}

		switch {
		case strings.HasPrefix(line, "+"):
			if changed[currentPath] == nil {
				changed[currentPath] = make(map[int]struct{})
			}
			changed[currentPath][newLine] = struct{}{}
			newLine++
		case strings.HasPrefix(line, "-"):
			oldLine++
		default:
			oldLine++
			newLine++
		}
	}

	return changed
}

func parseInlineHunkStart(header string) (oldStart, newStart int) {
	matches := inlineHunkHeaderPattern.FindStringSubmatch(header)
	if len(matches) != 3 {
		return 0, 0
	}
	oldStart, _ = strconv.Atoi(matches[1])
	newStart, _ = strconv.Atoi(matches[2])
	return oldStart, newStart
}

func normalizeInlinePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	if path == "/dev/null" {
		return ""
	}
	return path
}
