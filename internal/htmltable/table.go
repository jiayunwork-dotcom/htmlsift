package htmltable

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"golang.org/x/net/html"

	"htmlsift/internal/htmlparse"
)

var ErrNoTable = errors.New("htmltable: no table found")

type Table struct {
	Caption string
	Headers []string
	Rows    [][]string
}

func (t *Table) NumCols() int {
	max := len(t.Headers)
	for _, row := range t.Rows {
		if len(row) > max {
			max = len(row)
		}
	}
	return max
}

func (t *Table) NumRows() int { return len(t.Rows) }

func (t *Table) Cell(r, c int) string {
	if r < 0 || r >= len(t.Rows) {
		return ""
	}
	if c < 0 || c >= len(t.Rows[r]) {
		return ""
	}
	return t.Rows[r][c]
}

func (t *Table) Column(c int) []string {
	var out []string
	for _, row := range t.Rows {
		if c < len(row) {
			out = append(out, row[c])
		} else {
			out = append(out, "")
		}
	}
	return out
}

func ExtractAll(d *htmlparse.Doc) ([]*Table, error) {
	if d == nil || d.Root == nil {
		return nil, ErrNoTable
	}
	tableNodes := htmlparse.FindByTag(d, "table")
	if len(tableNodes) == 0 {
		return nil, ErrNoTable
	}
	var tables []*Table
	for _, tn := range tableNodes {
		tables = append(tables, extractTable(tn))
	}
	return tables, nil
}

func ExtractFirst(d *htmlparse.Doc) (*Table, error) {
	tables, err := ExtractAll(d)
	if err != nil {
		return nil, err
	}
	return tables[0], nil
}

func extractTable(tableNode *html.Node) *Table {
	t := &Table{}
	for c := tableNode.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "caption" {
			t.Caption = cellText(c)
			break
		}
	}
	var headerRows []*html.Node
	var bodyRows []*html.Node
	_ = htmlparse.Walk(tableNode, func(n *html.Node) error {
		if n.Type != html.ElementNode || n.Data != "tr" {
			return nil
		}
		inThead := false
		for p := n.Parent; p != nil; p = p.Parent {
			if p.Type == html.ElementNode && p.Data == "thead" {
				inThead = true
				break
			}
		}
		if inThead {
			headerRows = append(headerRows, n)
		} else {
			bodyRows = append(bodyRows, n)
		}
		return nil
	})
	if len(headerRows) > 0 {
		t.Headers = extractRow(headerRows[0])
	} else if len(bodyRows) > 0 {
		first := extractRow(bodyRows[0])
		t.Headers = first
		bodyRows = bodyRows[1:]
	}
	for _, tr := range bodyRows {
		t.Rows = append(t.Rows, extractRow(tr))
	}
	return t
}

func extractRow(tr *html.Node) []string {
	var cells []string
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, cellText(c))
		}
	}
	return cells
}

func cellText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.TextNode {
				sb.WriteString(c.Data)
			} else if c.Type == html.ElementNode {
				walk(c)
			}
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

func (t *Table) WriteCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	if len(t.Headers) > 0 {
		if err := cw.Write(t.Headers); err != nil {
			return err
		}
	}
	for _, row := range t.Rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func (t *Table) ToCSVString() (string, error) {
	var sb strings.Builder
	if err := t.WriteCSV(&sb); err != nil {
		return "", err
	}
	return sb.String(), nil
}
