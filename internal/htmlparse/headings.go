package htmlparse

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

type Heading struct {
	Level int
	Text  string
	ID    string
}

func ExtractHeadings(d *Doc) []Heading {
	if d == nil || d.Root == nil {
		return nil
	}
	var headings []Heading
	_ = Walk(d.Root, func(n *html.Node) error {
		if n.Type != html.ElementNode {
			return nil
		}
		level := headingLevel(n.Data)
		if level == 0 {
			return nil
		}
		headings = append(headings, Heading{
			Level: level,
			Text:  InnerText(n),
			ID:    Attr(n, "id"),
		})
		return nil
	})
	return headings
}

func headingLevel(tag string) int {
	switch tag {
	case "h1":
		return 1
	case "h2":
		return 2
	case "h3":
		return 3
	case "h4":
		return 4
	case "h5":
		return 5
	case "h6":
		return 6
	}
	return 0
}

type TOCEntry struct {
	Level    int
	Text     string
	Anchor   string
	Children []TOCEntry
}

func GenerateTOC(headings []Heading) []TOCEntry {
	var toc []TOCEntry
	var stack []*[]TOCEntry
	stack = append(stack, &toc)
	prevLevel := 0

	for _, h := range headings {
		entry := TOCEntry{
			Level:  h.Level,
			Text:   h.Text,
			Anchor: h.ID,
		}
		if h.Level > prevLevel && prevLevel > 0 && len(*stack[len(stack)-1]) > 0 {
			parent := &(*stack[len(stack)-1])[len(*stack[len(stack)-1])-1].Children
			stack = append(stack, parent)
		} else if h.Level < prevLevel {
			for len(stack) > 1 && h.Level <= prevLevel {
				stack = stack[:len(stack)-1]
				prevLevel--
			}
		}
		*stack[len(stack)-1] = append(*stack[len(stack)-1], entry)
		prevLevel = h.Level
	}
	return toc
}

func ValidateHeadingHierarchy(headings []Heading) []string {
	if len(headings) == 0 {
		return nil
	}
	var warnings []string
	if headings[0].Level != 1 {
		warnings = append(warnings, fmt.Sprintf("first heading is h%d, expected h1", headings[0].Level))
	}
	for i := 1; i < len(headings); i++ {
		diff := headings[i].Level - headings[i-1].Level
		if diff > 1 {
			warnings = append(warnings, fmt.Sprintf("heading level skipped: h%d → h%d at %q",
				headings[i-1].Level, headings[i].Level, headings[i].Text))
		}
	}
	return warnings
}

func HeadingCount(headings []Heading) map[int]int {
	counts := make(map[int]int)
	for _, h := range headings {
		counts[h.Level]++
	}
	return counts
}

func HeadingsAsText(headings []Heading) string {
	var sb strings.Builder
	for _, h := range headings {
		indent := strings.Repeat("  ", h.Level-1)
		sb.WriteString(indent)
		sb.WriteString(h.Text)
		sb.WriteByte('\n')
	}
	return sb.String()
}
