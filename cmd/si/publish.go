package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/publish"
	"github.com/felipeg48/simple-ideas/internal/render"
)

// si publish: a deck to LinkedIn as a document post (its PDF), an article as
// a LinkedIn link post or to dev.to / Hashnode. Always a dry run (the exact
// requests are printed, secrets masked) unless --yes.

func runPublish(rest []string, o *options, stdout, stderr io.Writer) int {
	if len(rest) == 0 {
		return usageErr(stderr, "publish needs <file.md> --to linkedin|devto|hashnode|ghost|wordpress, or login|logout <service>")
	}
	switch rest[0] {
	case "login":
		if len(rest) != 2 {
			return usageErr(stderr, "publish login needs linkedin, devto, hashnode, ghost or wordpress")
		}
		return publishLogin(rest[1], o, stdout, stderr)
	case "logout":
		return publishLogout(rest[1:], stdout, stderr)
	}
	if len(rest) != 1 {
		return usageErr(stderr, "publish needs one <file.md>")
	}
	path := rest[0]
	d, err := deck.Parse(path)
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	if len(d.Problems) > 0 {
		fmt.Fprintf(stderr, "si: %s has problems; run si check %s first\n", path, path)
		return exitProblems
	}
	creds, err := publish.Load()
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	var dry io.Writer = stdout
	if o.yes {
		dry = nil
	} else {
		fmt.Fprintln(stdout, "Dry run: nothing is sent. These are the requests si would make (add --yes to send them):")
		fmt.Fprintln(stdout)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	canonical := firstNonBlank(o.canonical, o.url, d.Meta.Canonical)
	switch o.to {
	case "linkedin":
		if o.copy {
			return copyRich(d, stdout, stderr)
		}
		if o.yes && creds.LinkedIn.Token == "" {
			return needLogin(stderr, "linkedin")
		}
		if o.yes && creds.LinkedIn.Expires > 0 && time.Now().Unix() > creds.LinkedIn.Expires {
			fmt.Fprintln(stderr, "si: the LinkedIn login expired (they last 60 days): run si publish login linkedin")
			return exitProblems
		}
		l := publish.NewLinkedIn(firstNonBlank(creds.LinkedIn.Token, "<not logged in>"), firstNonBlank(creds.LinkedIn.Person, "urn:li:person:<you>"))
		p := publish.Post{Text: postText(d)}
		if d.IsDoc() {
			if canonical == "" {
				return usageErr(stderr, "an article goes to LinkedIn as a link post: give its public address with --url (or canonical: in the front matter)")
			}
			p.URL, p.LinkTitle, p.LinkDesc = canonical, d.Title(), d.Meta.Description
			if f := coverFile(d); f != "" {
				if raw, err := os.ReadFile(f); err == nil {
					p.Thumb, p.ThumbName, p.ThumbType = raw, filepath.Base(f), mime.TypeByExtension(strings.ToLower(filepath.Ext(f)))
					fmt.Fprintf(stdout, "(picture: %s)\n\n", f)
				} else {
					fmt.Fprintf(stderr, "si: cover %s: %v\n", f, err)
				}
			}
		} else {
			p.DocTitle = d.Title() + ".pdf"
			if o.yes {
				pdf, err := deckPDF(path, d.Name, stderr)
				if err != nil {
					fmt.Fprintf(stderr, "si: %v\n", err)
					return exitProblems
				}
				p.PDF = pdf
			} else {
				fmt.Fprintf(stdout, "(with --yes, si first renders %s.pdf with headless Chrome)\n\n", d.Name)
			}
		}
		id, err := l.Publish(ctx, p, dry)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		if o.yes {
			fmt.Fprintf(stdout, "posted to LinkedIn: https://www.linkedin.com/feed/update/%s/\n", id)
		}
	case "devto", "hashnode", "ghost", "wordpress":
		if !d.IsDoc() {
			return usageErr(stderr, o.to+" takes articles (type: article, guide, page…); a deck goes to LinkedIn")
		}
		a, warns := articleFor(d, o.baseURL, canonical)
		a.Public = o.public
		for _, w := range warns {
			fmt.Fprintln(stderr, "si: warning:", w)
		}
		var h bytes.Buffer
		if err := render.NewParser().Convert([]byte(a.Body), &h); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		a.HTML = h.String()
		var u string
		switch o.to {
		case "ghost":
			if o.yes && (creds.Ghost.Key == "" || creds.Ghost.Site == "") {
				return needLogin(stderr, "ghost")
			}
			u, err = (&publish.Ghost{Site: firstNonBlank(creds.Ghost.Site, "https://your-site.example"), Key: firstNonBlank(creds.Ghost.Key, "<not logged in>")}).Publish(ctx, a, dry)
		case "wordpress":
			if o.yes && (creds.WordPress.Password == "" || creds.WordPress.Site == "") {
				return needLogin(stderr, "wordpress")
			}
			if len(a.Tags) > 0 {
				fmt.Fprintln(stderr, "si: note: WordPress tags are not sent (it wants tag ids); add them in WordPress")
			}
			u, err = (&publish.WordPress{Site: firstNonBlank(creds.WordPress.Site, "https://your-site.example"), User: firstNonBlank(creds.WordPress.User, "<user>"), Password: firstNonBlank(creds.WordPress.Password, "<not logged in>")}).Publish(ctx, a, dry)
		case "devto":
			if o.yes && creds.DevTo.Key == "" {
				return needLogin(stderr, "devto")
			}
			u, err = publish.NewDevTo(firstNonBlank(creds.DevTo.Key, "<not logged in>")).Publish(ctx, a, dry)
		default:
			if o.yes && (creds.Hashnode.Token == "" || creds.Hashnode.Host == "") {
				return needLogin(stderr, "hashnode")
			}
			h := publish.NewHashnode(firstNonBlank(creds.Hashnode.Token, "<not logged in>"))
			pub := creds.Hashnode.Publication
			if pub == "" {
				pub, err = h.Publication(ctx, firstNonBlank(creds.Hashnode.Host, "<your blog host>"), dry)
			}
			if err == nil {
				u, err = h.Publish(ctx, pub, a, dry)
			}
		}
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		if o.yes {
			fmt.Fprintf(stdout, "published: %s\n", u)
		}
	case "":
		return usageErr(stderr, "publish needs --to linkedin, devto, hashnode, ghost or wordpress")
	default:
		return usageErr(stderr, fmt.Sprintf("unknown --to %q (use linkedin, devto, hashnode, ghost or wordpress)", o.to))
	}
	return exitOK
}

func needLogin(stderr io.Writer, svc string) int {
	fmt.Fprintf(stderr, "si: not logged in to %s: run si publish login %s\n", svc, svc)
	return exitProblems
}

func firstNonBlank(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// postText is the LinkedIn text: linkedin:, then description:, then the title.
func postText(d *deck.Deck) string {
	return firstNonBlank(d.Meta.LinkedIn, d.Meta.Description, d.Title())
}

// deckPDF renders the deck to PDF through si build --pdf (headless Chrome).
func deckPDF(path, name string, stderr io.Writer) ([]byte, error) {
	tmp, err := os.MkdirTemp("", "si-publish-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	if code := run([]string{"build", path, "--pdf", "--output", tmp}, io.Discard, stderr); code != exitOK {
		return nil, fmt.Errorf("could not build %s", path)
	}
	pdf, err := os.ReadFile(filepath.Join(tmp, "pdfs", name+".pdf"))
	if err != nil {
		return nil, fmt.Errorf("no PDF (si build --pdf needs Google Chrome or Chromium): %v", err)
	}
	return pdf, nil
}

var (
	mdURLRE  = regexp.MustCompile(`(!?\[[^\]]*\]\()([^)\s]+)`)
	schemeRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// articleFor is the Markdown sent to dev.to / Hashnode: the body with its
// includes expanded, the title heading dropped, relative links and images
// made absolute against base (warnings when there is no base).
// coverFile is the picture of a LinkedIn link post: cover: in the front
// matter, else the article's first local picture (PNG, JPEG or GIF; LinkedIn
// takes no others). "" for none.
func coverFile(d *deck.Deck) string {
	ok := func(f string) bool {
		switch strings.ToLower(filepath.Ext(f)) {
		case ".png", ".jpg", ".jpeg", ".gif":
			return true
		}
		return false
	}
	if c := strings.TrimSpace(d.Meta.Cover); c != "" {
		if !filepath.IsAbs(c) {
			c = filepath.Join(filepath.Dir(d.Path), c)
		}
		return c
	}
	for _, s := range d.Slides {
		for _, g := range mdURLRE.FindAllStringSubmatch(s.Body, -1) {
			if strings.HasPrefix(g[1], "!") && !schemeRE.MatchString(g[2]) && ok(g[2]) {
				return filepath.Join(s.Dir, filepath.FromSlash(g[2]))
			}
		}
	}
	return ""
}

func articleFor(d *deck.Deck, base, canonical string) (publish.Article, []string) {
	var parts []string
	for _, s := range d.Slides {
		parts = append(parts, s.Body)
	}
	body := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if first, rest, ok := strings.Cut(body, "\n"); ok && strings.TrimSpace(strings.TrimPrefix(first, "# ")) == strings.TrimSpace(d.Title()) && strings.HasPrefix(first, "# ") {
		body = strings.TrimSpace(rest)
	}
	var warns []string
	body = mdURLRE.ReplaceAllStringFunc(body, func(m string) string {
		g := mdURLRE.FindStringSubmatch(m)
		target := g[2]
		if schemeRE.MatchString(target) || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "/") {
			return m
		}
		if base == "" {
			if strings.HasPrefix(g[1], "!") {
				warns = append(warns, fmt.Sprintf("image %s is relative: give --base-url (where the article's files are online) or it will not show", target))
			}
			return m
		}
		if strings.HasSuffix(strings.ToLower(target), ".md") {
			target = strings.TrimSuffix(target, filepath.Ext(target)) + ".html"
		}
		return g[1] + strings.TrimRight(base, "/") + "/" + strings.TrimPrefix(target, "./")
	})
	var tags []string
	for _, t := range strings.Split(d.Meta.Tags, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return publish.Article{Title: d.Title(), Body: body, Description: d.Meta.Description, Canonical: canonical, Tags: tags}, warns
}

// copyRich puts the article (or a deck's post text) on the clipboard as rich
// text, for LinkedIn's article editor, which has no API (macOS: textutil + pbcopy).
func copyRich(d *deck.Deck, stdout, stderr io.Writer) int {
	pb, err1 := exec.LookPath("pbcopy")
	tu, err2 := exec.LookPath("textutil")
	if err1 != nil || err2 != nil {
		fmt.Fprintln(stderr, "si: --copy needs macOS (pbcopy and textutil)")
		return exitProblems
	}
	src := postText(d)
	if d.IsDoc() {
		a, _ := articleFor(d, "", "")
		src = "# " + a.Title + "\n\n" + a.Body
	}
	var h bytes.Buffer
	if err := render.NewParser().Convert([]byte(src), &h); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	conv := exec.Command(tu, "-stdin", "-stdout", "-format", "html", "-convert", "rtf", "-inputencoding", "UTF-8")
	conv.Stdin = strings.NewReader("<meta charset=\"utf-8\">" + h.String())
	rtf, err := conv.Output()
	if err != nil {
		fmt.Fprintf(stderr, "si: textutil: %v\n", err)
		return exitProblems
	}
	cp := exec.Command(pb, "-Prefer", "rtf")
	cp.Stdin = bytes.NewReader(rtf)
	if err := cp.Run(); err != nil {
		fmt.Fprintf(stderr, "si: pbcopy: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "copied %q as rich text: paste it into LinkedIn's article editor (linkedin.com/article/new)\n", d.Title())
	return exitOK
}

func publishLogin(svc string, o *options, stdout, stderr io.Writer) int {
	c, err := publish.Load()
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	switch svc {
	case "linkedin":
		l := publish.NewLinkedIn(o.token, "")
		var expires int64
		if o.token == "" {
			id, secret := firstNonBlank(o.clientID, os.Getenv("SI_LINKEDIN_CLIENT_ID")), firstNonBlank(o.clientSecret, os.Getenv("SI_LINKEDIN_CLIENT_SECRET"))
			if id == "" || secret == "" {
				fmt.Fprintln(stderr, "si: LinkedIn login needs your app's --client-id and --client-secret (or SI_LINKEDIN_CLIENT_ID / SI_LINKEDIN_CLIENT_SECRET),\n    or a ready token: --token <token> from linkedin.com/developers/tools/oauth/token-generator. See docs/publish.md.")
				return exitUsage
			}
			tok, exp, err := linkedinOAuth(ctx, l, id, secret, o.port, !o.noOpen, stdout)
			if err != nil {
				fmt.Fprintf(stderr, "si: %v\n", err)
				return exitProblems
			}
			l.Token, expires = tok, exp
		}
		person, err := l.Me(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitProblems
		}
		c.LinkedIn.Token, c.LinkedIn.Person = l.Token, person
		c.LinkedIn.Expires = 0
		if expires > 0 {
			c.LinkedIn.Expires = time.Now().Unix() + expires
		}
	case "devto":
		key := firstNonBlank(o.token, os.Getenv("SI_DEVTO_KEY"))
		if key == "" {
			fmt.Fprintln(stderr, "si: dev.to login needs --token <API key> (dev.to/settings/extensions)")
			return exitUsage
		}
		c.DevTo.Key = key
	case "hashnode":
		if o.token == "" || o.url == "" {
			fmt.Fprintln(stderr, "si: Hashnode login needs --token <personal access token> (hashnode.com/settings/developer) and --url <your blog address>")
			return exitUsage
		}
		host := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(o.url, "https://"), "http://"), "/")
		pub, err := publish.NewHashnode(o.token).Publication(ctx, host, nil)
		if err != nil {
			fmt.Fprintf(stderr, "si: %v (the Hashnode API needs a Pro plan on the blog)\n", err)
			return exitProblems
		}
		c.Hashnode.Token, c.Hashnode.Host, c.Hashnode.Publication = o.token, host, pub
	case "ghost":
		if o.token == "" || o.url == "" {
			fmt.Fprintln(stderr, "si: Ghost login needs --url <your site> and --token <Admin API key> (Settings › Integrations › Add custom integration)")
			return exitUsage
		}
		if err := publish.CheckGhostKey(o.token); err != nil {
			fmt.Fprintf(stderr, "si: %v\n", err)
			return exitUsage
		}
		c.Ghost.Site, c.Ghost.Key = strings.TrimRight(o.url, "/"), o.token
	case "wordpress":
		if o.token == "" || o.url == "" || o.user == "" {
			fmt.Fprintln(stderr, "si: WordPress login needs --url <your site>, --user <your user name> and --token <application password> (Users › Profile › Application Passwords)")
			return exitUsage
		}
		c.WordPress.Site, c.WordPress.User, c.WordPress.Password = strings.TrimRight(o.url, "/"), o.user, o.token
	default:
		return usageErr(stderr, fmt.Sprintf("unknown service %q (use linkedin, devto, hashnode, ghost or wordpress)", svc))
	}
	if err := publish.Save(c); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	p, _ := publish.Path()
	fmt.Fprintf(stdout, "logged in to %s; the login is in %s (only you can read it)\n", svc, p)
	return exitOK
}

// linkedinOAuth runs the 3-legged flow with a loopback redirect
// (http://localhost:<port>/callback, which the LinkedIn app must list).
func linkedinOAuth(ctx context.Context, l *publish.LinkedIn, clientID, secret string, port int, open bool, stdout io.Writer) (string, int64, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", 0, fmt.Errorf("listen on port %d: %v (use --port)", port, err)
	}
	defer ln.Close()
	redirect := fmt.Sprintf("http://localhost:%d/callback", port)
	state := publish.NewState()
	type answer struct {
		code string
		err  error
	}
	got := make(chan answer, 1)
	mux := http.NewServeMux()
	mux.Handle("/callback", publish.Callback(state, func(code string, err error) {
		select {
		case got <- answer{code, err}:
		default:
		}
	}))
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go srv.Serve(ln)
	defer srv.Close()
	u := l.AuthURL(clientID, redirect, state)
	fmt.Fprintf(stdout, "Open this page to let si post for you (it waits for the answer):\n  %s\n", u)
	if open {
		if p, err := exec.LookPath("open"); err == nil {
			exec.Command(p, u).Start()
		}
	}
	select {
	case a := <-got:
		if a.err != nil {
			return "", 0, a.err
		}
		return l.Exchange(ctx, clientID, secret, a.code, redirect)
	case <-ctx.Done():
		return "", 0, fmt.Errorf("no answer from LinkedIn in time")
	}
}

func publishLogout(rest []string, stdout, stderr io.Writer) int {
	c, err := publish.Load()
	if err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	svcs := rest
	if len(svcs) == 0 {
		svcs = []string{"linkedin", "devto", "hashnode", "ghost", "wordpress"}
	}
	for _, s := range svcs {
		switch s {
		case "linkedin":
			c.LinkedIn = publish.Creds{}.LinkedIn
		case "devto":
			c.DevTo = publish.Creds{}.DevTo
		case "hashnode":
			c.Hashnode = publish.Creds{}.Hashnode
		case "ghost":
			c.Ghost = publish.Creds{}.Ghost
		case "wordpress":
			c.WordPress = publish.Creds{}.WordPress
		default:
			return usageErr(stderr, fmt.Sprintf("unknown service %q", s))
		}
	}
	if err := publish.Save(c); err != nil {
		fmt.Fprintf(stderr, "si: %v\n", err)
		return exitProblems
	}
	fmt.Fprintf(stdout, "logged out of %s\n", strings.Join(svcs, ", "))
	return exitOK
}
