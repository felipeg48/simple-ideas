package studio

import (
	"net"
	"net/http"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/serve"
)

// Options configures a Studio. Addr is the real listen address (host:port).
type Options struct {
	CSS                             []string
	Theme, Conf, Title, Token, Addr string
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
	(&api{ws: ws}).routes(mux)
	mux.Handle(assetsPrefix, http.StripPrefix("/_si/studio/", http.FileServerFS(assets.StudioFS)))
	mux.HandleFunc("/_si/studio/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, assets.StudioFS, "index.html")
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/_si/studio/", http.StatusFound)
	})
	mux.Handle("/", srv.Handler())

	hosts := []string{opt.Addr}
	if _, port, err := net.SplitHostPort(opt.Addr); err == nil {
		hosts = []string{"127.0.0.1:" + port, "localhost:" + port}
	}
	return &Studio{srv: srv, h: NewGateway(opt.Token, hosts).Wrap(mux)}, nil
}

func (s *Studio) Handler() http.Handler { return s.h }

func (s *Studio) Watch(stop <-chan struct{}) { s.srv.Watch(stop) }

// Close ends any running terminal sessions.
func (s *Studio) Close() { s.srv.CloseTerminals() }
