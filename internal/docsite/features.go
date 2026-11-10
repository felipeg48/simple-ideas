package docsite

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/felipeg48/simple-ideas/examples"
)

// A feature page shows one kind of si document: what it is for, the example
// running live, the Markdown behind it (read from examples/, so the page
// cannot drift from the files si ships), its key options, the commands and
// the rules pages that cover it.
type feature struct {
	Slug, Title, Lede, What string
	Live                    func() string // the live part, relative to features/
	Open                    string        // the whole example, built in the site
	Source                  string        // the example file whose Markdown is shown
	Lines                   int           // how much of it (0: all)
	Options                 [][2]string
	Commands                string
	More                    string   // more sections, Markdown, after the options
	Rules                   []string // RULES.md section numbers
}

// Features are the main kinds of document, in the order the site lists them.
var Features = []feature{
	{Slug: "decks", Title: "Decks",
		Lede: "Talks that fill any screen: build steps, animated canvases, a teleprompter that scrolls with you and a remote on your phone.",
		What: "A deck is a Markdown file cut into slides by `---` lines. Every slide reflows to the screen it is on, from a phone to a 4K projector; one SVG canvas can change state slide after slide, and the teleprompter reads you the `Notes:` of each slide at your pace.",
		Live: func() string { return "<!-- embed: ../live/showcase.md 1-4 -->" }, Open: "../live/showcase.md",
		Source: "showcase.md", Lines: 46,
		Options: [][2]string{
			{"`<!-- layout: two-col -->`", "center, two-col, split, quote, big-number, image"},
			{"`<!-- step -->`", "reveal the next block on a click"},
			{"`<!-- svg: journey.svg +paid -->`", "a canvas whose parts change state slide by slide"},
			{"`Notes: (2 min)`", "the teleprompter script, with `[CLICK]` cues"},
			{"`transition:`", "fade, slide, zoom, none"},
			{"`<!-- poll: id · a · b -->`", "a live poll the room answers on their phones ([your audience](audience.md))"},
		},
		Commands: "si new talk                      # a starter deck\nsi play talk.md --open           # present, live reload, p for the teleprompter\nsi play talk.md --remote --https # teleprompter, draw pad and clicker on your phone (features/remote)\nsi play talk.md --audience       # the room follows, votes and asks (features/audience)\nsi build talk.md --pdf --mp4     # one HTML file, plus a PDF and a video\nsi build talk.md --handout       # a handout PDF: slides only, or =lines, =points, =notes\nsi build talk.md --narration latest # the talk as video, with your recorded voice\nsi rehearse talk.md              # planned vs spoken per slide",
		More:     strings.ReplaceAll(decksMore, "§", "`"),
		Rules:    []string{"1", "3", "5", "9", "10", "11", "14", "16"}},
	{Slug: "articles", Title: "Articles",
		Lede: "One readable page: a header, a table of contents, code with copy buttons, callouts, mind maps and code windows in the text.",
		What: "An article is a Markdown file with `type: article`. It reads like a blog post, prints well, publishes to LinkedIn, dev.to or Hashnode with `si publish`, and can hold anything a slide can: canvases, mind maps, code shows.",
		Live: func() string { return "<!-- embed: ../live/article/retry.md full -->" }, Open: "../live/article/retry.md",
		Source: "article/retry.md", Lines: 40,
		Options: [][2]string{
			{"`type: article`", "a page with a header, a lede and a table of contents"},
			{"`description:`", "the lede under the title, and the search snippet"},
			{"`> [!TIP]`", "callouts: note, tip, important, warning, caution"},
			{"`cover:` `tags:`", "the picture and tags `si publish` sends"},
		},
		Commands: "si new post --type article\nsi play post.md --open\nsi build post.md\nsi publish post.md --to devto     # a dry run until you add --yes",
		Rules:    []string{"19", "13", "26"}},
	{Slug: "guides", Title: "Tutorials and guides",
		Lede: "A folder of pages with a sidebar, steps you can tick, a runbook of commands, and AsciiDoc pages next to Markdown ones.",
		What: "A guide is a folder with a `_nav.md`: its list is the sidebar, every page gets a breadcrumb and a pager, and a `## Step 3:` heading becomes a numbered step. Pages can be Markdown or AsciiDoc (`.adoc`); both feed the same renderer.",
		Live: func() string { return "<!-- embed: ../live/guide/ -->" }, Open: "../live/guide/",
		Source: "guide/_nav.md",
		Options: [][2]string{
			{"`_nav.md`", "the sidebar: a nested list of pages and sections"},
			{"`## Step 1: …`", "numbered steps with a badge"},
			{"`_header.md` `_footer.md`", "shared header and footer"},
			{"`.adoc` pages", "AsciiDoc pages in the same guide"},
		},
		Commands: "si new tutorial --type guide\nsi play tutorial/ --open\nsi build tutorial/",
		Rules:    []string{"20", "18", "12"}},
	{Slug: "mindmaps", Title: "Mind maps",
		Lede: "A nested list becomes a map you can fold, search and fly into: 2D or 3D, with linked maps one click away.",
		What: "A mind map is a Markdown list under a `# Title`, as a page (`type: mindmap`) or a ```` ```mindmap ```` fence anywhere. A topic written `→ payments.md` opens that map with a zoom; `view: 3d` builds the same list in blocks, bricks or toy shapes.",
		Live: func() string {
			return mapFence("mindmap/warehouse.md", "") + "\n\n" + mapFence("mindmap/thursday.md", "") + "\n\n" + mapFence("mindmap/payments.md", "")
		},
		Open:   "../live/mindmap/thursday.md",
		Source: "mindmap/thursday.md",
		Options: [][2]string{
			{"`layout:`", "map, right, org, radial"},
			{"`lines:`", "curved, straight, orthogonal, tapered"},
			{"`view: 3d` `kind:`", "blocks, bricks or toy, iso or perspective camera"},
			{"`→ other.md`", "a linked map: drill-down, inlined in the page"},
			{"`[!sticky]` ```` ```go ````", "sticky notes and code topics"},
		},
		Commands: "si new ideas --type mindmap\nsi --web .                       # the Studio's Map view: Tab, Enter, drag\nsi build ideas.md --svg --png --pdf --mp4\nsi mindmap import notes.opml     # also FreeMind, XMind, Mermaid",
		Rules:    []string{"24"}},
	{Slug: "books", Title: "Books",
		Lede: "A reference with chapters, figures and tables numbered for you, an index, a glossary and one printable volume.",
		What: "A book is a folder whose `_nav.md` says `type: book`: chapters become numbered, figures and tables get numbers you can cross-reference, `((term))` builds the index, and the whole book prints as one volume.",
		Live: func() string { return "<!-- embed: ../live/book/ -->" }, Open: "../live/book/",
		Source: "book/01-journey.md",
		Options: [][2]string{
			{"`type: book`", "in `_nav.md`: chapters, numbering, a cover"},
			{"`((term))`", "an index entry"},
			{"`[](#fig-…)`", "a numbered cross-reference"},
			{"`a-glossary.md`", "appendices with letters"},
		},
		Commands: "si new manual --type book\nsi play manual/ --open\nsi build manual/ --pdf",
		Rules:    []string{"21"}},
	{Slug: "code", Title: "Code windows and shows",
		Lede: "Code in a macOS, Windows or terminal window, with callouts you place by hand and lines that glide from one step to the next.",
		What: "Any code fence becomes a window with `window=mac`. Fences that share an `id=` become one show: the lines that stay glide to their new place, new ones fade in. On a slide each step is a click; in a page it gets ← → and Play.",
		Live: func() string { return "<!-- embed: ../live/code/retry.md full -->" }, Open: "../live/code/retry.md",
		Source: "code/retry.md", Lines: 60,
		Options: [][2]string{
			{"`window=`", "mac, windows, terminal, none"},
			{"`theme=` `backdrop=`", "night, paper, dusk · dusk, ocean, mint, graphite"},
			{"`numbers` `hl=4,9`", "line numbers, highlighted lines"},
			{"`id=retry` `transition=`", "a show: morph, fade, slide"},
			{"`<!-- callout: 9 text at=60,30 -->`", "a numbered note placed by hand"},
		},
		Commands: "si --web .                       # the Studio's Window view\nsi code retry.md --png --scale 2\nsi code retry.md --gif --mp4",
		Rules:    []string{"25", "13"}},
}

// writeFeatures writes features/<slug>.md for every feature.
func writeFeatures(write func(rel, text string) error, refs map[string]Section) error {
	for _, f := range Features {
		src, err := fs.ReadFile(examples.FS, f.Source)
		if err != nil {
			return fmt.Errorf("feature %s: %w", f.Slug, err)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "---\ntitle: %s\ndescription: %s\n---\n\n# %s\n\n%s\n\n", f.Title, f.Lede, f.Title, f.What)
		fmt.Fprintf(&b, "## Live\n\n%s\n\n[Open the whole example](%s)\n\n", f.Live(), f.Open)
		fmt.Fprintf(&b, "## The Markdown behind it\n\n`examples/%s`%s:\n\n%s\n\n", f.Source, partOf(f.Lines, string(src)), withAdoc(fence("markdown", excerpt(string(src), f.Lines)), nil))
		b.WriteString("## Key options\n\n| Write | What it does |\n|---|---|\n")
		for _, o := range f.Options {
			fmt.Fprintf(&b, "| %s | %s |\n", o[0], o[1])
		}
		if f.More != "" {
			b.WriteString("\n" + f.More)
		}
		fmt.Fprintf(&b, "\n## Commands\n\n%s\n\n## The rules\n\n", fence("bash", f.Commands))
		for _, n := range f.Rules {
			if s, ok := refs[n]; ok {
				fmt.Fprintf(&b, "- [%s. %s](../rules/%s.md)\n", s.N, s.Title, s.Slug)
			}
		}
		if err := write("features/"+f.Slug+".md", b.String()); err != nil {
			return err
		}
	}
	return nil
}

func partOf(lines int, src string) string {
	if lines > 0 && strings.Count(src, "\n") > lines {
		return fmt.Sprintf(", its first %d lines", lines)
	}
	return ""
}

func excerpt(src string, lines int) string {
	ls := strings.Split(strings.TrimRight(src, "\n"), "\n")
	if lines > 0 && len(ls) > lines {
		ls = append(ls[:lines], "…")
	}
	return strings.Join(ls, "\n")
}

// fence wraps text in a code fence longer than any backtick run inside it.
func fence(lang, text string) string {
	n := 3
	for _, m := range regexp.MustCompile("`+").FindAllString(text, -1) {
		n = max(n, len(m)+1)
	}
	t := strings.Repeat("`", n)
	return t + lang + "\n" + text + "\n" + t
}

var (
	fmKeyRE    = regexp.MustCompile(`(?m)^(layout|lines|nodes|map-theme|view|camera|floors|kind|flow):\s*(\S+)\s*$`)
	mapLinkRE  = regexp.MustCompile(`→\s*(slides:\s*)?([^\s#{][^\s{]*)`)
	mapImageRE = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
)

// mapFence turns an example mind map page into a ```mindmap fence for a
// feature page: its front matter options on the fence line, its relative
// links and pictures rebased from examples/<dir> to features/.
func mapFence(file, extra string) string {
	raw, _ := fs.ReadFile(examples.FS, file)
	text := string(raw)
	fm, body := "", text
	if strings.HasPrefix(text, "---\n") {
		if end := strings.Index(text[4:], "\n---\n"); end >= 0 {
			fm, body = text[4:4+end], text[4+end+5:]
		}
	}
	var opts []string
	for _, m := range fmKeyRE.FindAllStringSubmatch(fm, -1) {
		k := m[1]
		if k == "map-theme" {
			k = "theme"
		}
		opts = append(opts, k+"="+m[2])
	}
	if extra != "" {
		opts = append(opts, extra)
	}
	base := "../live/" + path.Dir(file)
	rebase := func(p string) string {
		if strings.Contains(p, "://") || strings.HasPrefix(p, "#") {
			return p
		}
		return path.Join(base, p)
	}
	body = mapLinkRE.ReplaceAllStringFunc(body, func(s string) string {
		m := mapLinkRE.FindStringSubmatch(s)
		return "→ " + m[1] + rebase(m[2])
	})
	body = mapImageRE.ReplaceAllStringFunc(body, func(s string) string {
		m := mapImageRE.FindStringSubmatch(s)
		return "![" + m[1] + "](" + rebase(m[2]) + ")"
	})
	return fence("mindmap "+strings.Join(opts, " "), strings.TrimSpace(body))
}

// decksMore is the Decks page's part on presenting: keys, overview, find,
// draw.
const decksMore = `## While you present

Click the live deck above and try these; they work in every deck si builds,
from a laptop, a projector or §file://§.

| Key | What it does |
|---|---|
| → Space · ← | next step or slide · back |
| §o§ | **overview**: every slide at once, one column per section; arrows move, Enter opens |
| §/§ | **find**: search slide titles and text, or type a number to jump; it lists the other pages of the folder too |
| §a§ | **draw** on the slide; §c§ colour, §e§ erase, §z§ undo, §x§ clear, Esc stops |
| §p§ | the **teleprompter**: your §Notes:§ at your pace, timers, the next slide |
| §t§ | a **break timer** to project: pick the minutes, it counts down |
| Shift+R | in the teleprompter: **record** your voice with the slide timings (under §si play§) |
| §d§ · §f§ · §?§ | light or dark · full screen · every key |

**Draw with the trackpad.** In draw mode, hold ⌥ Option (Alt) and move: the
cursor draws where it goes, no click needed, so one finger on the trackpad
circles a number on the slide. Let go of Option and the cursor moves without
drawing. Prefer another key? §draw.hold: shift§ (or §ctrl§, §meta§, §off§) in
§si.conf§. A pen on a tablet works too, with pressure, and from your phone
the [draw pad](remote.md) puts what you draw on the projected slide.

**Overview.** §o§ shows the whole talk as a map: each §#§ section (or
included module) is a column headed by its title, so you can jump to the
Q&A backup slide or back to the agenda without paging through.

**Find.** §/§ opens a search over the deck: type a word and the matching
slides list with their text, best title matches first; type §12§ to jump to
slide 12. Below the slides it offers the other decks, labs and PDFs of the
folder, and the §links.*§ you add in §si.conf§.

**Record your talk.** Under §si play§ the teleprompter's **● Record** keeps
your voice and when each slide and step began. §si build talk.md --narration
latest§ then films the deck with your voice, every view held as long as you
spoke on it, and §si rehearse talk.md§ shows per slide the time you planned,
the time you took and your words per minute. It all stays on your computer.

**Your keys.** Every key can be changed in §si.conf§ (§keys.draw: w§), and
any feature turned off (§feature.find: off§).

`
