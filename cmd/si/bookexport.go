package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/felipeg48/simple-ideas/internal/bookexport"
	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Book exports (si build <book> --md --docx --rtf --epub, RULES.md §21): the
// book (or a guide, or an article) read once into bookexport's model and
// written in each format asked for, beside the HTML.

// wantsExport reports whether any document export flag is on.
func (o *options) wantsExport() bool { return o.md || o.docx || o.rtf || o.epub }

// exportDocs writes b as base.md (with one file per chapter, in md/), base.docx,
// base.rtf and base.epub into out. It returns false when a format failed.
func exportDocs(b *bookexport.Book, out, base string, o *options, stdout, stderr io.Writer) bool {
	ok := true
	report := func(rel string, err error) {
		if err != nil {
			fmt.Fprintf(stderr, "si: %s: %v\n", rel, err)
			ok = false
			return
		}
		fmt.Fprintf(stdout, "  %-40s (%s)\n", filepath.ToSlash(filepath.Join(filepath.Base(out), rel)), filepath.Ext(rel)[1:])
	}
	var ro bookexport.Options
	if o.docx || o.rtf { // Word and RTF cannot show SVG: Chrome draws it as a PNG
		if chrome := findChrome(); chrome != "" {
			r := &rasterizer{chrome: chrome}
			defer r.close()
			ro.Raster = r.raster
		} else {
			fmt.Fprintln(stderr, "si: no Chrome: SVG pictures become placeholders in the .docx and .rtf")
		}
	}
	if o.md {
		files, err := bookexport.WriteMarkdown(b, filepath.Join(out, "md"), base)
		if err == nil {
			report(filepath.Join("md", base+".md"), nil)
			if len(files) > 1 {
				fmt.Fprintf(stdout, "  %-40s (%d chapters)\n", filepath.ToSlash(filepath.Join(filepath.Base(out), "md"))+"/", len(files)-1)
			}
		} else {
			report(filepath.Join("md", base+".md"), err)
		}
	}
	if o.docx {
		data, err := bookexport.Docx(b, ro)
		if err == nil {
			err = os.WriteFile(filepath.Join(out, base+".docx"), data, 0o644)
		}
		report(base+".docx", err)
	}
	if o.rtf {
		report(base+".rtf", os.WriteFile(filepath.Join(out, base+".rtf"), []byte(bookexport.RTF(b, ro)), 0o644))
	}
	if o.epub {
		data, err := bookexport.EPUB(b, ro)
		if err == nil {
			err = os.WriteFile(filepath.Join(out, base+".epub"), data, 0o644)
		}
		report(base+".epub", err)
	}
	return ok
}

// exportGuide exports a book or guide; a site or course has no reading order
// to export.
func exportGuide(g *deck.Guide, out string, o *options, stdout, stderr io.Writer) bool {
	if g.IsSite() || g.IsCourse() {
		fmt.Fprintln(stderr, "si: --md, --docx, --rtf and --epub export books, guides and articles, not a site or a course")
		return false
	}
	b, err := bookexport.FromGuide(g)
	if err != nil {
		fmt.Fprintf(stderr, "si: export: %v\n", err)
		return false
	}
	base := "guide"
	if g.IsBook() {
		base = "book"
	}
	return exportDocs(b, out, base, o, stdout, stderr)
}

// exportable is a page --md / --docx / --rtf / --epub can export on its own.
func exportable(d *deck.Deck) bool {
	switch d.Meta.Type {
	case "article", "guide", "lab", "page":
		return true
	}
	return false
}

// rasterizer draws SVG files as PNGs (twice the CSS size) in one headless Chrome.
type rasterizer struct {
	chrome string
	c      *cdp.Client
	err    error
}

func (r *rasterizer) raster(path string) ([]byte, int, int, error) {
	w, h := bookexport.SVGSize(path)
	if r.c == nil && r.err == nil {
		r.c, r.err = cdp.Launch(r.chrome, w, h, "--allow-file-access-from-files")
		if r.err == nil {
			r.c.Scale = 2
			_, r.err = r.c.OpenPage("", w, h)
		}
	}
	if r.err != nil {
		return nil, 0, 0, r.err
	}
	if err := r.c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": w, "height": h, "deviceScaleFactor": 2, "mobile": false}, nil); err != nil {
		return nil, 0, 0, err
	}
	abs, _ := filepath.Abs(path)
	if err := r.c.Call("Page.navigate", map[string]any{"url": fileURL(abs)}, nil); err != nil {
		return nil, 0, 0, err
	}
	waitLoaded(r.c)
	png, err := r.c.Screenshot()
	return png, w, h, err
}

func (r *rasterizer) close() {
	if r.c != nil {
		r.c.Close()
	}
}

func waitLoaded(c *cdp.Client) {
	for i := 0; i < 200; i++ {
		if v, _ := c.Eval("document.readyState"); string(v) == `"complete"` {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	c.Eval(`document.fonts?document.fonts.ready.then(function(){return 1}):1`)
}

// bookPDF prints the one-file book (book.html) as the book's PDF in headless
// Chrome: A4 pages with the page number in the footer and the chapter's
// title at the top, the cover and a title page unnumbered, each part on a
// page of its own, every chapter on a new page. It prints twice: the first
// print tells on which page every chapter and section starts (the PDF's
// named destinations), the second writes those numbers into the contents.
func bookPDF(chrome, htmlPath, pdfPath string, b *bookexport.Book) error {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return err
	}
	if p, err := filepath.Abs(pdfPath); err == nil {
		pdfPath = p
	}
	c, err := cdp.Launch(chrome, 1200, 900, "--allow-file-access-from-files")
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := c.OpenPage("", 1200, 900); err != nil {
		return err
	}
	c.Call("Emulation.setEmulatedMedia", map[string]any{"features": []map[string]string{{"name": "prefers-color-scheme", "value": "light"}}}, nil)
	if err := c.Call("Page.navigate", map[string]any{"url": fileURL(abs) + "#print"}, nil); err != nil {
		return err
	}
	waitLoaded(c)
	opts, _ := json.Marshal(map[string]string{"rights": b.Rights()})
	if _, err := c.Eval("(" + bookPrintJS + ")(" + string(opts) + ")"); err != nil {
		return fmt.Errorf("print layout: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	print := func() ([]byte, error) {
		var r struct{ Data string }
		if err := c.Call("Page.printToPDF", map[string]any{"printBackground": true, "preferCSSPageSize": true, "generateDocumentOutline": true, "generateTaggedPDF": true}, &r); err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(r.Data)
	}
	pdf, err := print()
	if err != nil {
		return err
	}
	if pages := bookexport.PDFPages(pdf); len(pages) > 0 {
		m, _ := json.Marshal(pages)
		if _, err := c.Eval("window.siBookPages(" + string(m) + ")"); err == nil {
			if again, err := print(); err == nil {
				pdf = again
			}
		}
	}
	return os.WriteFile(pdfPath, pdf, 0o644)
}

// bookPrintJS lays book.html out for paper: page rules (A4, margins, the
// page number, a running chapter title per chapter through named pages),
// the cover and a title page, part pages, and page-number slots in the
// contents ("000" for the measuring print, filled by siBookPages).
const bookPrintJS = `function(o){
var d=document,b=d.body,cs=getComputedStyle(b),bg=cs.backgroundColor,font=cs.fontFamily,muted=getComputedStyle(d.documentElement).getPropertyValue('--si-muted')||'#777';
var q=function(s){return JSON.stringify(s)};
var mark='font:9pt '+font+';color:'+muted;
var css=['@page{size:A4;margin:22mm 20mm 24mm;background:'+bg+';@bottom-center{content:counter(page);'+mark+'}}',
'@page si-front{@bottom-center{content:none}}',
'@page si-toc{@top-left{content:"Contents";'+mark+'}}',
'html,body{background:transparent!important}',
'.si-doc-guidebar,.si-book-bar,.si-book-start,.si-book-time,.si-doc-foot,.si-doc-head,.si-wm,.si-doc-search,.si-book-az{display:none!important}',
'.si-doc-layout,.si-doc-body{margin:0!important;padding:0!important;max-width:none!important;width:auto!important}',
'.si-book-cover{page:si-front;min-height:0;height:230mm;display:grid;align-content:center;break-after:page}',
'.si-book-cover .si-book-plate{box-shadow:none}',
'.si-book-titlepage{page:si-front;break-before:page;break-after:page;height:230mm;display:flex;flex-direction:column;justify-content:flex-end}',
'.si-book-titlepage .si-book-rights{margin-top:1.5em;color:'+muted+'}',
'#si-book-contents{page:si-toc}',
'.si-book-partpage{page:si-front;break-before:page;break-after:page;height:230mm;display:grid;align-content:center;text-align:center}',
'.si-book-partpage h1{font-size:2.4em}',
'.si-book-pg{display:inline-block;min-width:3ch;text-align:right;margin-left:auto;padding-left:1em;font-variant-numeric:tabular-nums;color:'+muted+'}',
'.si-book-ch>.si-book-ct{flex:1}',
'.si-book-ch ol li a{display:flex}',
'pre,figure,.si-figure,.si-image,.si-codeblock,.si-callouts,table,.si-callout,.si-alert{break-inside:avoid}',
'h1,h2,h3{break-after:avoid}'];
var cover=d.querySelector('.si-book-cover'),col=cover&&cover.querySelector('.si-book-colophon');
if(cover&&(col||o.rights)){var t=d.createElement('section');t.className='si-book-titlepage';if(col)t.appendChild(col);
 if(o.rights){var r=d.createElement('p');r.className='si-book-rights';r.textContent=o.rights;t.appendChild(r)}cover.after(t)}
var toc=d.getElementById('si-book-contents');
d.querySelectorAll('.si-book-part').forEach(function(p){var h=p.querySelector('h2'),a=p.querySelector('.si-book-ch a.si-book-ct');
 if(!h||!a)return;var s=d.getElementById(a.getAttribute('href').slice(1));if(!s)return;
 var pp=d.createElement('section');pp.className='si-book-partpage';var h1=d.createElement('h1');h1.textContent=h.textContent;pp.appendChild(h1);s.before(pp)});
d.querySelectorAll('section.si-book-chapter').forEach(function(s,i){
 var t=s.querySelector('.si-doc-title'),k=s.querySelector('.si-book-chap'),head=(k?k.textContent+' · ':'')+(t?t.textContent:'');
 s.style.page='si-ch'+i;css.push('@page si-ch'+i+'{@top-left{content:'+q(head)+';'+mark+'}}')});
if(toc)toc.querySelectorAll('a[href^="#"]').forEach(function(a){var sp=d.createElement('span');sp.className='si-book-pg';sp.dataset.id=a.getAttribute('href').slice(1);sp.textContent='000';
 if(a.classList.contains('si-book-ct'))a.after(sp);else a.appendChild(sp)});
var st=d.createElement('style');st.textContent=css.join('\n');d.head.appendChild(st);
window.siBookPages=function(m){d.querySelectorAll('.si-book-pg').forEach(function(sp){sp.textContent=m[sp.dataset.id]||''})};
return 1}`
