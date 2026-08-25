package htmllink

import (
	"net/url"
	"strings"
)

type LinkGraph struct {
	Nodes map[string]bool
	Edges map[string][]string
}

func NewLinkGraph() *LinkGraph {
	return &LinkGraph{
		Nodes: make(map[string]bool),
		Edges: make(map[string][]string),
	}
}

func (g *LinkGraph) AddPage(pageURL string, links []Link) {
	g.Nodes[pageURL] = true
	for _, l := range links {
		target := l.Resolved
		if target == "" {
			target = l.Href
		}
		g.Nodes[target] = true
		g.Edges[pageURL] = append(g.Edges[pageURL], target)
	}
}

func (g *LinkGraph) OutDegree(pageURL string) int {
	return len(g.Edges[pageURL])
}

func (g *LinkGraph) InDegree(target string) int {
	count := 0
	for _, dests := range g.Edges {
		for _, d := range dests {
			if d == target {
				count++
			}
		}
	}
	return count
}

func (g *LinkGraph) NumNodes() int { return len(g.Nodes) }

func (g *LinkGraph) NumEdges() int {
	n := 0
	for _, dests := range g.Edges {
		n += len(dests)
	}
	return n
}

func ExternalLinks(links []Link, baseHost string) []Link {
	lowerBase := strings.ToLower(baseHost)
	var out []Link
	for _, l := range links {
		target := l.Resolved
		if target == "" {
			target = l.Href
		}
		u, err := url.Parse(target)
		if err != nil || u.Host == "" {
			continue
		}
		if strings.ToLower(u.Host) != lowerBase {
			out = append(out, l)
		}
	}
	return out
}

func InternalLinks(links []Link, baseHost string) []Link {
	lowerBase := strings.ToLower(baseHost)
	var out []Link
	for _, l := range links {
		target := l.Resolved
		if target == "" {
			target = l.Href
		}
		u, err := url.Parse(target)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Host)
		if host == lowerBase || host == "" {
			out = append(out, l)
		}
	}
	return out
}

func BrokenLinkCandidates(links []Link) []Link {
	var out []Link
	for _, l := range links {
		c := Classify(l.Href)
		if c == ClassJavaScript || c == ClassData || c == ClassUnknown {
			out = append(out, l)
		}
	}
	return out
}

func CountByTag(links []Link) map[string]int {
	counts := make(map[string]int)
	for _, l := range links {
		counts[l.Tag]++
	}
	return counts
}

func HasDuplicates(links []Link) bool {
	seen := make(map[string]bool)
	for _, l := range links {
		key := l.Resolved
		if key == "" {
			key = l.Href
		}
		if seen[key] {
			return true
		}
		seen[key] = true
	}
	return false
}
