// hover.go copies session.Inspect into the Wails hover DTO.

package lsp

import "paradox-modding-tools/services/internal/session"

// Hover returns the card at (line, UTF-8 column), or nil when there is none.
func Hover(s *session.Session, path string, line, col int) *HoverResult {
	ins := s.Inspect(path, line, col)
	if ins == nil || ins.Kind == "" {
		return nil
	}
	return hoverFromInspect(ins)
}

func hoverFromInspect(ins *session.Inspect) *HoverResult {
	h := &HoverResult{
		Kind:              ins.Kind,
		Key:               ins.Key,
		Hint:              ins.Hint,
		Docs:              ins.Docs,
		Body:              ins.Body,
		More:              ins.More,
		Owner:             ins.Owner,
		Usage:             ins.Usage,
		Origin:            ins.Origin,
		OriginName:        ins.OriginName,
		Rel:               ins.Rel,
		Path:              ins.Path,
		Line:              ins.Line,
		Col:               ins.Col,
		VanillaOriginName: ins.VanillaOriginName,
		VanillaRel:        ins.VanillaRel,
		VanillaPath:       ins.VanillaPath,
		VanillaLine:       ins.VanillaLine,
		VanillaCol:        ins.VanillaCol,
	}
	if len(ins.Values) > 0 {
		h.Values = make([]HoverValue, len(ins.Values))
		for i, v := range ins.Values {
			h.Values[i] = HoverValue{Text: v.Text, Count: v.Count}
		}
	}
	return h
}
