package tui

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPictureProtocol(t *testing.T) {
	for _, c := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"TERM": "xterm-kitty"}, "kitty"},
		{map[string]string{"TERM_PROGRAM": "ghostty"}, "kitty"},
		{map[string]string{"TERM_PROGRAM": "iTerm.app"}, "iterm"},
		{map[string]string{"TERM_PROGRAM": "iTerm.app", "TMUX": "/tmp/tmux"}, ""},
		{map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, ""},
		{map[string]string{"TERM": "xterm-kitty", "SI_PICTURES": "halfblock"}, ""},
		{map[string]string{"SI_PICTURES": "iterm"}, "iterm"},
	} {
		for _, k := range []string{"TERM", "TERM_PROGRAM", "TMUX", "STY", "KITTY_WINDOW_ID", "LC_TERMINAL", "SI_PICTURES"} {
			t.Setenv(k, c.env[k])
		}
		if got := PictureProtocol(); got != c.want {
			t.Errorf("%v: %q, want %q", c.env, got, c.want)
		}
	}
}

func TestRealPicturesReserveAndDraw(t *testing.T) {
	dir := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for x := 0; x < 40; x++ {
		img.Set(x, 5, color.RGBA{255, 0, 0, 255})
	}
	f, _ := os.Create(filepath.Join(dir, "red.png"))
	png.Encode(f, img)
	f.Close()
	os.WriteFile(filepath.Join(dir, "talk.md"), []byte("# T\n\n---\n\n## Picture\n\n![A red line](red.png)\n"), 0o644)
	d2, _ := parsePage(filepath.Join(dir, "talk.md"), dir)
	for _, proto := range []string{"kitty", "iterm"} {
		pics := &Pictures{Proto: proto}
		screen, _ := Slide(d2, 1, 0, 100, 30, PaletteFor("", ""), pics)
		if strings.Contains(screen, "▀") {
			t.Errorf("%s: no half blocks when the terminal draws pictures", proto)
		}
		clean, places := pics.placements(screen)
		if len(places) != 1 || strings.ContainsRune(clean, rune(markerBase)) {
			t.Fatalf("%s: one placement, marker gone: %v", proto, places)
		}
		pl := places[0]
		if pl.cols <= 0 || pl.rows <= 0 || pl.row < 2 || pl.col < 2 {
			t.Errorf("%s: placement %+v", proto, pl)
		}
		seq := pics.draw(places)
		want := map[string]string{"kitty": "\x1b_Ga=T,f=100,c=", "iterm": "\x1b]1337;File=inline=1;"}[proto]
		if !strings.Contains(seq, want) || !strings.HasPrefix(seq[strings.Index(seq, "\x1b7"):], "\x1b7\x1b[") {
			t.Errorf("%s: sequence %.80q", proto, seq)
		}
		if !strings.Contains(plain(clean), "A red line") {
			t.Errorf("%s: the caption stays", proto)
		}
	}
	if seq := kittyImage(make([]byte, 9000), 10, 5); strings.Count(seq, "\x1b_G") != 3 || !strings.Contains(seq, "m=0;") {
		t.Errorf("kitty chunks: %d", strings.Count(seq, "\x1b_G"))
	}
}
