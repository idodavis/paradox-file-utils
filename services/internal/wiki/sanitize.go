// sanitize.go allowlists wiki HTML and strips chrome (TOC, cites, [top] links).
package wiki

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	moddingHead = regexp.MustCompile(`(?i)^(user )?modding$`)
	dropClass   = regexp.MustCompile(`(?i)(navbox|thumb|infobox|mw-editsection|noprint|metadata|\btoc\b|mw-portlet-toc|sidebar-toc|\breferences?\b)`)
	dropHead    = regexp.MustCompile(`(?i)^(table of )?contents$|^see also$|^references$|^external links$|^further reading$`)
	moveHead    = regexp.MustCompile(`(?i)^scripting tools$`)
	citeFrag    = regexp.MustCompile(`(?i)^(cite_note|cite_ref)`)
	citeMark    = regexp.MustCompile(`^(\[\d+\])+$`)
)

var allowed = map[atom.Atom]bool{
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true,
	atom.P: true, atom.Br: true, atom.Hr: true, atom.Blockquote: true,
	atom.Ul: true, atom.Ol: true, atom.Li: true, atom.Dl: true, atom.Dt: true, atom.Dd: true,
	atom.Table: true, atom.Thead: true, atom.Tbody: true, atom.Tfoot: true,
	atom.Tr: true, atom.Th: true, atom.Td: true, atom.Caption: true,
	atom.Pre: true, atom.Code: true, atom.A: true,
	atom.Strong: true, atom.Em: true, atom.B: true, atom.I: true,
	atom.Sup: true, atom.Sub: true, atom.Small: true,
}

func classOf(n *html.Node) string {
	for _, a := range n.Attr {
		if a.Key == "class" {
			return a.Val
		}
	}
	return ""
}

func dropNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.Img, atom.Figure, atom.Style, atom.Script, atom.Noscript, atom.Link, atom.Meta:
		return true
	}
	if strings.EqualFold(attrVal(n, "id"), "toc") || citeID(attrVal(n, "id")) {
		return true
	}
	if citeMarker(n) || topLink(n) || (n.DataAtom == atom.A && citeHref(attrVal(n, "href"))) {
		return true
	}
	return dropClass.MatchString(classOf(n))
}

func citeID(id string) bool {
	return citeFrag.MatchString(id)
}

func citeHref(href string) bool {
	h := strings.TrimSpace(href)
	if i := strings.IndexByte(h, '#'); i >= 0 {
		h = h[i+1:]
	}
	return citeID(h)
}

func pruneCites(root *html.Node) {
	if root == nil {
		return
	}
	var next *html.Node
	for c := root.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if dropCiteNode(c) {
			root.RemoveChild(c)
			continue
		}
		pruneCites(c)
		if emptySup(c) {
			root.RemoveChild(c)
		}
	}
}

func citeMarker(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	switch n.DataAtom {
	case atom.A, atom.Sup:
		return citeMark.MatchString(strings.TrimSpace(nodeText(n)))
	default:
		return false
	}
}

func dropCiteNode(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if citeID(attrVal(n, "id")) || citeMarker(n) || topLink(n) {
		return true
	}
	return n.DataAtom == atom.A && citeHref(attrVal(n, "href"))
}

// topLink is a MediaWiki "[top]" / href="#top" jump, not a heading named Top.
func topLink(n *html.Node) bool {
	if n.Type != html.ElementNode || n.DataAtom != atom.A {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(nodeText(n)), "[top]") {
		return true
	}
	h := attrVal(n, "href")
	if i := strings.IndexByte(h, '#'); i >= 0 {
		return strings.EqualFold(h[i+1:], "top")
	}
	return false
}

func emptySup(n *html.Node) bool {
	if n.Type != html.ElementNode || n.DataAtom != atom.Sup {
		return false
	}
	return strings.TrimSpace(nodeText(n)) == ""
}

func rewriteHref(href, wikiBase string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "data:") {
		return ""
	}
	if strings.HasPrefix(href, "#") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/wiki/") {
		return strings.TrimRight(wikiBase, "/") + "/" + strings.TrimPrefix(href, "/wiki/")
	}
	if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
		return strings.TrimRight(wikiBase, "/") + href
	}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	return ""
}

const dumpTableRows = 20

func countTr(n *html.Node) int {
	if n == nil {
		return 0
	}
	ntr := 0
	if n.Type == html.ElementNode && n.DataAtom == atom.Tr {
		ntr = 1
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		ntr += countTr(c)
	}
	return ntr
}

func appendAllowed(dst, n *html.Node, wikiBase string) {
	if n == nil || dropNode(n) {
		return
	}
	switch n.Type {
	case html.TextNode:
		dst.AppendChild(&html.Node{Type: html.TextNode, Data: n.Data})
	case html.ElementNode:
		if n.DataAtom == atom.Table && countTr(n) >= dumpTableRows {
			return
		}
		if n.DataAtom == atom.Span || !allowed[n.DataAtom] {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				appendAllowed(dst, c, wikiBase)
			}
			return
		}
		out := &html.Node{Type: html.ElementNode, Data: n.Data, DataAtom: n.DataAtom}
		if headingRank(n) > 0 {
			if id := firstID(n); id != "" {
				out.Attr = []html.Attribute{{Key: "id", Val: id}}
			}
		}
		if n.DataAtom == atom.A {
			for _, a := range n.Attr {
				if a.Key == "href" {
					if h := rewriteHref(a.Val, wikiBase); h != "" {
						out.Attr = []html.Attribute{{Key: "href", Val: h}}
					}
					break
				}
			}
		}
		if n.DataAtom == atom.Td || n.DataAtom == atom.Th {
			for _, a := range n.Attr {
				if a.Key == "colspan" || a.Key == "rowspan" {
					out.Attr = append(out.Attr, a)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			appendAllowed(out, c, wikiBase)
		}
		dst.AppendChild(out)
	default:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			appendAllowed(dst, c, wikiBase)
		}
	}
}

func renderChildren(n *html.Node) string {
	if n == nil {
		return ""
	}
	var buf bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&buf, c)
	}
	return strings.TrimSpace(buf.String())
}

func findBody(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && n.DataAtom == atom.Body {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if b := findBody(c); b != nil {
			return b
		}
	}
	return n
}

func findOutput(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && strings.Contains(classOf(n), "mw-parser-output") {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if o := findOutput(c); o != nil {
			return o
		}
	}
	return nil
}

// sanitize returns allowlisted HTML (no images) and <pre> snippet texts.
func sanitize(raw, wikiBase string) (clean string, snippets []string) {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return "", nil
	}
	src := findOutput(doc)
	if src == nil {
		src = findBody(doc)
	}
	dst := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	if src != nil {
		for c := src.FirstChild; c != nil; c = c.NextSibling {
			appendAllowed(dst, c, wikiBase)
		}
	}
	reshapeTree(dst)
	return renderChildren(dst), collectPre(dst)
}

// reshape drops leftover wiki chrome and moves Scripting Tools to the end.
func reshape(cleanHTML string) (string, []Section) {
	root := fragmentRoot(cleanHTML)
	if root == nil {
		return "", nil
	}
	reshapeTree(root)
	secs := sectionsFrom(root)
	return renderChildren(root), secs
}

func takeChildren(root *html.Node) []*html.Node {
	var kids []*html.Node
	for c := root.FirstChild; c != nil; {
		n := c
		c = c.NextSibling
		root.RemoveChild(n)
		kids = append(kids, n)
	}
	return kids
}

func reshapeTree(root *html.Node) {
	if root == nil {
		return
	}
	kids := takeChildren(root)
	var keep, moved [][]*html.Node
	for i := 0; i < len(kids); {
		n := kids[i]
		r := headingRank(n)
		if r == 0 {
			keep = append(keep, []*html.Node{n})
			i++
			continue
		}
		end := i + 1
		for end < len(kids) {
			if er := headingRank(kids[end]); er > 0 && er <= r {
				break
			}
			end++
		}
		block := kids[i:end]
		line := headingLine(n)
		switch {
		case dropHead.MatchString(line):
		case moveHead.MatchString(line):
			moved = append(moved, block)
		default:
			keep = append(keep, block)
		}
		i = end
	}
	for _, blocks := range [][][]*html.Node{keep, moved} {
		for _, b := range blocks {
			for _, n := range b {
				root.AppendChild(n)
			}
		}
	}
	pruneCites(root)
}

func sectionsFrom(root *html.Node) []Section {
	if root == nil {
		return nil
	}
	var out []Section
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if r := headingRank(n); r > 0 {
			line := headingLine(n)
			if line == "" {
				return
			}
			id := firstID(n)
			if id == "" {
				id = strings.ReplaceAll(strings.TrimSpace(line), " ", "_")
				n.Attr = append(n.Attr, html.Attribute{Key: "id", Val: id})
			}
			out = append(out, Section{TocLevel: r, Line: line, Anchor: id})
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

func collectPre(n *html.Node) []string {
	var out []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Pre {
			text := strings.TrimSpace(nodeText(n))
			if text != "" {
				out = append(out, text)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}

func nodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(nodeText(c))
	}
	return b.String()
}

func headingRank(n *html.Node) int {
	if n.Type != html.ElementNode {
		return 0
	}
	switch n.DataAtom {
	case atom.H1:
		return 1
	case atom.H2:
		return 2
	case atom.H3:
		return 3
	case atom.H4:
		return 4
	case atom.H5:
		return 5
	case atom.H6:
		return 6
	}
	return 0
}

func headingLine(n *html.Node) string {
	return strings.TrimSpace(nodeText(n))
}

func fragmentRoot(htmlSrc string) *html.Node {
	nodes, err := html.ParseFragment(strings.NewReader(htmlSrc), &html.Node{
		Type:     html.ElementNode,
		Data:     "div",
		DataAtom: atom.Div,
	})
	if err != nil {
		return nil
	}
	root := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	return root
}

// extractModding keeps the Modding / User modding heading and deeper children.
func extractModding(cleanHTML string) string {
	root := fragmentRoot(cleanHTML)
	if root == nil {
		return ""
	}
	dst := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	collecting := false
	rank := 0
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if r := headingRank(c); r > 0 {
			line := strings.TrimSpace(headingLine(c))
			if !collecting && moddingHead.MatchString(line) {
				collecting = true
				rank = r
				dst.AppendChild(cloneTree(c))
				continue
			}
			if collecting && r <= rank {
				break
			}
		}
		if collecting {
			dst.AppendChild(cloneTree(c))
		}
	}
	return renderChildren(dst)
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func firstID(n *html.Node) string {
	if id := attrVal(n, "id"); id != "" {
		return id
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if id := firstID(c); id != "" {
			return id
		}
	}
	return ""
}

func cloneTree(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	out := &html.Node{
		Type:      n.Type,
		Data:      n.Data,
		DataAtom:  n.DataAtom,
		Namespace: n.Namespace,
	}
	if n.Attr != nil {
		out.Attr = append([]html.Attribute(nil), n.Attr...)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out.AppendChild(cloneTree(c))
	}
	return out
}
