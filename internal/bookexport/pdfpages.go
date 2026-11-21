package bookexport

import (
	"regexp"
	"strconv"
)

var (
	pdfObjRE   = regexp.MustCompile(`(?s)(?:^|\n)(\d+) 0 obj\s*(.*?)endobj`)
	pdfCatRE   = regexp.MustCompile(`/Type\s*/Catalog\b`)
	pdfPagesRE = regexp.MustCompile(`/Pages\s+(\d+) 0 R`)
	pdfDestsRE = regexp.MustCompile(`/Dests\s+(\d+) 0 R`)
	pdfKidsRE  = regexp.MustCompile(`/Kids\s*\[([^\]]*)\]`)
	pdfRefRE   = regexp.MustCompile(`(\d+) 0 R`)
	pdfDestRE  = regexp.MustCompile(`/((?:[^\s/\[\]()<>{}%]|#[0-9A-Fa-f]{2})+)\s*\[\s*(\d+) 0 R`)
	pdfHexRE   = regexp.MustCompile(`#([0-9A-Fa-f]{2})`)
)

// PDFPages reads a PDF Chrome printed (objects not compressed into streams)
// and returns the page, from 1, of each named destination: the ids of the
// elements the page's links point to. The book PDF uses it to write page
// numbers into its contents on a second print. Nil when it cannot tell.
func PDFPages(pdf []byte) map[string]int {
	objs := map[string]string{}
	for _, m := range pdfObjRE.FindAllSubmatch(pdf, -1) {
		objs[string(m[1])] = string(m[2])
	}
	root, dests := "", ""
	for _, o := range objs {
		if m := pdfPagesRE.FindStringSubmatch(o); m != nil && pdfCatRE.MatchString(o) {
			root = m[1]
			if d := pdfDestsRE.FindStringSubmatch(o); d != nil {
				dests = d[1]
			}
			break
		}
	}
	if root == "" || dests == "" {
		return nil
	}
	page := map[string]int{} // page object -> number
	n := 0
	seen := map[string]bool{}
	var walk func(id string)
	walk = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		o := objs[id]
		k := pdfKidsRE.FindStringSubmatch(o)
		if k == nil {
			n++
			page[id] = n
			return
		}
		for _, r := range pdfRefRE.FindAllStringSubmatch(k[1], -1) {
			walk(r[1])
		}
	}
	walk(root)
	out := map[string]int{}
	for _, m := range pdfDestRE.FindAllStringSubmatch(objs[dests], -1) {
		if p := page[m[2]]; p > 0 {
			name := pdfHexRE.ReplaceAllStringFunc(m[1], func(h string) string {
				v, _ := strconv.ParseUint(h[1:], 16, 8)
				return string(rune(v))
			})
			out[name] = p
		}
	}
	return out
}
