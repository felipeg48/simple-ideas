package fontsub

import (
	"bytes"
	"compress/zlib"
	"os"
	"testing"
)

// tiny is a TrueType font with four glyphs: .notdef, A (a simple glyph),
// B (a composite of A) and C (simple), mapped by a format 4 cmap.
func tiny() []byte {
	simple := func(fill byte) []byte { // one contour, one point
		g := []byte{0, 1, 0, 0, 0, 0, 0, 10, 0, 10, 0, 0, 0, 0, 1, fill, fill}
		for len(g)%4 != 0 {
			g = append(g, 0)
		}
		return g
	}
	comp := []byte{0xFF, 0xFF, 0, 0, 0, 0, 0, 10, 0, 10, 0x00, 0x02, 0, 1, 0, 0} // ARGS_ARE_XY_VALUES, glyph 1, args 0 0
	glyphs := [][]byte{simple(0), simple(0xAA), comp, simple(0xCC)}
	var glyf []byte
	loca := []byte{}
	for _, g := range glyphs {
		loca = be.AppendUint32(loca, uint32(len(glyf)))
		glyf = append(glyf, g...)
	}
	loca = be.AppendUint32(loca, uint32(len(glyf)))
	head := make([]byte, 54)
	be.PutUint32(head, 0x00010000)
	be.PutUint32(head[12:], 0x5F0F3CF5)
	be.PutUint16(head[18:], 1000)
	be.PutUint16(head[50:], 1) // long loca
	maxp := make([]byte, 6)
	be.PutUint32(maxp, 0x00005000)
	be.PutUint16(maxp[4:], 4)
	// cmap: one (3,1) format 4 subtable: 'A'-'C' → 1-3, and the 0xFFFF end
	sub := []byte{0, 4, 0, 32, 0, 0, 0, 4, 0, 4, 0, 1, 0, 0,
		0, 'C', 0xFF, 0xFF, // ends
		0, 0, // reserved
		0, 'A', 0xFF, 0xFF, // starts
		0, 0, 0, 1, // deltas
		0, 0, 0, 0} // range offsets
	d := 1 - 'A'
	be.PutUint16(sub[24:], uint16(d)) // delta for A-C: A → 1
	cmap := append([]byte{0, 0, 0, 1, 0, 3, 0, 1, 0, 0, 0, 12}, sub...)
	return write([]table{{"head", head}, {"maxp", maxp}, {"loca", loca}, {"glyf", glyf}, {"cmap", cmap}, {"GSUB", []byte{0, 1, 0, 0}}})
}

func glyphs(t *testing.T, f []byte) [][]byte {
	t.Helper()
	ts, err := parse(f)
	if err != nil {
		t.Fatal(err)
	}
	loca, glyf := find(ts, "loca"), find(ts, "glyf")
	var out [][]byte
	for i := 0; i+8 <= len(loca); i += 4 {
		out = append(out, glyf[be.Uint32(loca[i:]):be.Uint32(loca[i+4:])])
	}
	return out
}

func TestSubset(t *testing.T) {
	f := tiny()
	got, err := Subset(f, "B")
	if err != nil {
		t.Fatal(err)
	}
	g := glyphs(t, got)
	if len(g) != 4 || len(g[0]) == 0 || len(g[1]) == 0 || len(g[2]) == 0 || len(g[3]) != 0 {
		t.Errorf("B keeps .notdef, its part A and itself, and empties C: %v", g)
	}
	ts, _ := parse(got)
	if find(ts, "GSUB") != nil || find(ts, "cmap") == nil {
		t.Error("GSUB goes, cmap stays")
	}
	if m, err := lookup(find(ts, "cmap"), "ABC"); err != nil || len(m) != 1 || m['B'] != 2 {
		t.Errorf("the new cmap maps only B (to 2): %v %v", m, err)
	}
	if sum(got) != 0xB1B0AFBA {
		t.Errorf("whole-font checksum %x", sum(got))
	}
	if _, err := Subset([]byte("OTTO........"), "x"); err != ErrUnsupported {
		t.Errorf("CFF: %v", err)
	}
	// WOFF 1: the same font, every table compressed
	ts, _ = parse(f)
	var w bytes.Buffer
	w.Write([]byte("wOFF\x00\x01\x00\x00"))
	w.Write(make([]byte, 4))
	w.Write([]byte{0, byte(len(ts))})
	w.Write(make([]byte, 44-14))
	dir := make([]byte, 20*len(ts))
	var body bytes.Buffer
	for i, tb := range ts {
		var z bytes.Buffer
		zw := zlib.NewWriter(&z)
		zw.Write(tb.data)
		zw.Close()
		e := dir[20*i:]
		copy(e, tb.tag)
		be.PutUint32(e[4:], uint32(44+len(dir)+body.Len()))
		data := z.Bytes()
		if len(data) >= len(tb.data) {
			data = tb.data
		}
		be.PutUint32(e[8:], uint32(len(data)))
		be.PutUint32(e[12:], uint32(len(tb.data)))
		body.Write(data)
	}
	w.Write(dir)
	w.Write(body.Bytes())
	back, err := FromWOFF(w.Bytes())
	if err != nil || len(glyphs(t, back)) != 4 {
		t.Fatalf("WOFF: %v", err)
	}
	if !Shaped("שלום") || Shaped("Ação · 1936 – “x”") {
		t.Error("Shaped")
	}
}

// TestSubsetRealFont: a system TrueType font cut to a word is much smaller.
func TestSubsetRealFont(t *testing.T) {
	f, err := os.ReadFile("/System/Library/Fonts/Supplemental/Arial.ttf")
	if err != nil {
		t.Skip("no Arial here")
	}
	got, err := Subset(f, "Hitos en Algoritmos 1936")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) > len(f)/3 {
		t.Errorf("%d bytes of %d", len(got), len(f))
	}
	if g := glyphs(t, got); len(g) < 100 {
		t.Errorf("glyph ids kept: %d", len(g))
	}
}
