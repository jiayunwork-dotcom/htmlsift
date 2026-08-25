package htmlparse

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Doc struct {
	Root *html.Node
}

type Stats struct {
	Elements   int
	TextNodes  int
	Comments   int
	MaxDepth   int
	Tags       map[string]int
	Links      int
	Images     int
	TotalBytes int
}

func Parse(input string) (*Doc, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("htmlparse: empty input")
	}
	root, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return nil, fmt.Errorf("htmlparse: %w", err)
	}
	return &Doc{Root: root}, nil
}

func ParseFragment(input, context string) ([]*html.Node, error) {
	if context == "" {
		context = "body"
	}
	ctx := &html.Node{
		Type:     html.ElementNode,
		Data:     context,
		DataAtom: atom.Lookup([]byte(context)),
	}
	nodes, err := html.ParseFragment(strings.NewReader(input), ctx)
	if err != nil {
		return nil, fmt.Errorf("htmlparse: %w", err)
	}
	return nodes, nil
}

func (d *Doc) Render() (string, error) {
	if d == nil || d.Root == nil {
		return "", fmt.Errorf("htmlparse: nil document")
	}
	var buf bytes.Buffer
	if err := html.Render(&buf, d.Root); err != nil {
		return "", fmt.Errorf("htmlparse: render: %w", err)
	}
	return buf.String(), nil
}

func RenderNode(n *html.Node) (string, error) {
	if n == nil {
		return "", fmt.Errorf("htmlparse: nil node")
	}
	var buf bytes.Buffer
	if err := html.Render(&buf, n); err != nil {
		return "", fmt.Errorf("htmlparse: render node: %w", err)
	}
	return buf.String(), nil
}

func Walk(n *html.Node, fn func(*html.Node) error) error {
	if n == nil {
		return nil
	}
	if err := fn(n); err != nil {
		return err
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if err := Walk(c, fn); err != nil {
			return err
		}
	}
	return nil
}

func (d *Doc) Walk(fn func(*html.Node) error) error {
	if d == nil || d.Root == nil {
		return fmt.Errorf("htmlparse: nil document")
	}
	return Walk(d.Root, fn)
}

func (d *Doc) Stats() Stats {
	st := Stats{Tags: map[string]int{}}
	if d == nil || d.Root == nil {
		return st
	}
	depth := 0
	_ = Walk(d.Root, func(n *html.Node) error {
		switch n.Type {
		case html.ElementNode:
			st.Elements++
			if st.Tags == nil {
				st.Tags = map[string]int{}
			}
			st.Tags[n.Data]++
			if n.Data == "a" {
				st.Links++
			}
			if n.Data == "img" {
				st.Images++
			}
		case html.TextNode:
			st.TextNodes++
			st.TotalBytes += len(n.Data)
		case html.CommentNode:
			st.Comments++
		}
		if depth < nodeDepth(n) {
			depth = nodeDepth(n)
		}
		return nil
	})
	st.MaxDepth = depth
	return st
}

func nodeDepth(n *html.Node) int {
	d := 0
	for p := n.Parent; p != nil; p = p.Parent {
		d++
	}
	return d
}

func hasAttr(n *html.Node, key string) bool {
	for _, a := range n.Attr {
		if a.Key == key {
			return true
		}
	}
	return false
}

func Attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func FindByTag(d *Doc, tag string) []*html.Node {
	var out []*html.Node
	if d == nil || d.Root == nil {
		return out
	}
	_ = Walk(d.Root, func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == tag {
			out = append(out, n)
		}
		return nil
	})
	return out
}

var HiddenTags = map[string]bool{
	"script":   true,
	"style":    true,
	"noscript": true,
	"template": true,
	"head":     true,
}

func VisibleText(d *Doc) string {
	var sb strings.Builder
	var collect func(n *html.Node)
	collect = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && HiddenTags[c.Data] {
				continue
			}
			if c.Type == html.TextNode {
				sb.WriteString(c.Data)
				sb.WriteByte(' ')
			}
			collect(c)
		}
	}
	if d != nil && d.Root != nil {
		collect(d.Root)
	}
	return collapseSpace(sb.String())
}

func CollapseSpace(s string) string {
	return collapseSpace(s)
}

func collapseSpace(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return strings.Join(fields, " ")
}

func ReadAll(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("htmlparse: read: %w", err)
	}
	return string(b), nil
}
