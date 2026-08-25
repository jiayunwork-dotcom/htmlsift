package htmlsanitize

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/unicode/norm"

	"htmlsift/internal/htmlparse"
)

type Policy struct {
	Elements           map[string]bool
	Attrs              map[string]map[string]bool
	Schemes            map[string]bool
	AllowDataImages    bool
	StripComments      bool
	RequireRelNofollow bool
}

var safeImageDataTypes = []string{"image/png", "image/jpeg", "image/gif", "image/webp"}

func DefaultPolicy() Policy {
	elements := []string{
		"p", "div", "span", "br", "hr",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"b", "i", "u", "s", "em", "strong", "small", "sub", "sup", "mark",
		"ul", "ol", "li", "dl", "dt", "dd",
		"blockquote", "pre", "code", "kbd", "samp", "var",
		"table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption",
		"a", "img", "figure", "figcaption",
		"section", "article", "aside", "header", "footer", "nav", "main",
		"details", "summary", "time",
	}
	el := map[string]bool{}
	for _, e := range elements {
		el[e] = true
	}
	attrs := map[string]map[string]bool{
		"*": {"title": true, "class": true, "id": true, "lang": true},
		"a": {
			"href": true, "rel": true, "target": true,
			"download": true, "hreflang": true,
		},
		"img":        {"src": true, "alt": true, "width": true, "height": true, "loading": true},
		"source":     {"src": true, "type": true, "srcset": true, "media": true},
		"td":         {"colspan": true, "rowspan": true, "align": true},
		"th":         {"colspan": true, "rowspan": true, "align": true, "scope": true},
		"ol":         {"start": true, "reversed": true},
		"time":       {"datetime": true},
		"blockquote": {"cite": true},
		"details":    {"open": true},
	}
	return Policy{
		Elements:           el,
		Attrs:              attrs,
		Schemes:            map[string]bool{"http": true, "https": true, "mailto": true, "tel": true},
		AllowDataImages:    true,
		StripComments:      true,
		RequireRelNofollow: true,
	}
}

type Report struct {
	RemovedElements int
	RemovedAttrs    int
	RemovedURLs     int
	KeptElements    int
	TextBytes       int
}

func (p Policy) SanitizeReport(input string) (string, Report, error) {
	d, err := htmlparse.Parse(input)
	if err != nil {
		return "", Report{}, fmt.Errorf("htmlsanitize: %w", err)
	}
	rep := &Report{}
	if err := apply(p, d, rep); err != nil {
		return "", Report{}, fmt.Errorf("htmlsanitize: %w", err)
	}
	out, err := d.Render()
	if err != nil {
		return "", Report{}, fmt.Errorf("htmlsanitize: %w", err)
	}
	return out, *rep, nil
}

func (p Policy) Sanitize(input string) (string, error) {
	d, err := htmlparse.Parse(input)
	if err != nil {
		return "", fmt.Errorf("htmlsanitize: %w", err)
	}
	if err := p.Apply(d); err != nil {
		return "", fmt.Errorf("htmlsanitize: %w", err)
	}
	out, err := d.Render()
	if err != nil {
		return "", fmt.Errorf("htmlsanitize: %w", err)
	}
	return out, nil
}

func (p Policy) SanitizeFragment(input string) (string, error) {
	nodes, err := htmlparse.ParseFragment(input, "body")
	if err != nil {
		return "", fmt.Errorf("htmlsanitize: %w", err)
	}
	var sb strings.Builder
	for _, n := range nodes {
		clean := sanitizeNode(p, n, nil)
		if clean == nil {
			continue
		}
		s, err := htmlparse.RenderNode(clean)
		if err != nil {
			return "", fmt.Errorf("htmlsanitize: %w", err)
		}
		sb.WriteString(s)
	}
	return sb.String(), nil
}

func (p Policy) Apply(d *htmlparse.Doc) error {
	return apply(p, d, nil)
}

func apply(p Policy, d *htmlparse.Doc, rep *Report) error {
	if d == nil || d.Root == nil {
		return fmt.Errorf("htmlsanitize: nil document")
	}
	var walk func(n *html.Node) error
	walk = func(n *html.Node) error {
		if n.Type == html.ElementNode && isWrapper(n.Data) {
			for c := n.FirstChild; c != nil; {
				next := c.NextSibling
				if err := walk(c); err != nil {
					return err
				}
				c = next
			}
			return nil
		}
		clean := sanitizeNode(p, n, rep)
		if clean == nil {
			if rep != nil {
				rep.RemovedElements++
			}
			if n.Parent != nil {
				n.Parent.RemoveChild(n)
			}
			return nil
		}
		if rep != nil && n.Type == html.ElementNode {
			rep.KeptElements++
		}
		if clean != n && n.Parent != nil {
			n.Parent.InsertBefore(clean, n)
			n.Parent.RemoveChild(n)
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			if err := walk(c); err != nil {
				return err
			}
			c = next
		}
		return nil
	}
	return walk(d.Root)
}

func isWrapper(name string) bool {
	switch name {
	case "html", "head", "body":
		return true
	}
	return false
}

func sanitizeNode(p Policy, n *html.Node, rep *Report) *html.Node {
	switch n.Type {
	case html.TextNode:
		n.Data = norm.NFC.String(n.Data)
		if rep != nil {
			rep.TextBytes += len(n.Data)
		}
		return n
	case html.CommentNode:
		if p.StripComments {
			return nil
		}
		return n
	case html.DoctypeNode:
		return n
	case html.ElementNode:
		return sanitizeElement(p, n, rep)
	}
	return n
}

func sanitizeElement(p Policy, n *html.Node, rep *Report) *html.Node {
	name := strings.ToLower(n.Data)
	if !p.Elements[name] {
		return unwrapDisallowed(p, n, rep)
	}
	n.Data = name

	allowed := p.Attrs[name]
	global := p.Attrs["*"]
	kept := make([]html.Attribute, 0, len(n.Attr))
	for _, a := range n.Attr {
		key := strings.ToLower(a.Key)
		if isEventAttr(key) {
			if rep != nil {
				rep.RemovedAttrs++
			}
			continue
		}
		if !(allowed[key] || global[key]) {
			if rep != nil {
				rep.RemovedAttrs++
			}
			continue
		}
		if !p.attrValueSafe(name, key, a.Val) {
			if rep != nil {
				rep.RemovedURLs++
			}
			continue
		}
		kept = append(kept, html.Attribute{Key: a.Key, Val: norm.NFC.String(a.Val)})
	}
	if name == "a" && p.RequireRelNofollow {
		hasHref := false
		hasRel := false
		for _, a := range kept {
			if a.Key == "href" {
				hasHref = true
			}
			if a.Key == "rel" {
				hasRel = true
				if !strings.Contains(strings.ToLower(a.Val), "nofollow") {
					a.Val = a.Val + " nofollow"
					for i := range kept {
						if kept[i].Key == "rel" {
							kept[i].Val = a.Val
							break
						}
					}
				}
			}
		}
		if hasHref && !hasRel {
			kept = append(kept, html.Attribute{Key: "rel", Val: "nofollow"})
		}
	}
	n.Attr = kept
	return n
}

func isPhrasing(name string) bool {
	switch name {
	case "a", "span", "b", "i", "u", "s", "em", "strong", "small",
		"sub", "sup", "mark", "abbr", "q", "cite", "code", "kbd", "time":
		return true
	}
	return false
}

func isEventAttr(key string) bool {
	if strings.HasPrefix(key, "on") {
		return true
	}
	switch key {
	case "style", "formaction", "action", "srcdoc", "xlink:href":
		return true
	}
	return false
}

func (p Policy) attrValueSafe(element, key, val string) bool {
	val = strings.TrimSpace(val)
	if val == "" {
		return true
	}
	switch key {
	case "href", "src", "cite", "poster":
		if !p.urlAllowed(val, element, key) {
			return false
		}
	case "srcset":
		for _, part := range strings.Split(val, ",") {
			fields := strings.Fields(part)
			if len(fields) == 0 {
				continue
			}
			if !p.urlAllowed(fields[0], element, key) {
				return false
			}
		}
	}
	return true
}

func (p Policy) urlAllowed(v, element, key string) bool {
	v = strings.TrimSpace(v)
	if v == "" || strings.HasPrefix(v, "#") || strings.HasPrefix(v, "/") {
		return true
	}
	scheme, rest, ok := splitScheme(v)
	if !ok {
		return true
	}
	low := strings.ToLower(scheme)
	if p.Schemes[low] {
		return true
	}
	if low == "data" && p.AllowDataImages && element == "img" && key == "src" {
		return dataImageAllowed(rest)
	}
	return false
}

func dataImageAllowed(rest string) bool {
	head := strings.SplitN(rest, ";", 2)[0]
	head = strings.ToLower(strings.TrimSpace(head))
	for _, t := range safeImageDataTypes {
		if head == t {
			return true
		}
	}
	return false
}

func splitScheme(s string) (string, string, bool) {
	idx := strings.IndexByte(s, ':')
	if idx <= 0 {
		return "", "", false
	}
	head := s[:idx]
	for i := 0; i < len(head); i++ {
		c := head[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z':
		case i > 0 && c >= '0' && c <= '9':
		case i > 0 && (c == '+' || c == '-' || c == '.'):
		default:
			return "", "", false
		}
	}
	return head, s[idx+1:], true
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	var rec func(*html.Node)
	rec = func(x *html.Node) {
		if x.Type == html.TextNode {
			sb.WriteString(x.Data)
			sb.WriteByte(' ')
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			rec(c)
		}
	}
	rec(n)
	return sb.String()
}

func unwrapDisallowed(p Policy, n *html.Node, rep *Report) *html.Node {
	parent := n.Parent
	if parent == nil {
		return nil
	}
	if isPhrasing(n.Data) {
		text := htmlparse.CollapseSpace(textContent(n))
		if text == "" {
			return nil
		}
		t := &html.Node{Type: html.TextNode, Data: norm.NFC.String(text)}
		parent.InsertBefore(t, n)
		parent.RemoveChild(n)
		return nil
	}
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		n.RemoveChild(c)
		parent.InsertBefore(c, n)
		c = next
	}
	parent.RemoveChild(n)
	if rep != nil {
		rep.RemovedElements++
	}
	return nil
}
