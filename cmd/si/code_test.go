package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestEncodeGIF(t *testing.T) {
	var frames []image.Image
	for _, c := range []color.RGBA{{200, 80, 60, 255}, {40, 60, 200, 255}, {240, 240, 240, 255}} {
		img := image.NewRGBA(image.Rect(0, 0, 20, 10))
		for y := 0; y < 10; y++ {
			for x := 0; x < 20; x++ {
				img.Set(x, y, c)
			}
		}
		frames = append(frames, img)
	}
	data, err := encodeGIF(frames, []int{220, 6, 220})
	if err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil || len(g.Image) != 3 || g.Delay[0] != 220 || g.Delay[1] != 6 {
		t.Fatalf("gif: %v %d %v", err, len(g.Image), g.Delay)
	}
	if r, _, _, _ := g.Image[1].At(5, 5).RGBA(); r>>8 > 60 {
		t.Errorf("the second frame should stay blue, red=%d", r>>8)
	}
}

func TestWindowSource(t *testing.T) {
	text := "# T\n\n---\n\n## A\n\n```go id=r window=mac\na()\n```\n<!-- callout: 1 one -->\n\n---\n\n## B\n\n```go id=r window=mac\na()\nb()\n```\n\n```sh window=terminal\nls\n```\n\n```go\nplain()\n```\n"
	src, n := windowSource(text, deck.Meta{Theme: "paper"}, "")
	if n != 2 || strings.Count(src, "id=r") != 2 || !strings.Contains(src, "callout: 1 one") || strings.Contains(src, "plain()") || !strings.Contains(src, "theme: paper") {
		t.Fatalf("%d\n%s", n, src)
	}
	if _, n := windowSource(text, deck.Meta{}, "r"); n != 1 {
		t.Errorf("--id r: %d shows", n)
	}
}
