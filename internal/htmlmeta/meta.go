package htmlmeta

import (
	"strings"

	"golang.org/x/net/html"

	"htmlsift/internal/htmlparse"
)

type Meta struct {
	Title       string
	Description string
	Keywords    []string
	Charset     string
	Language    string
	Canonical   string
	Author      string
	Robots      string
	OpenGraph   map[string]string
	Twitter     map[string]string
	OtherMeta   map[string]string
}

func Extract(d *htmlparse.Doc) *Meta {
	m := &Meta{
		OpenGraph: make(map[string]string),
		Twitter:   make(map[string]string),
		OtherMeta: make(map[string]string),
	}
	if d == nil || d.Root == nil {
		return m
	}
	_ = d.Walk(func(n *html.Node) error {
		if n.Type != html.ElementNode {
			return nil
		}
		switch n.Data {
		case "title":
			if text := textContent(n); text != "" {
				m.Title = text
			}
		case "html":
			if lang := htmlparse.Attr(n, "lang"); lang != "" {
				m.Language = lang
			}
		case "meta":
			extractMeta(n, m)
		case "link":
			rel := strings.ToLower(htmlparse.Attr(n, "rel"))
			href := htmlparse.Attr(n, "href")
			if rel == "canonical" && href != "" {
				m.Canonical = href
			}
		}
		return nil
	})
	return m
}

func extractMeta(n *html.Node, m *Meta) {
	name := strings.ToLower(htmlparse.Attr(n, "name"))
	property := strings.ToLower(htmlparse.Attr(n, "property"))
	content := htmlparse.Attr(n, "content")
	charset := htmlparse.Attr(n, "charset")

	if charset != "" {
		m.Charset = charset
		return
	}
	httpEquiv := strings.ToLower(htmlparse.Attr(n, "http-equiv"))
	if httpEquiv == "content-type" && content != "" {
		if idx := strings.Index(strings.ToLower(content), "charset="); idx >= 0 {
			m.Charset = strings.TrimSpace(content[idx+8:])
		}
		return
	}

	switch name {
	case "description":
		m.Description = content
	case "keywords":
		m.Keywords = splitKeywords(content)
	case "author":
		m.Author = content
	case "robots":
		m.Robots = content
	default:
		if name != "" {
			m.OtherMeta[name] = content
		}
	}

	if strings.HasPrefix(property, "og:") {
		m.OpenGraph[property] = content
	} else if strings.HasPrefix(property, "twitter:") {
		m.Twitter[property] = content
	}
}

func splitKeywords(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return strings.TrimSpace(sb.String())
}

func (m *Meta) HasOpenGraph() bool { return len(m.OpenGraph) > 0 }

func (m *Meta) HasTwitterCard() bool { return len(m.Twitter) > 0 }

func (m *Meta) OGTitle() string {
	if t, ok := m.OpenGraph["og:title"]; ok && t != "" {
		return t
	}
	return m.Title
}

func (m *Meta) OGDescription() string {
	if d, ok := m.OpenGraph["og:description"]; ok && d != "" {
		return d
	}
	return m.Description
}
