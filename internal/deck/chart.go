package deck

import (
	"fmt"
	"sort"
	"strings"
)

// Charts: a ```chart fence in a slide or a document page. The info string
// names the type (bar, column, line, area, donut, pie, progress) and the
// options; the body is a Markdown table, labels in the first column and one
// or more columns of numbers after. With steps every number column is one
// build step and the chart animates from one to the next; sort keeps the rows
// in order of value (the bars glide to their new places); race is sort plus
// steps, each step rescaled to its own top value. Several chart fences in a
// row with the same id= are one show, as code windows are. Drawn by
// render/chart.go with the infographic chart code, animated by web/chart.js.

// ChartTypes are the kinds a ```chart fence draws: the flat ones an
// infographic block draws too (IGCharts), and three solid ones in CSS 3D.
var ChartTypes = append(append([]string{}, IGCharts...), "bar3d", "donut3d", "funnel3d")

// Chart is one ```chart fence, read.
type Chart struct {
	Type                       string
	Title, Unit, ID, Explode   string
	Steps, Sort, Race, Stacked bool
	Head                       []string // the number columns' headers: series, or steps
	Labels                     []string
	Text                       [][]string // [row][column], the cells as written
	Vals                       [][]float64
}

// ParseChart reads a chart fence's options and table. Problems are si check
// errors; the chart is still drawn as far as it can be.
func ParseChart(attrs map[string]string, body string) (*Chart, []string) {
	c := &Chart{}
	var probs []string
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := attrs[k]
		switch {
		case k == "type" || (v == "" && contains(ChartTypes, k)):
			t := k
			if k == "type" {
				t = v
			}
			if !contains(ChartTypes, t) {
				probs = append(probs, fmt.Sprintf("chart: unknown type %q (use %s)", t, strings.Join(ChartTypes, ", ")))
			} else if c.Type != "" && c.Type != t {
				probs = append(probs, fmt.Sprintf("chart: two types, %s and %s: keep one", c.Type, t))
			} else {
				c.Type = t
			}
		case k == "title":
			c.Title = v
		case k == "unit":
			c.Unit = v
		case k == "id":
			c.ID = v
		case k == "explode":
			c.Explode = v
		case k == "steps":
			c.Steps = true
		case k == "sort":
			c.Sort = true
		case k == "race":
			c.Race, c.Steps, c.Sort = true, true, true
		case k == "stacked":
			c.Stacked = true
		case v == "":
			probs = append(probs, fmt.Sprintf("chart: unknown type %q (use %s; options steps, sort, race, stacked, title=, unit=, id=, explode=)", k, strings.Join(ChartTypes, ", ")))
		default:
			probs = append(probs, fmt.Sprintf("chart: unknown option %s= (use title=, unit=, id=, explode=, type=)", k))
		}
	}
	if c.Type == "" {
		c.Type = "bar"
	}
	if c.Stacked && c.Type != "bar" && c.Type != "column" {
		probs = append(probs, fmt.Sprintf("chart: stacked is for bar and column charts, not %s", c.Type))
		c.Stacked = false
	}
	head, rows, ok := IGTable(body)
	if !ok || len(rows) == 0 {
		return c, append(probs, "chart: write a Markdown table, labels in the first column and numbers after (| Cause | 2024 |)")
	}
	if len(head) < 2 {
		return c, append(probs, "chart: the table needs a column of numbers after the labels")
	}
	c.Head = head[1:]
	pie := c.Type == "donut" || c.Type == "pie" || c.Type == "progress" || c.Type == "donut3d" || c.Type == "funnel3d"
	for _, r := range rows {
		c.Labels = append(c.Labels, r[0])
		var vs []float64
		for _, cell := range r[1:] {
			v, ok := IGNumber(cell)
			switch {
			case !ok:
				probs = append(probs, fmt.Sprintf("chart: %q (row %q) is not a number", cell, r[0]))
			case v < 0 && pie:
				probs = append(probs, fmt.Sprintf("chart: %q (row %q): a %s takes no negative numbers", cell, r[0], c.Type))
				v = 0
			}
			vs = append(vs, v)
		}
		c.Vals = append(c.Vals, vs)
		c.Text = append(c.Text, append([]string(nil), r[1:]...))
	}
	if c.Explode != "" {
		switch {
		case c.Type != "donut" && c.Type != "pie" && c.Type != "donut3d":
			probs = append(probs, fmt.Sprintf("chart: explode= is for donut, pie and donut3d charts, not %s", c.Type))
			c.Explode = ""
		case c.ExplodeRow() < 0:
			probs = append(probs, fmt.Sprintf("chart: explode=%q names no row (use one of %s)", c.Explode, strings.Join(c.Labels, ", ")))
			c.Explode = ""
		}
	}
	if c.Steps && len(c.Head) < 2 {
		what := "steps"
		if c.Race {
			what = "race"
		}
		probs = append(probs, fmt.Sprintf("chart: %s needs two or more number columns, one per step (| Cause | 2024 | 2025 |)", what))
		c.Steps, c.Race = false, false
	}
	return c, probs
}

// ChartValue is a cell as shown: the cell as written, with unit= after a
// bare number.
func (c *Chart) ChartValue(row, col int) string {
	t := c.Text[row][col]
	if c.Unit != "" && t != "" && strings.Trim(t, "0123456789.,−- ") == "" {
		return t + c.Unit
	}
	return t
}

// ExplodeRow is the row explode= names (-1 when none).
func (c *Chart) ExplodeRow() int {
	for i, l := range c.Labels {
		if c.Explode != "" && strings.EqualFold(strings.TrimSpace(l), strings.TrimSpace(c.Explode)) {
			return i
		}
	}
	return -1
}
