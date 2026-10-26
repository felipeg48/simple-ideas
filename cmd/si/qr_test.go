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
