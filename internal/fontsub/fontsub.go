// Package fontsub cuts a TrueType font down to the glyphs a text uses (for
// si build --svg, which embeds its fonts) and reads WOFF 1 files into the
// TrueType/OpenType font inside. Standard library only.
//
// The subset keeps every glyph id (so hmtx and kern stay valid as they are),
// empties the outlines of glyphs the text does not reach (a composite glyph
// keeps its parts) and maps only the text's characters (a new format 4 cmap).
// GSUB goes (a ligature or an alternate could point at an emptied glyph), and
// so do GPOS and GDEF (an SVG's text keeps its width through textLength), DSIG
// (the font changed) and the glyph names (post version 3).
// ponytail: glyph ids kept, so hmtx/loca stay full size (a few KB); renumber
// glyphs if that ever matters. CFF (OTTO), variable fonts (gvar) and text in
// scripts that need shaping are refused: the caller embeds them whole.
package fontsub

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
)

var be = binary.BigEndian

// ErrUnsupported: a font this package does not cut (embed it whole).
var ErrUnsupported = errors.New("fontsub: not a static TrueType font")

type table struct {
	tag  string
	data []byte
}

// parse reads an sfnt's tables.
func parse(f []byte) ([]table, error) {
	if len(f) < 12 {
		return nil, ErrUnsupported
	}
	switch be.Uint32(f) {
	case 0x00010000, 0x74727565: // TrueType, 'true'
	default:
		return nil, ErrUnsupported
	}
	n := int(be.Uint16(f[4:]))
	if len(f) < 12+16*n {
		return nil, ErrUnsupported
	}
	var out []table
	for i := range n {
		e := f[12+16*i:]
		off, ln := int(be.Uint32(e[8:])), int(be.Uint32(e[12:]))
		if off < 0 || ln < 0 || off+ln > len(f) {
			return nil, fmt.Errorf("fontsub: table %q out of the file", e[:4])
		}
		out = append(out, table{string(e[:4]), f[off : off+ln]})
	}
	return out, nil
}

func find(ts []table, tag string) []byte {
	for _, t := range ts {
		if t.tag == tag {
			return t.data
		}
	}
	return nil
}

// Shaped reports whether s has text in a script that needs shaping (Arabic,
// Hebrew, Indic, Thai…): a font for it is better embedded whole.
func Shaped(s string) bool {
	for _, r := range s {
		if r >= 0x0590 && r < 0x1E00 || r >= 0xA800 && r < 0xABFF || r >= 0xFB1D && r < 0xFEFF {
			return true
		}
	}
	return false
}

// Subset returns font f with only the outlines that the runes of text need.
func Subset(f []byte, text string) ([]byte, error) {
	ts, err := parse(f)
	if err != nil {
		return nil, err
	}
	if find(ts, "gvar") != nil || find(ts, "CFF ") != nil || find(ts, "CFF2") != nil {
		return nil, ErrUnsupported
	}
	head, maxp, loca, glyf, cmap := find(ts, "head"), find(ts, "maxp"), find(ts, "loca"), find(ts, "glyf"), find(ts, "cmap")
	if len(head) < 54 || len(maxp) < 6 || loca == nil || glyf == nil || cmap == nil {
		return nil, ErrUnsupported
	}
	ng, long := int(be.Uint16(maxp[4:])), be.Uint16(head[50:]) == 1
	offs := make([]int, ng+1)
	for i := range offs {
		switch {
		case long && 4*i+4 <= len(loca):
			offs[i] = int(be.Uint32(loca[4*i:]))
		case !long && 2*i+2 <= len(loca):
			offs[i] = 2 * int(be.Uint16(loca[2*i:]))
		default:
			return nil, errors.New("fontsub: short loca")
		}
		if offs[i] > len(glyf) {
			return nil, errors.New("fontsub: loca past glyf")
		}
	}
	gids, err := lookup(cmap, text)
	if err != nil {
		return nil, err
	}
	keep := map[int]bool{}
	var add func(g int)
	add = func(g int) {
		if g < 0 || g >= ng || keep[g] || offs[g] > offs[g+1] {
			return
		}
		keep[g] = true
		for _, c := range components(glyf[offs[g]:offs[g+1]]) {
			add(c)
		}
	}
	add(0)
	for _, g := range gids {
		add(g)
	}
	var ng2 bytes.Buffer
	loca2 := make([]byte, 0, len(loca))
	for g := 0; g < ng; g++ {
		at := ng2.Len()
		if long {
			loca2 = be.AppendUint32(loca2, uint32(at))
		} else {
			loca2 = be.AppendUint16(loca2, uint16(at/2))
		}
		if keep[g] {
			ng2.Write(glyf[offs[g]:offs[g+1]])
			for ng2.Len()%4 != 0 {
				ng2.WriteByte(0)
			}
		}
	}
	if long {
		loca2 = be.AppendUint32(loca2, uint32(ng2.Len()))
	} else {
		loca2 = be.AppendUint16(loca2, uint16(ng2.Len()/2))
	}
	var out []table
	for _, t := range ts {
		switch t.tag {
		case "GSUB", "GPOS", "GDEF", "DSIG":
			continue
		case "glyf":
			t.data = ng2.Bytes()
		case "loca":
			t.data = loca2
		case "cmap":
			t.data = cmap4(gids)
		case "post":
			if len(t.data) >= 32 { // version 3: no glyph names
				t.data = append(be.AppendUint32(nil, 0x00030000), t.data[4:32]...)
			}
		}
		out = append(out, t)
	}
	return write(out), nil
}

// cmap4 is a cmap with one format 4 subtable (3,1) for the runes in m: one
// segment per rune, plus the closing 0xFFFF one.
func cmap4(m map[rune]int) []byte {
	var rs []rune
	for r := range m {
		if r <= 0xFFFE {
			rs = append(rs, r)
		}
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })
	segs := len(rs) + 1
	es := 0
	for 1<<(es+1) <= segs {
		es++
	}
	ends, starts, deltas := []byte{}, []byte{}, []byte{}
	for _, r := range rs {
		ends = be.AppendUint16(ends, uint16(r))
		starts = be.AppendUint16(starts, uint16(r))
		deltas = be.AppendUint16(deltas, uint16(m[r]-int(r)))
	}
	ends = be.AppendUint16(ends, 0xFFFF)
	starts = be.AppendUint16(starts, 0xFFFF)
	deltas = be.AppendUint16(deltas, 1)
	sub := be.AppendUint16(nil, 4)
	sub = be.AppendUint16(sub, uint16(16+8*segs))
	sub = be.AppendUint16(sub, 0)
	sub = be.AppendUint16(sub, uint16(2*segs))
	sub = be.AppendUint16(sub, uint16(2<<es))
	sub = be.AppendUint16(sub, uint16(es))
	sub = be.AppendUint16(sub, uint16(2*segs-2<<es))
	sub = append(sub, ends...)
	sub = append(sub, 0, 0)
	sub = append(sub, starts...)
	sub = append(sub, deltas...)
	sub = append(sub, make([]byte, 2*segs)...) // idRangeOffsets: all 0
	return append([]byte{0, 0, 0, 1, 0, 3, 0, 1, 0, 0, 0, 12}, sub...)
}

// components are the glyphs a composite glyph is made of.
func components(g []byte) []int {
	if len(g) < 10 || int16(be.Uint16(g)) >= 0 {
		return nil
	}
	var out []int
	p := 10
	for p+4 <= len(g) {
		flags := be.Uint16(g[p:])
		out = append(out, int(be.Uint16(g[p+2:])))
		p += 4
		if flags&1 != 0 { // ARG_1_AND_2_ARE_WORDS
			p += 4
		} else {
			p += 2
		}
		switch {
		case flags&0x8 != 0: // WE_HAVE_A_SCALE
			p += 2
		case flags&0x40 != 0: // X_AND_Y_SCALE
			p += 4
		case flags&0x80 != 0: // TWO_BY_TWO
			p += 8
		}
		if flags&0x20 == 0 { // MORE_COMPONENTS
			break
		}
	}
	return out
}

// lookup maps the runes of text to glyph ids through a Unicode cmap
// subtable (format 12 or 4).
func lookup(cmap []byte, text string) (map[rune]int, error) {
	if len(cmap) < 4 {
		return nil, ErrUnsupported
	}
	n := int(be.Uint16(cmap[2:]))
	best, bestRank := -1, 0
	for i := range n {
		e := 4 + 8*i
		if e+8 > len(cmap) {
			break
		}
		pid, eid, off := be.Uint16(cmap[e:]), be.Uint16(cmap[e+2:]), int(be.Uint32(cmap[e+4:]))
		if off+2 > len(cmap) {
			continue
		}
		format, rank := be.Uint16(cmap[off:]), 0
		switch {
		case format == 12 && (pid == 3 && eid == 10 || pid == 0):
			rank = 2
		case format == 4 && (pid == 3 && eid == 1 || pid == 0):
			rank = 1
		}
		if rank > bestRank {
			best, bestRank = off, rank
		}
	}
	if best < 0 {
		return nil, ErrUnsupported
	}
	st := cmap[best:]
	runes := map[rune]bool{' ': true}
	for _, r := range text {
		runes[r] = true
	}
	out := map[rune]int{}
	if bestRank == 2 {
		if len(st) < 16 {
			return nil, ErrUnsupported
		}
		groups := int(be.Uint32(st[12:]))
		for r := range runes {
			for i := range groups {
				g := st[16+12*i:]
				if len(g) < 12 {
					break
				}
				lo, hi := rune(be.Uint32(g)), rune(be.Uint32(g[4:]))
				if r >= lo && r <= hi {
					out[r] = int(be.Uint32(g[8:])) + int(r-lo)
					break
				}
			}
		}
		return out, nil
	}
	if len(st) < 14 {
		return nil, ErrUnsupported
	}
	segs := int(be.Uint16(st[6:])) / 2
	if len(st) < 16+8*segs {
		return nil, ErrUnsupported
	}
	ends, starts, deltas, ranges := st[14:], st[16+2*segs:], st[16+4*segs:], st[16+6*segs:]
	for r := range runes {
		if r > 0xFFFF {
			continue
		}
		for i := range segs {
			end, start := rune(be.Uint16(ends[2*i:])), rune(be.Uint16(starts[2*i:]))
			if r > end {
				continue
			}
			if r < start {
				break
			}
			delta, ro := be.Uint16(deltas[2*i:]), int(be.Uint16(ranges[2*i:]))
			if ro == 0 {
				out[r] = int(uint16(r) + delta)
				break
			}
			at := 16 + 6*segs + 2*i + ro + 2*int(r-start)
			if at+2 <= len(st) {
				if g := be.Uint16(st[at:]); g != 0 {
					out[r] = int(g + delta)
				}
			}
			break
		}
	}
	return out, nil
}

// write lays tables out as an sfnt with fresh checksums.
func write(ts []table) []byte {
	sort.Slice(ts, func(i, j int) bool { return ts[i].tag < ts[j].tag })
	n := len(ts)
	es := 0
	for 1<<(es+1) <= n {
		es++
	}
	var b bytes.Buffer
	hdr := make([]byte, 12+16*n)
	be.PutUint32(hdr, 0x00010000)
	be.PutUint16(hdr[4:], uint16(n))
	be.PutUint16(hdr[6:], uint16(16<<es))
	be.PutUint16(hdr[8:], uint16(es))
	be.PutUint16(hdr[10:], uint16(16*n-16<<es))
	b.Write(hdr)
	headAt := -1
	for i, t := range ts {
		data := t.data
		if t.tag == "head" && len(data) >= 12 {
			data = append([]byte{}, data...)
			be.PutUint32(data[8:], 0) // checkSumAdjustment, set below
			headAt = b.Len()
		}
		e := hdr[12+16*i:]
		copy(e, t.tag)
		be.PutUint32(e[4:], sum(data))
		be.PutUint32(e[8:], uint32(b.Len()))
		be.PutUint32(e[12:], uint32(len(data)))
		b.Write(data)
		for b.Len()%4 != 0 {
			b.WriteByte(0)
		}
	}
	out := b.Bytes()
	copy(out, hdr)
	if headAt >= 0 {
		be.PutUint32(out[headAt+8:], 0xB1B0AFBA-sum(out))
	}
	return out
}

func sum(b []byte) uint32 {
	var s uint32
	for i := 0; i < len(b); i += 4 {
		var w [4]byte
		copy(w[:], b[i:])
		s += be.Uint32(w[:])
	}
	return s
}

// FromWOFF returns the sfnt font inside a WOFF 1 file.
func FromWOFF(w []byte) ([]byte, error) {
	if len(w) < 44 || string(w[:4]) != "wOFF" {
		return nil, ErrUnsupported
	}
	n := int(be.Uint16(w[12:]))
	var ts []table
	for i := range n {
		e := w[44+20*i:]
		if len(e) < 20 {
			return nil, errors.New("fontsub: short WOFF directory")
		}
		off, cl, ol := int(be.Uint32(e[4:])), int(be.Uint32(e[8:])), int(be.Uint32(e[12:]))
		if off+cl > len(w) {
			return nil, errors.New("fontsub: WOFF table out of the file")
		}
		data := w[off : off+cl]
		if cl < ol {
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			if data, err = io.ReadAll(io.LimitReader(r, int64(ol))); err != nil {
				return nil, err
			}
		}
		ts = append(ts, table{string(e[:4]), data})
	}
	return write(ts), nil
}
