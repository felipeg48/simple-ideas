package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTimelinePDFIsText: the paged PDF is printed, not pictures: one page per
// group of eras, the words in it as text (a font, no page-sized picture).
func TestTimelinePDFIsText(t *testing.T) {
	if findChrome() == "" || testing.Short() {
		t.Skip("needs Chrome")
	}
	dir := t.TempDir()
	src := "---\ntype: timeline\ntitle: Ferrocarriles\n---\n\n"
	for _, era := range []string{"Vapor", "Diésel", "Alta velocidad"} {
		src += "## " + era + "\n\n"
		for i := range 7 {
			src += fmt.Sprintf("### 19%d%d · Hito %s %d\nUn tren nuevo.\n\n", len(era)%10, i, era, i)
		}
	}
	os.WriteFile(filepath.Join(dir, "rail.md"), []byte(src), 0o644)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"build", "--pdf", "-o", filepath.Join(dir, "out"), filepath.Join(dir, "rail.md")}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "out", "pdfs", "rail.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	all := string(raw)
	for _, s := range regexp.MustCompile(`(?s)stream\r?\n(.*?)endstream`).FindAllStringSubmatch(all, -1) {
		if r, err := zlib.NewReader(strings.NewReader(s[1])); err == nil {
			b, _ := io.ReadAll(r)
			all += string(b)
		}
	}
	if n := len(regexp.MustCompile(`/Type\s*/Page\b`).FindAllString(all, -1)); n != 3 {
		t.Errorf("pages: %d, want 3", n)
	}
	if !regexp.MustCompile(`/FontFile|/Type\s*/Font`).MatchString(all) {
		t.Error("no font: the pages are pictures, not text")
	}
	if regexp.MustCompile(`/Width (1[6-9]\d\d|[2-9]\d\d\d)\b`).MatchString(all) {
		t.Error("a page-sized picture in the PDF")
	}
}

func TestTimelinePages(t *testing.T) {
	era := func(name string, n int) string {
		s := "## " + name + "\n\n"
		for i := 0; i < n; i++ {
			s += "### 19" + string(rune('0'+i)) + "0 · M\nText.\n\n"
		}
		return s
	}
	src := "---\ntype: timeline\ntitle: T\nclosing: The end.\nrequire: 20 dates\n---\n\nIntro.\n\n" +
		era("A", 5) + era("B", 5) + era("C", 6) + "## Fuentes\n\n- Book.\n"
	pages := timelinePages(src, 12)
	if len(pages) != 2 {
		t.Fatalf("want 2 pages, got %d", len(pages))
	}
	if !strings.Contains(pages[0], "Intro.") || !strings.Contains(pages[0], "## A") || !strings.Contains(pages[0], "## B") || strings.Contains(pages[0], "## C") {
		t.Fatalf("page 1: %s", pages[0])
	}
	for i, p := range pages {
		if strings.Contains(p, "require:") || !strings.Contains(p, "title: T") {
			t.Fatalf("page %d front matter: %s", i+1, p)
		}
	}
	if strings.Contains(pages[0], "closing:") || strings.Contains(pages[0], "Fuentes") {
		t.Fatal("closing and sources belong on the last page")
	}
	if !strings.Contains(pages[1], "closing: The end.") || !strings.Contains(pages[1], "## Fuentes") || !strings.Contains(pages[1], "## C") {
		t.Fatalf("last page: %s", pages[1])
	}
	if got := timelinePages(era("A", 3), 12); len(got) != 1 {
		t.Fatalf("a short timeline stays one page, got %d", len(got))
	}
}
