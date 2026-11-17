package main

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/fontsub"
	"github.com/felipeg48/simple-ideas/internal/render"
)

var (
	svgFaceRE  = regexp.MustCompile(`@font-face\s*\{[^}]*\}`)
	svgSrcRE   = regexp.MustCompile(`src:\s*url\(\s*["']?data:font/[\w-]+;base64,([A-Za-z0-9+/=]+)["']?\s*\)(\s*format\([^)]*\))?\s*;?`)
	svgPropRE  = func(p string) *regexp.Regexp { return regexp.MustCompile(p + `:\s*["']?([^;"'}]+)`) }
	svgFamRE   = svgPropRE("font-family")
	svgWghtRE  = svgPropRE("font-weight")
	svgStyleRE = svgPropRE("font-style")
	svgRangeRE = regexp.MustCompile(`unicode-range:[^;}]*;?`)
)

// svgFonts cuts the fonts an exported SVG embeds down to the characters its
// text uses (fontsub: TrueType, and WOFF 1 read into TrueType). A WOFF2 font
// (what si inlines from Google Fonts) is fetched again from Google as
// TrueType when the deck's fonts: takes it from there; one TrueType file then
// serves every unicode-range subset of that face. What cannot be cut (WOFF2
// from elsewhere, CFF, variable fonts, scripts that need shaping) stays
// whole, and a note says so.
func svgFonts(svg string, google map[string]bool) (string, []string) {
	text, ok := svgText(svg)
	if !ok {
		return svg, nil
	}
	var notes []string
	done := map[string]bool{}
	out := svgFaceRE.ReplaceAllStringFunc(svg, func(rule string) string {
		m := svgSrcRE.FindStringSubmatch(rule)
		if m == nil {
			return rule
		}
		font, err := base64.StdEncoding.DecodeString(m[1])
		if err != nil {
			return rule
		}
		fam, weight, style := prop(svgFamRE, rule, ""), prop(svgWghtRE, rule, "400"), prop(svgStyleRE, rule, "normal")
		weight = strings.Fields(weight)[0]
		name := fam + " " + weight
		refetched := false
		switch {
		case len(font) >= 4 && string(font[:4]) == "wOFF":
			if f, err := fontsub.FromWOFF(font); err == nil {
				font = f
			}
		case len(font) >= 4 && string(font[:4]) == "wOF2":
			key := strings.ToLower(fam) + "|" + weight + "|" + style
			if done[key] {
				return "" // the TrueType file above covers every subset
			}
			if !google[strings.ToLower(fam)] {
				notes = append(notes, name+": WOFF2 only, embedded whole")
				return rule
			}
			ttf, err := render.GoogleTTF(fam, weight, style)
			if err != nil {
				notes = append(notes, fmt.Sprintf("%s: embedded whole (no TrueType copy: %v)", name, err))
				return rule
			}
			font, refetched, done[key] = ttf, true, true
		}
		if fontsub.Shaped(text) {
			notes = append(notes, name+": embedded whole (text that needs shaping)")
			return rule
		}
		sub, err := fontsub.Subset(font, text)
		if err != nil {
			notes = append(notes, name+": embedded whole (not a static TrueType font)")
			return rule
		}
		if refetched {
			rule = svgRangeRE.ReplaceAllString(rule, "")
		}
		src := `src: url("data:font/ttf;base64,` + base64.StdEncoding.EncodeToString(sub) + `") format("truetype");`
		return strings.Replace(rule, m[0], src, 1)
	})
	return out, notes
}

func prop(re *regexp.Regexp, rule, def string) string {
	if m := re.FindStringSubmatch(rule); m != nil && strings.TrimSpace(m[1]) != "" {
		return strings.TrimSpace(m[1])
	}
	return def
}

// svgText is every character of an SVG's text (not its style sheets).
func svgText(svg string) (string, bool) {
	d := xml.NewDecoder(strings.NewReader(svg))
	var b strings.Builder
	style := 0
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return b.String(), true
		}
		if err != nil {
			return "", false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "style" || t.Name.Local == "script" {
				style++
			}
		case xml.EndElement:
			if t.Name.Local == "style" || t.Name.Local == "script" {
				style--
			}
		case xml.CharData:
			if style == 0 {
				b.Write(t)
			}
		}
	}
}
