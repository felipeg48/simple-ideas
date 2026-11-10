package webdemo

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/assets"
	"github.com/felipeg48/simple-ideas/internal/render"
)

//go:embed termpage.js
var termPageJS string

// term is one terminal of a demo: a real shell in the script's folder under
// a pseudo-terminal (pipes on Windows), shown by a page of its own.
type term struct {
	name    string
	opts    map[string]string
	rw      io.ReadWriteCloser
	cmd     *exec.Cmd
	pipes   bool // no pseudo-terminal: si echoes what it types and asks for the exit status itself
	out     *output
	done    chan struct{}
	runAt   int    // the plain text's length when the last run began
	ran     bool   // a run has begun
	runs    int    // commands typed
	base    int    // marks seen when the first command was typed (idle prompts: the first, redraws on resize)
	id      string // on the terminal server
	session string // the page's DevTools session
	once    sync.Once
}

// shellRC writes si's prompt for sh, bash and zsh into dir: the folder's
// name and $, the last exit status in an invisible OSC 133;D mark (how
// `run … · wait` knows a command ended and how), job control off (a
// program shares the shell's process group, so closing the terminal stops
// it), and no ! history expansion (a command types as written). The user's
// own rc files are not read: the prompt looks the same on every machine.
func shellRC(dir string) error {
	files := map[string]string{
		".zshrc": "unsetopt PROMPT_SP MONITOR BEEP BANG_HIST\nHISTFILE=\nPROMPT=$'%{\\e]133;D;%?\\a%}%F{cyan}%1~%f $ '\n",
		"bashrc": "set +m +H\nunset HISTFILE PROMPT_COMMAND\nPS1='\\[\\e]133;D;$?\\a\\]\\[\\e[36m\\]\\W\\[\\e[0m\\] \\$ '\n",
		"shrc":   "set +m\ncase $- in *H*) set +H;; esac\nPS1=\"$(printf '\\033]133;D;')\"'$?'\"$(printf '\\007')$(basename \"$PWD\") \"'$ '\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// shellCommand is the terminal's shell (shell:, else $SHELL, else sh; cmd on
// Windows) with si's prompt from rc.
func shellCommand(shell, dir, rc string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		if shell == "" {
			shell = "cmd"
		}
		c := exec.Command(shell, "/Q", "/K")
		c.Dir = dir
		return c
	}
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	if shell == "" {
		shell = "sh"
	}
	env := append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor", "BASH_SILENCE_DEPRECATION_WARNING=1")
	var c *exec.Cmd
	switch strings.TrimSuffix(filepath.Base(shell), ".exe") {
	case "zsh":
		c = exec.Command(shell, "-d", "-i")
		env = append(env, "ZDOTDIR="+rc)
	case "bash":
		c = exec.Command(shell, "--noprofile", "--rcfile", filepath.Join(rc, "bashrc"), "-i")
	default:
		c = exec.Command(shell, "-i")
		env = append(env, "ENV="+filepath.Join(rc, "shrc"))
	}
	c.Dir, c.Env = dir, env
	return c
}

// openShell starts a terminal's shell and keeps its output.
func openShell(name, shell, dir, rc string, mask func(string) string) (*term, error) {
	t := &term{name: name, cmd: shellCommand(shell, dir, rc), out: newOutput(mask, nil), done: make(chan struct{})}
	rw, pipes, err := startShell(t.cmd, 100, 30)
	if err != nil {
		return nil, fmt.Errorf("terminal %q: %v", name, err)
	}
	t.rw, t.pipes = rw, pipes
	go func() {
		io.Copy(t.out, rw) // until the shell is gone
		t.out.close()
	}()
	go func() {
		t.cmd.Wait()
		close(t.done)
	}()
	if pipes {
		t.send("echo \x1b]133;D;0\x07\r\n") // the first prompt's mark
	}
	return t, nil
}

func (t *term) send(s string) error {
	_, err := io.WriteString(t.rw, s)
	return err
}

// prompt waits for the shell's first prompt.
func (t *term) prompt(within time.Duration) error {
	end := time.Now().Add(within)
	for {
		if _, ok := t.out.code(0); ok {
			return nil
		}
		if time.Now().After(end) {
			return &failure{expected: "terminal " + strconv.Quote(t.name) + " shows its prompt", found: "no prompt within " + within.String(), output: t.out.tail(40)}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// since is the output after the command line of the last run (all of it
// before the first run).
func (t *term) since() string {
	s := t.out.text(t.runAt)
	if !t.ran {
		return s
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[i+1:]
	}
	return "" // the command line is not back yet
}

// close hangs up the shell and what runs in it (SIGHUP and SIGTERM to its
// process group), then kills what is left after stopGrace.
func (t *term) close() {
	t.once.Do(func() {
		pid := t.cmd.Process.Pid
		signalGroup(pid, true)
		select {
		case <-t.done:
		case <-time.After(stopGrace):
		}
		killGroup(pid)
		t.rw.Close()
		<-t.done
	})
}

// termServer serves the terminal pages and their output on the loopback
// address, every URL under a random key.
type termServer struct {
	base  string // http://127.0.0.1:port/<key>
	srv   *http.Server
	mu    sync.Mutex
	terms map[string]*term
	n     int
}

func newTermServer() (*termServer, error) {
	k := make([]byte, 16)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	key := hex.EncodeToString(k)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &termServer{base: "http://" + ln.Addr().String() + "/" + key, terms: map[string]*term{}}
	mux := http.NewServeMux()
	mux.Handle("GET /"+key+"/a/", http.StripPrefix("/"+key+"/a/", http.FileServer(http.FS(assets.TermFS))))
	mux.HandleFunc("GET /"+key+"/t/{id}", s.page)
	mux.HandleFunc("GET /"+key+"/o/{id}", s.stream)
	mux.HandleFunc("POST /"+key+"/s/{id}", s.size)
	s.srv = &http.Server{Handler: mux}
	go s.srv.Serve(ln)
	return s, nil
}

func (s *termServer) close() { s.srv.Close() }

func (s *termServer) add(t *term) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	t.id = strconv.Itoa(s.n)
	s.terms[t.id] = t
}

func (s *termServer) get(r *http.Request) *term {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terms[r.PathValue("id")]
}

func (s *termServer) url(t *term) string { return s.base + "/t/" + t.id }

// page is the terminal scene: the code window frame, xterm.js inside.
func (s *termServer) page(w http.ResponseWriter, r *http.Request) {
	t := s.get(r)
	if t == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, termHTML(t, s.base))
}

// termHTML is the page of t (base: the server's keyed URL).
func termHTML(t *term, base string) string {
	o := t.opts
	win := firstOf(o["window"], "terminal")
	title := firstOf(o["title"], t.name)
	cfg := map[string]any{"out": base + "/o/" + t.id, "size": base + "/s/" + t.id}
	if m := sizeRE.FindStringSubmatch(o["size"]); m != nil {
		cfg["cols"], cfg["rows"] = atoi(m[1]), atoi(m[2])
	}
	js, _ := json.Marshal(cfg)
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title>
<link rel="stylesheet" href="` + base + `/a/xterm.css">
<style>:root{` + tokens() + `}
` + assets.CodewinCSS + `
html,body{margin:0;height:100%;background:#14161a;overflow:hidden}
.si-cw{margin:0;block-size:100vh;font-size:clamp(13px,1.15vw,22px);container-type:size}
.si-cw .si-cw-stage{block-size:100%;border-radius:0}
.si-cw .si-cw-win{inline-size:100%;block-size:100%}
.si-cw[data-backdrop="none"] .si-cw-win{border-radius:0}
.si-cw .si-cw-body{flex:1;min-block-size:0;padding:.7em .4em .5em 1em;overflow:hidden;line-height:normal;background:var(--si-cw-bg)}
.si-cw .si-cw-prompt{display:none}
#si-term{block-size:100%}
.xterm .xterm-viewport{overflow-y:hidden}
</style></head><body>
<figure class="si-cw si-cw-on" data-window="` + html.EscapeString(win) + `" data-cw-theme="` + html.EscapeString(firstOf(o["theme"], "night")) + `" data-backdrop="` + html.EscapeString(firstOf(o["backdrop"], "graphite")) + `" aria-label="` + html.EscapeString(title) + `">
<div class="si-cw-stage"><div class="si-cw-win">` + render.WindowChrome(win, title) + `<div class="si-cw-body"><div id="si-term"></div></div></div></div></figure>
<script src="` + base + `/a/xterm.js"></script><script src="` + base + `/a/addon-fit.js"></script>
<script>` + termPageJS + `(` + string(js) + `);</script>
</body></html>`
}

func firstOf(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

// stream sends the terminal's output from the start, then as it comes.
func (s *termServer) stream(w http.ResponseWriter, r *http.Request) {
	t := s.get(r)
	fl, ok := w.(http.Flusher)
	if t == nil || !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	fl.Flush()
	off := 0
	for {
		b, changed, closed := t.out.from(off)
		if len(b) > 0 {
			if _, err := w.Write(b); err != nil {
				return
			}
			fl.Flush()
			off += len(b)
			continue
		}
		if closed {
			return
		}
		select {
		case <-changed:
		case <-r.Context().Done():
			return
		}
	}
}

// size is the page telling the shell its columns and rows.
func (s *termServer) size(w http.ResponseWriter, r *http.Request) {
	t := s.get(r)
	if t == nil {
		http.NotFound(w, r)
		return
	}
	cols, _ := strconv.Atoi(r.URL.Query().Get("cols"))
	rows, _ := strconv.Atoi(r.URL.Query().Get("rows"))
	if cols > 0 && rows > 0 && cols < 1000 && rows < 1000 {
		resizeShell(t.rw, cols, rows)
	}
	w.WriteHeader(http.StatusNoContent)
}
