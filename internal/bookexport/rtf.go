package bookexport

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf16"
)

// RTF is the book as an RTF 1.x document: headings (as styles, every chapter
// on a new page), paragraphs with bold, italic, code and links, lists,
// simple tables, code blocks, captions and pictures (PNG or JPEG, an SVG
// through Options.Raster).
func RTF(b *Book, o Options) string {
	r := &rtf{o: o}
	r.w.WriteString(`{\rtf1\ansi\ansicpg1252\deff0\uc1` + "\n")
	r.w.WriteString(`{\fonttbl{\f0\froman\fcharset0 Georgia;}{\f1\fmodern\fcharset0 Courier New;}{\f2\fswiss\fcharset0 Helvetica;}}` + "\n")
	r.w.WriteString(`{\colortbl;\red85\green85\blue85;\red31\green95\blue173;\red242\green242\blue242;\red170\green170\blue170;}` + "\n")
	r.w.WriteString(`{\stylesheet{\s0\f0\fs22 Normal;}{\s1\f2\fs36\b\keepn heading 1;}{\s2\f2\fs28\b\keepn heading 2;}{\s3\f2\fs24\b\keepn heading 3;}{\s4\f2\fs56\b Title;}{\s5\f1\fs18 Code;}{\s6\f0\fs18\i\cf1\qc Caption;}{\s7\f0\fs22\li432 Quote;}}` + "\n")
	fmt.Fprintf(&r.w, `{\info{\title %s}{\author %s}}`+"\n", esc(b.Title), esc(b.Author))
	r.w.WriteString(`\paperw11906\paperh16838\margl1440\margr1440\margt1440\margb1440` + "\n")
	r.w.WriteString(`{\footer\pard\qc\f0\fs18{\field{\*\fldinst PAGE}{\fldrslt 1}}\par}` + "\n")
	r.par(`\s4\f2\fs56\b\sb2400\sa240`, []Run{{Text: b.Title}})
	if b.Subtitle != "" {
		r.par(`\s0\f0\fs30\cf1\sa480`, []Run{{Text: b.Subtitle}})
	}
	if by := b.Byline(); by != "" {
		r.par(`\s0\f0\fs22\sa120`, []Run{{Text: by}})
	}
	if rights := b.Rights(); rights != "" {
		r.par(`\s0\f0\fs22\sa120`, []Run{{Text: rights}})
	}
	r.blocks(b.Colophon, 0)
	// a contents page: the chapters (RTF has no field every reader fills)
	r.par(`\s1\f2\fs36\b\keepn\pagebb\sb240\sa240`, []Run{{Text: "Contents"}})
	for _, c := range b.Chapters {
		if c.Part != "" {
			r.par(`\s0\f0\fs22\b\sb120\sa60`, []Run{{Text: c.Part}})
		}
		r.par(`\s0\f0\fs22\li360\sa60`, []Run{{Text: c.Heading()}})
	}
	for _, c := range b.Chapters {
		if c.Part != "" {
			r.par(`\s1\f2\fs40\b\keepn\pagebb\sb2400\sa240`, []Run{{Text: c.Part}})
		}
		r.par(`\s1\f2\fs36\b\keepn\pagebb\sb480\sa240`, []Run{{Text: c.Heading()}})
		r.blocks(c.Blocks, 0)
	}
	r.w.WriteString("}\n")
	return r.w.String()
}

type rtf struct {
	o Options
	w strings.Builder
}

// esc writes text for RTF: braces and backslashes escaped, anything beyond
// ASCII as \uN? (UTF-16, signed), tabs as \tab, newlines as \line.
func esc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\\' || r == '{' || r == '}':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\line `)
		case r == '\t':
			b.WriteString(`\tab `)
		case r < 0x20:
		case r < 0x80:
			b.WriteRune(r)
		default:
			for _, u := range utf16.Encode([]rune{r}) {
				fmt.Fprintf(&b, `\u%d?`, int16(u))
			}
		}
	}
	return b.String()
}

func (r *rtf) par(fmtCodes string, runs []Run) {
	r.w.WriteString(`{\pard\plain` + fmtCodes + " ")
	r.runs(runs)
	r.w.WriteString("\\par}\n")
}

func (r *rtf) runs(runs []Run) {
	for _, run := range runs {
		codes := ""
		if run.Bold {
			codes += `\b`
		}
		if run.Italic {
			codes += `\i`
		}
		if run.Code {
			codes += `\f1\fs19`
		}
		t := esc(run.Text)
		if codes != "" {
			t = "{" + codes + " " + t + "}"
		}
		if run.Link != "" {
			t = `{\field{\*\fldinst HYPERLINK "` + esc(strings.ReplaceAll(run.Link, `"`, "%22")) + `"}{\fldrslt{\ul\cf2 ` + t + `}}}`
		}
		r.w.WriteString(t)
	}
}

func (r *rtf) blocks(bs []Block, lvl int) {
	for _, b := range bs {
		r.block(b, lvl)
	}
}

func (r *rtf) block(b Block, lvl int) {
	li := fmt.Sprintf(`\li%d`, 360*lvl)
	switch b.Kind {
	case Heading:
		runs := b.Runs
		if b.Num != "" {
			runs = append([]Run{{Text: b.Num + " "}}, runs...)
		}
		switch min(b.Level, 3) {
		case 2:
			r.par(`\s2\f2\fs28\b\keepn\sb360\sa120`, runs)
		default:
			r.par(`\s3\f2\fs24\b\keepn\sb240\sa120`, runs)
		}
	case Para:
		r.par(`\s0\f0\fs22\sa180`+li, b.Runs)
	case Code:
		r.w.WriteString(`{\pard\plain\s5\f1\fs18\keep\sb120\sa200\li` + fmt.Sprint(360*lvl+144) + `\cbpat3 ` + esc(CodeLines(b)) + "\\par}\n")
		for _, co := range b.Callouts {
			r.par(fmt.Sprintf(`\s0\f0\fs22\sa60\li%d\fi-360`, 360*lvl+360), append([]Run{{Text: "(" + co.N + ")\t", Bold: true}}, co.Runs...))
		}
	case Placeholder:
		r.par(`\s6\f0\fs18\i\cf1\qc\sa240`, b.Runs)
	case List:
		for i, it := range b.Items {
			mark := "\u2022\t"
			if b.Ordered {
				mark = fmt.Sprintf("%d.\t", b.Start+i)
			}
			for j, k := range it {
				if j == 0 && k.Kind == Para {
					r.par(fmt.Sprintf(`\s0\f0\fs22\sa60\li%d\fi-360`, 360*lvl+360), append([]Run{{Text: mark}}, k.Runs...))
					continue
				}
				r.block(k, lvl+1)
			}
		}
	case Table:
		cols := 0
		for _, row := range b.Rows {
			cols = max(cols, len(row))
		}
		if cols == 0 {
			return
		}
		width := (11906 - 2*1440) / cols
		for i, row := range b.Rows {
			r.w.WriteString(`\trowd\trgaph108\trleft0`)
			if i == 0 {
				r.w.WriteString(`\trhdr`)
			}
			for j := 1; j <= cols; j++ {
				fmt.Fprintf(&r.w, `\clbrdrt\brdrs\brdrw10\brdrcf4\clbrdrl\brdrs\brdrw10\brdrcf4\clbrdrb\brdrs\brdrw10\brdrcf4\clbrdrr\brdrs\brdrw10\brdrcf4\cellx%d`, width*j)
			}
			r.w.WriteString("\n")
			for j := 0; j < cols; j++ {
				r.w.WriteString(`\pard\intbl\f0\fs19 `)
				if j < len(row) {
					cell := row[j]
					if i == 0 {
						cell = append([]Run(nil), cell...)
						for k := range cell {
							cell[k].Bold = true
						}
					}
					r.runs(cell)
				}
				r.w.WriteString(`\cell` + "\n")
			}
			r.w.WriteString(`\row` + "\n")
		}
		r.w.WriteString(`\pard` + "\n")
		if b.Caption != "" {
			r.par(`\s6\f0\fs18\i\cf1\qc\sb120\sa240`, []Run{{Text: b.Caption}})
		}
	case Image:
		p, ok := r.o.load(b.Src, true)
		if !ok || (p.ext != "png" && p.ext != "jpeg") {
			r.par(`\s6\f0\fs18\i\cf1\qc\sa240`, []Run{{Text: "[Figure: " + firstNonEmpty(b.Caption, b.Alt, b.Src) + "]"}})
			return
		}
		pw, ph := fit(p.w, p.h, 451)
		blip := `\pngblip`
		if p.ext == "jpeg" {
			blip = `\jpegblip`
		}
		fmt.Fprintf(&r.w, `{\pard\plain\qc\keepn\sb120\sa60{\pict%s\picw%d\pich%d\picwgoal%d\pichgoal%d`+"\n", blip, p.w, p.h, int(pw*20), int(ph*20))
		h := hex.EncodeToString(p.data)
		for len(h) > 128 {
			r.w.WriteString(h[:128] + "\n")
			h = h[128:]
		}
		r.w.WriteString(h + "}\\par}\n")
		if b.Caption != "" {
			r.par(`\s6\f0\fs18\i\cf1\qc\sa240`, []Run{{Text: b.Caption}})
		}
	case Admonition:
		label := []Run{{Text: b.Label + ": ", Bold: true}}
		rest := b.Blocks
		if len(rest) > 0 && rest[0].Kind == Para {
			label, rest = append(label, rest[0].Runs...), rest[1:]
		}
		r.par(`\s7\f0\fs22\li432\sa180`, label)
		r.blocks(rest, lvl+1)
	case Quote:
		for _, k := range b.Blocks {
			if k.Kind == Para {
				r.par(`\s7\f0\fs22\i\li432\sa180`, k.Runs)
			} else {
				r.block(k, lvl+1)
			}
		}
	case Rule:
		r.w.WriteString(`{\pard\plain\brdrb\brdrs\brdrw10\brdrcf4\sa180\par}` + "\n")
	}
}
