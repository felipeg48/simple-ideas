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
		for _, sheet := range root.Kids {
			for _, top := range sheet.Kids {
				if top.XName.Local == "topic" {
					t := xmind8(top)
					out = append(out, &Map{Title: t.Text, Topics: t.Kids})
				}
			}
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
// takes every map, one sheet each.
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
	default:
		return nil, fmt.Errorf("unknown format %q (use opml, freemind, xmind or mermaid)", format)
	}
	return b.Bytes(), nil
}

// mermaidText keeps a topic plain for Mermaid: brackets would start a shape.
func mermaidText(s string) string {
	if strings.ContainsAny(s, "()[]{}") {
		return `["` + strings.ReplaceAll(s, `"`, "'") + `"]`
	}
	return s
}
