package htmlmeta

import (
	"encoding/json"
	"strings"

	"golang.org/x/net/html"

	"htmlsift/internal/htmlparse"
)

type JSONLD struct {
	Type    string
	Context string
	Raw     map[string]interface{}
}

func ExtractJSONLD(d *htmlparse.Doc) []JSONLD {
	if d == nil || d.Root == nil {
		return nil
	}
	var results []JSONLD
	scripts := htmlparse.FindByTag(d, "script")
	for _, s := range scripts {
		if htmlparse.Attr(s, "type") != "application/ld+json" {
			continue
		}
		text := scriptText(s)
		if text == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(text), &obj); err != nil {
			continue
		}
		ld := JSONLD{Raw: obj}
		if t, ok := obj["@type"].(string); ok {
			ld.Type = t
		}
		if c, ok := obj["@context"].(string); ok {
			ld.Context = c
		}
		results = append(results, ld)
	}
	return results
}

func scriptText(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return strings.TrimSpace(sb.String())
}

func HasJSONLD(d *htmlparse.Doc) bool {
	return len(ExtractJSONLD(d)) > 0
}

type MicrodataItem struct {
	ItemType   string
	Properties map[string]string
}

func ExtractMicrodata(d *htmlparse.Doc) []MicrodataItem {
	if d == nil || d.Root == nil {
		return nil
	}
	var items []MicrodataItem
	_ = d.Walk(func(n *html.Node) error {
		if n.Type != html.ElementNode {
			return nil
		}
		if !hasItemScope(n) {
			return nil
		}
		item := MicrodataItem{
			ItemType:   htmlparse.Attr(n, "itemtype"),
			Properties: make(map[string]string),
		}
		htmlparse.Walk(n, func(child *html.Node) error {
			if child.Type != html.ElementNode {
				return nil
			}
			prop := htmlparse.Attr(child, "itemprop")
			if prop == "" {
				return nil
			}
			val := htmlparse.Attr(child, "content")
			if val == "" {
				val = htmlparse.Attr(child, "href")
			}
			if val == "" {
				val = htmlparse.Attr(child, "src")
			}
			if val == "" {
				val = textContent(child)
			}
			item.Properties[prop] = val
			return nil
		})
		items = append(items, item)
		return nil
	})
	return items
}

func hasItemScope(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "itemscope" {
			return true
		}
	}
	return false
}

func StructuredDataSummary(d *htmlparse.Doc) map[string]int {
	summary := make(map[string]int)
	jsonld := ExtractJSONLD(d)
	for _, ld := range jsonld {
		summary["jsonld:"+ld.Type]++
	}
	microdata := ExtractMicrodata(d)
	for _, md := range microdata {
		summary["microdata:"+md.ItemType]++
	}
	return summary
}
