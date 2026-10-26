package main

import (
	"fmt"
	"strings"
)

// A tiny byte-mode QR encoder (versions 1-4, error-correction level L) — enough
// for a LAN URL. It exists so `si play` can show a QR to open the deck on a
// phone, with no third-party dependency (stdlib only, per the project rules).
// The Reed-Solomon / GF(256) math and the BCH format bits are computed, not
// table-guessed, so they are self-consistent and unit-testable.

// GF(256) with primitive polynomial 0x11D.
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

// rsGen returns the generator polynomial for n error-correction codewords.
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

// rsEncode returns the n Reed-Solomon EC codewords for data.
func rsEncode(data []byte, n int) []byte {
	gen := rsGen(n)
	res := make([]byte, len(data)+n)
	copy(res, data)
	for i := 0; i < len(data); i++ {
		coef := res[i]
		if coef != 0 {
			for j, g := range gen {
				res[i+j] ^= gfMul(g, coef)
			}
		}
	}
	return res[len(data):]
}

type qrVer struct{ size, dataCW, ecCW, align int }

// versions 1-4 at EC level L (single EC block each — no interleaving needed).
var qrVers = []qrVer{
	{21, 19, 7, 0},
	{25, 34, 10, 18},
	{29, 55, 15, 22},
	{33, 80, 20, 26},
}

// qrCodewords builds the full data+EC codeword stream for s.
func qrCodewords(s string) (qrVer, []byte, error) {
	data := []byte(s)
	var v qrVer
	found := false
	for _, ver := range qrVers {
		if len(data)+2 <= ver.dataCW { // 4-bit mode + 8-bit count ≈ 2 bytes of header
			v, found = ver, true
			break
		}
	}
	if !found {
		return qrVer{}, nil, fmt.Errorf("too long for a small QR (max %d bytes)", qrVers[len(qrVers)-1].dataCW-2)
	}
	var bits []bool
	push := func(val, n int) {
		for i := n - 1; i >= 0; i-- {
			bits = append(bits, (val>>i)&1 == 1)
		}
	}
	push(0b0100, 4) // byte mode
	push(len(data), 8)
	for _, b := range data {
		push(int(b), 8)
	}
	if term := v.dataCW*8 - len(bits); term > 4 { // terminator (up to 4 zero bits)
		term = 4
		push(0, term)
	} else if term > 0 {
		push(0, term)
	}
	for len(bits)%8 != 0 {
		bits = append(bits, false)
	}
	cw := make([]byte, 0, v.dataCW)
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i+j] {
				b |= 1 << (7 - j)
			}
		}
		cw = append(cw, b)
	}
	for pad := []byte{0xEC, 0x11}; len(cw) < v.dataCW; {
		cw = append(cw, pad[(len(cw)-0)%2])
	}
	return v, append(cw, rsEncode(cw, v.ecCW)...), nil
}

// formatBits returns the 15-bit format information for EC level L and a mask.
func formatBits(mask int) int {
	d := (0b01 << 3) | mask // EC level L = 0b01, then 3 mask bits
	rem := d << 10
	for i := 14; i >= 10; i-- {
		if rem&(1<<i) != 0 {
			rem ^= 0x537 << (i - 10)
		}
	}
	return ((d << 10) | rem) ^ 0x5412
}

type qrMatrix struct {
	size int
	mod  []bool // dark?
	fn   []bool // function pattern (not data, not masked)?
}

func (m *qrMatrix) set(r, c int, dark, fn bool) {
	m.mod[r*m.size+c] = dark
	m.fn[r*m.size+c] = fn
}

func maskCond(mask, r, c int) bool {
	switch mask {
	case 0:
		return (r+c)%2 == 0
	case 1:
		return r%2 == 0
	case 2:
		return c%3 == 0
	case 3:
		return (r+c)%3 == 0
	case 4:
		return (r/2+c/3)%2 == 0
	case 5:
		return (r*c)%2+(r*c)%3 == 0
	case 6:
		return ((r*c)%2+(r*c)%3)%2 == 0
	default:
		return ((r+c)%2+(r*c)%3)%2 == 0
	}
}

// qrString returns an ANSI-coloured QR code for s (two spaces per module, white
// background, a quiet zone), ready to print to a terminal. Correct polarity on
// any terminal theme because each module paints its own background colour.
func qrString(s string) (string, error) {
	v, code, err := qrCodewords(s)
	if err != nil {
		return "", err
	}
	size := v.size
	m := &qrMatrix{size: size, mod: make([]bool, size*size), fn: make([]bool, size*size)}

	finder := func(r, c int) {
		for dr := -1; dr <= 7; dr++ {
			for dc := -1; dc <= 7; dc++ {
				rr, cc := r+dr, c+dc
				if rr < 0 || rr >= size || cc < 0 || cc >= size {
					continue
				}
				dark := dr >= 0 && dr <= 6 && dc >= 0 && dc <= 6 &&
					(dr == 0 || dr == 6 || dc == 0 || dc == 6 || (dr >= 2 && dr <= 4 && dc >= 2 && dc <= 4))
				m.set(rr, cc, dark, true)
			}
		}
	}
	finder(0, 0)
	finder(0, size-7)
	finder(size-7, 0)
	// timing patterns
	for i := 8; i < size-8; i++ {
		m.set(6, i, i%2 == 0, true)
		m.set(i, 6, i%2 == 0, true)
	}
	// dark module
	m.set(size-8, 8, true, true)
	// alignment pattern (one, for v2-4)
	if v.align > 0 {
		a := v.align
		for dr := -2; dr <= 2; dr++ {
			for dc := -2; dc <= 2; dc++ {
				dark := dr == -2 || dr == 2 || dc == -2 || dc == 2 || (dr == 0 && dc == 0)
				m.set(a+dr, a+dc, dark, true)
			}
		}
	}
	// reserve format-info areas (filled after masking)
	for i := 0; i < 9; i++ {
		if !m.fn[8*size+i] {
			m.set(8, i, false, true)
		}
		if !m.fn[i*size+8] {
			m.set(i, 8, false, true)
		}
	}
	for i := 0; i < 8; i++ {
		m.set(8, size-1-i, false, true)
		m.set(size-1-i, 8, false, true)
	}

	// place data bits in the upward/downward zigzag
	bitAt := func(idx int) bool { return idx < len(code)*8 && code[idx/8]&(1<<(7-idx%8)) != 0 }
	idx := 0
	up := true
	for col := size - 1; col > 0; col -= 2 {
		if col == 6 {
			col = 5
		}
		for i := 0; i < size; i++ {
			row := i
			if up {
				row = size - 1 - i
			}
			for c := 0; c < 2; c++ {
				cc := col - c
				if !m.fn[row*size+cc] {
					m.mod[row*size+cc] = bitAt(idx)
					idx++
				}
			}
		}
		up = !up
	}

	// pick the mask with the lowest penalty
	best, bestPen := 0, 1<<31
	for mask := 0; mask < 8; mask++ {
		pen := penalty(m, mask)
		if pen < bestPen {
			best, bestPen = mask, pen
		}
	}
	// apply the chosen mask to data modules
	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if !m.fn[r*size+c] && maskCond(best, r, c) {
				m.mod[r*size+c] = !m.mod[r*size+c]
			}
		}
	}
	// write format info for the chosen mask
	f := formatBits(best)
	bit := func(i int) bool { return (f>>i)&1 == 1 }
	put := func(r, c int, b bool) { m.mod[r*size+c] = b }
	for i := 0; i <= 5; i++ {
		put(8, i, bit(i))
	}
	put(8, 7, bit(6))
	put(8, 8, bit(7))
	put(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		put(14-i, 8, bit(i))
	}
	for i := 0; i < 8; i++ {
		put(size-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		put(8, size-15+i, bit(i))
	}

	return renderQR(m), nil
}

// penalty scores a mask applied to the data modules (QR rules 1-3; rule 4 is
// minor and omitted — rules 1-3 dominate scannability for a short URL).
func penalty(m *qrMatrix, mask int) int {
	size := m.size
	at := func(r, c int) bool {
		d := m.mod[r*size+c]
		if !m.fn[r*size+c] && maskCond(mask, r, c) {
			d = !d
		}
		return d
	}
	pen := 0
	// rule 1: runs of 5+ same-colour in a row/column
	for r := 0; r < size; r++ {
		runC, runR := 1, 1
		for c := 1; c < size; c++ {
			if at(r, c) == at(r, c-1) {
				runC++
			} else {
				runC = 1
			}
			if runC == 5 {
				pen += 3
			} else if runC > 5 {
				pen++
			}
			if at(c, r) == at(c-1, r) {
				runR++
			} else {
				runR = 1
			}
			if runR == 5 {
				pen += 3
			} else if runR > 5 {
				pen++
			}
		}
	}
	// rule 2: 2x2 blocks of the same colour
	for r := 0; r < size-1; r++ {
		for c := 0; c < size-1; c++ {
			v := at(r, c)
			if at(r, c+1) == v && at(r+1, c) == v && at(r+1, c+1) == v {
				pen += 3
			}
		}
	}
	return pen
}

// renderQR draws the matrix with ANSI background colours: white quiet zone and
// light modules, black dark modules, two spaces per module.
func renderQR(m *qrMatrix) string {
	const (
		light = "\x1b[107m  \x1b[0m" // bright white bg
		dark  = "\x1b[40m  \x1b[0m"  // black bg
	)
	q := 2 // quiet zone, in modules
	var b strings.Builder
	line := func(f func(c int) bool, n int) {
		for i := 0; i < q; i++ {
			b.WriteString(light)
		}
		for c := 0; c < n; c++ {
			if f(c) {
				b.WriteString(dark)
			} else {
				b.WriteString(light)
			}
		}
		for i := 0; i < q; i++ {
			b.WriteString(light)
		}
		b.WriteByte('\n')
	}
	blank := func(c int) bool { return false }
	for i := 0; i < q; i++ {
		line(blank, m.size)
	}
	for r := 0; r < m.size; r++ {
		line(func(c int) bool { return m.mod[r*m.size+c] }, m.size)
	}
	for i := 0; i < q; i++ {
		line(blank, m.size)
	}
	return b.String()
}
