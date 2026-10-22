package deck

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Background is a parsed `background:` value (slide directive or front matter):
//
//	<colour|gradient|image path|video path|none> [fit=cover|contain] [position=center|top|left-top ...]
//	[opacity=0..1] [blur=N] [overlay=auto|dark|light|none] [poster=image path]
type Background struct {
	Value    string // colour, gradient or path
	Kind     string // colour | image | video | none
	Fit      string
	Position string
	Opacity  string
	Blur     string
	Overlay  string
	Poster   string
}

var (
	videoExtRE = regexp.MustCompile(`(?i)\.(mp4|webm|mov|m4v|ogv)$`)
	bgOptRE    = regexp.MustCompile(`^(fit|position|opacity|blur|overlay|poster)=(\S+)$`)
)

// BackgroundOptions lists the accepted background options.
var BackgroundOptions = []string{"fit", "position", "opacity", "blur", "overlay", "poster"}

// ParseBackground splits a background value into the value and its options
// and validates both. Paths are not checked here (see CheckBackground).
func ParseBackground(v string) (Background, error) {
	f := strings.Fields(v)
	var b Background
	end := len(f)
	for end > 0 && bgOptRE.MatchString(f[end-1]) {
		end--
	}
	for _, opt := range f[end:] {
		m := bgOptRE.FindStringSubmatch(opt)
		val := m[2]
		switch m[1] {
		case "fit":
			if val != "cover" && val != "contain" {
				return b, fmt.Errorf("background fit=%s: use cover or contain", val)
			}
			b.Fit = val
		case "position":
			b.Position = strings.ReplaceAll(val, "-", " ")
		case "opacity":
			x, err := strconv.ParseFloat(val, 64)
			if err != nil || x < 0 || x > 1 {
				return b, fmt.Errorf("background opacity=%s: use a number from 0 to 1", val)
			}
			b.Opacity = val
		case "blur":
			x, err := strconv.ParseFloat(val, 64)
			if err != nil || x < 0 || x > 100 {
				return b, fmt.Errorf("background blur=%s: use a number from 0 to 100", val)
			}
			b.Blur = val
		case "overlay":
			if val != "auto" && val != "dark" && val != "light" && val != "none" {
				return b, fmt.Errorf("background overlay=%s: use auto, dark, light or none", val)
			}
			b.Overlay = val
		case "poster":
			b.Poster = val
		}
	}
	b.Value = strings.Join(f[:end], " ")
	switch {
	case b.Value == "":
		return b, fmt.Errorf("background needs a value: a colour, a gradient, an image or a video path")
	case b.Value == "none":
		b.Kind = "none"
	case isPath(b.Value) && videoExtRE.MatchString(b.Value):
		b.Kind = "video"
	case IsImagePath(b.Value):
		b.Kind = "image"
	default:
		b.Kind = "colour"
	}
	if b.Kind == "colour" && (b.Fit != "" || b.Blur != "" || b.Poster != "") {
		return b, fmt.Errorf("background %q: fit, blur and poster apply to images and videos only", b.Value)
	}
	return b, nil
}

func isPath(v string) bool { return !strings.ContainsAny(v, "() ") }

// Files returns the files a background refers to.
func (b Background) Files() []string {
	var out []string
	if b.Kind == "image" || b.Kind == "video" {
		out = append(out, b.Value)
	}
	if b.Poster != "" {
		out = append(out, b.Poster)
	}
	return out
}

func (d *Deck) checkBackground(file string, line int, dir, v string) {
	b, err := ParseBackground(v)
	if err != nil {
		d.problem(file, line, "%v", err)
		return
	}
	for _, f := range b.Files() {
		if !exists(filepath.Join(dir, f)) {
			d.problem(file, line, "missing file: %s", f)
		}
	}
}

// ImageAttrs is a parsed `<!-- image: ... -->` directive.
type ImageAttrs struct {
	Width    string // CSS length or percentage of the content width
	Align    string // left | right | center
	Ratio    string // aspect ratio box, e.g. 16:9
	Fit      string // cover | contain (with ratio)
	Position string // object-position
}

var (
	imgOptRE   = regexp.MustCompile(`^(width|align|ratio|fit|position)=(\S+)$`)
	widthRE    = regexp.MustCompile(`^\d+(\.\d+)?(%|em|rem|cqi|cqw|ch|vw)$`)
	ratioRE    = regexp.MustCompile(`^\d+(\.\d+)?[:/]\d+(\.\d+)?$`)
	fixedPxRE  = regexp.MustCompile(`px$`)
	ImageOpts  = []string{"width", "align", "ratio", "fit", "position"}
	imageAlign = []string{"left", "right", "center"}
)

// ParseImageAttrs parses the options of the image directive.
func ParseImageAttrs(v string) (ImageAttrs, error) {
	var a ImageAttrs
	for _, f := range strings.Fields(v) {
		m := imgOptRE.FindStringSubmatch(f)
		if m == nil {
			return a, fmt.Errorf("image option %q: use %s as key=value", f, strings.Join(ImageOpts, ", "))
		}
		val := m[2]
		switch m[1] {
		case "width":
			if fixedPxRE.MatchString(val) {
				return a, fmt.Errorf("image width=%s: fixed pixel sizes break the fluid layout; use %%, em or cqi", val)
			}
			if !widthRE.MatchString(val) {
				return a, fmt.Errorf("image width=%s: use a percentage or a relative length (40%%, 20em, 30cqi)", val)
			}
			a.Width = val
		case "align":
			if msg := oneOf("image align", val, imageAlign); msg != "" {
				return a, fmt.Errorf("%s", msg)
			}
			a.Align = val
		case "ratio":
			if !ratioRE.MatchString(val) {
				return a, fmt.Errorf("image ratio=%s: use W:H, e.g. 16:9", val)
			}
			a.Ratio = strings.Replace(val, ":", "/", 1)
		case "fit":
			if val != "cover" && val != "contain" {
				return a, fmt.Errorf("image fit=%s: use cover or contain", val)
			}
			a.Fit = val
		case "position":
			a.Position = strings.ReplaceAll(val, "-", " ")
		}
	}
	return a, nil
}
