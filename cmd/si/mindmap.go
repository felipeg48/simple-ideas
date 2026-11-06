package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/mindmap"
)

// mapSVG is one map as the runtime exports it (#si-export).
type mapSVG struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	W     int    `json:"w"`
	H     int    `json:"h"`
	SVG   string `json:"svg"`
	Own   bool   `json:"own"` // from the page itself: SVG and PNG files; the PDF has the linked maps too
}

var exportRE = regexp.MustCompile(`(?s)<pre id="si-export"[^>]*>(.*?)</pre>`)

// mapSVGs has headless Chrome load a built mind map page at #si-export, where
// mindmap.js writes every map as a standalone SVG (2D, styles and fonts inside).
func mapSVGs(chrome, htmlPath string) ([]mapSVG, error) {
	abs, err := filepath.Abs(htmlPath)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, chrome, "--headless=new", "--disable-gpu", "--virtual-time-budget=4000",
		"--window-size=1600,1000", "--dump-dom", "file://"+filepath.ToSlash(abs)+"#si-export").Output()
	if err != nil {
		return nil, fmt.Errorf("chrome: %v", err)
	}
	m := exportRE.FindSubmatch(out)
	if m == nil {
		return nil, fmt.Errorf("the page did not export its maps")
	}
	var maps []mapSVG
	if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &maps); err != nil {
		return nil, err
	}
	return maps, nil
}

// exportMindmap writes <name>-<id>.svg / .png per map and pdfs/<name>.pdf
// (one page per map, map › topics linking to their page) next to the HTML.
func exportMindmap(chrome, htmlPath, out, name string, svg, png, pdf bool, scale float64, stdout io.Writer) (pdfRel string, err error) {
	maps, err := mapSVGs(chrome, htmlPath)
	if err != nil {
		return "", err
	}
	tmp, err := os.MkdirTemp("", "si-map-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	for _, m := range maps {
		if !m.Own {
			continue
		}
		base := name + "-" + m.ID
		if svg {
			p := filepath.Join(out, base+".svg")
			if err := os.WriteFile(p, []byte(m.SVG), 0o644); err != nil {
				return "", err
			}
			fmt.Fprintf(stdout, "  %-28s (svg, %d × %d)\n", base+".svg", m.W, m.H)
		}
		if png {
			page := filepath.Join(tmp, base+".html")
			doc := `<!doctype html><meta charset="utf-8"><style>html,body{margin:0;background:transparent}svg{display:block}</style>` + m.SVG
			if err := os.WriteFile(page, []byte(doc), 0o644); err != nil {
				return "", err
			}
			p, _ := filepath.Abs(filepath.Join(out, base+".png"))
			if err := chromeRun(chrome, "--screenshot="+p, fmt.Sprintf("--window-size=%d,%d", m.W, m.H),
				fmt.Sprintf("--force-device-scale-factor=%g", scale), "--hide-scrollbars", "--default-background-color=00000000", "file://"+filepath.ToSlash(page)); err != nil {
				return "", fmt.Errorf("png %s: %v", base, err)
			}
			fmt.Fprintf(stdout, "  %-28s (png, %d × %d)\n", base+".png", int(float64(m.W)*scale), int(float64(m.H)*scale))
		}
	}
	if !pdf {
		return "", nil
	}
	// one page per map, each its own size; links between pages are #map-<id> anchors
	var b strings.Builder
	b.WriteString(`<!doctype html><meta charset="utf-8"><style>html,body{margin:0}section{break-after:page;overflow:hidden}svg{display:block}`)
	for i, m := range maps {
		fmt.Fprintf(&b, "@page p%d{size:%dpx %dpx;margin:0}#map-%s{page:p%d;width:%dpx;height:%dpx}", i, m.W, m.H, m.ID, i, m.W, m.H)
	}
	b.WriteString("</style>")
	for _, m := range maps {
		fmt.Fprintf(&b, `<section id="map-%s">%s</section>`, m.ID, m.SVG)
	}
	page := filepath.Join(tmp, name+"-maps.html")
	if err := os.WriteFile(page, []byte(b.String()), 0o644); err != nil {
		return "", err
	}
	rel := filepath.Join("pdfs", name+".pdf")
	p, _ := filepath.Abs(filepath.Join(out, rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	if err := chromeRun(chrome, "--no-pdf-header-footer", "--print-to-pdf="+p, "file://"+filepath.ToSlash(page)); err != nil {
		return "", fmt.Errorf("pdf: %v", err)
	}
	fmt.Fprintf(stdout, "  %-28s (pdf, %d page%s)\n", rel, len(maps), map[bool]string{true: "", false: "s"}[len(maps) == 1])
	return filepath.ToSlash(rel), nil
}

func chromeRun(chrome string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, chrome, append([]string{"--headless=new", "--disable-gpu"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// runMindmap is `si mindmap import <file> [-o out.md]` and
// `si mindmap export <map.md> --to opml|freemind|xmind|mermaid [--map id] [-o file]`.
func runMindmap(args []string, o *options, stdout, stderr io.Writer) int {
	if len(args) != 2 || (args[0] != "import" && args[0] != "export") {
		return usageErr(stderr, "mindmap takes import <file.opml|.mm|.xmind|.mmd> or export <map.md> --to opml|freemind|xmind|mermaid")
	}
	raw, err := os.ReadFile(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if args[0] == "import" {
		d, err := mindmap.Import(filepath.Base(args[1]), raw)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		dest := o.output
		if dest == "" {
			dest = strings.TrimSuffix(args[1], filepath.Ext(args[1])) + ".md"
		}
		if _, err := os.Stat(dest); err == nil {
			fmt.Fprintf(stderr, "si: %s exists; pick another name with -o\n", dest)
			return exitProblems
		}
		if err := os.WriteFile(dest, []byte(d.String()), 0o644); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		maps, topics := mindmap.Count(d.String())
		fmt.Fprintf(stdout, "si: %s → %s (%d map%s, %d topics)\n", args[1], dest, maps, map[bool]string{true: "", false: "s"}[maps == 1], topics)
		return exitOK
	}
	ext := map[string]string{"opml": ".opml", "freemind": ".mm", "xmind": ".xmind", "mermaid": ".mmd"}[o.to]
	if ext == "" {
		return usageErr(stderr, "mindmap export needs --to opml|freemind|xmind|mermaid")
	}
	data, err := mindmap.Export(mindmap.Parse(string(raw)), o.to, o.mapID)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	dest := o.output
	if dest == "" {
		dest = strings.TrimSuffix(args[1], filepath.Ext(args[1])) + ext
	}
	if dest == "-" {
		stdout.Write(data)
		return exitOK
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "si: %s → %s\n", args[1], dest)
	return exitOK
}
