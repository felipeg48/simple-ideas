package render

import (
	"strings"
	"testing"
)

func TestVideoPlayer(t *testing.T) {
	page := func(body string) string { return "---\ntype: article\n---\n# V\n\n" + body }

	// inline: a normal <video controls> with a poster, no dialog
	res, _ := renderFiles(t, map[string]string{"i.md": page("```video src=\"https://x/clip.mp4\" poster=\"https://x/p.jpg\" mode=inline title=\"Demo\"\nA caption.\n```\n")}, "i.md")
	h := string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("inline: %s", problemsText(res))
	}
	if strings.Count(h, `<figure class="si-vp"`) != 1 || !strings.Contains(h, `data-vp-mode="inline"`) {
		t.Errorf("inline: one figure expected")
	}
	if !strings.Contains(h, `<video class="si-vp-video" controls`) {
		t.Errorf("inline: a <video controls> is expected")
	}
	if strings.Contains(h, "<dialog") {
		t.Errorf("inline: no dialog expected")
	}
	if !strings.Contains(h, `<figcaption class="si-vp-cap">A caption.</figcaption>`) {
		t.Errorf("inline: caption missing")
	}
	if !strings.Contains(h, "si video player:") {
		t.Errorf("inline: the script should be inlined")
	}

	// modal: a poster button that opens a <dialog> lightbox, a close control, the script
	res, _ = renderFiles(t, map[string]string{"m.md": page("```video src=\"https://x/clip.mp4\" poster=\"https://x/p.jpg\" mode=modal title=\"Demo\"\n```\n")}, "m.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("modal: %s", problemsText(res))
	}
	if !strings.Contains(h, `data-vp-mode="modal"`) || !strings.Contains(h, `<dialog class="si-vp-modal"`) {
		t.Errorf("modal: a <dialog> lightbox is expected")
	}
	if !strings.Contains(h, `class="si-vp-close"`) {
		t.Errorf("modal: a close control is expected")
	}
	if !strings.Contains(h, `class="si-vp-open"`) || !strings.Contains(h, `<img class="si-vp-poster"`) {
		t.Errorf("modal: a clickable poster is expected")
	}
	if !strings.Contains(h, "si video player:") {
		t.Errorf("modal: the script should be inlined")
	}

	// facade: a poster only, the <video> built on click; no eager <video> in the output
	res, _ = renderFiles(t, map[string]string{
		"f.md":     page("```video src=\"clip.mp4\" poster=\"p.jpg\" mode=facade title=\"Demo\"\n```\n"),
		"clip.mp4": "fake mp4 bytes",
		"p.jpg":    "fake jpg bytes",
	}, "f.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("facade: %s", problemsText(res))
	}
	if !strings.Contains(h, `data-vp-mode="facade"`) || !strings.Contains(h, `data-si-vp-src="clip.mp4"`) {
		t.Errorf("facade: the src should be on a data attribute, resolved: %s", h)
	}
	if strings.Contains(h, "<video") {
		t.Errorf("facade: no eager <video> should be emitted")
	}

	// degrade: only a poster, no src — poster shown, play affordance disabled, no error
	res, _ = renderFiles(t, map[string]string{"d.md": page("```video poster=\"https://x/p.jpg\" mode=inline title=\"Demo\"\n```\n")}, "d.md")
	h = string(res.HTML)
	if len(res.Problems) > 0 {
		t.Fatalf("degrade: %s", problemsText(res))
	}
	if !strings.Contains(h, `data-vp-disabled`) || strings.Contains(h, "<video") {
		t.Errorf("degrade: a poster-only block must not emit a <video>")
	}
	if !strings.Contains(h, `<img class="si-vp-poster"`) {
		t.Errorf("degrade: the poster should still show")
	}

	// an unknown mode is a problem (and falls back to inline)
	res, _ = renderFiles(t, map[string]string{"b.md": page("```video src=\"https://x/clip.mp4\" mode=carousel\n```\n")}, "b.md")
	if !strings.Contains(problemsText(res), `video mode "carousel"`) {
		t.Errorf("unknown mode should be a problem: %s", problemsText(res))
	}

	// a plain fence stays plain; no video player scripts
	res, _ = renderFiles(t, map[string]string{"p.md": page("```go\na()\n```\n")}, "p.md")
	if h := string(res.HTML); strings.Contains(h, `class="si-vp`) || strings.Contains(h, "si video player:") {
		t.Errorf("a plain fence must not become a video player")
	}
}
