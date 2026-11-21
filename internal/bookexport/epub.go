package bookexport

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"fmt"
	"html"
	"os"
	"strings"
	"time"
)

// EPUB is the book as an EPUB 3 file: the mimetype first and stored, the
// container, the package (content.opf), a navigation document, a title page,
// one XHTML file per part and chapter, a stylesheet and the pictures (SVG
// kept as SVG).
func EPUB(b *Book, o Options) ([]byte, error) {
	order, name := names(b)
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	w, err := z.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		return nil, err
	}
	w.Write([]byte("application/epub+zip"))
	add := func(path string, data []byte) error {
		w, err := z.Create(path)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	lang := b.Lang
	if lang == "" {
		lang = "en"
	}
	type item struct{ id, href, media, props string }
	var manifest []item
	var spine []string
	page := func(id, title, body string) error {
		href := id + ".xhtml"
		manifest = append(manifest, item{id, href, "application/xhtml+xml", ""})
		spine = append(spine, id)
		return add("OEBPS/"+href, []byte(xhtmlPage(lang, title, body)))
	}
	var t strings.Builder
	fmt.Fprintf(&t, `<section class="title"><h1>%s</h1>`, h(b.Title))
	if b.Subtitle != "" {
		fmt.Fprintf(&t, `<p class="subtitle">%s</p>`, h(b.Subtitle))
	}
	if by := b.Byline(); by != "" {
		fmt.Fprintf(&t, `<p class="byline">%s</p>`, h(by))
	}
	if r := b.Rights(); r != "" {
		fmt.Fprintf(&t, `<p class="rights">%s</p>`, h(r))
	}
	t.WriteString(xhtmlBlocks(b.Colophon, name) + `</section>`)
	if err := page("title", b.Title, t.String()); err != nil {
		return nil, err
	}
	var nav strings.Builder
	open := false // a part's list is open
	for i, c := range b.Chapters {
		if c.Part != "" {
			if open {
				nav.WriteString(`</ol></li>`)
			}
			id := fmt.Sprintf("part%d", i+1)
			if err := page(id, c.Part, `<section class="part"><h1>`+h(c.Part)+`</h1></section>`); err != nil {
				return nil, err
			}
			fmt.Fprintf(&nav, `<li><a href="%s.xhtml">%s</a><ol>`, id, h(c.Part))
			open = true
		}
		id := fmt.Sprintf("ch%02d", i+1)
		var secs strings.Builder
		for _, bl := range c.Blocks {
			if bl.Kind == Heading && bl.Level == 2 {
				t := runsText(bl.Runs)
				if bl.Num != "" {
					t = bl.Num + " " + t
				}
				fmt.Fprintf(&secs, `<li><a href="%s.xhtml#%s">%s</a></li>`, id, h(bl.ID), h(t))
			}
		}
		fmt.Fprintf(&nav, `<li><a href="%s.xhtml">%s</a>`, id, h(c.Heading()))
		if secs.Len() > 0 {
			nav.WriteString(`<ol>` + secs.String() + `</ol>`)
		}
		nav.WriteString(`</li>`)
		if err := page(id, c.Heading(), `<section class="chapter"><h1>`+h(c.Heading())+`</h1>`+xhtmlBlocks(c.Blocks, name)+`</section>`); err != nil {
			return nil, err
		}
		_ = i
	}
	if open {
		nav.WriteString(`</ol></li>`)
	}
	navDoc := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="` + h(lang) + `" xml:lang="` + h(lang) + `"><head><meta charset="utf-8"/><title>Contents</title><link rel="stylesheet" href="style.css"/></head><body>` +
		`<nav epub:type="toc" id="toc"><h1>Contents</h1><ol><li><a href="title.xhtml">` + h(b.Title) + `</a></li>` + nav.String() + `</ol></nav></body></html>`
	if err := add("OEBPS/nav.xhtml", []byte(navDoc)); err != nil {
		return nil, err
	}
	manifest = append(manifest, item{"nav", "nav.xhtml", "application/xhtml+xml", "nav"}, item{"css", "style.css", "text/css", ""})
	if err := add("OEBPS/style.css", []byte(epubCSS)); err != nil {
		return nil, err
	}
	for i, src := range order {
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		media := map[string]string{".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp"}[strings.ToLower(extOf(src))]
		if media == "" {
			continue
		}
		manifest = append(manifest, item{fmt.Sprintf("img%d", i+1), "images/" + name[src], media, ""})
		if err := add("OEBPS/images/"+name[src], data); err != nil {
			return nil, err
		}
	}
	if err := add("META-INF/container.xml", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`)); err != nil {
		return nil, err
	}
	sum := sha1.Sum([]byte(b.Title + "\x00" + b.Author + "\x00" + b.Edition))
	uid := fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
	var opf strings.Builder
	opf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uid" xml:lang="` + h(lang) + `"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/">`)
	fmt.Fprintf(&opf, `<dc:identifier id="uid">%s</dc:identifier><dc:title>%s</dc:title><dc:language>%s</dc:language>`, uid, h(b.Title), h(lang))
	if b.Author != "" {
		fmt.Fprintf(&opf, `<dc:creator>%s</dc:creator>`, h(b.Author))
	}
	if r := b.Rights(); r != "" {
		fmt.Fprintf(&opf, `<dc:rights>%s</dc:rights>`, h(r))
	}
	fmt.Fprintf(&opf, `<meta property="dcterms:modified">%s</meta></metadata><manifest>`, time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	for _, it := range manifest {
		props := ""
		if it.props != "" {
			props = ` properties="` + it.props + `"`
		}
		fmt.Fprintf(&opf, `<item id="%s" href="%s" media-type="%s"%s/>`, it.id, h(it.href), it.media, props)
	}
	opf.WriteString(`</manifest><spine>`)
	for _, id := range spine {
		fmt.Fprintf(&opf, `<itemref idref="%s"/>`, id)
	}
	opf.WriteString(`</spine></package>`)
	if err := add("OEBPS/content.opf", []byte(opf.String())); err != nil {
		return nil, err
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func h(s string) string { return html.EscapeString(s) }

func xhtmlPage(lang, title, body string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="` + h(lang) + `" xml:lang="` + h(lang) + `"><head><meta charset="utf-8"/><title>` + h(title) + `</title><link rel="stylesheet" href="style.css"/></head><body>` + body + `</body></html>`
}

func runsText(runs []Run) string {
	var b strings.Builder
	for _, r := range runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

// xhtmlRuns writes runs as XHTML inline text.
func xhtmlRuns(runs []Run) string {
	var b strings.Builder
	for _, r := range runs {
		if r.Text == "\n" {
			b.WriteString("<br/>")
			continue
		}
		t := h(r.Text)
		if r.Code {
			t = "<code>" + t + "</code>"
		}
		if r.Italic {
			t = "<em>" + t + "</em>"
		}
		if r.Bold {
			t = "<strong>" + t + "</strong>"
		}
		if r.Link != "" {
			t = `<a href="` + h(r.Link) + `">` + t + `</a>`
		}
		b.WriteString(t)
	}
	return b.String()
}

func xhtmlBlocks(bs []Block, name map[string]string) string {
	var w strings.Builder
	for _, b := range bs {
		w.WriteString(xhtmlBlock(b, name))
	}
	return w.String()
}

func xhtmlBlock(b Block, name map[string]string) string {
	id := ""
	if b.ID != "" {
		id = ` id="` + h(b.ID) + `"`
	}
	switch b.Kind {
	case Heading:
		t := xhtmlRuns(b.Runs)
		if b.Num != "" {
			t = `<span class="num">` + b.Num + `</span> ` + t
		}
		return fmt.Sprintf("<h%d%s>%s</h%d>\n", b.Level, id, t, b.Level)
	case Para:
		return "<p>" + xhtmlRuns(b.Runs) + "</p>\n"
	case Code:
		cls := ""
		if b.Lang != "" {
			cls = ` class="language-` + h(b.Lang) + `"`
		}
		out := "<pre><code" + cls + ">" + h(CodeLines(b)) + "</code></pre>\n"
		if len(b.Callouts) > 0 {
			out += `<ol class="callouts">`
			for _, co := range b.Callouts {
				out += `<li value="` + h(co.N) + `">` + xhtmlRuns(co.Runs) + `</li>`
			}
			out += "</ol>\n"
		}
		return out
	case Placeholder:
		return `<p class="placeholder">` + xhtmlRuns(b.Runs) + "</p>\n"
	case List:
		tag, start := "ul", ""
		if b.Ordered {
			tag = "ol"
			if b.Start > 1 {
				start = fmt.Sprintf(` start="%d"`, b.Start)
			}
		}
		var w strings.Builder
		w.WriteString("<" + tag + start + ">")
		for _, it := range b.Items {
			w.WriteString("<li>" + xhtmlBlocks(it, name) + "</li>")
		}
		w.WriteString("</" + tag + ">\n")
		return w.String()
	case Table:
		var w strings.Builder
		w.WriteString("<table" + id + ">")
		if b.Caption != "" {
			w.WriteString("<caption>" + h(b.Caption) + "</caption>")
		}
		for i, r := range b.Rows {
			cell := "td"
			if i == 0 {
				cell = "th"
				w.WriteString("<thead>")
			}
			w.WriteString("<tr>")
			for _, c := range r {
				w.WriteString("<" + cell + ">" + xhtmlRuns(c) + "</" + cell + ">")
			}
			w.WriteString("</tr>")
			if i == 0 {
				w.WriteString("</thead><tbody>")
			}
		}
		if len(b.Rows) > 0 {
			w.WriteString("</tbody>")
		}
		w.WriteString("</table>\n")
		return w.String()
	case Image:
		src := name[b.Src]
		if src == "" {
			return `<p class="placeholder">[Figure: ` + h(firstNonEmpty(b.Caption, b.Alt, b.Src)) + "]</p>\n"
		}
		out := "<figure" + id + `><img src="images/` + h(src) + `" alt="` + h(b.Alt) + `"/>`
		if b.Caption != "" {
			out += "<figcaption>" + h(b.Caption) + "</figcaption>"
		}
		return out + "</figure>\n"
	case Admonition:
		return `<aside class="admonition ` + strings.ToLower(b.Label) + `" epub:type="note"><p class="label">` + h(b.Label) + "</p>" + xhtmlBlocks(b.Blocks, name) + "</aside>\n"
	case Quote:
		return "<blockquote>" + xhtmlBlocks(b.Blocks, name) + "</blockquote>\n"
	case Rule:
		return "<hr/>\n"
	}
	return ""
}

const epubCSS = `body{font-family:Georgia,serif;line-height:1.5;margin:0 5%}
h1,h2,h3,h4{font-family:Helvetica,Arial,sans-serif;line-height:1.2;page-break-after:avoid}
h1{font-size:1.8em;margin:1.5em 0 1em}
.num{color:#9a3a1f}
section.title{text-align:center;margin-top:20%}
section.title h1{font-size:2.2em}
.subtitle{font-size:1.2em;color:#555}
.byline,.rights{color:#555}
section.part h1{text-align:center;margin-top:35%}
pre{font-family:"Courier New",monospace;font-size:.8em;background:#f2f2f2;padding:.8em;white-space:pre-wrap;page-break-inside:avoid}
code{font-family:"Courier New",monospace;font-size:.9em}
ol.callouts{font-size:.95em}
table{border-collapse:collapse;width:100%;margin:1em 0;font-size:.9em}
th,td{border:1px solid #aaa;padding:.3em .5em;text-align:left;vertical-align:top}
caption,figcaption{caption-side:bottom;font-size:.85em;font-style:italic;color:#555;text-align:center;margin:.4em 0}
figure{margin:1em 0;text-align:center;page-break-inside:avoid}
figure img{max-width:100%}
aside.admonition,blockquote{border-left:4px solid #bbb;margin:1em 0;padding:.2em 1em;color:#333}
aside .label{font-weight:bold;margin-bottom:0}
.placeholder{font-style:italic;color:#555;text-align:center}
`
