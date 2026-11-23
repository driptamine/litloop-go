package httpapi

import (
	"html"
	"regexp"
)

var TagsRE = regexp.MustCompile(`<[^>]*>`)

func Clean(text string) string	{ return html.UnescapeString(TagsRE.ReplaceAllString(text, "")) }
func Truncate(text string, length int) string {
	runes := []rune(text)
	if len(runes) > length {
		return string(runes[:length])
	}
	return text
}

var SlugRE = regexp.MustCompile(`[^\p{L}\p{N}]+`)
