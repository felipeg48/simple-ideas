package webdemo

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
)

// The take's edits, applied when its frames are laid out at 30 fps
// (sequence): transitions at the scene cuts the take recorded (the old
// scene's last frame held, then blended into the new one) and camera moves
// (a crop rectangle per frame, eased, scaled back to the frame). Both are
// drawn in Go on the decoded JPEG planes; the length of the video never
// changes, so the voice, captions, badges and chapters keep their times.

// cutAt is a scene change in a take: the old scene's last moment and the
// new one's first (take seconds; a page load lies between them), the
// transition, and the cursor (CSS pixels; X < 0 = none) for circle.
type cutAt struct {
	From, To float64
	Tr       Transition
	X, Y     float64
}

// move is a camera move in a take: zoom (to a box), pan (to a box, at the
// current size) or out (back to the whole picture).
type move struct {
	At, Dur float64 // take seconds
	Kind    string  // zoom, pan, out
	B       box     // CSS pixels (DPR ignored: frames are scaled by frame/viewport)
}

// rectF is a rectangle in frame pixels.
type rectF struct{ X, Y, W, H float64 }

// camKey is a resolved camera move on the video's clock.
type camKey struct {
	Start, Dur float64
	From, To   rectF
}

// effects are a take's cuts and moves, with what they need to be drawn.
type effects struct {
	cuts  []cutAt
	moves []move
	view  rect    // what the camera frames when it is out: the region, or the whole frame
	scale float64 // frame pixels per CSS pixel
}

func (fx *effects) none() bool {
	if fx == nil || len(fx.moves) > 0 {
		return fx == nil
	}
	for _, c := range fx.cuts {
		if c.Tr.Kind != "cut" {
			return false
		}
	}
	return true
}

// ease is smoothstep: a move starts and lands gently.
func ease(p float64) float64 {
	p = clampf(p, 0, 1)
	return p * p * (3 - 2*p)
}

func lerpRect(a, b rectF, e float64) rectF {
	return rectF{a.X + (b.X-a.X)*e, a.Y + (b.Y-a.Y)*e, a.W + (b.W-a.W)*e, a.H + (b.H-a.H)*e}
}

// maxZoom is how far the camera goes in: a third of the picture's width.
const maxZoom = 3.0

// fitRect places a w×h rectangle centered on cx, cy, kept inside view.
func fitRect(cx, cy, w, h float64, view rectF) rectF {
	w, h = min(w, view.W), min(h, view.H)
	x := clampf(cx-w/2, view.X, view.X+view.W-w)
	y := clampf(cy-h/2, view.Y, view.Y+view.H-h)
	return rectF{x, y, w, h}
}

// zoomRect is the camera's rectangle for a box (frame pixels): the box with
// a margin, grown to the view's aspect, no closer than maxZoom, inside the
// view.
func zoomRect(b rectF, view rectF, margin float64) rectF {
	w, h := b.W+2*margin, b.H+2*margin
	a := view.W / view.H
	if w/h < a {
		w = h * a
	} else {
		h = w / a
	}
	if w < view.W/maxZoom {
		w, h = view.W/maxZoom, view.H/maxZoom
	}
	if w > view.W {
		w, h = view.W, view.H
	}
	return fitRect(b.X+b.W/2, b.Y+b.H/2, w, h, view)
}

// camAt is the camera's rectangle at video time o.
func camAt(keys []camKey, o float64, view rectF) rectF {
	r := view
	for _, k := range keys {
		if o < k.Start {
			break
		}
		if k.Dur <= 0 || o >= k.Start+k.Dur {
			r = k.To
		} else {
			r = lerpRect(k.From, k.To, ease((o-k.Start)/k.Dur))
		}
	}
	return r
}

// resolveMoves puts a take's moves on the video's clock (fs: the
// fast-forwarded waits) as rectangles in frame pixels; a move keeps its
// length, wherever it falls.
func resolveMoves(moves []move, fs []fast, view rectF, scale float64) []camKey {
	var keys []camKey
	last := view // where the camera is heading: pan keeps its size
	for _, m := range moves {
		start := outT(fs, m.At)
		from := camAt(keys, start, view)
		b := rectF{m.B.X * scale, m.B.Y * scale, m.B.W * scale, m.B.H * scale}
		to := view
		switch m.Kind {
		case "zoom":
			to = zoomRect(b, view, max(24*scale, 0.15*max(b.W, b.H)))
		case "pan":
			to = fitRect(b.X+b.W/2, b.Y+b.H/2, last.W, last.H, view)
		}
		keys = append(keys, camKey{start, m.Dur, from, to})
		last = to
	}
	return keys
}

// slideShift is how far the old (a) and the new (b) pictures have moved at
// eased progress e of a slide: the new one comes in from the side the name
// points away from (slide-left: from the right, moving left).
func slideShift(kind string, e, w, h float64) (adx, ady, bdx, bdy float64) {
	switch kind {
	case "slide-left":
		return -w * e, 0, w * (1 - e), 0
	case "slide-right":
		return w * e, 0, -w * (1 - e), 0
	case "slide-up":
		return 0, -h * e, 0, h * (1 - e)
	case "slide-down":
		return 0, h * e, 0, -h * (1 - e)
	}
	return 0, 0, 0, 0
}

// circleRadius is the open circle's radius at eased progress e: from 0 to
// the farthest corner from cx, cy.
func circleRadius(e, cx, cy, w, h float64) float64 {
	far := 0.0
	for _, c := range [][2]float64{{0, 0}, {w, 0}, {0, h}, {w, h}} {
		far = max(far, math.Hypot(c[0]-cx, c[1]-cy))
	}
	return e * far
}

// xform places a picture: scaled by S about the frame's centre, then
// shifted by DX, DY.
type xform struct{ S, DX, DY float64 }

var still = xform{S: 1}

// src is where output pixel x, y samples the picture.
func (t xform) src(x, y, w, h float64) (float64, float64) {
	return (x-t.DX-w/2)/t.S + w/2, (y-t.DY-h/2)/t.S + h/2
}

// blend is transition kind at progress p (0…1) on a w×h frame: how each
// picture is placed and how much of the new one shows at x, y (0…1).
func blend(kind string, p, w, h, cx, cy float64) (a, b xform, mix func(x, y float64) float64) {
	e := ease(p)
	a, b = still, still
	inB := func(x, y float64) bool {
		sx, sy := b.src(x, y, w, h)
		return sx >= 0 && sy >= 0 && sx < w && sy < h
	}
	edge := func(d float64) float64 { return clampf(d/2+.5, 0, 1) } // a 2-pixel soft edge
	switch kind {
	case "slide-left", "slide-right", "slide-up", "slide-down":
		a.DX, a.DY, b.DX, b.DY = slideShift(kind, e, w, h)
		return a, b, func(x, y float64) float64 {
			if inB(x, y) {
				return 1
			}
			return 0
		}
	case "wipe-left":
		return a, b, func(x, y float64) float64 { return edge(x - w*(1-e)) }
	case "wipe-right":
		return a, b, func(x, y float64) float64 { return edge(w*e - x) }
	case "wipe-up":
		return a, b, func(x, y float64) float64 { return edge(y - h*(1-e)) }
	case "wipe-down":
		return a, b, func(x, y float64) float64 { return edge(h*e - y) }
	case "pop":
		b.S = 0.92 + 0.08*e
	case "zoom":
		a.S, b.S = 1+0.3*e, 0.85+0.15*e
	case "circle":
		r := circleRadius(e, cx, cy, w, h)
		return a, b, func(x, y float64) float64 { return edge(r - math.Hypot(x-cx, y-cy)) }
	}
	return a, b, func(x, y float64) float64 {
		if !inB(x, y) {
			return 0
		}
		return e
	}
}

// plane is one of a YCbCr picture's planes; fx, fy are luma pixels per
// plane pixel.
type plane struct {
	pix          []byte
	stride, w, h int
	fx, fy       float64
}

func planesOf(m *image.YCbCr) [3]plane {
	w, h := m.Rect.Dx(), m.Rect.Dy()
	cw, ch := w, h
	switch m.SubsampleRatio {
	case image.YCbCrSubsampleRatio420:
		cw, ch = (w+1)/2, (h+1)/2
	case image.YCbCrSubsampleRatio422:
		cw = (w + 1) / 2
	case image.YCbCrSubsampleRatio440:
		ch = (h + 1) / 2
	}
	fx, fy := float64(w)/float64(cw), float64(h)/float64(ch)
	return [3]plane{{m.Y, m.YStride, w, h, 1, 1}, {m.Cb, m.CStride, cw, ch, fx, fy}, {m.Cr, m.CStride, cw, ch, fx, fy}}
}

// at samples the plane at luma coordinates x, y (bilinear, clamped).
func (p plane) at(x, y float64) float64 {
	x, y = clampf(x/p.fx, 0, float64(p.w-1)), clampf(y/p.fy, 0, float64(p.h-1))
	x0, y0 := int(x), int(y)
	x1, y1 := min(x0+1, p.w-1), min(y0+1, p.h-1)
	tx, ty := x-float64(x0), y-float64(y0)
	r0, r1 := p.pix[y0*p.stride:], p.pix[y1*p.stride:]
	top := float64(r0[x0])*(1-tx) + float64(r0[x1])*tx
	bot := float64(r1[x0])*(1-tx) + float64(r1[x1])*tx
	return top*(1-ty) + bot*ty
}

// asYCbCr gives a decoded JPEG as YCbCr (a grey or CMYK one is converted).
func asYCbCr(m image.Image) *image.YCbCr {
	if y, ok := m.(*image.YCbCr); ok && (y.SubsampleRatio == image.YCbCrSubsampleRatio420 || y.SubsampleRatio == image.YCbCrSubsampleRatio422 ||
		y.SubsampleRatio == image.YCbCrSubsampleRatio444 || y.SubsampleRatio == image.YCbCrSubsampleRatio440) {
		return y
	}
	b := m.Bounds()
	out := image.NewYCbCr(image.Rect(0, 0, b.Dx(), b.Dy()), image.YCbCrSubsampleRatio444)
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			c := color.YCbCrModel.Convert(m.At(b.Min.X+x, b.Min.Y+y)).(color.YCbCr)
			i := y*out.YStride + x
			out.Y[i], out.Cb[i], out.Cr[i] = c.Y, c.Cb, c.Cr
		}
	}
	return out
}

func newLike(m *image.YCbCr) *image.YCbCr {
	return image.NewYCbCr(image.Rect(0, 0, m.Rect.Dx(), m.Rect.Dy()), m.SubsampleRatio)
}

func byteOf(v float64) byte { return byte(clampf(v+.5, 0, 255)) }

// camera shows the c rectangle of src's view (frame pixels) over the whole
// view; outside the view (cut away by the region crop) the frame is kept.
func camera(src *image.YCbCr, view rect, c rectF) *image.YCbCr {
	if c == (rectF{float64(view.X), float64(view.Y), float64(view.W), float64(view.H)}) {
		return src
	}
	dst := newLike(src)
	k := c.W / float64(view.W)
	sp, dp := planesOf(src), planesOf(dst)
	for n := range 3 {
		s, d := sp[n], dp[n]
		for j := 0; j < d.h; j++ {
			Y := float64(j) * d.fy
			in := Y >= float64(view.Y) && Y < float64(view.Y+view.H)
			sy := c.Y + (Y-float64(view.Y))*k
			row := d.pix[j*d.stride:]
			for i := 0; i < d.w; i++ {
				X := float64(i) * d.fx
				if in && X >= float64(view.X) && X < float64(view.X+view.W) {
					row[i] = byteOf(s.at(c.X+(X-float64(view.X))*k, sy))
				} else {
					row[i] = s.pix[j*s.stride+i]
				}
			}
		}
	}
	return dst
}

// compose draws transition kind at progress p from a (old) to b (new);
// cx, cy: the circle's centre (frame pixels).
func compose(a, b *image.YCbCr, kind string, p, cx, cy float64) *image.YCbCr {
	w, h := float64(a.Rect.Dx()), float64(a.Rect.Dy())
	ta, tb, mix := blend(kind, p, w, h, cx, cy)
	dst := newLike(a)
	ap, bp, dp := planesOf(a), planesOf(b), planesOf(dst)
	for n := range 3 {
		pa, pb, d := ap[n], bp[n], dp[n]
		for j := 0; j < d.h; j++ {
			Y := float64(j) * d.fy
			row := d.pix[j*d.stride:]
			for i := 0; i < d.w; i++ {
				X := float64(i) * d.fx
				m := mix(X, Y)
				v := 0.0
				if m < 1 {
					sx, sy := ta.src(X, Y, w, h)
					v += pa.at(sx, sy) * (1 - m)
				}
				if m > 0 {
					sx, sy := tb.src(X, Y, w, h)
					v += pb.at(sx, sy) * m
				}
				row[i] = byteOf(v)
			}
		}
	}
	return dst
}

// lastAt is the index of the last frame that came at or before t (-1: none).
func lastAt(frames []frame, t float64) int {
	return sort.Search(len(frames), func(i int) bool { return frames[i].At > t }) - 1
}

// shot is what output frame k shows: frame A under camera CA, and, during a
// transition, frame B under CB mixed in at progress P (B < 0: none).
type shot struct {
	A, B   int
	CA, CB rectF
	P      float64
	Kind   string
	CX, CY float64
}

// plan decides every output frame of a d-second take: which frames, which
// camera, which transition.
func plan(frames []frame, d float64, fs []fast, fx *effects, w, h int) []shot {
	n := max(1, int(outT(fs, d)*fps+.5))
	view := rectF{float64(fx.view.X), float64(fx.view.Y), float64(fx.view.W), float64(fx.view.H)}
	keys := resolveMoves(fx.moves, fs, view, fx.scale)
	cuts := append([]cutAt(nil), fx.cuts...)
	sort.Slice(cuts, func(i, j int) bool { return cuts[i].From < cuts[j].From })
	out := make([]shot, n)
	j := 0
	for k := range out {
		o := float64(k) / fps
		t := inT(fs, o)
		for j+1 < len(frames) && frames[j+1].At <= t {
			j++
		}
		s := shot{A: j, B: -1, CA: camAt(keys, o, view)}
		for _, c := range cuts {
			ia := lastAt(frames, c.From)
			if c.Tr.Kind == "cut" || ia < 0 {
				continue
			}
			of, ot := outT(fs, c.From), outT(fs, c.To)
			if o >= of && o < ot { // the page loads: the old scene held
				s.A, s.CA = ia, camAt(keys, of, view)
				break
			}
			if o >= ot && o < ot+c.Tr.Dur {
				ib := j
				if frames[ib].At < c.To { // the new scene's first frame may come late
					if f := lastAt(frames, c.To-1e-9) + 1; f < len(frames) {
						ib = f
					}
				}
				cx, cy := float64(w)/2, float64(h)/2
				if c.X >= 0 {
					cx, cy = c.X*fx.scale, c.Y*fx.scale
				}
				s = shot{A: ia, B: ib, CA: camAt(keys, of, view), CB: s.CA, P: (o - ot) / c.Tr.Dur, Kind: c.Tr.Kind, CX: cx, CY: cy}
				break
			}
		}
		out[k] = s
	}
	return out
}

func decodeJPEG(file string) (*image.YCbCr, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	m, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	return asYCbCr(m), nil
}

// drawShot draws one shot into dst.
func drawShot(frames []frame, s shot, view rect, dst string) error {
	a, err := decodeJPEG(frames[s.A].File)
	if err != nil {
		return err
	}
	out := camera(a, view, s.CA)
	if s.B >= 0 {
		b, err := decodeJPEG(frames[s.B].File)
		if err != nil {
			return err
		}
		if b.Rect.Dx() != a.Rect.Dx() || b.Rect.Dy() != a.Rect.Dy() { // a scene of another size (a terminal tab): stretched, as ffmpeg would
			b = resize(b, a.Rect.Dx(), a.Rect.Dy())
		}
		out = compose(out, camera(b, view, s.CB), s.Kind, s.P, s.CX, s.CY)
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: 92}); err != nil {
		return err
	}
	return os.WriteFile(dst, buf.Bytes(), 0o644)
}

func link(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, 0o644)
}

// drawShots writes the sequence's frames: a plain shot links its frame, a
// shot the same as the one before links that one's file, the rest are drawn
// on every core.
func drawShots(frames []frame, shots []shot, view rect, dir string) error {
	full := rectF{float64(view.X), float64(view.Y), float64(view.W), float64(view.H)}
	name := func(k int) string { return filepath.Join(dir, fmt.Sprintf("seq%05d.jpg", k)) }
	var jobs []int
	links := map[int]int{} // output frame -> the drawn frame it repeats
	for k, s := range shots {
		switch {
		case s.B < 0 && s.CA == full:
			if err := link(frames[s.A].File, name(k)); err != nil {
				return err
			}
		case k > 0 && s.B < 0 && shots[k-1].B < 0 && shots[k-1] == s && !(s.CA == full):
			if p, ok := links[k-1]; ok {
				links[k] = p
			} else {
				links[k] = k - 1
			}
		default:
			jobs = append(jobs, k)
		}
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	ch := make(chan int)
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := range ch {
				if err := drawShot(frames, shots[k], view, name(k)); err != nil {
					mu.Lock()
					first = cmpErr(first, err)
					mu.Unlock()
				}
			}
		}()
	}
	for _, k := range jobs {
		ch <- k
	}
	close(ch)
	wg.Wait()
	if first != nil {
		return first
	}
	for k, p := range links {
		if err := link(name(p), name(k)); err != nil {
			return err
		}
	}
	return nil
}

func cmpErr(a, b error) error {
	if a != nil {
		return a
	}
	return b
}

// resize stretches m to w×h.
func resize(m *image.YCbCr, w, h int) *image.YCbCr {
	dst := image.NewYCbCr(image.Rect(0, 0, w, h), m.SubsampleRatio)
	kx, ky := float64(m.Rect.Dx())/float64(w), float64(m.Rect.Dy())/float64(h)
	sp, dp := planesOf(m), planesOf(dst)
	for n := range 3 {
		s, d := sp[n], dp[n]
		for j := 0; j < d.h; j++ {
			row := d.pix[j*d.stride:]
			for i := 0; i < d.w; i++ {
				row[i] = byteOf(s.at(float64(i)*d.fx*kx, float64(j)*d.fy*ky))
			}
		}
	}
	return dst
}
