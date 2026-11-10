package docsite

import (
	"regexp"
	"strings"
)

// tours are the picture tours of the feature pages: what you get, shown, with
// the command that makes it, before the live example. Pictures are in
// docs/site/features/img/<slug>/ (headless screenshots of the live examples).
// § stands for a backtick.
var tours = map[string]string{
	"articles": `
§§§bash
si new retry --type article     # retry.md, a starter article
si play retry.md --open         # write it with live reload
si build retry.md -o html/      # one HTML file; --pdf for print
§§§

![An article: a lede, the byline, reading time and a contents column](img/articles/top.jpg)

## Everything a long read needs

<div class="tour-grid">
<figure><img src="img/articles/map.jpg" alt="A mind map in the text" loading="lazy"><figcaption>A §§§mindmap§ fence in the text, folding and searchable</figcaption></figure>
<figure><img src="img/articles/map3d.jpg" alt="A 3D mind map in the text" loading="lazy"><figcaption>The same in 3D: §view=3d kind=bricks§</figcaption></figure>
<figure><img src="img/articles/tabs.jpg" alt="Code in tabs, Java and Python, with callouts" loading="lazy"><figcaption>Code in tabs with highlighted lines and numbered callouts; the reader's tab choice follows them</figcaption></figure>
<figure><img src="img/articles/terminal.jpg" alt="A command in a terminal window" loading="lazy"><figcaption>A command in a terminal window: §window=terminal§</figcaption></figure>
</div>

## Light, dark, and a phone

<div class="tour-grid tour-wide">
<figure><img src="img/articles/dark.jpg" alt="The article in dark mode" loading="lazy"><figcaption>Dark mode follows the reader</figcaption></figure>
<figure><img src="img/articles/phone.jpg" alt="The article on a phone" loading="lazy"><figcaption>On a phone, the contents fold away</figcaption></figure>
</div>

Publish it where your readers are, always a dry run first:

§§§bash
si publish retry.md --to devto        # also hashnode, ghost, wordpress
si publish retry.md --to linkedin --url https://you.dev/retry
§§§
`,
	"guides": `
§§§bash
si new order-guide --type guide   # a folder with _nav.md and its first pages
si serve order-guide --open       # read it as your readers will, live
si build order-guide -o html/     # one HTML file per page
§§§

![A guide: the sidebar with sections and reading progress, the page, and its contents](img/guides/top.jpg)

The menu is a Markdown list in §_nav.md§; every link and anchor is checked by
§si check§.

<div class="tour-grid">
<figure><img src="img/guides/install.jpg" alt="An install page with numbered steps and commands" loading="lazy"><figcaption>Numbered steps with commands to copy</figcaption></figure>
<figure><img src="img/guides/step.jpg" alt="A step page with Previous and Next" loading="lazy"><figcaption>Sections, breadcrumbs, Previous / Next</figcaption></figure>
<figure><img src="img/guides/adoc.jpg" alt="A page written in AsciiDoc, in dark mode" loading="lazy"><figcaption>A page written in AsciiDoc beside the Markdown ones (§reset.adoc§), in dark mode</figcaption></figure>
<figure><img src="img/guides/phone.jpg" alt="The guide on a phone" loading="lazy"><figcaption>On a phone, the menu folds into one button</figcaption></figure>
</div>
`,
	"books": `
§§§bash
si new ops-reference --type book    # a folder with a cover, a preface and a chapter
si serve ops-reference --open
si build ops-reference -o html/ --pdf
§§§

<div class="tour-grid tour-wide-even">
<figure><img src="img/books/cover.jpg" alt="The book's cover page" loading="lazy"><figcaption>A cover, with the edition and links to read</figcaption></figure>
<figure><img src="img/books/cover-dark.jpg" alt="The cover in dark mode" loading="lazy"><figcaption>The same cover in dark mode</figcaption></figure>
</div>

<div class="tour-grid tour-3">
<figure><img src="img/books/contents.jpg" alt="The table of contents with parts and chapters" loading="lazy"><figcaption>Parts and numbered chapters</figcaption></figure>
<figure><img src="img/books/chapter.jpg" alt="A chapter with numbered sections and a code listing" loading="lazy"><figcaption>Numbered sections, figures and listings, cross-references</figcaption></figure>
<figure><img src="img/books/index.jpg" alt="The index" loading="lazy"><figcaption>An index from the terms you mark</figcaption></figure>
</div>

The whole book is also one page and one PDF, for print.
`,
	"mindmaps": `
§§§bash
si new ideas --type mindmap        # ideas.md: a list that becomes a map
si play ideas.md --open
si build ideas.md --svg --png --pdf --mp4
§§§

<div class="tour-grid tour-wide-even">
<figure><img src="img/mindmaps/2d.jpg" alt="A mind map with pictures, badges, a sticky note and code" loading="lazy"><figcaption>Topics with pictures, badges, sticky notes, code and links to slides</figcaption></figure>
<figure><img src="img/mindmaps/3d.jpg" alt="The same map in 3D with linked maps as floors" loading="lazy"><figcaption>The same list in 3D: §view: 3d§; linked maps float above as floors</figcaption></figure>
</div>

<div class="tour-grid tour-3">
<figure><img src="img/mindmaps/find.jpg" alt="Searching the map for pay" loading="lazy"><figcaption><kbd>/</kbd> searches every topic and every linked map</figcaption></figure>
<figure><img src="img/mindmaps/play.jpg" alt="Play walking through the branches" loading="lazy"><figcaption><b>Play</b> walks the branches one by one, for a talk</figcaption></figure>
<figure><img src="img/mindmaps/3d-dark.jpg" alt="The 3D map in dark mode" loading="lazy"><figcaption>Dark mode, and three kinds of 3D: blocks, bricks, toy</figcaption></figure>
</div>

Edit it by dragging in the Studio's Map view (§si --web .§), or bring one in:
§si mindmap import notes.opml§ (also FreeMind, XMind, Mermaid).
`,
	"code": `
§§§bash
si new retry --type codeshow       # retry.md: a code show, step by step
si play retry.md --open
si code retry.md --png --gif --mp4 # each step as a picture, or the show as a video
§§§

<div class="tour-grid tour-wide-even">
<figure><img src="img/code/first.jpg" alt="The first step of a code show with one callout" loading="lazy"><figcaption>Step 1: one attempt, with a callout on the line that matters</figcaption></figure>
<figure><img src="img/code/last.jpg" alt="The last step with highlighted lines and two callouts" loading="lazy"><figcaption>Step 4: the code morphs into place; highlighted lines, two callouts</figcaption></figure>
</div>

<div class="tour-grid tour-3">
<figure><img src="img/code/light.jpg" alt="The code show in a light page" loading="lazy"><figcaption>In an article, with ← → and Play</figcaption></figure>
<figure><img src="img/code/windows.jpg" alt="A Windows-style window with a paper theme" loading="lazy"><figcaption>§window=windows theme=paper backdrop=ocean§</figcaption></figure>
<figure><img src="img/code/terminal.jpg" alt="A terminal window with a command and its output" loading="lazy"><figcaption>§window=terminal backdrop=graphite§</figcaption></figure>
</div>

On a slide, each step is a click and the window morphs between slides. Place
callouts by dragging them in the Studio's Window view.
`,
}

// tourCSS lays out a tour's pictures: a grid that stacks on a phone.
const tourCSS = `
<style>
.tour-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,18em),1fr));gap:1.2em;margin:1.2em 0}
.tour-grid.tour-3{grid-template-columns:repeat(auto-fit,minmax(min(100%,14em),1fr))}
.tour-grid.tour-wide{grid-template-columns:minmax(0,2.6fr) minmax(0,1fr);align-items:start}
.tour-grid.tour-wide-even{grid-template-columns:repeat(auto-fit,minmax(min(100%,20em),1fr))}
@media (max-width:40em){.tour-grid.tour-wide{grid-template-columns:1fr}}
.tour-grid figure{margin:0}
.tour-grid img{display:block;width:100%;height:auto;border:1px solid var(--si-line);border-radius:var(--si-radius);box-shadow:0 .4em 1.2em -.6em rgb(0 0 0 / .25)}
.tour-grid img[src*="phone"]{max-width:16em;margin-inline:auto}
.tour-grid figcaption{font-size:.85em;color:var(--si-muted);margin-top:.5em;line-height:1.45}
kbd{font-family:var(--si-font-mono);font-size:.85em;padding:.05em .4em;border:1px solid var(--si-line);border-bottom-width:2px;border-radius:.3em;background:var(--si-surface)}
</style>
`

// tour is the picture tour of a feature page ("" when it has none).
func tour(slug string) string {
	t, ok := tours[slug]
	if !ok {
		return ""
	}
	// inside a caption (HTML, so no Markdown) a §code§ span becomes <code>
	t = captionRE.ReplaceAllStringFunc(t, func(c string) string { return captionCodeRE.ReplaceAllString(c, "<code>$1</code>") })
	return strings.ReplaceAll(t, "§", "`") + tourCSS
}

var (
	captionRE     = regexp.MustCompile(`<figcaption>.*?</figcaption>`)
	captionCodeRE = regexp.MustCompile(`§+([^§]+?)§+`)
)
