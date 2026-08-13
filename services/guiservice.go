// Package services: GuiService parses .gui files into a structural preview tree.
package services

import (
	"strconv"
	"strings"

	parser "paradox-modding-tools/services/internal/parser"
)

// GuiNode is a simplified widget node for structural preview.
type GuiNode struct {
	Kind       string            `json:"kind"`
	Name       string            `json:"name,omitempty"`
	Props      map[string]string `json:"props,omitempty"`
	Children   []GuiNode         `json:"children,omitempty"`
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	Text       string            `json:"text,omitempty"`
}

// GuiPreview is the parse result for a .gui file.
type GuiPreview struct {
	Path  string    `json:"path"`
	Roots []GuiNode `json:"roots"`
	Error string    `json:"error,omitempty"`
}

// GuiService builds structural GUI previews from .gui files.
type GuiService struct{}

// PreviewGui parses path and returns a simplified widget tree.
func (g *GuiService) PreviewGui(path string) (*GuiPreview, error) {
	f, err := parser.ParseFile(path)
	if err != nil {
		return &GuiPreview{Path: path, Error: err.Error()}, nil
	}
	var roots []GuiNode
	for _, e := range f.Entries {
		if e.Expression == nil {
			continue
		}
		roots = append(roots, exprToGuiNode(e.Expression)...)
	}
	return &GuiPreview{Path: path, Roots: roots}, nil
}

func exprToGuiNode(ex *parser.Expression) []GuiNode {
	if ex == nil {
		return nil
	}
	key := strings.ToLower(ex.Key)
	if key == "types" && ex.Object != nil {
		var out []GuiNode
		for _, te := range ex.Object.Entries {
			if te.Expression == nil {
				continue
			}
			n := objectToNode(te.Expression)
			n.Kind = "type"
			if te.Expression.Key != "" {
				n.Name = te.Expression.Key
			}
			out = append(out, n)
		}
		return out
	}
	if key == "template" && ex.Object != nil {
		n := objectToNode(ex)
		n.Kind = "template"
		n.Name = firstChildName(ex.Object)
		return []GuiNode{n}
	}
	return []GuiNode{objectToNode(ex)}
}

func firstChildName(obj *parser.Object) string {
	if obj == nil || len(obj.Entries) == 0 || obj.Entries[0].Expression == nil {
		return ""
	}
	return obj.Entries[0].Expression.Key
}

func objectToNode(ex *parser.Expression) GuiNode {
	n := GuiNode{
		Kind:  widgetKind(ex),
		Name:  ex.Key,
		Props: map[string]string{},
	}
	if ex.Object == nil {
		if ex.Literal != nil {
			n.Text = literalString(ex.Literal)
		}
		return n
	}
	for _, e := range ex.Object.Entries {
		if e.Expression == nil {
			continue
		}
		child := e.Expression
		ck := strings.ToLower(child.Key)
		switch ck {
		case "size":
			w, h := parseSize(child)
			n.Width, n.Height = w, h
		case "text", "text_enabled", "tooltip":
			n.Text = literalString(child.Literal)
			n.Props[child.Key] = n.Text
		default:
			if child.Object != nil {
				cn := objectToNode(child)
				if cn.Kind == "unknown" && child.Key != "" {
					cn.Kind = child.Key
				}
				n.Children = append(n.Children, cn)
			} else if child.Literal != nil {
				n.Props[child.Key] = literalString(child.Literal)
			}
		}
	}
	return n
}

func widgetKind(ex *parser.Expression) string {
	if ex.Object == nil {
		return "unknown"
	}
	// type name = widget { ... }  → operator target may be in Value/Object differently;
	// Participle stores right-hand identifier as nested; fall back to key.
	for _, e := range ex.Object.Entries {
		if e.Expression != nil {
			k := strings.ToLower(e.Expression.Key)
			switch k {
			case "widget", "button", "hbox", "vbox", "overlappingcontainer",
				"container", "scrollarea", "fixedgridbox", "flowcontainer",
				"background", "icon", "text_single", "texteditbox":
				return k
			}
		}
	}
	k := strings.ToLower(ex.Key)
	switch k {
	case "widget", "button", "hbox", "vbox", "container", "scrollarea":
		return k
	}
	return "widget"
}

func parseSize(ex *parser.Expression) (int, int) {
	if ex.Object == nil {
		return 0, 0
	}
	var nums []int
	for _, e := range ex.Object.Entries {
		if e.Expression != nil && e.Expression.Literal != nil && e.Expression.Literal.Number != nil {
			nums = append(nums, int(*e.Expression.Literal.Number))
		}
		if e.Expression != nil && e.Expression.Key != "" {
			if n, err := strconv.Atoi(e.Expression.Key); err == nil {
				nums = append(nums, n)
			}
		}
	}
	if len(nums) >= 2 {
		return nums[0], nums[1]
	}
	return 0, 0
}

func literalString(lit *parser.Literal) string {
	if lit == nil {
		return ""
	}
	if lit.String != nil {
		return strings.Trim(*lit.String, `"`)
	}
	if lit.Number != nil {
		return strconv.FormatFloat(*lit.Number, 'f', -1, 64)
	}
	if lit.Boolean != nil {
		return *lit.Boolean
	}
	return ""
}
