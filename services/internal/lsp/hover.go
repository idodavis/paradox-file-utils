// hover.go resolves the word at a position to overlay def, field-doc prose, or
// cached wiki text.

package lsp

import (
	"strconv"
	"strings"

	"paradox-modding-tools/services/internal/session"
)

// Hover returns hover text at (line, UTF-8 column), or nil when there is none.
func Hover(s *session.Session, path string, line, col int) *HoverResult {
	src := fileText(s, path)
	if src == "" {
		return nil
	}
	off := offsetOf(src, line, col)
	word, start, end := wordAt(src, off)
	if word == "" {
		return nil
	}
	rg := byteRange(src, start, end)
	if d := s.Resolve(word); d != nil {
		body := d.Type + " `" + d.Key + "`"
		if extra := docsFor(s, word); extra != "" {
			body += "\n\n" + extra
		}
		body += "\n\n" + d.Path + ":" + strconv.Itoa(d.Line+1)
		return &HoverResult{Contents: body, Range: &rg}
	}
	if extra := docsFor(s, word); extra != "" {
		return &HoverResult{Contents: extra, Range: &rg}
	}
	if w := s.Wiki(word); w != "" {
		return &HoverResult{Contents: w, Range: &rg}
	}
	return nil
}

func docsFor(s *session.Session, key string) string {
	c := s.Cache()
	if c == nil || c.FieldDocs == nil {
		return ""
	}
	if d := c.FieldDocs[key]; d != "" {
		return d
	}
	return c.FieldDocs[strings.ToLower(key)]
}
