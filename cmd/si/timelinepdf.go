package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// A landscape timeline's PDFs (si build --pdf): pdfs/<name>.pdf has several
// 16:9 pages in the same look, a few eras on each so the text stays readable
// (title on every page, the closing line and sources on the last);
// pdfs/<name>-compact.pdf is the whole timeline on one page, as the PNG.
// Pages are pictures of the rendered timeline (text not selectable).

// tlPerPage is about how many milestones fit one readable 16:9 page.
const tlPerPage = 12

// timelinePages splits a timeline's Markdown into one source per page: whole
// eras, greedily up to perPage milestones a page (an era bigger than that gets
// a page of its own). Text before the first era stays on page 1; sources and
// `closing:` go on the last page; `require:` is dropped (a page is not the
// whole timeline). One page back when nothing needs splitting.
func timelinePages(src string, perPage int) []string {
	front, body := "", src
	if strings.HasPrefix(src, "---\n") {
		if i := strings.Index(src[4:], "\n---\n"); i >= 0 {
			front, body = src[4:4+i+1], src[4+i+5:]
		}
	}
	var intro string
	var eras, sources []string
	parts := strings.Split("\n"+body, "\n## ")
	intro = strings.TrimPrefix(parts[0], "\n")
	for _, p := range parts[1:] {
		sec := "## " + p
		head := strings.ToLower(strings.SplitN(p, "\n", 2)[0])
		if strings.Contains(head, "· sources") || strings.Contains(head, "·sources") || isSourcesTitle(head) {
			sources = append(sources, sec)
		} else {
			eras = append(eras, sec)
		}
	}
	var groups [][]string
	n := 0
	for _, e := range eras {
		m := strings.Count("\n"+e, "\n### ")
		if len(groups) == 0 || (n > 0 && n+m > perPage) {
			groups = append(groups, nil)
			n = 0
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], e)
		n += m
	}
	if len(groups) <= 1 {
		return []string{src}
	}
	var keep, closingLine []string
	for _, l := range strings.Split(strings.TrimSuffix(front, "\n"), "\n") {
		k := strings.TrimSpace(strings.SplitN(l, ":", 2)[0])
		switch k {
		case "closing":
			closingLine = append(closingLine, l)
		case "require":
		default:
			keep = append(keep, l)
		}
	}
	var pages []string
	for i, g := range groups {
		fm := append([]string{}, keep...)
		last := i == len(groups)-1
		if last {
			fm = append(fm, closingLine...)
		}
		var b strings.Builder
		b.WriteString("---\n" + strings.Join(fm, "\n") + "\n---\n")
		if i == 0 && strings.TrimSpace(intro) != "" {
			b.WriteString(intro + "\n")
		}
		for _, e := range g {
			b.WriteString(strings.TrimRight(e, "\n") + "\n\n")
		}
		if last {
			for _, s := range sources {
				b.WriteString(strings.TrimRight(s, "\n") + "\n\n")
			}
		}
		pages = append(pages, b.String())
	}
	return pages
}

func isSourcesTitle(head string) bool {
	h := strings.TrimSpace(strings.SplitN(head, "·", 2)[0])
	switch h {
	case "sources", "source", "fuentes", "references", "referencias", "bibliography":
		return true
	}
	return false
}

// exportTimelinePDF writes the paged PDF and the compact one; it returns the
// paged PDF's path relative to out.
func exportTimelinePDF(chrome, dest, out string, d *deck.Deck, scale float64, stdout io.Writer) (string, error) {
	if err := os.MkdirAll(filepath.Join(out, "pdfs"), 0o755); err != nil {
		return "", err
	}
	scale = max(scale, 2)
	compact, w, h, err := timelineShot(chrome, dest, d, scale)
	if err != nil {
		return "", err
	}
	src, err := os.ReadFile(d.Path)
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "si-timeline-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	var pages [][]byte
	srcs := timelinePages(string(src), tlPerPage)
	if len(srcs) == 1 {
		pages = [][]byte{compact}
	}
	for i, ps := range srcs {
		if len(srcs) == 1 {
			break
		}
		// beside the original, so its pictures resolve; removed at once
		md := filepath.Join(filepath.Dir(d.Path), fmt.Sprintf(".si-%s-page%d.md", d.Name, i+1))
		if err := os.WriteFile(md, []byte(ps), 0o644); err != nil {
			return "", err
		}
		pd, err := deck.Parse(md)
		os.Remove(md)
		if err != nil {
			return "", err
		}
		res := render.Deck(pd, render.Options{Media: render.MediaInline})
		html := filepath.Join(tmp, fmt.Sprintf("page%d.html", i+1))
		if err := os.WriteFile(html, res.HTML, 0o644); err != nil {
			return "", err
		}
		png, _, _, err := timelineShot(chrome, html, pd, scale)
		if err != nil {
			return "", err
		}
		pages = append(pages, png)
	}
	rel := filepath.Join("pdfs", d.Name+".pdf")
	if err := picturesPDF(chrome, tmp, pages, w, h, filepath.Join(out, rel)); err != nil {
		return "", err
	}
	crel := filepath.Join("pdfs", d.Name+"-compact.pdf")
	if err := picturesPDF(chrome, tmp, [][]byte{compact}, w, h, filepath.Join(out, crel)); err != nil {
		return "", err
	}
	fmt.Fprintf(stdout, "  %-28s (pdf, %d pages of 16:9)\n  %-28s (pdf, the whole timeline on one page)\n", rel, len(pages), crel)
	return filepath.ToSlash(rel), nil
}

// picturesPDF prints PNG pictures as a PDF, one w×h (CSS px) page each.
func picturesPDF(chrome, tmp string, pngs [][]byte, w, h int, pdfPath string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "<!doctype html><style>@page{size:%dpx %dpx;margin:0}html,body{margin:0}img{display:block;width:%dpx;height:%dpx;break-after:page}img:last-child{break-after:auto}</style>", w, h, w, h)
	for i, p := range pngs {
		name := fmt.Sprintf("pic%d.png", i+1)
		if err := os.WriteFile(filepath.Join(tmp, name), p, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(&b, `<img src="%s">`, name)
	}
	page := filepath.Join(tmp, "pictures.html")
	if err := os.WriteFile(page, []byte(b.String()), 0o644); err != nil {
		return err
	}
	c, err := cdp.Launch(chrome, w, h)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := c.OpenPage("", w, h); err != nil {
		return err
	}
	if err := c.Call("Page.navigate", map[string]any{"url": "file://" + filepath.ToSlash(page)}, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		if v, _ := c.Eval(`document.readyState==="complete"&&[...document.images].every(function(i){return i.complete})`); string(v) == "true" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	var r struct{ Data string }
	if err := c.Call("Page.printToPDF", map[string]any{"printBackground": true, "preferCSSPageSize": true}, &r); err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(r.Data)
	if err != nil {
		return err
	}
	return os.WriteFile(pdfPath, raw, 0o644)
}
