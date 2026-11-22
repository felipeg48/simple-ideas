package render

import (
	"fmt"
	"html"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Status boards (`type: status`, RULES.md §36): the meta pill row, the KPI
// tiles, the overall bar, the Shipped chips, then one card per area with its
// bar and its Done / Missing / Improve columns, then the notes. Every count,
// percentage and bar width comes from deck.Status, so none can drift from
// the lists.

var statusCols = [3]string{"done", "missing", "improve"}

// statusBody is the board's HTML, below the page header.
func (dr *deckRenderer) statusBody(sl slugger) string {
	st := dr.d.Status
	if st == nil {
		return ""
	}
	total := st.Total()
	var b strings.Builder
	b.WriteString(`<div class="si-status">` + "\n")
	if len(st.Pills) > 0 || dr.d.Meta.Updated != "" {
		b.WriteString(`<p class="si-status-meta">`)
		if u := dr.d.Meta.Updated; u != "" {
			fmt.Fprintf(&b, `<span class="si-status-pill">Updated <b>%s</b></span>`, html.EscapeString(u))
		}
		for _, p := range st.Pills {
			tone := ""
			if p.Tone != "plain" {
				tone = ` data-tone="` + html.EscapeString(p.Tone) + `"`
			}
			label := ""
			if p.Label != "" {
				label = dr.inline(p.Label) + ": "
			}
			fmt.Fprintf(&b, `<span class="si-status-pill"%s>%s<b>%s</b></span>`, tone, label, dr.inline(p.Value))
		}
		b.WriteString("</p>\n")
	}
	if len(st.Tiles) > 0 {
		fill := strings.NewReplacer("{done}", strconv.Itoa(total.Done), "{missing}", strconv.Itoa(total.Missing),
			"{improve}", strconv.Itoa(total.Improve), "{open}", strconv.Itoa(total.Missing+total.Improve),
			"{pct}", strconv.Itoa(total.Percent())+"%", "{areas}", strconv.Itoa(len(st.Areas)), "{items}", strconv.Itoa(total.Items()))
		b.WriteString(`<div class="si-status-tiles">`)
		for _, t := range st.Tiles {
			attrs := ""
			if t.Gate {
				attrs += " data-gate"
			}
			if t.Tone != "" {
				attrs += ` data-tone="` + t.Tone + `"`
			}
			fmt.Fprintf(&b, `<div class="si-status-tile"%s><div class="si-status-num">%s</div>`, attrs, dr.inline(fill.Replace(t.Num)))
			if t.Label != "" {
				fmt.Fprintf(&b, `<div class="si-status-lbl">%s</div>`, dr.inline(fill.Replace(t.Label)))
			}
			if t.Sub != "" {
				fmt.Fprintf(&b, `<div class="si-status-sub">%s</div>`, dr.inline(fill.Replace(t.Sub)))
			}
			b.WriteString("</div>")
		}
		b.WriteString("</div>\n")
	}
	if len(st.Areas) > 0 {
		fmt.Fprintf(&b, `<section class="si-status-overall" aria-label="Overall progress"><div class="si-status-head"><span class="si-status-pct">%d%%</span><span class="si-status-lbl">Overall done</span><span class="si-status-count"><b>%d</b> done · <b>%d</b> missing · <b>%d</b> improve · %s</span></div>`,
			total.Percent(), total.Done, total.Missing, total.Improve, plural(len(st.Areas), "area"))
		b.WriteString(statusBar("Overall", total))
		b.WriteString(`<p class="si-status-key">`)
		for i, n := range []int{total.Done, total.Missing, total.Improve} {
			fmt.Fprintf(&b, `<span><i data-col="%s"></i>%s <b>%d</b></span>`, statusCols[i], deck.StatusColumns[i], n)
		}
		b.WriteString("</p></section>\n")
	}
	if s := st.Shipped; s != nil {
		fmt.Fprintf(&b, `<section class="si-status-shipped" id="%s"><h2>%s</h2><p class="si-status-chips">`, sl.slug(plain(dr.inline(s.Title))), dr.inline(s.Title))
		for _, it := range s.Cols[0] {
			b.WriteString(`<span class="si-status-chip">` + dr.inline(it.Text))
			for _, m := range it.Marks {
				fmt.Fprintf(&b, ` <span class="si-status-hash">%s</span>`, html.EscapeString(m.Text))
			}
			b.WriteString("</span>")
		}
		b.WriteString("</p></section>\n")
	}
	if st.Intro.Body != "" {
		b.WriteString(dr.statusNote(st.Intro, sl))
	}
	if len(st.Areas) > 0 {
		b.WriteString(`<div class="si-status-areas">` + "\n")
		for _, a := range st.Areas {
			c := a.Counts()
			fmt.Fprintf(&b, `<section class="si-status-area" id="%s"><h2>%s`, sl.slug(plain(dr.inline(a.Title))), dr.inline(a.Title))
			if a.Scope != "" {
				fmt.Fprintf(&b, ` <span class="si-status-scope">%s</span>`, dr.inline(a.Scope))
			}
			b.WriteString("</h2>\n")
			fmt.Fprintf(&b, `<div class="si-status-abar">%s<p class="si-status-ameta"><span>%d done · %d missing · %d improve</span><span class="si-status-p">%d%% done</span></p></div>`+"\n",
				statusBar(plain(dr.inline(a.Title)), c), c.Done, c.Missing, c.Improve, c.Percent())
			b.WriteString(`<div class="si-status-cols">`)
			for i, items := range a.Cols {
				fmt.Fprintf(&b, `<div class="si-status-col" data-col="%s"><h3><span class="si-status-dot"></span>%s</h3>`, statusCols[i], deck.StatusColumns[i])
				if len(items) == 0 {
					b.WriteString(`<p class="si-status-empty">Nothing here</p></div>`)
					continue
				}
				b.WriteString("<ul>")
				for _, it := range items {
					b.WriteString("<li>" + dr.inline(it.Text) + statusMarks(it.Marks) + "</li>")
				}
				b.WriteString("</ul></div>")
			}
			b.WriteString("</div></section>\n")
		}
		b.WriteString("</div>\n")
	}
	for _, n := range st.Notes {
		b.WriteString(dr.statusNote(n, sl))
	}
	b.WriteString("</div>\n")
	return b.String()
}

// statusBar is a stacked Done / Missing / Improve bar; its label carries
// the numbers.
func statusBar(name string, c deck.StatusCounts) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="si-status-bar" role="img" aria-label="%s: %d%% done; %d done, %d missing, %d improve">`,
		html.EscapeString(name), c.Percent(), c.Done, c.Missing, c.Improve)
	for i, w := range c.Shares() {
		if w > 0 {
			fmt.Fprintf(&b, `<span class="si-status-seg" data-col="%s" style="width:%s%%"></span>`, statusCols[i], strconv.FormatFloat(w, 'f', -1, 64))
		}
	}
	b.WriteString("</div>")
	return b.String()
}

// statusMarks draws an item's trailing marks: a priority tag, a badge
// (partial, gated, uncommitted, improvement) or a mono code (version · hash).
func statusMarks(marks []deck.StatusMark) string {
	var b strings.Builder
	for _, m := range marks {
		t := html.EscapeString(m.Text)
		switch m.Kind {
		case "tag":
			fmt.Fprintf(&b, ` <span class="si-status-tag">%s</span>`, t)
		case "code":
			fmt.Fprintf(&b, ` <span class="si-status-code">%s</span>`, t)
		default:
			fmt.Fprintf(&b, ` <span class="si-status-badge" data-kind="%s">%s</span>`, m.Kind, t)
		}
	}
	return b.String()
}

// statusNote renders a note section (or the intro) as document Markdown.
func (dr *deckRenderer) statusNote(n deck.StatusNote, sl slugger) string {
	title := ""
	s := &deck.Slide{Body: n.Body, Line: n.Line, File: dr.d.Path, Dir: filepath.Dir(dr.d.Path)}
	h, _ := dr.docBody(s, sl, &title)
	head := ""
	if n.Title != "" {
		head = fmt.Sprintf(`<h2 id="%s">%s</h2>`, sl.slug(plain(dr.inline(n.Title))), dr.inline(n.Title))
	}
	return `<section class="si-status-note">` + head + h + "</section>\n"
}

func plural(n int, what string) string {
	if n == 1 {
		return "1 " + what
	}
	return fmt.Sprintf("%d %ss", n, what)
}
