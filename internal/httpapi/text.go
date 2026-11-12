package httpapi

import (
	"html"
	"regexp"
)

var TagsRE = regexp.MustCompile(`<[^>]*>`)

func Clean(s string) string { return html.UnescapeString(TagsRE.ReplaceAllString(s, "")) }
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

var SlugRE = regexp.MustCompile(`[^\p{L}\p{N}]+`)
