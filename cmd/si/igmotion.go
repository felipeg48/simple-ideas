package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// Animated infographics as GIF and MP4 (si build x.md --gif --mp4): the built
// page opens in headless Chrome with ?si-ig-film, where infographic.js does
// not play but waits to be told the time (window.siIG.seek); every frame is
// the frame of the infographic at k/fps seconds, cut to the frame's box. The
// clock is the page's own, so the frames are exact and the same on every run.
// --size phone,square,... renders the file again at each size.

// igFilmCSS is the CSS size a frame of ratio w:h is filmed at: a portrait one
// 540 wide (a phone's layout), a square 900, a landscape slide 1280.
func igFilmCSS(w, h float64) (int, int) {
	cw := 1280
	switch {
	case w < h:
		cw = 540
	case w == h:
		cw = 900
	}
	return cw, int(math.Round(float64(cw) * h / w))
}

// igVideoWidth is the full width of an MP4: an exact size's own, else 1080
// (portrait, square) or 1920 (landscape).
func igVideoWidth(aspect string, w, h float64) int {
	if m := regexp.MustCompile(`^(\d+)x\d+$`).FindStringSubmatch(aspect); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	if w > h {
		return 1920
	}
	return 1080
}

const igGIFWidth = 720

// igVariant is one size of an infographic: its deck, page and output name.
type igVariant struct {
	d    *deck.Deck
	html string
	name string
	tmp  bool
}

// exportInfographicAll writes the pictures, PDF and animations of a built
// infographic, at its own size or at each of --size.
func exportInfographicAll(chrome string, d *deck.Deck, ro render.Options, dest, out string, o *options, stdout, stderr io.Writer) (pdfRel string, err error) {
	vs := []igVariant{{d: d, html: dest, name: d.Name}}
	if o.sizeSet {
		vs = nil
		for _, s := range strings.Split(o.size, ",") {
			s = strings.TrimSpace(s)
			if _, _, ok := deck.IGSize(s); !ok {
				return "", fmt.Errorf("--size %q: use phone, story, portrait, square, landscape or an exact size (1080x1350)", s)
			}
			v := *d
			v.Meta.Aspect, v.Meta.Orientation = s, ""
			v.Name = d.Name + "-" + s
			p := filepath.Join(out, "."+v.Name+".html") // beside the page: its links still work
			if err := os.WriteFile(p, render.Deck(&v, ro).HTML, 0o644); err != nil {
				return "", err
			}
			vs = append(vs, igVariant{d: &v, html: p, name: v.Name, tmp: true})
		}
	}
	for _, v := range vs {
		if v.tmp {
			defer os.Remove(v.html)
		}
		if o.png || o.pdf {
			rel, err := exportInfographic(chrome, v.html, out, v.d, o.png, o.pdf, o.scale, stdout)
			if err != nil {
				return "", err
			}
			if pdfRel == "" {
				pdfRel = rel
			}
		}
		if o.gif || o.mp4 {
			if err := filmInfographic(chrome, v, out, o, stdout); err != nil {
				return pdfRel, err
			}
		}
	}
	return pdfRel, nil
}

// igRatioOf is the width:height of an infographic's frame.
func igRatioOf(d *deck.Deck) (float64, float64) {
	if w, h, ok := render.IGRatio(d.Meta.Aspect); ok {
		return w, h
	}
	if d.Meta.IGPortrait() {
		return 9, 16
	}
	return 16, 9
}

type igPage struct {
	D, X, Y, W, H float64
	Loop          bool
}

// filmInfographic writes <name>.gif and/or <name>.mp4 of one size.
func filmInfographic(chrome string, v igVariant, out string, o *options, stdout io.Writer) error {
	rw, rh := igRatioOf(v.d)
	cw, ch := igFilmCSS(rw, rh)
	c, err := cdp.Launch(chrome, cw, ch)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err := c.OpenPage("", cw, ch); err != nil {
		return err
	}
	abs, _ := filepath.Abs(v.html)
	url := "file://" + filepath.ToSlash(abs) + "?si-ig-film"
	if !v.d.IsDoc() {
		url += "#1.999" // the slide, every build step shown
	}
	if err := c.Call("Page.navigate", map[string]any{"url": url}, nil); err != nil {
		return err
	}
	for i := 0; i < 200; i++ {
		if r, _ := c.Eval("document.readyState"); string(r) == `"complete"` {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := c.Eval(`document.fonts ? document.fonts.ready.then(function(){return 1}) : 1`); err != nil {
		return err
	}
	if _, err := c.Eval(hideChrome); err != nil {
		return err
	}
	const measure = `(function(){var f=document.querySelector('.si-ig-frame');if(!f)return null;window.siInfographicFit&&siInfographicFit();var r=f.getBoundingClientRect();
return {D:window.siIG?siIG.duration:0,Loop:!!(window.siIG&&siIG.loop),X:r.left+scrollX,Y:r.top+scrollY,W:r.width,H:r.height}})()`
	var pg igPage
	raw, err := c.Eval(measure)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &pg); err != nil || pg.W == 0 {
		return fmt.Errorf("the page has no infographic frame")
	}
	if pg.D <= 0 {
		return fmt.Errorf("%s is not animated: add animate: build (or loop) to its front matter for --gif and --mp4", v.d.Name)
	}
	if need := int(math.Ceil(pg.Y + pg.H)); need > ch { // a poster taller than its aspect: film all of it
		if err := c.Call("Emulation.setDeviceMetricsOverride", map[string]any{"width": cw, "height": need, "deviceScaleFactor": 1, "mobile": false}, nil); err != nil {
			return err
		}
		time.Sleep(300 * time.Millisecond)
		if raw, err = c.Eval(measure); err == nil {
			json.Unmarshal(raw, &pg)
		}
	}
	seek := func(t float64) error { _, err := c.Eval(fmt.Sprintf("siIG.seek(%g)", t)); return err }
	shot := func(scale float64) ([]byte, error) {
		var r struct{ Data string }
		clip := map[string]any{"x": pg.X, "y": pg.Y, "width": pg.W, "height": pg.H, "scale": scale}
		if err := c.Call("Page.captureScreenshot", map[string]any{"format": "png", "clip": clip, "captureBeyondViewport": true}, &r); err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(r.Data)
	}
	scaleFor := func(width int) float64 {
		if o.scaleSet {
			return o.scale
		}
		return float64(width) / pg.W
	}
	fps := 15
	if o.fps > 0 {
		fps = o.fps
	}
	if o.gif {
		// a GIF has 256 colours: the frame's soft glows would band, so it is flat
		if _, err := c.Eval(`document.documentElement.classList.add('si-ig-flat')`); err != nil {
			return err
		}
		scale := scaleFor(igGIFWidth)
		n := max(1, int(math.Ceil(pg.D*float64(fps)-1e-9)))
		var frames []image.Image
		var delays []int
		for k := 0; k < n; k++ {
			if err := seek(float64(k) / float64(fps)); err != nil {
				return err
			}
			b, err := shot(scale)
			if err != nil {
				return err
			}
			img, err := png.Decode(bytes.NewReader(b))
			if err != nil {
				return err
			}
			frames = append(frames, img)
			delays = append(delays, gifDelay(k, fps))
		}
		if !pg.Loop { // built once: the last frame stays
			frames = append(frames, frames[len(frames)-1])
			delays = append(delays, 300)
			if err := seek(pg.D); err == nil {
				if b, err := shot(scale); err == nil {
					if img, err := png.Decode(bytes.NewReader(b)); err == nil {
						frames[len(frames)-1] = img
					}
				}
			}
		}
		data, err := encodeAnimGIF(frames, delays, pg.Loop)
		if err != nil {
			return err
		}
		p := filepath.Join(out, v.name+".gif")
		if err := os.WriteFile(p, data, 0o644); err != nil {
			return err
		}
		b := frames[0].Bounds()
		fmt.Fprintf(stdout, "  %-28s (gif, %d × %d, %d fps, %gs, %s, %s)\n", v.name+".gif", b.Dx(), b.Dy(), fps, pg.D, map[bool]string{true: "loops", false: "plays once"}[pg.Loop], igMB(len(data)))
	}
	if o.mp4 {
		if _, err := c.Eval(`document.documentElement.classList.remove('si-ig-flat')`); err != nil {
			return err
		}
		tmp, err := os.MkdirTemp("", "si-igfilm-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		scale := scaleFor(igVideoWidth(v.d.Meta.Aspect, rw, rh))
		n := max(1, int(math.Ceil(pg.D*videoFPS-1e-9)))
		var video []videoFrame
		for k := 0; k <= n; k++ {
			t := math.Min(float64(k)/videoFPS, pg.D)
			if pg.Loop && k == n {
				break
			}
			if err := seek(t); err != nil {
				return err
			}
			b, err := shot(scale)
			if err != nil {
				return err
			}
			p := filepath.Join(tmp, fmt.Sprintf("f%05d.png", k))
			if err := os.WriteFile(p, b, 0o644); err != nil {
				return err
			}
			video = append(video, videoFrame{PNG: p, Dur: 1.0 / videoFPS})
		}
		if !pg.Loop {
			video[len(video)-1].Dur = 2 // the finished infographic, held
		}
		p := filepath.Join(out, v.name+".mp4")
		if err := encodeMP4(video, p); err != nil {
			return err
		}
		st, _ := os.Stat(p)
		fmt.Fprintf(stdout, "  %-28s (mp4, %d px wide, %gs, %s)\n", v.name+".mp4", int(math.Round(pg.W*scale))/2*2, pg.D, igMB(int(st.Size())))
	}
	return nil
}

func igMB(n int) string {
	if n < 1<<20 {
		return fmt.Sprintf("%d KB", n/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
}

// gifDelay is frame k's delay in hundredths of a second at fps, rounded so
// the delays add up to the real time.
func gifDelay(k, fps int) int {
	return int(math.Round(float64(k+1)*100/float64(fps)) - math.Round(float64(k)*100/float64(fps)))
}

// encodeAnimGIF encodes frames as an animated GIF, small: one adaptive
// palette for all (32 greys and the 223 most used other colours, 5 bits a channel, no
// dithering, with a ramp of greys: flat cards and fades stay clean), and every frame after
// the first only as the box of pixels that changed, the unchanged ones in it
// transparent; a frame identical to the one before lengthens it. loop: play
// forever, else once (the last frame stays).
// ponytail: a popularity palette, not median cut; good for themed vector
// frames, upgrade if photos ever fill an animated infographic.
func encodeAnimGIF(frames []image.Image, delays []int, loop bool) ([]byte, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames")
	}
	b := frames[0].Bounds()
	w, h := b.Dx(), b.Dy()
	var count [1 << 15]int
	keyed := make([][]uint16, len(frames)) // each pixel's 5-bit colour
	for i, f := range frames {
		keyed[i] = rgb15(f, w, h)
		for j := 0; j < len(keyed[i]); j += 3 {
			count[keyed[i][j]]++
		}
	}
	var keys []int
	for k, n := range count {
		if n > 0 {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return count[keys[i]] > count[keys[j]] })
	pal := make(color.Palette, 0, 256)
	for v := 0; v < 32; v++ { // a ramp of greys, so fades between neutrals never take a hue
		g := uint8(v<<3 | v>>2)
		pal = append(pal, color.RGBA{g, g, g, 255})
		count[v<<10|v<<5|v] = 0
	}
	n := 0
	for _, k := range keys {
		if n == 255-32 {
			break
		}
		if count[k] == 0 {
			continue
		}
		n++
		r, g, bl := uint8(k>>10&31), uint8(k>>5&31), uint8(k&31)
		pal = append(pal, color.RGBA{r<<3 | r>>2, g<<3 | g>>2, bl<<3 | bl>>2, 255})
	}
	clear := uint8(len(pal))
	pal = append(pal, color.RGBA{}) // transparent: the pixels a frame leaves as they were
	var lut [1 << 15]uint8
	for k := range lut {
		r, g, bl := int(k>>10&31)<<3, int(k>>5&31)<<3, int(k&31)<<3
		best, bd := 0, math.MaxInt
		for i, c := range pal[:clear] {
			cc := c.(color.RGBA)
			dr, dg, db := r-int(cc.R), g-int(cc.G), bl-int(cc.B)
			if d := 3*dr*dr + 4*dg*dg + 2*db*db; d < bd {
				best, bd = i, d
			}
		}
		lut[k] = uint8(best)
	}
	index := func(i int) []uint8 {
		out := make([]uint8, w*h)
		for j, k := range keyed[i] {
			out[j] = lut[k]
		}
		keyed[i] = nil
		return out
	}
	g := &gif.GIF{LoopCount: -1, Config: image.Config{ColorModel: pal, Width: w, Height: h}}
	if loop {
		g.LoopCount = 0
	}
	var prev []uint8
	for i := range frames {
		cur := index(i)
		if prev == nil {
			p := image.NewPaletted(image.Rect(0, 0, w, h), pal)
			copy(p.Pix, cur)
			g.Image, g.Delay, g.Disposal = append(g.Image, p), append(g.Delay, delays[i]), append(g.Disposal, gif.DisposalNone)
			prev = cur
			continue
		}
		x0, y0, x1, y1 := w, h, -1, -1
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if cur[y*w+x] != prev[y*w+x] {
					x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
				}
			}
		}
		if x1 < 0 { // nothing moved: the frame before stays longer
			g.Delay[len(g.Delay)-1] += delays[i]
			continue
		}
		p := image.NewPaletted(image.Rect(x0, y0, x1+1, y1+1), pal)
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				v := cur[y*w+x]
				if v == prev[y*w+x] {
					v = clear
				}
				p.Pix[(y-y0)*p.Stride+(x-x0)] = v
			}
		}
		g.Image, g.Delay, g.Disposal = append(g.Image, p), append(g.Delay, delays[i]), append(g.Disposal, gif.DisposalNone)
		prev = cur
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// rgb15 is every pixel of f (w×h from its corner) as 5 bits a channel.
func rgb15(f image.Image, w, h int) []uint16 {
	out := make([]uint16, w*h)
	fb := f.Bounds()
	var pix []uint8
	stride := 0
	switch m := f.(type) { // a screenshot is opaque: NRGBA and RGBA read the same
	case *image.NRGBA:
		pix, stride = m.Pix, m.Stride
	case *image.RGBA:
		pix, stride = m.Pix, m.Stride
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var r, g, b uint32
			if pix != nil {
				o := y*stride + x*4
				r, g, b = uint32(pix[o]), uint32(pix[o+1]), uint32(pix[o+2])
			} else {
				r, g, b, _ = f.At(fb.Min.X+x, fb.Min.Y+y).RGBA()
				r, g, b = r>>8, g>>8, b>>8
			}
			out[y*w+x] = uint16(r>>3)<<10 | uint16(g>>3)<<5 | uint16(b>>3)
		}
	}
	return out
}
