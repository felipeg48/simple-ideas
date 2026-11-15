package mindmap

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// Import reads an OPML, FreeMind (.mm), XMind (.xmind) or Mermaid mindmap
// (.mmd) file into a mind map page.
func Import(name string, data []byte) (*Doc, error) {
	var maps []*Map
	var err error
	switch strings.ToLower(filepath.Ext(name)) {
	case ".opml":
		maps, err = fromOPML(data)
	case ".mm":
		maps, err = fromFreeMind(data)
	case ".xmind":
		maps, err = fromXMind(data)
	case ".mmd", ".mermaid":
		maps, err = fromMermaid(string(data))
	default:
		return nil, fmt.Errorf("%s: unknown format (use .opml, .mm, .xmind or .mmd)", name)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if len(maps) == 0 {
		return nil, fmt.Errorf("%s: no topics", name)
	}
	d := &Doc{Front: []string{"---", "type: mindmap", "title: " + maps[0].Title, "---"}}
	for i, m := range maps {
		m.Heading = true
		if i > 0 {
			m.ID = slug(m.Title)
			d.Parts = append(d.Parts, Part{Text: []string{"", "---", ""}})
		}
		d.Parts = append(d.Parts, Part{Map: m})
	}
	return d, nil
}

// xml outline shared by OPML (outline/@text) and FreeMind (node/@TEXT)
type xnode struct {
	Attrs []xml.Attr
	Kids  []xnode
	Text  string
	XName xml.Name
}

func (n *xnode) UnmarshalXML(d *xml.Decoder, s xml.StartElement) error {
	n.XName, n.Attrs = s.Name, s.Attr
	var p struct {
		Kids []xnode `xml:",any"`
		Text string  `xml:",chardata"`
	}
	if err := d.DecodeElement(&p, &s); err != nil {
		return err
	}
	n.Kids, n.Text = p.Kids, p.Text
	return nil
}

func (n xnode) attr(names ...string) string {
	for _, want := range names {
		for _, a := range n.Attrs {
			if strings.EqualFold(a.Name.Local, want) {
				return a.Value
			}
		}
	}
	return ""
}

func xmlTopics(nodes []xnode, elem, text string, link ...string) []*Topic {
	var out []*Topic
	for _, n := range nodes {
		if n.XName.Local != elem {
			continue
		}
		t := &Topic{Text: oneLine(n.attr(text)), Target: n.attr(link...)}
		t.Kids = xmlTopics(n.Kids, elem, text, link...)
		if t.Text == "" && len(t.Kids) == 0 {
			continue
		}
		out = append(out, t)
	}
	return out
}

func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func fromOPML(data []byte) ([]*Map, error) {
	var root xnode
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	title, body := "", []xnode(nil)
	for _, k := range root.Kids {
		switch k.XName.Local {
		case "head":
			for _, h := range k.Kids {
				if h.XName.Local == "title" {
					title = h.attr("text")
				}
			}
		case "body":
			body = k.Kids
		}
	}
	ts := xmlTopics(body, "outline", "text", "url", "htmlUrl")
	if len(ts) == 1 { // one outline at the top: the central topic
		return []*Map{{Title: ts[0].Text, Topics: ts[0].Kids}}, nil
	}
	if title == "" {
		title = "Mind map"
	}
	return []*Map{{Title: title, Topics: ts}}, nil
}

func fromFreeMind(data []byte) ([]*Map, error) {
	var root xnode
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	ts := xmlTopics(root.Kids, "node", "TEXT", "LINK")
	if len(ts) == 0 {
		return nil, errors.New("no <node>")
	}
	return []*Map{{Title: ts[0].Text, Topics: ts[0].Kids}}, nil
}

type xmTopic struct {
	Title    string `json:"title"`
	Href     string `json:"href,omitempty"`
	Children *struct {
		Attached []xmTopic `json:"attached"`
	} `json:"children,omitempty"`
}

func (x xmTopic) topics() []*Topic {
	var out []*Topic
	if x.Children == nil {
		return nil
	}
	for _, c := range x.Children.Attached {
		out = append(out, &Topic{Text: oneLine(c.Title), Target: c.Href, Kids: c.topics()})
	}
	return out
}

func fromXMind(data []byte) ([]*Map, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	read := func(name string) []byte {
		for _, f := range z.File {
			if f.Name == name {
				r, err := f.Open()
				if err != nil {
					return nil
				}
				defer r.Close()
				b, _ := io.ReadAll(io.LimitReader(r, 32<<20))
				return b
			}
		}
		return nil
	}
	if b := read("content.json"); b != nil {
		var sheets []struct {
			Title     string  `json:"title"`
			RootTopic xmTopic `json:"rootTopic"`
		}
		if err := json.Unmarshal(b, &sheets); err != nil {
			return nil, err
		}
		var out []*Map
		for _, s := range sheets {
			out = append(out, &Map{Title: oneLine(s.RootTopic.Title), Topics: s.RootTopic.topics()})
		}
		return out, nil
	}
	if b := read("content.xml"); b != nil { // XMind 8
		var root xnode
		if err := xml.Unmarshal(b, &root); err != nil {
			return nil, err
		}
		var out []*Map
		roots := map[string]string{} // a sheet's central topic id → #its map
		for _, sheet := range root.Kids {
			for _, top := range sheet.Kids {
				if top.XName.Local == "topic" {
					t := xmind8(top)
					roots[top.attr("id")] = "#" + slug(t.Text)
					out = append(out, &Map{Title: t.Text, Topics: t.Kids})
				}
			}
		}
		var fix func([]*Topic)
		fix = func(ts []*Topic) { // xmind:#topic-id links to another sheet become → #map
			for _, t := range ts {
				if strings.HasPrefix(t.Target, "xmind:#") {
					t.Target = roots[strings.TrimPrefix(t.Target, "xmind:#")]
				}
				fix(t.Kids)
			}
		}
		for _, m := range out {
			fix(m.Topics)
		}
		return out, nil
	}
	return nil, errors.New("no content.json or content.xml")
}

func xmind8(n xnode) *Topic {
	t := &Topic{Target: n.attr("href")}
	for _, k := range n.Kids {
		switch k.XName.Local {
		case "title":
			t.Text = oneLine(k.Text)
		case "labels":
			for _, l := range k.Kids {
				if l.XName.Local == "label" && t.Badge == "" {
					t.Badge = oneLine(l.Text)
				}
			}
		case "children":
			for _, ts := range k.Kids {
				for _, c := range ts.Kids {
					if c.XName.Local == "topic" {
						t.Kids = append(t.Kids, xmind8(c))
					}
				}
			}
		}
	}
	return t
}

var (
	mmdShapeRE = regexp.MustCompile(`^[\w-]*\s*(\(\((.*)\)\)|\)\)(.*)\(\(|\{\{(.*)\}\}|\[(.*)\]|\((.*)\)|\)(.*)\()$`)
)

func fromMermaid(src string) ([]*Map, error) {
	type item struct {
		t      *Topic
		indent int
	}
	var stack []item
	var root *Topic
	seen := false
	for _, l := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		s := strings.TrimSpace(l)
		if s == "" || strings.HasPrefix(s, "%%") || strings.HasPrefix(s, "::") || strings.HasPrefix(s, ":::") {
			continue
		}
		if !seen {
			if s != "mindmap" {
				return nil, errors.New(`the first line is not "mindmap"`)
			}
			seen = true
			continue
		}
		text := s
		if m := mmdShapeRE.FindStringSubmatch(s); m != nil {
			for _, g := range m[2:] {
				if g != "" {
					text = g
					break
				}
			}
		}
		text = oneLine(strings.Trim(text, `"`+"`"))
		t, in := &Topic{Text: text}, indentOf(l)
		if root == nil {
			root, stack = t, []item{{t, in}}
			continue
		}
		for len(stack) > 1 && stack[len(stack)-1].indent >= in {
			stack = stack[:len(stack)-1]
		}
		p := stack[len(stack)-1].t
		p.Kids = append(p.Kids, t)
		stack = append(stack, item{t, in})
	}
	if root == nil {
		return nil, errors.New("no root")
	}
	return []*Map{{Title: root.Text, Topics: root.Kids}}, nil
}

// Export writes one map (the first, or mapID) in another format; XMind
// (xmind: Zen/JSON, xmind8: the legacy XML) takes every map, one sheet each.
func Export(d *Doc, format, mapID string) ([]byte, error) {
	maps := d.Maps()
	if len(maps) == 0 {
		return nil, errors.New("no maps")
	}
	title := func(m *Map) string {
		if !m.Heading && m.Title == "" {
			if t := d.FrontValue("title"); t != "" {
				return t
			}
			return "Mind map"
		}
		return m.Title
	}
	pick := maps[0]
	if mapID != "" {
		pick = nil
		for _, m := range maps {
			if m.ID == mapID || slug(m.Title) == mapID {
				pick = m
			}
		}
		if pick == nil {
			return nil, fmt.Errorf("no map #%s", mapID)
		}
		maps = []*Map{pick}
	}
	var b bytes.Buffer
	esc := func(s string) string {
		var w bytes.Buffer
		xml.EscapeText(&w, []byte(s))
		return strings.ReplaceAll(w.String(), `"`, "&#34;")
	}
	switch format {
	case "opml":
		fmt.Fprintf(&b, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<opml version=\"2.0\">\n<head><title>%s</title></head>\n<body>\n", esc(title(pick)))
		var w func([]*Topic, string)
		w = func(ts []*Topic, pad string) {
			for _, t := range ts {
				if t.Code != nil {
					continue
				}
				link := ""
				if strings.HasPrefix(t.Target, "http") {
					link = fmt.Sprintf(` url="%s"`, esc(t.Target))
				}
				if len(t.Kids) == 0 {
					fmt.Fprintf(&b, "%s<outline text=\"%s\"%s/>\n", pad, esc(t.Text), link)
					continue
				}
				fmt.Fprintf(&b, "%s<outline text=\"%s\"%s>\n", pad, esc(t.Text), link)
				w(t.Kids, pad+"  ")
				fmt.Fprintf(&b, "%s</outline>\n", pad)
			}
		}
		fmt.Fprintf(&b, "<outline text=\"%s\">\n", esc(title(pick)))
		w(pick.Topics, "  ")
		b.WriteString("</outline>\n</body>\n</opml>\n")
	case "freemind":
		b.WriteString("<map version=\"1.0.1\">\n")
		var w func([]*Topic, string)
		w = func(ts []*Topic, pad string) {
			for _, t := range ts {
				if t.Code != nil {
					continue
				}
				link := ""
				if strings.HasPrefix(t.Target, "http") {
					link = fmt.Sprintf(` LINK="%s"`, esc(t.Target))
				}
				fmt.Fprintf(&b, "%s<node TEXT=\"%s\"%s>\n", pad, esc(t.Text), link)
				w(t.Kids, pad+"  ")
				fmt.Fprintf(&b, "%s</node>\n", pad)
			}
		}
		fmt.Fprintf(&b, "<node TEXT=\"%s\">\n", esc(title(pick)))
		w(pick.Topics, "  ")
		b.WriteString("</node>\n</map>\n")
	case "mermaid":
		b.WriteString("mindmap\n")
		fmt.Fprintf(&b, "  root((%s))\n", mermaidText(title(pick)))
		var w func([]*Topic, string)
		w = func(ts []*Topic, pad string) {
			for _, t := range ts {
				if t.Code != nil {
					continue
				}
				fmt.Fprintf(&b, "%s%s\n", pad, mermaidText(t.Text))
				w(t.Kids, pad+"  ")
			}
		}
		w(pick.Topics, "    ")
	case "xmind":
		n := 0
		id := func() string { n++; return fmt.Sprintf("si%05d", n) }
		var topic func(text, href string, ts []*Topic) map[string]any
		topic = func(text, href string, ts []*Topic) map[string]any {
			t := map[string]any{"id": id(), "class": "topic", "title": text}
			if strings.HasPrefix(href, "http") {
				t["href"] = href
			}
			var kids []map[string]any
			for _, k := range ts {
				if k.Code == nil {
					kids = append(kids, topic(k.Text, k.Target, k.Kids))
				}
			}
			if len(kids) > 0 {
				t["children"] = map[string]any{"attached": kids}
			}
			return t
		}
		var sheets []map[string]any
		for _, m := range maps {
			root := topic(title(m), "", m.Topics)
			root["structureClass"] = "org.xmind.ui.map.unbalanced"
			sheets = append(sheets, map[string]any{"id": id(), "class": "sheet", "title": title(m), "rootTopic": root})
		}
		z := zip.NewWriter(&b)
		for _, f := range []struct {
			name string
			v    any
		}{{"content.json", sheets}, {"metadata.json", map[string]any{"creator": map[string]string{"name": "simple-ideas"}}},
			{"manifest.json", map[string]any{"file-entries": map[string]any{"content.json": map[string]any{}, "metadata.json": map[string]any{}}}}} {
			w, err := z.Create(f.name)
			if err != nil {
				return nil, err
			}
			if err := json.NewEncoder(w).Encode(f.v); err != nil {
				return nil, err
			}
		}
		if err := z.Close(); err != nil {
			return nil, err
		}
	case "xmind8":
		if err := writeXMind8(&b, maps, title); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown format %q (use opml, freemind, xmind, xmind8 or mermaid)", format)
	}
	return b.Bytes(), nil
}

// XMind 8 (legacy): content.xml in the xmind-2.0 namespace, meta.xml and
// META-INF/manifest.xml in a zip. Badges are labels, web links and links to
// another map (→ #id, to that sheet's central topic) are xlink:href, and code
// topics are notes on their parent.
const (
	xmind8NS   = "urn:xmind:xmap:xmlns:content:2.0"
	xmind8Meta = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<meta xmlns="urn:xmind:xmap:xmlns:meta:2.0" version="2.0"><Creator><Name>simple-ideas</Name></Creator></meta>
`
	xmind8Manifest = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<manifest xmlns="urn:xmind:xmap:xmlns:manifest:1.0" password-hint=""><file-entry full-path="content.xml" media-type="text/xml"/><file-entry full-path="META-INF/" media-type=""/><file-entry full-path="META-INF/manifest.xml" media-type="text/xml"/><file-entry full-path="meta.xml" media-type="text/xml"/></manifest>
`
)

type x8Content struct {
	XMLName xml.Name   `xml:"urn:xmind:xmap:xmlns:content:2.0 xmap-content"`
	Attrs   []xml.Attr `xml:",any,attr"`
	Sheets  []x8Sheet  `xml:"sheet"`
}

type x8Sheet struct {
	ID    string   `xml:"id,attr"`
	Topic *x8Topic `xml:"topic"`
	Title string   `xml:"title"`
}

type x8Topic struct {
	ID       string      `xml:"id,attr"`
	Struct   string      `xml:"structure-class,attr,omitempty"`
	Href     string      `xml:"xlink:href,attr,omitempty"`
	Title    string      `xml:"title"`
	Notes    *x8Notes    `xml:"notes,omitempty"`
	Labels   *x8Labels   `xml:"labels,omitempty"`
	Children *x8Children `xml:"children,omitempty"`
}

type x8Labels struct {
	Label []string `xml:"label"`
}

// x8Notes carries a topic's notes twice, as XMind 8 writes them: plain text
// and an html element of xhtml:p paragraphs (one per line, code kept
// monospaced), which XMind 8 shows formatted.
type x8Notes struct {
	Plain string `xml:"plain"`
	HTML  x8HTML `xml:"html"`
}

type x8HTML struct {
	P []x8P `xml:"xhtml:p"`
}

type x8P struct {
	Span *x8Span `xml:"xhtml:span,omitempty"`
}

type x8Span struct {
	Style string `xml:"style,attr"`
	Text  string `xml:",chardata"`
}

func x8NotesOf(notes []string) *x8Notes {
	n := &x8Notes{Plain: strings.Join(notes, "\n\n")}
	for i, note := range notes {
		if i > 0 {
			n.HTML.P = append(n.HTML.P, x8P{})
		}
		for _, l := range strings.Split(note, "\n") {
			n.HTML.P = append(n.HTML.P, x8P{Span: &x8Span{Style: "font-family:monospace", Text: l}})
		}
	}
	return n
}

type x8Children struct {
	Topics struct {
		Type  string     `xml:"type,attr"`
		Topic []*x8Topic `xml:"topic"`
	} `xml:"topics"`
}

func writeXMind8(w io.Writer, maps []*Map, title func(*Map) string) error {
	roots := map[string]string{} // map id or title slug → its central topic's id
	for i, m := range maps {
		id := fmt.Sprintf("si-map-%d", i+1)
		roots[slug(title(m))] = id
		if m.ID != "" {
			roots[m.ID] = id
		}
	}
	n := 0
	var topic func(t *Topic) *x8Topic
	topic = func(t *Topic) *x8Topic {
		n++
		x := &x8Topic{ID: fmt.Sprintf("si%05d", n), Title: t.Text}
		if t.Badge != "" {
			x.Labels = &x8Labels{Label: []string{t.Badge}}
		}
		if strings.HasPrefix(t.Target, "http") {
			x.Href = t.Target
		} else if id, ok := roots[strings.TrimPrefix(t.Target, "#")]; ok && strings.HasPrefix(t.Target, "#") {
			x.Href = "xmind:#" + id
		}
		var notes []string
		for _, k := range t.Kids {
			if k.Code != nil {
				notes = append(notes, strings.TrimRight(k.Code.Text, "\n"))
				continue
			}
			if x.Children == nil {
				x.Children = &x8Children{}
				x.Children.Topics.Type = "attached"
			}
			x.Children.Topics.Topic = append(x.Children.Topics.Topic, topic(k))
		}
		if len(notes) > 0 {
			x.Notes = x8NotesOf(notes)
		}
		return x
	}
	c := x8Content{Attrs: []xml.Attr{
		{Name: xml.Name{Local: "xmlns:fo"}, Value: "http://www.w3.org/1999/XSL/Format"},
		{Name: xml.Name{Local: "xmlns:svg"}, Value: "http://www.w3.org/2000/svg"},
		{Name: xml.Name{Local: "xmlns:xhtml"}, Value: "http://www.w3.org/1999/xhtml"},
		{Name: xml.Name{Local: "xmlns:xlink"}, Value: "http://www.w3.org/1999/xlink"},
		{Name: xml.Name{Local: "version"}, Value: "2.0"},
	}}
	for i, m := range maps {
		root := topic(&Topic{Text: title(m), Kids: m.Topics})
		root.ID = fmt.Sprintf("si-map-%d", i+1)
		root.Struct = "org.xmind.ui.map.unbalanced"
		c.Sheets = append(c.Sheets, x8Sheet{ID: fmt.Sprintf("si-sheet-%d", i+1), Topic: root, Title: title(m)})
	}
	body, err := xml.Marshal(c)
	if err != nil {
		return err
	}
	z := zip.NewWriter(w)
	for _, f := range []struct{ name, text string }{
		{"content.xml", `<?xml version="1.0" encoding="UTF-8" standalone="no"?>` + "\n" + string(body) + "\n"},
		{"meta.xml", xmind8Meta},
		{"META-INF/manifest.xml", xmind8Manifest},
	} {
		fw, err := z.Create(f.name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(fw, f.text); err != nil {
			return err
		}
	}
	return z.Close()
}

// mermaidText keeps a topic plain for Mermaid: brackets would start a shape.
func mermaidText(s string) string {
	if strings.ContainsAny(s, "()[]{}") {
		return `["` + strings.ReplaceAll(s, `"`, "'") + `"]`
	}
	return s
}
