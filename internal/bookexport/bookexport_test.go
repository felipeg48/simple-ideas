package bookexport

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func exampleBook(t *testing.T) *Book {
	t.Helper()
	g, err := deck.LoadGuide(filepath.Join("..", "..", "examples", "book"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := FromGuide(g)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestModelNumbersAndCrossReferences(t *testing.T) {
	b := exampleBook(t)
	if len(b.Chapters) != 5 {
		t.Fatalf("chapters = %d, want 5", len(b.Chapters))
	}
	if h := b.Chapters[1].Heading(); h != "Chapter 1 · The order journey" {
		t.Errorf("heading = %q", h)
	}
	if b.Chapters[1].Part != "Part I · Foundations" || b.Chapters[4].Kind != "Appendix" || b.Chapters[4].Num != "A" {
		t.Errorf("parts and appendices: %+v / %+v", b.Chapters[1].Part, b.Chapters[4])
	}
	md := Markdown(b, nil)
	for _, want := range []string{
		"## 1.1 Five stations",
		"Figure 1.1 shows them, and Table 1.1 names their owners",
		"*Figure 1.1 · The five stations, from checkout to the door*",
		"*Table 1.1 · Who owns each station*",
		"see Section 2.1.",
		"| Station | Owner | Answers within |",
		"```java\n",
		"var key = order.id();                                  // (1)",
		"1. The order id is the idempotency key: a retry is the same request.",
		"checkout, payment, warehouse", // index terms shown as plain words
		"computed at checkout",
		"The retry is logged", // (((hidden))) terms leave nothing
		"title: \"The Northwind Operations Reference\"",
		"# Part II · When it breaks",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q", want)
		}
	}
	for _, bad := range []string{"((", "<!--", "<1>", "Table:"} {
		if strings.Contains(md, bad) {
			t.Errorf("markdown keeps %q", bad)
		}
	}
}

func TestCodeLinesMarkers(t *testing.T) {
	b := Block{Kind: Code, Text: "a := 1 // <1>\nb := 2 /* <.> */\n<p>x</p> <!-- 3 -->\nplain", Callouts: []Callout{{N: "1"}}}
	got := CodeLines(b)
	want := "a := 1 // (1)\nb := 2 /* (2) */\n<p>x</p> <!-- (3) -->\nplain"
	if got != want {
		t.Errorf("CodeLines:\n%s\nwant\n%s", got, want)
	}
}

func TestAdmonitionsAndLists(t *testing.T) {
	c := newConv()
	c.file = "/x/a.md"
	c.slugs = map[string]bool{}
	bs := c.blocks("> [!WARNING]\n> Do not retry twice.\n\n1. one\n2. two\n   - nested\n\n<!-- steps: focus -->\n\n```chart\nbar\n```\n", "/x", "/x/a.md", "", "")
	if len(bs) != 3 || bs[0].Kind != Admonition || bs[0].Label != "Warning" || bs[1].Kind != List || bs[2].Kind != Placeholder {
		t.Fatalf("blocks = %+v", bs)
	}
	md := mdBlocks(bs, nil)
	for _, want := range []string{"> **Warning:** Do not retry twice.", "1. one\n2. two\n   - nested", "Figure: a chart (in the HTML and PDF editions)"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown lacks %q in\n%s", want, md)
		}
	}
}

func unzip(t *testing.T, data []byte) (*zip.Reader, map[string][]byte) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range zr.File {
		r, _ := f.Open()
		files[f.Name], _ = io.ReadAll(r)
		r.Close()
	}
	return zr, files
}

func wellFormed(t *testing.T, name string, data []byte) {
	t.Helper()
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		_, err := d.Token()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("%s is not well-formed XML: %v", name, err)
		}
	}
}

func TestDocx(t *testing.T) {
	b := exampleBook(t)
	raster := func(string) ([]byte, int, int, error) { return tinyPNG(), 2, 2, nil }
	data, err := Docx(b, Options{Raster: raster})
	if err != nil {
		t.Fatal(err)
	}
	_, files := unzip(t, data)
	for _, f := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/styles.xml", "word/numbering.xml",
		"word/settings.xml", "word/footer1.xml", "word/_rels/document.xml.rels", "docProps/core.xml", "word/media/stations.png"} {
		if files[f] == nil {
			t.Errorf("no %s", f)
		}
	}
	for name, body := range files {
		if strings.HasSuffix(name, ".xml") || strings.HasSuffix(name, ".rels") {
			wellFormed(t, name, body)
		}
	}
	doc := string(files["word/document.xml"])
	for _, want := range []string{`TOC \o "1-3" \h \z \u`, `w:val="Heading1"`, `w:val="Code"`, `w:val="TableGrid"`, `r:embed="`, "Figure 1.1 · The five stations"} {
		if !strings.Contains(doc, want) {
			t.Errorf("document.xml lacks %q", want)
		}
	}
	if !strings.Contains(string(files["word/settings.xml"]), `<w:updateFields w:val="true"/>`) {
		t.Error("settings.xml does not update the TOC on open")
	}
}

func TestRTF(t *testing.T) {
	b := exampleBook(t)
	r := RTF(b, Options{Raster: func(string) ([]byte, int, int, error) { return tinyPNG(), 2, 2, nil }})
	if !strings.HasPrefix(r, `{\rtf1\ansi`) || !strings.HasSuffix(strings.TrimSpace(r), "}") {
		t.Fatal("not an RTF document")
	}
	if strings.Count(r, "{")-strings.Count(r, `\{`) != strings.Count(r, "}")-strings.Count(r, `\}`) {
		t.Error("unbalanced braces")
	}
	for _, want := range []string{`\pngblip`, `\trowd`, `\cell`, `\row`, `\pagebb`, `\b `, `\f1`, `Chapter 1 \u183?`} {
		if !strings.Contains(r, want) {
			t.Errorf("rtf lacks %q", want)
		}
	}
}

func TestEPUB(t *testing.T) {
	b := exampleBook(t)
	data, err := EPUB(b, Options{})
	if err != nil {
		t.Fatal(err)
	}
	zr, files := unzip(t, data)
	if zr.File[0].Name != "mimetype" || zr.File[0].Method != zip.Store || string(files["mimetype"]) != "application/epub+zip" {
		t.Fatalf("mimetype must come first, stored: %+v", zr.File[0].FileHeader)
	}
	for _, f := range []string{"META-INF/container.xml", "OEBPS/content.opf", "OEBPS/nav.xhtml", "OEBPS/style.css", "OEBPS/images/stations.svg", "OEBPS/title.xhtml"} {
		if files[f] == nil {
			t.Errorf("no %s", f)
		}
	}
	for name, body := range files {
		if strings.HasSuffix(name, ".xhtml") || strings.HasSuffix(name, ".opf") || strings.HasSuffix(name, ".xml") {
			wellFormed(t, name, body)
		}
	}
	if !strings.Contains(string(files["OEBPS/nav.xhtml"]), "Chapter 1 · The order journey") {
		t.Error("nav lacks the chapters")
	}
}

func TestWriteMarkdownCopiesImages(t *testing.T) {
	b := exampleBook(t)
	dir := t.TempDir()
	files, err := WriteMarkdown(b, dir, "book")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 6 || filepath.Base(files[0]) != "book.md" {
		t.Errorf("files = %v", files)
	}
	if _, err := os.Stat(filepath.Join(dir, "images", "stations.svg")); err != nil {
		t.Error(err)
	}
	ch, _ := os.ReadFile(filepath.Join(dir, "01-journey.md"))
	if !strings.HasPrefix(string(ch), "# Chapter 1 · The order journey\n") || !strings.Contains(string(ch), "](images/stations.svg)") {
		t.Errorf("01-journey.md:\n%s", ch)
	}
}

func TestPDFPages(t *testing.T) {
	pdf := []byte("1 0 obj\n<</Type /Catalog\n/Pages 2 0 R\n/Dests 9 0 R>>\nendobj\n" +
		"2 0 obj\n<</Type /Pages\n/Count 3\n/Kids [3 0 R 7 0 R]>>\nendobj\n" +
		"7 0 obj\n<</Type /Pages\n/Count 2\n/Kids [4 0 R 5 0 R]\n/Parent 2 0 R>>\nendobj\n" +
		"3 0 obj\n<</Type /Page>>\nendobj\n4 0 obj\n<</Type /Page>>\nendobj\n5 0 obj\n<</Type /Page>>\nendobj\n" +
		"9 0 obj\n<</c-one [3 0 R /XYZ 0 0 0]\n/c-two-x#2Dy [5 0 R /XYZ 0 0 0]>>\nendobj\n")
	got := PDFPages(pdf)
	if got["c-one"] != 1 || got["c-two-x-y"] != 3 || len(got) != 2 {
		t.Errorf("PDFPages = %v", got)
	}
}

func tinyPNG() []byte {
	return []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\xff\xff?\x00\x05\xfe\x02\xfe\xa7\x35\x81\x84\x00\x00\x00\x00IEND\xaeB`\x82")
}
