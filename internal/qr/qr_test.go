package qr

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRSGenerator(t *testing.T) {
	// published generator polynomial for 7 EC codewords, in alpha exponents
	want := []int{0, 87, 229, 146, 149, 238, 102, 21}
	g := rsGen(7)
	for i, e := range want {
		if g[i] != gfExp[e] {
			t.Errorf("rsGen(7)[%d]=%d want a^%d", i, g[i], e)
		}
	}
}

func TestFormatWords(t *testing.T) {
	// published format information, mask 0..7, per level
	want := map[Level][]int{
		L: {0x77C4, 0x72F3, 0x7DAA, 0x789D, 0x662F, 0x6318, 0x6C41, 0x6976},
		M: {0x5412, 0x5125, 0x5E7C, 0x5B4B, 0x45F9, 0x40CE, 0x4F97, 0x4AA0},
		Q: {0x355F, 0x3068, 0x3F31, 0x3A06, 0x24B4, 0x2183, 0x2EDA, 0x2BED},
		H: {0x1689, 0x13BE, 0x1CE7, 0x19D0, 0x0762, 0x0255, 0x0D0C, 0x083B},
	}
	for l, ws := range want {
		for m, w := range ws {
			if got := FormatWord(l, m); got != w {
				t.Errorf("FormatWord(%s, %d)=0x%04X want 0x%04X", l, m, got, w)
			}
		}
	}
}

// Byte capacities from the spec's capacity table.
func TestCapacities(t *testing.T) {
	for _, c := range []struct {
		ver  int
		l    Level
		want int
	}{{1, L, 17}, {1, M, 14}, {1, Q, 11}, {1, H, 7}, {7, L, 154}, {10, M, 213}, {25, Q, 715}, {40, L, 2953}, {40, M, 2331}, {40, Q, 1663}, {40, H, 1273}} {
		if got := MaxBytes(c.ver, c.l); got != c.want {
			t.Errorf("MaxBytes(%d, %s)=%d want %d", c.ver, c.l, got, c.want)
		}
	}
	if _, err := Encode(make([]byte, 2954), Options{}); err == nil {
		t.Error("2954 bytes must not fit")
	}
}

// The format and version information sit where a reader looks for them; read
// back from the spec positions, both copies agree.
func TestFormatAndVersionPlacement(t *testing.T) {
	c, err := Encode([]byte(strings.Repeat("a", 150)), Options{Level: M})
	if err != nil {
		t.Fatal(err)
	}
	if c.Version < 7 {
		t.Fatalf("want a version 7+ code, got %d", c.Version)
	}
	s := c.Size
	var one, two int
	put := func(w *int, i int, d bool) {
		if d {
			*w |= 1 << i
		}
	}
	for i := 0; i <= 5; i++ {
		put(&one, i, c.Dark(8, i))
	}
	put(&one, 6, c.Dark(8, 7))
	put(&one, 7, c.Dark(8, 8))
	put(&one, 8, c.Dark(7, 8))
	for i := 9; i < 15; i++ {
		put(&one, i, c.Dark(14-i, 8))
	}
	for i := 0; i < 8; i++ {
		put(&two, i, c.Dark(s-1-i, 8))
	}
	for i := 8; i < 15; i++ {
		put(&two, i, c.Dark(8, s-15+i))
	}
	if one != two || one != FormatWord(c.Level, c.Mask) {
		t.Errorf("format: %015b / %015b, want %015b", one, two, FormatWord(c.Level, c.Mask))
	}
	// version information: bottom-left block, bit i at (i/3, s-11+i%3)
	var ver int
	for i := 0; i < 18; i++ {
		put(&ver, i, c.Dark(i/3, s-11+i%3))
	}
	if ver>>12 != c.Version {
		t.Errorf("version information %018b does not name version %d", ver, c.Version)
	}
	known := map[int]int{7: 0x07C94, 8: 0x085BC, 9: 0x09A99, 10: 0x0A4D3}
	if w, ok := known[c.Version]; ok && ver != w {
		t.Errorf("version %d information 0x%05X want 0x%05X", c.Version, ver, w)
	}
}

// A picture over the centre fits when it is small and is refused when it is
// big; covering a finder is always refused.
func TestCover(t *testing.T) {
	c, _ := Encode([]byte("https://simple-ideas.org/rules/"), Options{Level: H})
	if _, _, _, _, ok, _ := c.Hole(0.2, 0.8); !ok {
		t.Error("a 20 % picture should fit a level-H code")
	}
	if _, _, _, _, ok, _ := c.Hole(0.6, 0.8); ok {
		t.Error("a 60 % picture must not fit")
	}
	if ok, _ := c.Cover(0, 0, 3, 3, 1); ok {
		t.Error("covering a finder must be refused")
	}
}

// goldenCases are the codes checked by a real reader (TestVision, macOS) and
// pinned in testdata/golden.txt for every other machine.
func goldenCases() []struct {
	name string
	data string
	l    Level
	min  int
} {
	var out []struct {
		name string
		data string
		l    Level
		min  int
	}
	for _, v := range []int{1, 2, 4, 5, 6, 7, 9, 10, 13, 15, 20, 26, 27, 32, 33, 40} {
		for _, l := range []Level{L, M, Q, H} {
			n := MaxBytes(v, l) // fill the version, so every block and remainder is used
			data := strings.Repeat("si: Markdown in, decks out. ", n/28+1)[:n]
			out = append(out, struct {
				name string
				data string
				l    Level
				min  int
			}{fmt.Sprintf("v%02d-%s", v, l), data, l, v})
		}
	}
	return out
}

func modules(c *Code) string {
	var b strings.Builder
	for y := 0; y < c.Size; y++ {
		for x := 0; x < c.Size; x++ {
			if c.Dark(x, y) {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String())))
}

// Every golden code still comes out module for module the same as the one a
// real reader decoded (QR_UPDATE=1 rewrites the file after TestVision passes).
func TestGolden(t *testing.T) {
	path := filepath.Join("testdata", "golden.txt")
	got := map[string]string{}
	var lines []string
	for _, g := range goldenCases() {
		c, err := Encode([]byte(g.data), Options{Level: g.l, MinVersion: g.min})
		if err != nil {
			t.Fatal(err)
		}
		if c.Version != g.min || c.Level != g.l {
			t.Fatalf("%s: got v%d-%s", g.name, c.Version, c.Level)
		}
		got[g.name] = modules(c)
		lines = append(lines, g.name+" "+got[g.name])
	}
	if os.Getenv("QR_UPDATE") != "" {
		os.MkdirAll("testdata", 0o755)
		os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		name, sum, _ := strings.Cut(sc.Text(), " ")
		if got[name] != sum {
			t.Errorf("%s: modules changed", name)
		}
		n++
	}
	if n != len(got) {
		t.Errorf("golden has %d codes, want %d", n, len(got))
	}
}

// TestVision decodes every golden code (square and dots, with a picture in
// the middle at level H) with Apple's Vision framework. macOS only, and only
// with QR_VISION=1: it is how the golden file was made trustworthy.
func TestVision(t *testing.T) {
	if os.Getenv("QR_VISION") == "" || runtime.GOOS != "darwin" {
		t.Skip("QR_VISION=1 on macOS")
	}
	dir := t.TempDir()
	want := map[string]string{}
	scale := 6 // pixels per module
	if v := os.Getenv("QR_SCALE"); v != "" {
		fmt.Sscan(v, &scale)
	}
	write := func(name, data string, c *Code, s Style) {
		f, _ := os.Create(filepath.Join(dir, name+".png"))
		png.Encode(f, c.Image(s, (c.Size+8)*scale))
		f.Close()
		want[name] = data
	}
	for _, g := range goldenCases() {
		c, _ := Encode([]byte(g.data), Options{Level: g.l, MinVersion: g.min})
		write(g.name, g.data, c, Style{})
		if g.min%5 == 0 || g.min < 3 {
			write(g.name+"-dots", g.data, c, Style{Dots: true})
		}
	}
	for _, u := range []string{"https://simple-ideas.org/", "WIFI:T:WPA;S:Office;P:correct horse;;", strings.Repeat("logo ", 40)} {
		c, _ := Encode([]byte(u), Options{Level: H})
		x0, y0, x1, y1, ok, _ := c.Hole(0.22, 0.8)
		if !ok {
			t.Fatalf("hole refused for %q", u)
		}
		write(fmt.Sprintf("logo-%d", len(u)), u, c, Style{Dots: true, Logo: &Logo{X0: x0, Y0: y0, X1: x1, Y1: y1}})
	}
	out, err := exec.Command("swift", "testdata/vision.swift", dir).CombinedOutput()
	if err != nil {
		t.Fatalf("swift: %v\n%s", err, out)
	}
	read := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name, text, _ := strings.Cut(line, "\t")
		read[name] = text
	}
	for name, w := range want {
		if read[name] != w {
			t.Errorf("%s: read %q", name, read[name])
		}
	}
}
