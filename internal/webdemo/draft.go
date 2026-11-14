package webdemo

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/cdp"
)

// si demo draft: a first promo script from a site, read headless. Same
// origin only, robots.txt obeyed (Disallow, Allow, Crawl-delay), a page
// limit, nothing clicked or submitted: pages are only loaded and read.

//go:embed draft.js
var draftJS string

// DraftOptions are how a draft reads a site.
type DraftOptions struct {
	Chrome string
	Pages  int       // the most pages read (0: 6)
	W, H   int       // the viewport (0: 1440x900)
	Log    io.Writer // one line per page (nil: quiet)
	Client *http.Client
}

// pageInfo is what draft.js reads from one page.
type pageInfo struct {
	URL      string `json:"-"`
	Title    string
	Hero     *draftBlock
	Sections []draftBlock
	Cookie   *draftLink
	Links    []struct{ Href, Text string }
}

type draftBlock struct {
	Heading, Sel, Sentence string
	CTA                    *draftLink
}

type draftLink struct{ Text, Sel string }

// robots is what robots.txt says to every crawler (or to si).
type robots struct {
	rules []robotRule
	delay time.Duration
}

type robotRule struct {
	allow  bool
	prefix string
}

// parseRobots reads the groups for user-agent * and si (si's own wins).
func parseRobots(txt string) robots {
	groups := map[string]*robots{}
	var cur []string
	inRules := false
	sc := bufio.NewScanner(strings.NewReader(txt))
	for sc.Scan() {
		l, _, _ := strings.Cut(sc.Text(), "#")
		k, v, ok := strings.Cut(l, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		switch k {
		case "user-agent":
			if inRules {
				cur, inRules = nil, false
			}
			cur = append(cur, strings.ToLower(v))
		case "allow", "disallow", "crawl-delay":
			inRules = true
			for _, ua := range cur {
				g := groups[ua]
				if g == nil {
					g = &robots{}
					groups[ua] = g
				}
				switch {
				case k == "crawl-delay":
					if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
						g.delay = time.Duration(min(f, 10) * float64(time.Second))
					}
				case v != "":
					g.rules = append(g.rules, robotRule{k == "allow", v})
				case k == "disallow": // Disallow: (empty) allows everything
				}
			}
		}
	}
	if g := groups["si"]; g != nil {
		return *g
	}
	if g := groups["*"]; g != nil {
		return *g
	}
	return robots{}
}

// allowed: the longest matching rule wins (Allow on a tie); * and $ as in
// the robots.txt standard.
func (rb robots) allowed(p string) bool {
	best, allow := -1, true
	for _, r := range rb.rules {
		if n := robotMatch(r.prefix, p); n > best || (n == best && r.allow) {
			if n >= 0 {
				best, allow = n, r.allow
			}
		}
	}
	return allow
}

// robotMatch is the length of rule when it matches p (-1: it does not).
func robotMatch(rule, p string) int {
	end := strings.HasSuffix(rule, "$")
	parts := strings.Split(strings.TrimSuffix(rule, "$"), "*")
	if !strings.HasPrefix(p, parts[0]) {
		return -1
	}
	at := len(parts[0])
	for _, part := range parts[1:] {
		i := strings.Index(p[at:], part)
		if i < 0 {
			return -1
		}
		at += i + len(part)
	}
	if end && (at != len(p) && (len(parts) == 1 || !strings.HasSuffix(p, parts[len(parts)-1]))) {
		return -1
	}
	return len(rule)
}

// skipExt are links that are files, not pages.
var skipExt = map[string]bool{".pdf": true, ".zip": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".svg": true,
	".webp": true, ".mp4": true, ".mp3": true, ".xml": true, ".json": true, ".txt": true, ".css": true, ".js": true, ".ico": true, ".dmg": true, ".exe": true}

// Draft reads the site at start and writes a first type: demo promo script.
func Draft(start string, o DraftOptions) (string, error) {
	u, err := url.Parse(start)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("%q: give the site's address (https://example.com)", start)
	}
	u.Fragment = ""
	origin := u.Scheme + "://" + u.Host
	if o.Pages <= 0 {
		o.Pages = 6
	}
	if o.W == 0 {
		o.W, o.H = 1440, 900
	}
	if o.Log == nil {
		o.Log = io.Discard
	}
	hc := o.Client
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	var rb robots
	if resp, err := hc.Get(origin + "/robots.txt"); err == nil {
		if resp.StatusCode == 200 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
			rb = parseRobots(string(b))
		}
		resp.Body.Close()
	}
	if !rb.allowed(pathOf(u)) {
		return "", fmt.Errorf("%s: robots.txt asks crawlers to keep out", start)
	}
	c, err := cdp.Launch(o.Chrome, o.W, o.H)
	if err != nil {
		return "", err
	}
	defer c.Close()
	var ctx struct{ BrowserContextID string }
	if err := c.CallIn("", "Target.createBrowserContext", map[string]any{"disposeOnDetach": true}, &ctx); err != nil {
		return "", err
	}
	if _, err := c.OpenPage(ctx.BrowserContextID, o.W, o.H); err != nil {
		return "", err
	}
	delay := max(rb.delay, 300*time.Millisecond)
	queue, seen := []string{u.String()}, map[string]bool{pathOf(u): true}
	var pages []*pageInfo
	for len(queue) > 0 && len(pages) < o.Pages {
		next := queue[0]
		queue = queue[1:]
		if len(pages) > 0 {
			time.Sleep(delay)
		}
		p, err := readPage(c, next)
		if err != nil {
			if len(pages) == 0 {
				return "", err
			}
			fmt.Fprintf(o.Log, "  ✗ %s: %v\n", next, err)
			continue
		}
		fmt.Fprintf(o.Log, "  ✓ %s: %d section(s)\n", next, len(p.Sections))
		pages = append(pages, p)
		for _, l := range p.Links {
			lu, err := url.Parse(l.Href)
			if err != nil || lu.Scheme+"://"+lu.Host != origin || lu.RawQuery != "" || skipExt[strings.ToLower(path.Ext(lu.Path))] {
				continue
			}
			k := pathOf(lu)
			if seen[k] || !rb.allowed(k) {
				continue
			}
			seen[k] = true
			lu.Fragment = ""
			queue = append(queue, lu.String())
		}
	}
	return draftScript(origin, pages), nil
}

func pathOf(u *url.URL) string {
	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}
	return strings.TrimSuffix(p, "/index.html")
}

// readPage loads a page, lets its scripts draw, and reads it.
func readPage(c *cdp.Client, u string) (*pageInfo, error) {
	var nav struct{ ErrorText string }
	if err := c.Call("Page.navigate", map[string]any{"url": u}, &nav); err != nil {
		return nil, err
	}
	if nav.ErrorText != "" {
		return nil, fmt.Errorf("%s", nav.ErrorText)
	}
	for end := time.Now().Add(15 * time.Second); ; time.Sleep(100 * time.Millisecond) {
		v, err := c.Eval("document.readyState")
		if err == nil && string(v) == `"complete"` {
			break
		}
		if time.Now().After(end) {
			return nil, fmt.Errorf("it did not finish loading")
		}
	}
	time.Sleep(600 * time.Millisecond) // a page drawn by script after load
	v, err := c.Eval(draftJS)
	if err != nil {
		return nil, err
	}
	p := &pageInfo{URL: u}
	return p, json.Unmarshal(v, p)
}

// draftScript writes the script: a feature for the home hero, one for each
// of its sections, one per other page, a closing call to action. Every Say:
// is the page's own words, marked for the author to polish.
func draftScript(origin string, pages []*pageInfo) string {
	home := pages[0]
	site := siteName(home.Title, origin)
	var b strings.Builder
	host := strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
	var closing *draftLink
	if home.Hero != nil && home.Hero.CTA != nil {
		closing = home.Hero.CTA
	}
	fmt.Fprintf(&b, "---\ntype: demo\ntitle: %s\napp: %s\nviewport: desktop\nsession: keep\nurl: %s\n", oneLine(site), origin, origin)
	if closing != nil {
		fmt.Fprintf(&b, "outro: %s · %s\n", oneLine(closing.Text), host)
	}
	b.WriteString("---\n\n# " + oneLine(site) + "\n\n")
	fmt.Fprintf(&b, "<!-- Drafted by si demo draft from %s on %s. Every Say: line is the site's own words, cut to a sentence: polish each one (marked draft) before filming, then run si demo test. -->\n\n",
		origin, time.Now().Format("2006-01-02"))
	if home.Cookie != nil && home.Cookie.Sel != "" {
		fmt.Fprintf(&b, "## Before\n\n- go /\n- click %s\n\n", home.Cookie.Sel)
	}
	say := func(parts ...string) {
		var s []string
		for _, p := range parts {
			if p = sentenceOf(p); p != "" {
				s = append(s, p)
			}
		}
		if len(s) == 0 {
			return
		}
		b.WriteString("<!-- draft: polish this line -->\nSay: " + strings.Join(s, " ") + "\n")
	}
	ctaSteps := func(c *draftLink) {
		if c != nil && c.Sel != "" {
			fmt.Fprintf(&b, "- hover %s\n- spotlight %s · label %s\n", c.Sel, c.Sel, strconv.Quote(c.Text))
		}
	}
	block := func(bl draftBlock, goTo string, first bool) {
		fmt.Fprintf(&b, "## %s\n\n", oneLine(bl.Heading))
		if bl.Sentence != "" {
			fmt.Fprintf(&b, "What it does: %s\n\n", oneLine(bl.Sentence))
		}
		say(bl.Heading, bl.Sentence)
		if goTo != "" {
			fmt.Fprintf(&b, "- go %s\n", goTo)
		}
		if first {
			fmt.Fprintf(&b, "- spotlight %s\n", bl.Sel)
		} else {
			fmt.Fprintf(&b, "- scroll to %s\n- spotlight %s\n", bl.Sel, bl.Sel)
		}
		ctaSteps(bl.CTA)
		b.WriteString("\n")
	}
	if home.Hero != nil && home.Hero.Sel != "" {
		block(*home.Hero, "/", true)
	}
	for i, s := range home.Sections {
		if i >= 5 {
			break
		}
		goTo := ""
		if i == 0 && home.Hero == nil {
			goTo = "/"
		}
		block(s, goTo, false)
	}
	for _, p := range pages[1:] {
		pu, _ := url.Parse(p.URL)
		where := pu.EscapedPath()
		if where == "" {
			where = "/"
		}
		if p.Hero != nil && p.Hero.Sel != "" {
			block(*p.Hero, where, true)
		} else if len(p.Sections) > 0 {
			block(p.Sections[0], where, false)
		}
	}
	b.WriteString("## " + oneLine(firstNonBlank(textOf(closing), "Get started")) + "\n\n")
	if closing != nil {
		b.WriteString("<!-- draft: polish this line -->\nSay: " + sentenceOf(closing.Text) + " " + host + ".\n- go /\n")
		ctaSteps(closing)
	} else {
		b.WriteString("<!-- draft: polish this line -->\nSay: " + host + ".\n- go /\n")
	}
	return b.String()
}

func textOf(l *draftLink) string {
	if l == nil {
		return ""
	}
	return l.Text
}

func firstNonBlank(a ...string) string {
	for _, s := range a {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// siteName is the site's name from its title ("Acme | Build faster" →
// "Acme"), else its host.
func siteName(title, origin string) string {
	for _, sep := range []string{" | ", " — ", " – ", " - ", " · ", ": "} {
		if a, _, ok := strings.Cut(title, sep); ok && strings.TrimSpace(a) != "" {
			return strings.TrimSpace(a)
		}
	}
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	return strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://")
}

// sentenceOf ends a text as a sentence (a heading gets its full stop).
func sentenceOf(s string) string {
	s = oneLine(s)
	if s == "" {
		return ""
	}
	if !strings.ContainsAny(s[len(s)-1:], ".!?…") {
		s += "."
	}
	return s
}

// oneLine keeps a text on one line of the script.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }
