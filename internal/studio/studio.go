package studio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/brand"
	"github.com/felipeg48/simple-ideas/internal/serve"
)

// Options configures a Studio. Addr is the real listen address (host:port).
type Options struct {
	CSS                             []string
	Theme, Conf, Title, Token, Addr string
	Version                         string // si's version, for About and Report a Problem
	// Run runs an si command line (the CLI's own entry point), for Build.
	Run func(args []string, out io.Writer) int
}

// Studio is the editor backend: API, UI and the deck server behind one gateway.
type Studio struct {
	srv *serve.Server
	h   http.Handler
}

// New opens dir as a workspace and builds the gated handler.
func New(dir string, opt Options) (*Studio, error) {
	ws, err := OpenWorkspace(dir)
	if err != nil {
		return nil, err
	}
	srv, err := serve.New(ws.Root, opt.CSS, opt.Theme, opt.Conf)
	if err != nil {
		return nil, err
	}
	srv.Dev, srv.Terminal, srv.Title = true, true, opt.Title

	mux := http.NewServeMux()
	(&api{ws: ws, run: opt.Run, css: opt.CSS, theme: opt.Theme, title: opt.Title}).routes(mux)
	mux.Handle(assetsPrefix, http.StripPrefix("/_si/studio/", http.FileServerFS(assets.StudioFS)))
	index, err := fs.ReadFile(assets.StudioFS, "index.html")
	if err != nil {
		return nil, err
	}
	// data-ws scopes the UI's localStorage keys (tabs, unsaved buffers) to
	// this folder, so two workspaces served from the same origin never mix
	sum := sha256.Sum256([]byte(ws.Root))
	index = bytes.Replace(index, []byte(`<div id="app">`),
		[]byte(`<div id="app" data-ws="`+hex.EncodeToString(sum[:6])+`">`), 1)
	index = bytes.Replace(index, []byte("</head>"), append(aboutJSON(opt.Version), "</head>"...), 1)
	mux.HandleFunc("/_si/studio/{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(index)
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/_si/studio/", http.StatusFound)
	})
	mux.Handle("/", srv.Handler())

	hosts := []string{opt.Addr}
	if _, port, err := net.SplitHostPort(opt.Addr); err == nil {
		// the listen address itself too, so --addr [::1]:PORT works
		hosts = []string{"127.0.0.1:" + port, "localhost:" + port, opt.Addr}
	}
	return &Studio{srv: srv, h: NewGateway(opt.Token, hosts).Wrap(mux)}, nil
}

func (s *Studio) Handler() http.Handler { return s.h }

func (s *Studio) Watch(stop <-chan struct{}) { s.srv.Watch(stop) }

// Close ends any running terminal sessions.
func (s *Studio) Close() { s.srv.CloseTerminals() }

// aboutJSON is what the Studio's Help menu and About dialog show, from
// internal/brand (the desktop app reaches the same through window.siStudio).
func aboutJSON(version string) []byte {
	commit, date := brand.Build()
	b, _ := json.Marshal(map[string]any{
		"name": brand.Name, "tagline": brand.Tagline, "about": brand.About, "version": version, "commit": commit, "date": date,
		"site": brand.Site, "repo": brand.Repo, "pages": brand.Pages, "os": brand.OS(), "credits": brand.Credits,
		"copyright": brand.Copyright, "license": brand.License,
		"bug": brand.IssueURL(false, version, "web Studio", brand.OS()), "bugDesktop": brand.IssueURL(false, version, "desktop app", brand.OS()),
		"feature": brand.IssueURL(true, version, "web Studio", ""), "featureDesktop": brand.IssueURL(true, version, "desktop app", ""),
	})
	return []byte(`<script id="si-about" type="application/json">` + string(bytes.ReplaceAll(b, []byte("</"), []byte(`<\/`))) + "</script>\n")
}
