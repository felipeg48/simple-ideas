// Package qr is si's QR code encoder: byte mode, versions 1-40, error
// correction levels L, M, Q and H, standard library only (CLAUDE.md rule #8).
// It also knows which data codeword each module carries, so a picture laid
// over the middle of a code can be checked against what the error correction
// can recover (Cover).
//
// The layout follows ISO/IEC 18004. Positions in the placement code are
// (x, y) = (column, row), as the spec draws them; the format bits were once
// written transposed and no reader could scan the codes, so the tests read
// them back from the spec positions.
package qr

import (
	"errors"
	"fmt"
)

// Level is an error correction level: about 7, 15, 25 or 30 % of the code can
// be lost and rebuilt.
type Level int

const (
	L Level = iota
	M
	Q
	H
)

// ParseLevel reads "L", "M", "Q" or "H" (any case).
func ParseLevel(s string) (Level, error) {
	switch s {
	case "L", "l":
		return L, nil
	case "M", "m":
		return M, nil
	case "Q", "q":
		return Q, nil
	case "H", "h":
		return H, nil
	}
	return 0, fmt.Errorf("error correction %q: use L, M, Q or H", s)
}

func (l Level) String() string { return "LMQH"[l : l+1] }

// formatBits is the level's 2-bit code in the format information.
func (l Level) formatBits() int { return [...]int{1, 0, 3, 2}[l] }

// Error correction codewords per block and number of blocks, by level and
// version (index 0 unused). ISO/IEC 18004 table 9.
var eccPerBlock = [4][41]int{
	{-1, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18, 20, 24, 26, 30, 22, 24, 28, 30, 28, 28, 28, 28, 30, 30, 26, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{-1, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28},
	{-1, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24, 28, 26, 24, 20, 30, 24, 28, 28, 26, 30, 28, 30, 30, 30, 30, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{-1, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28, 24, 28, 22, 24, 24, 30, 28, 28, 26, 28, 30, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
}

var eccBlocks = [4][41]int{
	{-1, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 6, 7, 8, 8, 9, 9, 10, 12, 12, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 24, 25},
	{-1, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49},
	{-1, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8, 8, 10, 12, 16, 12, 17, 16, 18, 21, 20, 23, 23, 25, 27, 29, 34, 34, 35, 38, 40, 43, 45, 48, 51, 53, 56, 59, 62, 65, 68},
	{-1, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8, 11, 11, 16, 16, 18, 16, 19, 21, 25, 25, 25, 34, 30, 32, 35, 37, 40, 42, 45, 48, 51, 54, 57, 60, 63, 66, 70, 74, 77, 81},
}

// rawModules is the number of data + error correction modules of a version:
// the whole grid less every function pattern.
func rawModules(ver int) int {
	n := (16*ver+128)*ver + 64
	if ver >= 2 {
		align := ver/7 + 2
		n -= (25*align-10)*align - 55
		if ver >= 7 {
			n -= 36 // two version information blocks
		}
	}
	return n
}

// DataCapacity is the number of data codewords of a version at a level.
func DataCapacity(ver int, l Level) int {
	return rawModules(ver)/8 - eccPerBlock[l][ver]*eccBlocks[l][ver]
}

// countBits is the length of the byte-mode character count.
func countBits(ver int) int {
	if ver <= 9 {
		return 8
	}
	return 16
}

// MaxBytes is the most bytes a version holds at a level in byte mode.
func MaxBytes(ver int, l Level) int {
	return (DataCapacity(ver, l)*8 - 4 - countBits(ver)) / 8
}

// ErrTooLong is returned when the data does not fit a version 40 code.
var ErrTooLong = errors.New("too long for a QR code")

// Code is an encoded QR code.
type Code struct {
	Version int
	Level   Level
	Size    int // modules per side, without the quiet zone
	Mask    int
	dark    []bool
	fn      []bool // function pattern (finders, timing, alignment, format, version)
	cw      []int  // index of the codeword each data module carries, -1 for none
	block   []int  // the block each codeword of the interleaved stream belongs to
}

// Dark reports whether the module at (x, y) is dark.
func (c *Code) Dark(x, y int) bool { return c.dark[y*c.Size+x] }

// Function reports whether (x, y) is part of a function pattern.
func (c *Code) Function(x, y int) bool { return c.fn[y*c.Size+x] }

// Finder reports whether (x, y) is inside one of the three 7x7 finder squares.
func (c *Code) Finder(x, y int) bool {
	in := func(a, b int) bool { return x >= a && x < a+7 && y >= b && y < b+7 }
	return in(0, 0) || in(c.Size-7, 0) || in(0, c.Size-7)
}

// Options choose the code: the lowest error correction level (the encoder
// raises it while the version stays the same) and the smallest version.
type Options struct {
	Level      Level
	MinVersion int
	Mask       int // -1 or 0: pick the best; 1-8 forces mask n-1 (tests)
}

// Encode makes a byte-mode QR code for data.
func Encode(data []byte, o Options) (*Code, error) {
	ver := o.MinVersion
	if ver < 1 {
		ver = 1
	}
	for ; ver <= 40; ver++ {
		if len(data) <= MaxBytes(ver, o.Level) {
			break
		}
	}
	if ver > 40 {
		return nil, fmt.Errorf("%w: %d bytes, at most %d at level %s", ErrTooLong, len(data), MaxBytes(40, o.Level), o.Level)
	}
	l := o.Level
	for l < H && len(data) <= MaxBytes(ver, l+1) { // more error correction for free
		l++
	}
	c := &Code{Version: ver, Level: l, Size: ver*4 + 17}
	n := c.Size * c.Size
	c.dark, c.fn, c.cw = make([]bool, n), make([]bool, n), make([]int, n)
	c.functionPatterns()
	c.place(c.codewords(data))
	mask := o.Mask - 1
	if mask < 0 {
		best := -1
		for m := 0; m < 8; m++ {
			c.applyMask(m)
			c.drawFormat(m)
			if p := c.penalty(); best < 0 || p < best {
				best, mask = p, m
			}
			c.applyMask(m) // undo (XOR)
		}
	}
	c.Mask = mask
	c.applyMask(mask)
	c.drawFormat(mask)
	return c, nil
}

func (c *Code) set(x, y int, dark bool) {
	c.dark[y*c.Size+x] = dark
	c.fn[y*c.Size+x] = true
}

func (c *Code) functionPatterns() {
	s := c.Size
	for i := 0; i < s; i++ { // timing
		c.set(6, i, i%2 == 0)
		c.set(i, 6, i%2 == 0)
	}
	finder := func(cx, cy int) { // centre (cx, cy), with its light separator
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := cx+dx, cy+dy
				if x < 0 || x >= s || y < 0 || y >= s {
					continue
				}
				d := max(abs(dx), abs(dy))
				c.set(x, y, d != 2 && d != 4)
			}
		}
	}
	finder(3, 3)
	finder(s-4, 3)
	finder(3, s-4)
	pos := alignPositions(c.Version)
	for i, y := range pos {
		for j, x := range pos {
			if (i == 0 && j == 0) || (i == 0 && j == len(pos)-1) || (i == len(pos)-1 && j == 0) {
				continue // the finders are there
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					c.set(x+dx, y+dy, max(abs(dx), abs(dy)) != 1)
				}
			}
		}
	}
	c.drawFormat(0) // reserves the format areas; redrawn with the real mask
	if c.Version >= 7 {
		rem := c.Version
		for i := 0; i < 12; i++ {
			rem = (rem << 1) ^ ((rem >> 11) * 0x1F25)
		}
		bits := c.Version<<12 | rem
		for i := 0; i < 18; i++ {
			b := (bits>>i)&1 == 1
			a, k := s-11+i%3, i/3
			c.set(a, k, b)
			c.set(k, a, b)
		}
	}
}

// alignPositions are the centre coordinates of the alignment patterns.
func alignPositions(ver int) []int {
	if ver == 1 {
		return nil
	}
	n := ver/7 + 2
	step := 26
	if ver != 32 {
		step = (ver*4 + n*2 + 1) / (n*2 - 2) * 2
	}
	pos := make([]int, n)
	pos[0] = 6
	for i, p := n-1, ver*4+10; i >= 1; i, p = i-1, p-step {
		pos[i] = p
	}
	return pos
}

// FormatWord is the 15-bit format information for a level and mask.
func FormatWord(l Level, mask int) int {
	d := l.formatBits()<<3 | mask
	rem := d
	for i := 0; i < 10; i++ {
		rem = (rem << 1) ^ ((rem >> 9) * 0x537)
	}
	return (d<<10 | rem) ^ 0x5412
}

func (c *Code) drawFormat(mask int) {
	f := FormatWord(c.Level, mask)
	bit := func(i int) bool { return (f>>i)&1 == 1 }
	s := c.Size
	for i := 0; i <= 5; i++ {
		c.set(8, i, bit(i))
	}
	c.set(8, 7, bit(6))
	c.set(8, 8, bit(7))
	c.set(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		c.set(14-i, 8, bit(i))
	}
	for i := 0; i < 8; i++ {
		c.set(s-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		c.set(8, s-15+i, bit(i))
	}
	c.set(8, s-8, true) // the dark module
}

// codewords builds the data codewords, adds each block's error correction and
// interleaves the blocks. It also records the block of every codeword.
func (c *Code) codewords(data []byte) []byte {
	ver, l := c.Version, c.Level
	capacity := DataCapacity(ver, l)
	var bits []bool
	push := func(v, n int) {
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, (v>>i)&1 == 1)
		}
	}
	push(0b0100, 4) // byte mode
	push(len(data), countBits(ver))
	for _, b := range data {
		push(int(b), 8)
	}
	push(0, min(4, capacity*8-len(bits))) // terminator
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}
	cw := make([]byte, 0, capacity)
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i+j] {
				b |= 1 << (7 - j)
			}
		}
		cw = append(cw, b)
	}
	for i := 0; len(cw) < capacity; i++ { // pad bytes alternate, starting with 0xEC
		cw = append(cw, []byte{0xEC, 0x11}[i%2])
	}
	nb, ecc := eccBlocks[l][ver], eccPerBlock[l][ver]
	raw := rawModules(ver) / 8
	short := nb - raw%nb // blocks one data codeword shorter than the rest
	shortLen := raw / nb
	var blocks [][]byte
	for i, k := 0, 0; i < nb; i++ {
		n := shortLen - ecc
		if i >= short {
			n++
		}
		d := cw[k : k+n]
		k += n
		b := append([]byte{}, d...)
		if i < short {
			b = append(b, 0) // placeholder, so every block's error correction lines up; skipped below
		}
		blocks = append(blocks, append(b, rsEncode(d, ecc)...))
	}
	var out []byte
	c.block = c.block[:0]
	for i := 0; i < len(blocks[nb-1]); i++ {
		for j, b := range blocks {
			if i == shortLen-ecc && j < short {
				continue // a short block has no codeword here
			}
			out = append(out, b[i])
			c.block = append(c.block, j)
		}
	}
	return out
}

// place lays the codewords into the grid in the two-column zigzag.
func (c *Code) place(code []byte) {
	s := c.Size
	for i := range c.cw {
		c.cw[i] = -1
	}
	i := 0
	for right := s - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for v := 0; v < s; v++ {
			for j := 0; j < 2; j++ {
				x := right - j
				y := v
				if (right+1)&2 == 0 {
					y = s - 1 - v // upward
				}
				if c.fn[y*s+x] {
					continue
				}
				if i < len(code)*8 {
					c.dark[y*s+x] = code[i/8]>>(7-i%8)&1 == 1
					c.cw[y*s+x] = i / 8
				}
				i++
			}
		}
	}
}

func (c *Code) applyMask(m int) {
	s := c.Size
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			if c.fn[y*s+x] {
				continue
			}
			var flip bool
			switch m {
			case 0:
				flip = (x+y)%2 == 0
			case 1:
				flip = y%2 == 0
			case 2:
				flip = x%3 == 0
			case 3:
				flip = (x+y)%3 == 0
			case 4:
				flip = (x/3+y/2)%2 == 0
			case 5:
				flip = x*y%2+x*y%3 == 0
			case 6:
				flip = (x*y%2+x*y%3)%2 == 0
			default:
				flip = ((x+y)%2+x*y%3)%2 == 0
			}
			if flip {
				c.dark[y*s+x] = !c.dark[y*s+x]
			}
		}
	}
}

// penalty scores the masked code by the four rules of the spec; the mask with
// the lowest score is used.
func (c *Code) penalty() int {
	s := c.Size
	at := func(x, y int) bool { return c.dark[y*s+x] }
	p := 0
	line := func(get func(i int) bool) {
		run := 0
		var prev bool
		for i := 0; i < s; i++ {
			v := get(i)
			if i > 0 && v == prev {
				run++
			} else {
				run = 1
			}
			if run == 5 {
				p += 3
			} else if run > 5 {
				p++
			}
			prev = v
		}
		// rule 3: 1:1:3:1:1 finder-like pattern with 4 light modules on a side
		for i := 0; i+11 <= s; i++ {
			a := [11]bool{}
			for k := range a {
				a[k] = get(i + k)
			}
			core := a[4] && !a[5] && a[6] && a[7] && a[8] && !a[9] && a[10]
			if core && !a[0] && !a[1] && !a[2] && !a[3] {
				p += 40
			}
			core = a[0] && !a[1] && a[2] && a[3] && a[4] && !a[5] && a[6]
			if core && !a[7] && !a[8] && !a[9] && !a[10] {
				p += 40
			}
		}
	}
	for y := 0; y < s; y++ {
		line(func(i int) bool { return at(i, y) })
	}
	for x := 0; x < s; x++ {
		line(func(i int) bool { return at(x, i) })
	}
	dark := 0
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			if at(x, y) {
				dark++
			}
			if x < s-1 && y < s-1 && at(x, y) == at(x+1, y) && at(x, y) == at(x, y+1) && at(x, y) == at(x+1, y+1) {
				p += 3
			}
		}
	}
	p += abs(dark*20-s*s*10) / (s * s) * 10 // rule 4: 10 per 5 % away from half dark
	return p
}

// Cover reports whether the code still reads when every module in the
// rectangle [x0, x1) x [y0, y1) is hidden (by a picture). A hidden module is
// an error the reader does not know about, so each block may lose at most
// half its error correction codewords; budget scales that (1 = all of it,
// lower leaves room for a real camera's misreads). It also refuses to hide
// any function pattern but an alignment pattern. worst is the highest share
// of a block's allowance used.
func (c *Code) Cover(x0, y0, x1, y1 int, budget float64) (ok bool, worst float64) {
	hit := map[int]bool{}
	for y := max(y0, 0); y < min(y1, c.Size); y++ {
		for x := max(x0, 0); x < min(x1, c.Size); x++ {
			i := y*c.Size + x
			if c.fn[i] {
				if c.Finder(x, y) || x == 6 || y == 6 || x <= 8 && y <= 8 || x <= 8 && y >= c.Size-8 || y <= 8 && x >= c.Size-8 ||
					c.Version >= 7 && (x < 6 && y >= c.Size-11 || y < 6 && x >= c.Size-11) {
					return false, 2 // finder, timing, format or version information
				}
				continue
			}
			if c.cw[i] >= 0 {
				hit[c.cw[i]] = true
			}
		}
	}
	per := map[int]int{}
	for w := range hit {
		per[c.block[w]]++
	}
	allow := float64(eccPerBlock[c.Level][c.Version]/2) * budget
	for _, n := range per {
		if f := float64(n) / allow; f > worst {
			worst = f
		}
	}
	return worst <= 1, worst
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// GF(256) with the primitive polynomial 0x11D, for Reed-Solomon.
var gfExp [512]byte
var gfLog [256]byte

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = byte(x)
		gfLog[byte(x)] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11D
		}
	}
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[int(gfLog[a])+int(gfLog[b])]
}

// rsGen returns the generator polynomial for n error correction codewords.
func rsGen(n int) []byte {
	g := []byte{1}
	for i := 0; i < n; i++ {
		next := make([]byte, len(g)+1)
		for j, c := range g {
			next[j] ^= c
			next[j+1] ^= gfMul(c, gfExp[i])
		}
		g = next
	}
	return g
}

// rsEncode returns the n Reed-Solomon error correction codewords for data.
func rsEncode(data []byte, n int) []byte {
	gen := rsGen(n)
	res := make([]byte, len(data)+n)
	copy(res, data)
	for i := range data {
		if coef := res[i]; coef != 0 {
			for j, g := range gen {
				res[i+j] ^= gfMul(g, coef)
			}
		}
	}
	return res[len(data):]
}
