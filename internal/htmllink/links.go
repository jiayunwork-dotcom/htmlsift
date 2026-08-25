package htmllink

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"golang.org/x/net/html"

	"htmlsift/internal/htmlparse"
)

type Link struct {
	Tag      string
	AttrKey  string
	Href     string
	Text     string
	Resolved string
}

type Class int

const (
	ClassUnknown Class = iota
	ClassHTTP
	ClassMailto
	ClassTel
	ClassData
	ClassJavaScript
	ClassFragment
	ClassRelative
)

func (c Class) String() string {
	switch c {
	case ClassHTTP:
		return "http"
	case ClassMailto:
		return "mailto"
	case ClassTel:
		return "tel"
	case ClassData:
		return "data"
	case ClassJavaScript:
		return "javascript"
	case ClassFragment:
		return "fragment"
	case ClassRelative:
		return "relative"
	}
	return "unknown"
}

var extractors = map[string]string{
	"a":      "href",
	"link":   "href",
	"area":   "href",
	"img":    "src",
	"script": "src",
	"iframe": "src",
	"source": "src",
	"video":  "src",
	"audio":  "src",
}

func Extract(d *htmlparse.Doc, base string) ([]Link, error) {
	if d == nil {
		return nil, fmt.Errorf("htmllink: nil document")
	}
	var out []Link
	if err := d.Walk(func(n *html.Node) error {
		if n.Type != html.ElementNode {
			return nil
		}
		attrKey, ok := extractors[n.Data]
		if !ok {
			return nil
		}
		href := htmlparse.Attr(n, attrKey)
		if href == "" {
			return nil
		}
		l := Link{
			Tag:     n.Data,
			AttrKey: attrKey,
			Href:    href,
			Text:    anchorText(n),
		}
		if base != "" {
			r, err := ResolveURL(base, href)
			if err != nil {
				return fmt.Errorf("htmllink: resolve %q against %q: %w", href, base, err)
			}
			l.Resolved = r
		}
		out = append(out, l)
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func anchorText(n *html.Node) string {
	var sb strings.Builder
	collectText(n, &sb)
	return htmlparse.CollapseSpace(sb.String())
}

func collectText(n *html.Node, sb *strings.Builder) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			sb.WriteString(c.Data)
			sb.WriteByte(' ')
		case html.ElementNode:
			collectText(c, sb)
		}
	}
}

func ResolveURL(base, ref string) (string, error) {
	if base == "" {
		return ref, nil
	}
	b, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("htmllink: %w", err)
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("htmllink: %w", err)
	}
	return b.ResolveReference(r).String(), nil
}

func Classify(href string) Class {
	t := strings.TrimSpace(href)
	switch {
	case t == "":
		return ClassUnknown
	case strings.HasPrefix(t, "#"):
		return ClassFragment
	}
	scheme, rest, ok := splitScheme(t)
	if !ok {
		return ClassRelative
	}
	switch strings.ToLower(scheme) {
	case "http", "https":
		return ClassHTTP
	case "mailto":
		return ClassMailto
	case "tel":
		return ClassTel
	case "data":
		return ClassData
	case "javascript", "vbscript", "livescript":
		return ClassJavaScript
	}
	_ = rest
	return ClassUnknown
}

func splitScheme(s string) (scheme, rest string, ok bool) {
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

func FilterByScheme(links []Link, allow ...Class) []Link {
	if len(allow) == 0 {
		return links
	}
	allowed := map[Class]bool{}
	for _, c := range allow {
		allowed[c] = true
	}
	var out []Link
	for _, l := range links {
		c := Classify(l.Href)
		if c == ClassRelative || c == ClassFragment || allowed[c] {
			out = append(out, l)
		}
	}
	return out
}

func UniqueByHref(links []Link) []Link {
	seen := map[string]bool{}
	var out []Link
	for _, l := range links {
		key := l.Href
		if l.Resolved != "" {
			key = l.Resolved
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Tag != out[j].Tag {
			return out[i].Tag < out[j].Tag
		}
		return out[i].Href < out[j].Href
	})
	return out
}

func AbsoluteLinks(links []Link) []Link {
	var out []Link
	for _, l := range links {
		c := Classify(l.Href)
		if c == ClassHTTP {
			out = append(out, l)
		}
	}
	return out
}

func GroupByHost(links []Link) map[string][]Link {
	groups := map[string][]Link{}
	for _, l := range links {
		u, err := url.Parse(l.Resolved)
		if err != nil || u.Host == "" {
			u, err = url.Parse(l.Href)
		}
		host := ""
		if err == nil {
			host = u.Host
		}
		groups[host] = append(groups[host], l)
	}
	return groups
}

func IsSameOrigin(base, ref string) (bool, error) {
	b, err := url.Parse(base)
	if err != nil {
		return false, fmt.Errorf("htmllink: %w", err)
	}
	r, err := url.Parse(ref)
	if err != nil {
		return false, fmt.Errorf("htmllink: %w", err)
	}
	if b.Scheme != r.Scheme {
		return false, nil
	}
	if !strings.EqualFold(b.Host, r.Host) {
		return false, nil
	}
	if b.Port() != r.Port() {
		return false, nil
	}
	return true, nil
}

func ValidBidi(s string) bool {
	for _, r := range s {
		if isBidiControl(r) {
			return false
		}
	}
	return true
}

func isBidiControl(r rune) bool {
	switch {
	case r >= 0x202A && r <= 0x202E:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0x061C:
		return true
	}
	return false
}

func LinkTextOK(text string, maxLen int) bool {
	if text == "" {
		return false
	}
	if maxLen > 0 && len([]rune(text)) > maxLen {
		return false
	}
	return ValidBidi(text)
}
