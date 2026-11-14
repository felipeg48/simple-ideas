package config

import (
	"strings"
	"testing"
)

const founderMap = `theme: harbor
structure:
  rabbitmq/
    md/           markdown
    slides/       deck
      pdf/        deck:pdf
    labs/         lab      # the hands-on part
    html/         site

      images/     image
      animations/ animation
      pdfs/       deck:pdf
    videos/       video
    dist/         html/:zip, deck:zip, course:scorm
# after the block
feature.draw: off
`

func TestStructureParse(t *testing.T) {
	c := Parse(strings.NewReader(founderMap))
	if len(c.Warnings) > 0 {
		t.Fatalf("warnings: %v", c.Warnings)
	}
	if c.Theme != "harbor" || c.Features["draw"] {
		t.Errorf("keys around the block: theme %q draw %v", c.Theme, c.Features["draw"])
	}
	s := c.Structure
	if s == nil || s.Project != "rabbitmq" {
		t.Fatalf("structure %+v", s)
	}
	var got []string
	for _, f := range s.Folders {
		var rs []string
		for _, r := range f.Recipes {
			rs = append(rs, r.String())
		}
		got = append(got, f.Path+"="+strings.Join(rs, ","))
	}
	want := "md=markdown slides=deck slides/pdf=deck:pdf labs=lab html=site html/images=image html/animations=animation html/pdfs=deck:pdf videos=video dist=html/:zip,deck:zip,course:scorm"
	if strings.Join(got, " ") != want {
		t.Errorf("folders\n got %s\nwant %s", strings.Join(got, " "), want)
	}
	if f := s.FoldersFor("deck", "pdf"); strings.Join(f, " ") != "slides/pdf html/pdfs" {
		t.Errorf("deck:pdf in %v", f)
	}
	if !s.Has("course", "scorm") || s.Has("lab", "zip") {
		t.Error("Has")
	}
}

func TestStructureNoTopFolder(t *testing.T) {
	c := Parse(strings.NewReader("structure:\n  slides/ deck\n  dist/ slides/:zip\n"))
	if len(c.Warnings) > 0 || c.Structure.Project != "" || len(c.Structure.Folders) != 2 {
		t.Fatalf("%v %+v", c.Warnings, c.Structure)
	}
}

func TestStructureErrors(t *testing.T) {
	for _, tc := range []struct{ conf, want string }{
		{"structure:\n  slides deck\n", "line 2: structure: a folder name ends in /"},
		{"structure:\n  slides/ decks\n", `line 2: structure: unknown kind "decks"`},
		{"structure:\n  slides/ deck:gif\n", `line 2: structure: unknown format "gif" for deck`},
		{"structure:\n  img/ image:png\n", `image has no formats`},
		{"structure:\n  dist/ html/:zip\n", "line 2: structure: html/:zip names a folder the map does not define"},
		{"structure:\n  html/ site:tar\n", `unknown format "tar" for site`},
		{"structure:\n  a/ deck\n    b/ lab\n   c/ site\n", "line 4: structure: c/ is indented 3, its sibling folders 4"},
		{"structure: slides/\n", "line 1: structure: takes an indented block"},
		{"structure:\n  a/b/ deck\n", "one folder per line"},
	} {
		c := Parse(strings.NewReader(tc.conf))
		if !strings.Contains(strings.Join(c.Warnings, "\n"), tc.want) {
			t.Errorf("%q: warnings %v, want %q", tc.conf, c.Warnings, tc.want)
		}
	}
}

func TestNoStructure(t *testing.T) {
	if c := Parse(strings.NewReader("theme: studio\n")); c.Structure != nil {
		t.Error("no block, no structure")
	}
}
