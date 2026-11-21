package bookexport

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// Word measures: A4 in twentieths of a point, 1 inch margins, and the text
// width in points for pictures.
const (
	docxPageW, docxPageH, docxMargin = 11906, 16838, 1440
	docxTextW                        = float64(docxPageW-2*docxMargin) / 20
)

// Docx is the book as a Word document (Office Open XML, written by hand):
// Title, Heading 1-3, Normal, Code, Caption, List Bullet / Number, Quote and
// Table Grid styles, pictures embedded, every chapter on a new page, page
// numbers in the footer, and a table of contents field Word fills on open.
func Docx(b *Book, o Options) ([]byte, error) {
	d := &docx{o: o, pics: map[string]string{}}
	body := d.document(b)
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	add := func(name, text string) error {
		w, err := z.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(text))
		return err
	}
	types := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
		`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
		`<Default Extension="xml" ContentType="application/xml"/>` +
		`<Default Extension="png" ContentType="image/png"/><Default Extension="jpeg" ContentType="image/jpeg"/><Default Extension="gif" ContentType="image/gif"/>` +
		`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>` +
		`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>` +
		`<Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>` +
		`<Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>` +
		`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>` +
		`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>` +
		`</Types>`
	rels := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>` +
		`</Relationships>`
	var drels strings.Builder
	drels.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
		`<Relationship Id="rIdStyles" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>` +
		`<Relationship Id="rIdNumbering" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>` +
		`<Relationship Id="rIdSettings" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/>` +
		`<Relationship Id="rIdFooter" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>`)
	for _, r := range d.rels {
		drels.WriteString(r)
	}
	drels.WriteString(`</Relationships>`)
	core := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">` +
		`<dc:title>` + x(b.Title) + `</dc:title><dc:creator>` + x(b.Author) + `</dc:creator>` +
		`<dcterms:created xsi:type="dcterms:W3CDTF">` + time.Now().UTC().Format(time.RFC3339) + `</dcterms:created></cp:coreProperties>`
	settings := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:defaultTabStop w:val="720"/><w:updateFields w:val="true"/><w:compat><w:compatSetting w:name="compatibilityMode" w:uri="http://schemas.microsoft.com/office/word" w:val="15"/></w:compat></w:settings>`
	footer := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:pPr><w:jc w:val="center"/></w:pPr>` +
		`<w:r><w:fldChar w:fldCharType="begin"/></w:r><w:r><w:instrText xml:space="preserve"> PAGE </w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r><w:r><w:t>1</w:t></w:r><w:r><w:fldChar w:fldCharType="end"/></w:r></w:p></w:ftr>`
	for _, f := range [][2]string{{"[Content_Types].xml", types}, {"_rels/.rels", rels}, {"docProps/core.xml", core},
		{"word/document.xml", body}, {"word/styles.xml", docxStyles}, {"word/numbering.xml", d.numbering()},
		{"word/settings.xml", settings}, {"word/footer1.xml", footer}, {"word/_rels/document.xml.rels", drels.String()}} {
		if err := add(f[0], f[1]); err != nil {
			return nil, err
		}
	}
	for _, m := range d.media {
		w, err := z.Create("word/media/" + m.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(m.data); err != nil {
			return nil, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type docx struct {
	o     Options
	w     strings.Builder
	rels  []string
	media []struct {
		name string
		data []byte
	}
	pics  map[string]string // image file -> relationship id
	taken map[string]bool
	nRel  int
	nPic  int
	lists []int // the ordered lists: each restarts at its start number
}

// x escapes text for XML (and drops characters XML cannot hold).
func x(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(s))
	return strings.ReplaceAll(b.String(), "&#xA;", "\n")
}

func (d *docx) document(b *Book) string {
	d.w.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><w:body>`)
	d.para("Title", []Run{{Text: b.Title}}, "")
	if b.Subtitle != "" {
		d.para("Subtitle", []Run{{Text: b.Subtitle}}, "")
	}
	if by := b.Byline(); by != "" {
		d.para("Normal", []Run{{Text: by}}, "")
	}
	if r := b.Rights(); r != "" {
		d.para("Normal", []Run{{Text: r}}, "")
	}
	d.blocks(b.Colophon, 0)
	// the contents: a field Word fills (settings.xml asks it to on open); until
	// then it lists the chapters
	d.para("TOCHeading", []Run{{Text: "Contents"}}, `<w:pageBreakBefore/>`)
	var entries []string
	for _, c := range b.Chapters {
		if c.Part != "" {
			entries = append(entries, c.Part)
		}
		entries = append(entries, c.Heading())
	}
	if len(entries) == 0 {
		entries = []string{" "}
	}
	for i, e := range entries {
		d.w.WriteString(`<w:p>`)
		if i == 0 {
			d.w.WriteString(`<w:r><w:fldChar w:fldCharType="begin" w:dirty="true"/></w:r><w:r><w:instrText xml:space="preserve"> TOC \o "1-3" \h \z \u </w:instrText></w:r><w:r><w:fldChar w:fldCharType="separate"/></w:r>`)
		}
		d.w.WriteString(`<w:r><w:t xml:space="preserve">` + x(e) + `</w:t></w:r>`)
		if i == len(entries)-1 {
			d.w.WriteString(`<w:r><w:fldChar w:fldCharType="end"/></w:r>`)
		}
		d.w.WriteString(`</w:p>`)
	}
	for _, c := range b.Chapters {
		if c.Part != "" {
			d.para("Heading1", []Run{{Text: c.Part}}, "")
		}
		d.para("Heading1", []Run{{Text: c.Heading()}}, "")
		d.blocks(c.Blocks, 0)
	}
	fmt.Fprintf(&d.w, `<w:sectPr><w:footerReference w:type="default" r:id="rIdFooter"/><w:pgSz w:w="%d" w:h="%d"/><w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`,
		docxPageW, docxPageH, docxMargin, docxMargin, docxMargin, docxMargin)
	return d.w.String()
}

// para writes one paragraph of style with runs; ppr adds paragraph properties.
func (d *docx) para(style string, runs []Run, ppr string) {
	d.w.WriteString(`<w:p><w:pPr><w:pStyle w:val="` + style + `"/>` + ppr + `</w:pPr>`)
	d.runs(runs)
	d.w.WriteString(`</w:p>`)
}

func (d *docx) runs(runs []Run) {
	for _, r := range runs {
		var rpr strings.Builder
		if r.Code {
			rpr.WriteString(`<w:rStyle w:val="CodeChar"/>`)
		} else if r.Link != "" {
			rpr.WriteString(`<w:rStyle w:val="Hyperlink"/>`)
		}
		if r.Bold {
			rpr.WriteString(`<w:b/>`)
		}
		if r.Italic {
			rpr.WriteString(`<w:i/>`)
		}
		run := `<w:r>`
		if rpr.Len() > 0 {
			run += `<w:rPr>` + rpr.String() + `</w:rPr>`
		}
		run += textRuns(r.Text) + `</w:r>`
		if r.Link != "" {
			d.nRel++
			id := fmt.Sprintf("rIdL%d", d.nRel)
			d.rels = append(d.rels, `<Relationship Id="`+id+`" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="`+xattr(r.Link)+`" TargetMode="External"/>`)
			run = `<w:hyperlink r:id="` + id + `">` + run + `</w:hyperlink>`
		}
		d.w.WriteString(run)
	}
}

// textRuns is text inside a run: line breaks and tabs as Word writes them.
func textRuns(t string) string {
	var b strings.Builder
	for i, line := range strings.Split(t, "\n") {
		if i > 0 {
			b.WriteString(`<w:br/>`)
		}
		for j, part := range strings.Split(line, "\t") {
			if j > 0 {
				b.WriteString(`<w:tab/>`)
			}
			if part != "" {
				b.WriteString(`<w:t xml:space="preserve">` + x(part) + `</w:t>`)
			}
		}
	}
	return b.String()
}

func xattr(s string) string {
	return strings.NewReplacer(`"`, "&quot;").Replace(x(s))
}

// blocks writes blocks; lvl is the list depth they sit in.
func (d *docx) blocks(bs []Block, lvl int) {
	for _, b := range bs {
		d.block(b, lvl)
	}
}

func (d *docx) block(b Block, lvl int) {
	ind := ""
	if lvl > 0 {
		ind = fmt.Sprintf(`<w:ind w:left="%d"/>`, 720*lvl)
	}
	switch b.Kind {
	case Heading:
		runs := b.Runs
		if b.Num != "" {
			runs = append([]Run{{Text: b.Num + " "}}, runs...)
		}
		d.para(fmt.Sprintf("Heading%d", min(b.Level, 3)), runs, "")
	case Para:
		d.para("Normal", b.Runs, ind)
	case Code, Placeholder:
		if b.Kind == Placeholder {
			d.para("Caption", b.Runs, ind)
			return
		}
		d.w.WriteString(`<w:p><w:pPr><w:pStyle w:val="Code"/>` + ind + `</w:pPr><w:r>` + textRuns(CodeLines(b)) + `</w:r></w:p>`)
		for _, co := range b.Callouts {
			d.para("Normal", append([]Run{{Text: "(" + co.N + ") ", Bold: true}}, co.Runs...), fmt.Sprintf(`<w:ind w:left="%d" w:hanging="360"/>`, 360+720*lvl))
		}
	case List:
		num := 1
		style := "ListBullet"
		if b.Ordered {
			d.lists = append(d.lists, max(b.Start, 1))
			num, style = len(d.lists)+1, "ListNumber"
		}
		for _, it := range b.Items {
			for i, k := range it {
				if i == 0 && k.Kind == Para {
					d.para(style, k.Runs, fmt.Sprintf(`<w:numPr><w:ilvl w:val="%d"/><w:numId w:val="%d"/></w:numPr>`, min(lvl, 8), num))
					continue
				}
				d.block(k, lvl+1)
			}
		}
	case Table:
		cols := 0
		for _, r := range b.Rows {
			cols = max(cols, len(r))
		}
		if cols == 0 {
			return
		}
		d.w.WriteString(`<w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="5000" w:type="pct"/><w:tblLook w:val="04A0" w:firstRow="1" w:lastRow="0" w:firstColumn="0" w:lastColumn="0" w:noHBand="0" w:noVBand="1"/></w:tblPr><w:tblGrid>`)
		for i := 0; i < cols; i++ {
			fmt.Fprintf(&d.w, `<w:gridCol w:w="%d"/>`, (docxPageW-2*docxMargin)/cols)
		}
		d.w.WriteString(`</w:tblGrid>`)
		for i, r := range b.Rows {
			d.w.WriteString(`<w:tr>`)
			if i == 0 {
				d.w.WriteString(`<w:trPr><w:tblHeader/></w:trPr>`)
			}
			for j := 0; j < cols; j++ {
				var cell []Run
				if j < len(r) {
					cell = r[j]
				}
				if i == 0 {
					cell = append([]Run(nil), cell...)
					for k := range cell {
						cell[k].Bold = true
					}
				}
				d.w.WriteString(`<w:tc><w:tcPr><w:tcW w:w="0" w:type="auto"/></w:tcPr>`)
				d.para("TableText", cell, "")
				d.w.WriteString(`</w:tc>`)
			}
			d.w.WriteString(`</w:tr>`)
		}
		d.w.WriteString(`</w:tbl>`)
		if b.Caption != "" {
			d.para("Caption", []Run{{Text: b.Caption}}, "")
		}
	case Image:
		if !d.picture(b) {
			d.para("Caption", []Run{{Text: "[Figure: " + firstNonEmpty(b.Caption, b.Alt, b.Src) + "]", Italic: true}}, "")
			return
		}
		if b.Caption != "" {
			d.para("Caption", []Run{{Text: b.Caption}}, "")
		}
	case Admonition:
		label := []Run{{Text: b.Label + ": ", Bold: true}}
		if len(b.Blocks) > 0 && b.Blocks[0].Kind == Para {
			d.para("Quote", append(label, b.Blocks[0].Runs...), ind)
			d.quoted(b.Blocks[1:], ind)
		} else {
			d.para("Quote", label, ind)
			d.quoted(b.Blocks, ind)
		}
	case Quote:
		d.quoted(b.Blocks, ind)
	case Rule:
		d.w.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="999999"/></w:pBdr></w:pPr></w:p>`)
	}
}

func (d *docx) quoted(bs []Block, ind string) {
	for _, k := range bs {
		if k.Kind == Para {
			d.para("Quote", k.Runs, ind)
		} else {
			d.block(k, 1)
		}
	}
}

// picture embeds an image paragraph, sized to the text width at most.
func (d *docx) picture(b Block) bool {
	id, ok := d.pics[b.Src]
	p, loaded := d.o.load(b.Src, true)
	if !loaded {
		return false
	}
	if !ok {
		d.nRel++
		id = fmt.Sprintf("rIdP%d", d.nRel)
		name := strings.TrimSuffix(baseName(b.Src), extOf(b.Src)) + "." + p.ext
		if d.taken == nil {
			d.taken = map[string]bool{}
		}
		for i := 2; d.taken[name]; i++ {
			name = fmt.Sprintf("%s-%d.%s", strings.TrimSuffix(baseName(b.Src), extOf(b.Src)), i, p.ext)
		}
		d.taken[name] = true
		d.media = append(d.media, struct {
			name string
			data []byte
		}{name, p.data})
		d.rels = append(d.rels, `<Relationship Id="`+id+`" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/`+xattr(name)+`"/>`)
		d.pics[b.Src] = id
	}
	pw, ph := fit(p.w, p.h, docxTextW)
	cx, cy := int(pw*12700), int(ph*12700) // EMU per point
	d.nPic++
	fmt.Fprintf(&d.w, `<w:p><w:pPr><w:pStyle w:val="Figure"/></w:pPr><w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="%d" name="Picture %d" descr="%s"/><wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="%d" name="Picture %d"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r></w:p>`,
		cx, cy, d.nPic, d.nPic, xattr(b.Alt), d.nPic, d.nPic, id, cx, cy)
	return true
}

// numbering: numId 1 is the bullet list; every ordered list gets its own
// numId (2, 3, …) on the decimal list, restarting at its start number.
func (d *docx) numbering() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:abstractNum w:abstractNumId="0"><w:multiLevelType w:val="hybridMultilevel"/>`)
	bullets := []string{"•", "◦", "▪"}
	for l := 0; l < 9; l++ {
		fmt.Fprintf(&b, `<w:lvl w:ilvl="%d"><w:start w:val="1"/><w:numFmt w:val="bullet"/><w:lvlText w:val="%s"/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="%d" w:hanging="360"/></w:pPr></w:lvl>`, l, bullets[l%3], 720*(l+1))
	}
	b.WriteString(`</w:abstractNum><w:abstractNum w:abstractNumId="1"><w:multiLevelType w:val="hybridMultilevel"/>`)
	fmts := []string{"decimal", "lowerLetter", "lowerRoman"}
	for l := 0; l < 9; l++ {
		fmt.Fprintf(&b, `<w:lvl w:ilvl="%d"><w:start w:val="1"/><w:numFmt w:val="%s"/><w:lvlText w:val="%%%d."/><w:lvlJc w:val="left"/><w:pPr><w:ind w:left="%d" w:hanging="360"/></w:pPr></w:lvl>`, l, fmts[l%3], l+1, 720*(l+1))
	}
	b.WriteString(`</w:abstractNum><w:num w:numId="1"><w:abstractNumId w:val="0"/></w:num>`)
	for i, start := range d.lists {
		fmt.Fprintf(&b, `<w:num w:numId="%d"><w:abstractNumId w:val="1"/><w:lvlOverride w:ilvl="0"><w:startOverride w:val="%d"/></w:lvlOverride></w:num>`, i+2, start)
	}
	b.WriteString(`</w:numbering>`)
	return b.String()
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

func extOf(p string) string {
	b := baseName(p)
	if i := strings.LastIndex(b, "."); i >= 0 {
		return b[i:]
	}
	return ""
}

var docxStyles = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">` +
	`<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Georgia" w:hAnsi="Georgia" w:eastAsia="Georgia" w:cs="Georgia"/><w:sz w:val="22"/><w:szCs w:val="22"/><w:lang w:val="en-US"/></w:rPr></w:rPrDefault>` +
	`<w:pPrDefault><w:pPr><w:spacing w:after="160" w:line="300" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>` +
	`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="2400" w:after="240"/></w:pPr><w:rPr><w:rFonts w:ascii="Helvetica Neue" w:hAnsi="Helvetica Neue"/><w:b/><w:sz w:val="56"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Subtitle"><w:name w:val="Subtitle"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:after="480"/></w:pPr><w:rPr><w:color w:val="555555"/><w:sz w:val="30"/></w:rPr></w:style>` +
	heading(1, 36, 480, `<w:pageBreakBefore/>`) + heading(2, 28, 360, "") + heading(3, 24, 240, "") +
	`<w:style w:type="paragraph" w:styleId="TOCHeading"><w:name w:val="TOC Heading"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:spacing w:before="240" w:after="240"/></w:pPr><w:rPr><w:rFonts w:ascii="Helvetica Neue" w:hAnsi="Helvetica Neue"/><w:b/><w:sz w:val="36"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Code"><w:name w:val="Code"/><w:basedOn w:val="Normal"/><w:qFormat/><w:pPr><w:keepLines/><w:shd w:val="clear" w:color="auto" w:fill="F2F2F2"/><w:spacing w:before="120" w:after="200" w:line="240" w:lineRule="auto"/><w:ind w:left="144" w:right="144"/></w:pPr><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/><w:sz w:val="18"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Caption"><w:name w:val="caption"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:after="240"/><w:jc w:val="center"/></w:pPr><w:rPr><w:i/><w:color w:val="555555"/><w:sz w:val="18"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Figure"><w:name w:val="Figure"/><w:basedOn w:val="Normal"/><w:next w:val="Caption"/><w:pPr><w:keepNext/><w:spacing w:before="120" w:after="60"/><w:jc w:val="center"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="ListBullet"><w:name w:val="List Bullet"/><w:basedOn w:val="Normal"/><w:qFormat/><w:pPr><w:numPr><w:numId w:val="1"/></w:numPr><w:spacing w:after="60"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="ListNumber"><w:name w:val="List Number"/><w:basedOn w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:after="60"/></w:pPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:pBdr><w:left w:val="single" w:sz="18" w:space="8" w:color="BBBBBB"/></w:pBdr><w:ind w:left="432" w:right="432"/></w:pPr><w:rPr><w:color w:val="404040"/></w:rPr></w:style>` +
	`<w:style w:type="paragraph" w:styleId="TableText"><w:name w:val="Table Text"/><w:basedOn w:val="Normal"/><w:pPr><w:spacing w:before="40" w:after="40" w:line="240" w:lineRule="auto"/></w:pPr><w:rPr><w:sz w:val="19"/></w:rPr></w:style>` +
	`<w:style w:type="character" w:default="1" w:styleId="DefaultParagraphFont"><w:name w:val="Default Paragraph Font"/><w:uiPriority w:val="1"/><w:semiHidden/></w:style>` +
	`<w:style w:type="character" w:styleId="CodeChar"><w:name w:val="Code Char"/><w:rPr><w:rFonts w:ascii="Courier New" w:hAnsi="Courier New" w:cs="Courier New"/><w:sz w:val="19"/><w:shd w:val="clear" w:color="auto" w:fill="F2F2F2"/></w:rPr></w:style>` +
	`<w:style w:type="character" w:styleId="Hyperlink"><w:name w:val="Hyperlink"/><w:rPr><w:color w:val="1F5FAD"/><w:u w:val="single"/></w:rPr></w:style>` +
	`<w:style w:type="table" w:default="1" w:styleId="TableNormal"><w:name w:val="Normal Table"/><w:semiHidden/><w:tblPr><w:tblInd w:w="0" w:type="dxa"/><w:tblCellMar><w:top w:w="0" w:type="dxa"/><w:left w:w="108" w:type="dxa"/><w:bottom w:w="0" w:type="dxa"/><w:right w:w="108" w:type="dxa"/></w:tblCellMar></w:tblPr></w:style>` +
	`<w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/><w:basedOn w:val="TableNormal"/><w:tblPr><w:tblBorders><w:top w:val="single" w:sz="4" w:space="0" w:color="AAAAAA"/><w:left w:val="single" w:sz="4" w:space="0" w:color="AAAAAA"/><w:bottom w:val="single" w:sz="4" w:space="0" w:color="AAAAAA"/><w:right w:val="single" w:sz="4" w:space="0" w:color="AAAAAA"/><w:insideH w:val="single" w:sz="4" w:space="0" w:color="AAAAAA"/><w:insideV w:val="single" w:sz="4" w:space="0" w:color="AAAAAA"/></w:tblBorders></w:tblPr></w:style>` +
	`</w:styles>`

func heading(n, size, before int, extra string) string {
	return fmt.Sprintf(`<w:style w:type="paragraph" w:styleId="Heading%d"><w:name w:val="heading %d"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:keepLines/>%s<w:spacing w:before="%d" w:after="120"/><w:outlineLvl w:val="%d"/></w:pPr><w:rPr><w:rFonts w:ascii="Helvetica Neue" w:hAnsi="Helvetica Neue"/><w:b/><w:sz w:val="%d"/></w:rPr></w:style>`,
		n, n, extra, before, n-1, size)
}
