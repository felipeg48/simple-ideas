package main

import (
	"strings"
	"testing"
)

func TestRSGenerator(t *testing.T) {
	// published generator polynomial for 7 EC codewords, in alpha-exponents
	wantExp := []int{0, 87, 229, 146, 149, 238, 102, 21}
	g := rsGen(7)
	if len(g) != len(wantExp) {
		t.Fatalf("rsGen(7) len=%d want %d", len(g), len(wantExp))
	}
	for i, e := range wantExp {
		if g[i] != gfExp[e] {
			t.Errorf("rsGen(7)[%d]=%d want a^%d=%d", i, g[i], e, gfExp[e])
		}
	}
}

func TestFormatBits(t *testing.T) {
	// published format-info strings for EC level L, masks 0..7
	want := []int{0x77C4, 0x72F3, 0x7DAA, 0x789D, 0x662F, 0x6318, 0x6C41, 0x6976}
	for mask, w := range want {
		if got := formatBits(mask); got != w {
			t.Errorf("formatBits(%d)=0x%04X want 0x%04X", mask, got, w)
		}
	}
}

func TestQRCodewordsCountAndVersion(t *testing.T) {
	v, code, err := qrCodewords("http://192.168.1.5:7070/")
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != v.dataCW+v.ecCW {
		t.Errorf("code len=%d want %d", len(code), v.dataCW+v.ecCW)
	}
	if v.size != 25 { // 24-byte URL -> version 2
		t.Errorf("version size=%d want 25 (v2)", v.size)
	}
	if _, _, err := qrCodewords(strings.Repeat("x", 200)); err == nil {
		t.Error("expected error for over-long input")
	}
}

func TestQRStringRenders(t *testing.T) {
	s, err := qrString("http://192.168.1.5:7070/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, "\x1b[40m") || !strings.Contains(s, "\x1b[107m") {
		t.Error("expected ANSI dark and light modules")
	}
}

// The longest short remote link (any IPv4 address and port, the longest role,
// a 128-bit hex key) must fit the QR codes qr.go makes.
func TestRemoteLinkFitsAQR(t *testing.T) {
	u := "http://255.255.255.255:65535/clicker?k=" + strings.Repeat("f", 32)
	if _, err := qrString(u); err != nil {
		t.Fatalf("%d-byte remote link: %v", len(u), err)
	}
}

// The format information (EC level + mask) sits where a reader looks for it:
// copy 1 down column 8 and along row 8 by the top-left finder, copy 2 split
// between the other two finders. Read back from the printed code, both copies
// must agree and be a valid level-L format word.
func TestQRFormatInfoPlacement(t *testing.T) {
	s, err := qrString("http://192.168.1.5:7071/clicker?k=" + strings.Repeat("a", 32))
	if err != nil {
		t.Fatal(err)
	}
	var m [][]bool
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		var row []bool
		for _, cell := range strings.Split(line, "\x1b[0m")[:strings.Count(line, "\x1b[0m")] {
			row = append(row, strings.HasPrefix(cell, "\x1b[40m"))
		}
		m = append(m, row)
	}
	const q = 2 // quiet zone
	size := len(m) - 2*q
	at := func(r, c int) bool { return m[r+q][c+q] }
	var one, two int
	set := func(f *int, i int, dark bool) {
		if dark {
			*f |= 1 << i
		}
	}
	for i := 0; i <= 5; i++ {
		set(&one, i, at(i, 8))
	}
	set(&one, 6, at(7, 8))
	set(&one, 7, at(8, 8))
	set(&one, 8, at(8, 7))
	for i := 9; i < 15; i++ {
		set(&one, i, at(8, 14-i))
	}
	for i := 0; i < 8; i++ {
		set(&two, i, at(8, size-1-i))
	}
	for i := 8; i < 15; i++ {
		set(&two, i, at(size-15+i, 8))
	}
	if one != two {
		t.Fatalf("format copies differ: %015b vs %015b", one, two)
	}
	for mask := 0; mask < 8; mask++ {
		if formatBits(mask) == one {
			return
		}
	}
	t.Fatalf("format word %015b is no level-L format", one)
}
