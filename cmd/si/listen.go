package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/tools"
	"github.com/felipeg48/simple-ideas/internal/webdemo"
)

// buildAndVoice is si build, then the voice of every Listen bar it wrote.
func buildAndVoice(src string, o *options, stdout, stderr io.Writer) int {
	start := time.Now().Add(-time.Second)
	code := build(src, o, stdout, stderr)
	if code == 0 {
		dir := src
		if fi, err := os.Stat(src); err == nil && !fi.IsDir() {
			dir = filepath.Dir(src)
		}
		out := o.output
		if out == "" {
			out = "html"
		}
		voiceListen(out, start, config.Load(o.conf, filepath.Join(dir, "si.conf")), o.voice, stdout, stderr)
	}
	return code
}

// voiceListen gives the Listen bar (feature: t2s) of every page this build
// wrote in out (changed since start) the build's voice: headless Chrome asks
// each page for its sentences (window.siT2S, the page's own split), the
// engine voices them (--voice, the page's voice:, si.conf voice:, else
// Kokoro when installed; the voice cache makes a rebuild quick), and each
// becomes <page>.voice/<hash>.m4a, listed in the page. With only the system
// voice, or no Chrome or ffmpeg, the page keeps the browser's voice.
func voiceListen(out string, start time.Time, cfg *config.Config, override string, stdout, stderr io.Writer) {
	var pages []string
	filepath.WalkDir(out, func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || !strings.HasSuffix(p, ".html") {
			return nil
		}
		if fi, err := e.Info(); err != nil || fi.ModTime().Before(start) {
			return nil
		}
		if b, err := os.ReadFile(p); err == nil && strings.Contains(string(b), render.T2SMark) && !strings.Contains(string(b), render.T2SMark+`"audio"`) {
			pages = append(pages, p)
		}
		return nil
	})
	if len(pages) == 0 {
		return
	}
	chrome, ff := findChrome(), tools.Find("ffmpeg")
	if chrome == "" || ff == "" {
		fmt.Fprintf(stderr, "  note: Listen pages speak with the browser's voice (the build's voice needs Chrome and ffmpeg)\n")
		return
	}
	c, err := cdp.Launch(chrome, 1200, 900, "--allow-file-access-from-files")
	if err != nil {
		fmt.Fprintf(stderr, "  warning: Listen: %v (the pages speak with the browser's voice)\n", err)
		return
	}
	defer c.Close()
	if _, err := c.OpenPage("", 1200, 900); err != nil {
		fmt.Fprintf(stderr, "  warning: Listen: %v\n", err)
		return
	}
	sy := &webdemo.Synth{FFmpeg: ff, Cache: webdemo.VoiceCache(), Log: stderr}
	for _, p := range pages {
		if err := voicePage(c, sy, ff, p, cfg, override, stdout); err != nil {
			fmt.Fprintf(stderr, "  warning: %s: Listen: %v (the page speaks with the browser's voice)\n", p, err)
		}
	}
}

func voicePage(c *cdp.Client, sy webdemo.Voicer, ff, page string, cfg *config.Config, override string, stdout io.Writer) error {
	html, err := os.ReadFile(page)
	if err != nil {
		return err
	}
	s := string(html)
	at := strings.Index(s, render.T2SMark) + len(render.T2SMark) - 1
	end := strings.Index(s[at:], ";\n")
	var set struct{ Engine string }
	if end < 0 || json.Unmarshal([]byte(s[at:at+end]), &set) != nil {
		return fmt.Errorf("no Listen settings")
	}
	v := webdemo.DefaultVoice()
	if spec := firstNonEmpty(override, set.Engine, cfg.Voice); spec != "" {
		if v, err = webdemo.ParseVoice(spec); err != nil {
			return err
		}
	}
	if v.Engine == "system" {
		return nil // the system voice is the browser's: nothing to add
	}
	if why := v.Ready(); why != "" {
		first, _, _ := strings.Cut(why, "\n")
		return fmt.Errorf("%s", first)
	}
	abs, _ := filepath.Abs(page)
	if err := c.Call("Page.navigate", map[string]any{"url": fileURL(abs)}, nil); err != nil {
		return err
	}
	waitLoaded(c)
	raw, err := c.Eval("window.siT2S ? window.siT2S.sentences() : null")
	if err != nil {
		return err
	}
	var texts []string
	if json.Unmarshal(raw, &texts) != nil || len(texts) == 0 {
		return nil
	}
	lines, _, err := sy.Voice(v, texts)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(page), ".html") + ".voice"
	dir := filepath.Join(filepath.Dir(page), base)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	keep := map[string]bool{}
	audio := make([]string, len(texts))
	size := 0
	for i, t := range texts {
		l, ok := lines[t]
		if !ok {
			return fmt.Errorf("no voice for %q", t)
		}
		h := sha1.Sum([]byte(v.String() + "\n" + t))
		name := hex.EncodeToString(h[:6]) + ".m4a"
		dest := filepath.Join(dir, name)
		if _, err := os.Stat(dest); err != nil { // a sentence voiced by an earlier build stays
			if err := encodeM4A(ff, l.WAV, dest); err != nil {
				return err
			}
		}
		if fi, err := os.Stat(dest); err == nil {
			size += int(fi.Size())
		}
		keep[name] = true
		audio[i] = base + "/" + name
	}
	if old, err := os.ReadDir(dir); err == nil { // sentences the page no longer has
		for _, e := range old {
			if !keep[e.Name()] {
				os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}
	list, _ := json.Marshal(audio)
	s = strings.Replace(s, render.T2SMark, render.T2SMark+`"audio":`+string(list)+",", 1)
	if err := os.WriteFile(page, []byte(s), 0o644); err != nil {
		return err
	}
	rel, _ := filepath.Rel(".", dir)
	fmt.Fprintf(stdout, "  %-28s (listen, %d sentences, %s, %d KB)\n", rel+"/", len(texts), v, size/1024)
	return nil
}
