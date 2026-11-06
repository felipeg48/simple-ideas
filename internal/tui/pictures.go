package tui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Real pictures in terminals that can show them: the Kitty graphics protocol
// (Kitty, Ghostty, WezTerm) or iTerm2's inline images (iTerm2, WezTerm). The
// slide reserves blank cells for the picture with a private-use marker in
// its first cell; after the frame is on screen the presenter draws each
// picture there with a raw escape sequence. Everywhere else, and inside tmux
// (which does not pass them through), pictures are half-block pixels.
// SI_PICTURES=kitty|iterm|halfblock overrides the guess.

// Pictures collects the pictures a slide reserved room for.
type Pictures struct {
	Proto string // kitty or iterm
	list  []pic
}

type pic struct {
	img        image.Image
	cols, rows int
}

const markerBase = 0xE100 // marker of picture n: U+E100+n

// PictureProtocol is the picture support of this terminal, "" for half blocks.
func PictureProtocol() string {
	switch v := os.Getenv("SI_PICTURES"); v {
	case "kitty", "iterm":
		return v
	case "halfblock", "off", "none":
		return ""
	}
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" {
		return ""
	}
	term, prog := os.Getenv("TERM"), os.Getenv("TERM_PROGRAM")
	switch {
	case strings.Contains(term, "kitty") || os.Getenv("KITTY_WINDOW_ID") != "" || prog == "ghostty" || strings.Contains(term, "ghostty"):
		return "kitty"
	case prog == "iTerm.app" || os.Getenv("LC_TERMINAL") == "iTerm2" || prog == "WezTerm":
		return "iterm"
	}
	return ""
}

// reserve records img and returns the blank block it will be drawn into.
func (p *Pictures) reserve(img image.Image, cols, rows int) []string {
	n := len(p.list)
	p.list = append(p.list, pic{img, cols, rows})
	out := make([]string, rows)
	for i := range out {
		out[i] = strings.Repeat(" ", cols)
	}
	out[0] = string(rune(markerBase+n)) + strings.Repeat(" ", cols-1)
	return out
}

// placement is where a reserved picture landed on the screen (0-based).
type placement struct {
	n, row, col, cols, rows int
}

// placements finds the markers in a rendered screen, replaces them with
// spaces, and returns where each picture goes.
func (p *Pictures) placements(screen string) (string, []placement) {
	if p == nil || len(p.list) == 0 {
		return screen, nil
	}
	lines := strings.Split(screen, "\n")
	var out []placement
	for y, l := range lines {
		for n := range p.list {
			m := string(rune(markerBase + n))
			i := strings.Index(l, m)
			if i < 0 {
				continue
			}
			col := ansi.StringWidth(l[:i])
			l = l[:i] + " " + l[i+len(m):]
			out = append(out, placement{n, y, col, p.list[n].cols, p.list[n].rows})
		}
		lines[y] = l
	}
	return strings.Join(lines, "\n"), out
}

// draw is the escape sequence that puts the placed pictures on screen.
func (p *Pictures) draw(places []placement) string {
	var b strings.Builder
	if p.Proto == "kitty" {
		b.WriteString("\x1b_Ga=d,d=A,q=2\x1b\\") // remove the last slide's pictures
	}
	for _, pl := range places {
		if pl.n >= len(p.list) || pl.cols <= 0 || pl.rows <= 0 {
			continue
		}
		data := encodePNG(p.list[pl.n].img, pl.cols*10, pl.rows*20)
		fmt.Fprintf(&b, "\x1b7\x1b[%d;%dH", pl.row+1, pl.col+1) // save the cursor, go to the block
		switch p.Proto {
		case "kitty":
			b.WriteString(kittyImage(data, pl.cols, pl.rows))
		case "iterm":
			fmt.Fprintf(&b, "\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=1:%s\x07",
				len(data), pl.cols, pl.rows, base64.StdEncoding.EncodeToString(data))
		}
		b.WriteString("\x1b8")
	}
	return b.String()
}

// kittyImage sends a PNG in 4096-byte base64 chunks, sized to cols × rows
// cells, without moving the cursor (C=1) and without replies (q=2).
func kittyImage(png []byte, cols, rows int) string {
	enc := base64.StdEncoding.EncodeToString(png)
	var b strings.Builder
	for i := 0; i < len(enc); i += 4096 {
		chunk := enc[i:min(i+4096, len(enc))]
		more := 0
		if i+4096 < len(enc) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(&b, "\x1b_Ga=T,f=100,c=%d,r=%d,C=1,q=2,m=%d;%s\x1b\\", cols, rows, more, chunk)
		} else {
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
	}
	return b.String()
}

// encodePNG scales img down to fit w × h pixels (never up) and encodes it.
func encodePNG(img image.Image, w, h int) []byte {
	bd := img.Bounds()
	if bd.Dx() > w || bd.Dy() > h {
		s := min(float64(w)/float64(bd.Dx()), float64(h)/float64(bd.Dy()))
		nw, nh := max(1, int(float64(bd.Dx())*s)), max(1, int(float64(bd.Dy())*s))
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		for y := 0; y < nh; y++ {
			for x := 0; x < nw; x++ {
				dst.Set(x, y, img.At(bd.Min.X+x*bd.Dx()/nw, bd.Min.Y+y*bd.Dy()/nh))
			}
		}
		img = dst
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
