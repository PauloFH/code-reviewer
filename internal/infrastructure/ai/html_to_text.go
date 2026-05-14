package ai

import (
	"regexp"
	"strings"
)

var (
	reTag     = regexp.MustCompile(`<[^>]+>`)
	reMultiNL = regexp.MustCompile(`\n{3,}`)
)

func htmlToPlainText(html string) string {
	s := html

	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "</h1>", "\n")
	s = strings.ReplaceAll(s, "</h2>", "\n")
	s = strings.ReplaceAll(s, "</h3>", "\n")
	s = strings.ReplaceAll(s, "</h4>", "\n")
	s = strings.ReplaceAll(s, "</li>", "\n")
	s = strings.ReplaceAll(s, "</tr>", "\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")

	s = strings.ReplaceAll(s, "<li>", "- ")

	s = reTag.ReplaceAllString(s, "")

	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")

	s = reMultiNL.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
