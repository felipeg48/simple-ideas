package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"image/color"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/qr"
	"github.com/felipeg48/simple-ideas/internal/sandbox"
	"github.com/felipeg48/simple-ideas/internal/serve"
	"github.com/felipeg48/simple-ideas/internal/tools"
)

// si class (RULES §37, v4.0.0): classes on the presenter's machine. A class
// is a course with a sandbox: (its own Docker box per learner), seats with
// codes, served through --tunnel or HTTPS; learners reach only their own
// box's shells. `si class` is the planner, a keyed loopback page; the
// subcommands are its twins for scripts.

const classUsage = `si class: classes on this machine, a Docker box per learner (RULES §37)

  si class [--no-open]             the planner: plan classes, seats and codes, start
                                   one, watch and reset the seats' boxes (a keyed
                                   page on this machine only)
  si class new <course> [--name N] [--seats 20 | --roster people.csv] [--image I]
               [--memory 2g] [--cpus 2] [--ttl 3h] [--idle 30m] [--expires YYYY-MM-DD]
                                   plan a class of a course folder (or a lab page)
                                   with a sandbox:; the roster is name,email[,code]
  si class list                    the classes planned here (.si/classes/)
  si class codes <id>              the seats and their codes
  si class start <id> [--tunnel | --https | --local] [--budget 48g] [--addr :8443]
                                   serve the class: a link per seat; the planner
                                   shows the boxes live; Ctrl+C removes them
  si class rm <id>                 forget a class (its codes stop working)

  --tunnel  a temporary public HTTPS address (cloudflared), for learners anywhere
  --https   this network, HTTPS with si's own certificate (a learner's device
            trusts it once at http://<address>/ca)
  --local   this machine only, to try a class yourself
  A seat's box is made when its learner first opens the course, from the
  course's sandbox: image (or --image), with its sandbox-limits: (or --memory,
  --cpus); --budget caps the memory of all boxes (default 80% of the sandbox
  host's). --trusted-sandbox lets boxes run privileged on a host without sysbox.
`

//go:embed classplanner.html
var classPlannerHTML string

type classFlags struct {
	name, roster, image, memory, ttl, idle, expires, budget, addr string
	cpus                                                          float64
	seats                                                         int
	tunnel, https, local, trusted, noOpen, help                   bool
}

func runClass(argv []string, stdout, stderr io.Writer) int {
	fl := flag.NewFlagSet("si class", flag.ContinueOnError)
	fl.SetOutput(io.Discard)
	var f classFlags
	fl.StringVar(&f.name, "name", "", "")
	fl.StringVar(&f.roster, "roster", "", "")
	fl.StringVar(&f.image, "image", "", "")
	fl.StringVar(&f.memory, "memory", "", "")
	fl.Float64Var(&f.cpus, "cpus", 0, "")
	fl.StringVar(&f.ttl, "ttl", "", "")
	fl.StringVar(&f.idle, "idle", "", "")
	fl.StringVar(&f.expires, "expires", "", "")
	fl.StringVar(&f.budget, "budget", "", "")
	fl.StringVar(&f.addr, "addr", "", "")
	fl.IntVar(&f.seats, "seats", 0, "")
	fl.BoolVar(&f.tunnel, "tunnel", false, "")
	fl.BoolVar(&f.https, "https", false, "")
	fl.BoolVar(&f.local, "local", false, "")
	fl.BoolVar(&f.trusted, "trusted-sandbox", false, "")
	fl.BoolVar(&f.noOpen, "no-open", false, "")
	fl.BoolVar(&f.help, "help", false, "")
	fl.BoolVar(&f.help, "h", false, "")
	var args []string
	for {
		if err := fl.Parse(argv); err != nil {
			fmt.Fprintf(stderr, "si class: %v\n\n%s", err, classUsage)
			return exitUsage
		}
		if argv = fl.Args(); len(argv) == 0 {
			break
		}
		args, argv = append(args, argv[0]), argv[1:]
	}
	if f.help {
		fmt.Fprint(stdout, classUsage)
		return exitOK
	}
	fail := func(err error) int {
		if err == nil {
			return exitOK
		}
		fmt.Fprintf(stderr, "si class: %v\n", err)
		return exitProblems
	}
	root, _ := os.Getwd()
	if len(args) == 0 {
		return fail(runPlanner(root, "", f, stdout, stderr))
	}
	switch args[0] {
	case "new":
		if len(args) != 2 {
			return usageErr(stderr, "si class new <course> [--seats N | --roster file.csv]")
		}
		var roster []byte
		if f.roster != "" {
			b, err := os.ReadFile(f.roster)
			if err != nil {
				return fail(err)
			}
			roster = b
		}
		c, err := newClass(root, classForm{Name: f.name, Course: args[1], Image: f.image, Memory: f.memory, CPUs: f.cpus,
			TTL: f.ttl, Idle: f.idle, Expires: f.expires, Seats: f.seats, Roster: string(roster)})
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "si class: %s, %d seats, in %s\n", c.ID, len(c.Seats), filepath.Join(".si", "classes", c.ID+".json"))
		printSeats(stdout, c, "")
		fmt.Fprintf(stdout, "  si class start %s --tunnel   (or --https on this network)\n", c.ID)
		return exitOK
	case "list", "ls":
		cs := sandbox.ListClasses(root)
		if len(cs) == 0 {
			fmt.Fprintln(stdout, "si class: no classes here yet (si class new <course>, or the planner: si class)")
			return exitOK
		}
		for _, c := range cs {
			joined := 0
			for _, s := range c.Seats {
				if !s.Joined.IsZero() {
					joined++
				}
			}
			fmt.Fprintf(stdout, "%-28s %-30s %3d seats, %d joined  %s\n", c.ID, c.Course, len(c.Seats), joined, c.Expires)
		}
		return exitOK
	case "codes":
		if len(args) != 2 {
			return usageErr(stderr, "si class codes <id>")
		}
		c, err := sandbox.LoadClass(root, args[1])
		if err != nil {
			return fail(err)
		}
		printSeats(stdout, c, "")
		return exitOK
	case "rm":
		if len(args) != 2 {
			return usageErr(stderr, "si class rm <id>")
		}
		return fail(sandbox.RemoveClass(root, args[1]))
	case "start":
		if len(args) != 2 {
			return usageErr(stderr, "si class start <id> [--tunnel | --https | --local]")
		}
		if _, err := sandbox.LoadClass(root, args[1]); err != nil {
			return fail(err)
		}
		return fail(runPlanner(root, args[1], f, stdout, stderr))
	}
	fmt.Fprintf(stderr, "si class: %q is not new, list, codes, start or rm\n\n%s", args[0], classUsage)
	return exitUsage
}

// printSeats prints a class's seats: code, name, and the link once the
// class runs.
func printSeats(w io.Writer, c *sandbox.Class, base string) {
	for i, s := range c.Seats {
		who := s.Name
		if who == "" {
			who = fmt.Sprintf("Seat %d", i+1)
		}
		if s.Email != "" {
			who += " <" + s.Email + ">"
		}
		link := ""
		if base != "" {
			link = "  " + base + "/go/" + sandbox.ShowCode(s.Code)
		}
		fmt.Fprintf(w, "  %s  %s%s\n", sandbox.ShowCode(s.Code), who, link)
	}
}

// classForm is a new class as the planner's form or si class new give it.
type classForm struct {
	Name, Course, Image, Memory, TTL, Idle, Expires, Roster string
	CPUs                                                    float64
	Seats                                                   int
}

// newClass plans a class and saves it.
func newClass(root string, f classForm) (*sandbox.Class, error) {
	course := filepath.Clean(f.Course)
	if filepath.IsAbs(course) {
		if r, err := filepath.Rel(root, course); err == nil && !strings.HasPrefix(r, "..") {
			course = r
		}
	}
	lb, err := classBox(root, course, f.Image)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(f.Name)
	if name == "" {
		name = filepath.Base(strings.TrimSuffix(course, filepath.Ext(course)))
	}
	c := &sandbox.Class{Name: name, Course: filepath.ToSlash(course), Image: strings.TrimSpace(f.Image), CPUs: f.CPUs,
		TTL: strings.TrimSpace(f.TTL), Idle: strings.TrimSpace(f.Idle), Expires: strings.TrimSpace(f.Expires), Created: time.Now()}
	if c.Image == lb.spec.Image {
		c.Image = ""
	}
	if m := strings.TrimSpace(f.Memory); m != "" {
		if c.Memory, err = deck.ParseBytes(m); err != nil || c.Memory < 64<<20 {
			return nil, fmt.Errorf("memory %q: want a size like 2g", m)
		}
	}
	base := sandbox.ClassID(name, c.Created)
	c.ID = base
	for i := 2; fileExists(filepath.Join(sandbox.ClassDir(root), c.ID+".json")); i++ {
		c.ID = fmt.Sprintf("%s-%d", base, i)
	}
	if f.Roster != "" {
		seats, err := sandbox.ReadRoster(strings.NewReader(f.Roster))
		if err != nil {
			return nil, err
		}
		if err := c.AddSeats(seats...); err != nil {
			return nil, err
		}
	}
	if f.Seats > 0 {
		if err := c.AddSeats(make([]sandbox.Seat, f.Seats)...); err != nil {
			return nil, err
		}
	}
	if len(c.Seats) == 0 {
		return nil, errors.New("a class needs seats: --seats N or --roster people.csv")
	}
	return c, sandbox.SaveClass(root, c)
}

// classBox is the box a class's course asks for (its sandbox:), the image
// replaced when the class names one.
func classBox(root, course, image string) (*labBox, error) {
	p := filepath.Join(root, filepath.FromSlash(course))
	if _, err := os.Stat(p); err != nil {
		return nil, fmt.Errorf("course %s: %v", course, err)
	}
	lb, err := findLabBox(p)
	if err != nil {
		return nil, err
	}
	if lb == nil {
		if image == "" {
			return nil, fmt.Errorf("%s has no sandbox: (in _course.md or a top-level lab); add one, or give the class an image", course)
		}
		dir := p
		if st, _ := os.Stat(p); st != nil && !st.IsDir() {
			dir = filepath.Dir(p)
		}
		lb = &labBox{spec: deck.Sandbox{Memory: deck.DefaultSandboxMemory, CPUs: deck.DefaultSandboxCPUs, Pids: deck.DefaultSandboxPids}, dir: dir, course: true}
	}
	if image != "" {
		lb.spec.Image = image
	}
	return lb, nil
}

// running is the class the planner serves now.
type running struct {
	id     string
	how    string // tunnel, https, local
	base   string // the learners' base URL
	caURL  string // --https: where a device trusts si's certificate
	room   *serve.Classroom
	srv    *serve.Server
	fleet  *sandbox.Fleet
	stop   func()
	budget int64
}

// planner is the si class page's state.
type planner struct {
	root    string
	key     string
	trusted bool
	budget  string
	addr    string
	out     io.Writer

	mu  sync.Mutex
	run *running
}

// runPlanner serves the planner on the loopback; with start, that class
// starts at once.
func runPlanner(root, start string, f classFlags, stdout, stderr io.Writer) error {
	var raw [16]byte
	rand.Read(raw[:])
	p := &planner{root: root, key: hex.EncodeToString(raw[:]), trusted: f.trusted, budget: f.budget, addr: f.addr, out: stdout}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	u := "http://" + ln.Addr().String() + "/?k=" + p.key
	hs := &http.Server{Handler: p.handler(), ReadHeaderTimeout: 10 * time.Second}
	go hs.Serve(ln)
	fmt.Fprintf(stdout, "si class: the planner at %s (this machine only; Ctrl+C stops)\n", u)
	if start != "" {
		how := "tunnel"
		switch {
		case f.https:
			how = "https"
		case f.local:
			how = "local"
		case !f.tunnel && tools.Find("cloudflared") == "":
			how = "https"
		}
		if err := p.start(start, how); err != nil {
			hs.Close()
			return err
		}
	}
	if !f.noOpen {
		openBrowser(u)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go p.reaper(ctx)
	<-ctx.Done()
	p.stopClass()
	hs.Close()
	return nil
}

// reaper removes the boxes past ttl: or idle:, and measures the boxes for
// the planner.
func (p *planner) reaper(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		p.mu.Lock()
		r := p.run
		p.mu.Unlock()
		if r == nil {
			continue
		}
		for _, code := range r.fleet.Reap(time.Now()) {
			r.room.Reset(code)
			fmt.Fprintf(p.out, "si class: removed the box of %s (ttl or idle)\n", sandbox.ShowCode(code))
		}
		mctx, c := context.WithTimeout(ctx, 10*time.Second)
		r.fleet.Measure(mctx)
		c()
	}
}

// start serves class id: its boxes on the sandbox host, the learners'
// listener on how (tunnel, https, local).
func (p *planner) start(id, how string) error {
	p.mu.Lock()
	busy := p.run != nil
	p.mu.Unlock()
	if busy {
		return errors.New("a class is running already: stop it first")
	}
	c, err := sandbox.LoadClass(p.root, id)
	if err != nil {
		return err
	}
	if c.Expired(time.Now()) {
		return fmt.Errorf("class %s ended on %s", c.ID, c.Expires)
	}
	lb, err := classBox(p.root, c.Course, c.Image)
	if err != nil {
		return err
	}
	spec := lb.spec
	if c.Memory > 0 {
		spec.Memory = c.Memory
	}
	if c.CPUs > 0 {
		spec.CPUs = c.CPUs
	}
	sock, err := sandboxSocket(lb.dir)
	if err != nil {
		return err
	}
	eng := sandbox.NewEngine(sock)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	info, err := eng.Info(ctx)
	if err != nil {
		return fmt.Errorf("the sandbox host does not answer (si sandbox host start): %v", err)
	}
	if _, ok := info.Runtimes[sandbox.SysboxRuntime]; !ok && !p.trusted {
		return sandbox.ErrNoSysbox
	}
	sandbox.Sweep(ctx, eng)
	budget := info.MemTotal * 8 / 10
	if p.budget != "" {
		if budget, err = deck.ParseBytes(p.budget); err != nil {
			return fmt.Errorf("--budget %q: want a size like 48g", p.budget)
		}
	}
	fleet := &sandbox.Fleet{Eng: eng, Spec: spec, Dir: lb.dir, Class: c.ID, Opt: sandbox.Options{Trusted: p.trusted}, Budget: budget}
	fleet.TTL, _ = time.ParseDuration(c.TTL)
	fleet.Idle, _ = time.ParseDuration(c.Idle)
	srv, err := serve.New(filepath.Join(p.root, filepath.FromSlash(c.Course)), nil, "", "")
	if err != nil {
		return err
	}
	srv.Class = true
	room := serve.NewClassroom(c, fleet)
	room.Joined = func(code string) {
		room.View(func(c *sandbox.Class) {
			if err := sandbox.SaveClass(p.root, c); err != nil {
				fmt.Fprintf(p.out, "si class: %v\n", err)
			}
			fmt.Fprintf(p.out, "si class: %s joined\n", c.Label(code))
		})
	}
	r := &running{id: c.ID, how: how, room: room, srv: srv, fleet: fleet, budget: budget}
	h := srv.ClassroomHandler(room)
	switch how {
	case "tunnel":
		base, stop, err := tunnelTo(h)
		if err != nil {
			return err
		}
		r.base, r.stop = base, stop
	case "https", "local":
		addr := p.addr
		if addr == "" {
			addr = "0.0.0.0:0"
			if how == "local" {
				addr = "127.0.0.1:0"
			}
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		ta := ln.Addr().(*net.TCPAddr)
		host := "127.0.0.1"
		if how == "https" {
			if host = lanIP(); host == "" {
				ln.Close()
				return errors.New("--https: this machine has no network address a learner could reach")
			}
		}
		hosts, tlsCfg, ca, err := httpsFor(&options{https: httpsFlag(map[bool]string{true: "local"}[how == "https"])}, []string{host})
		if err != nil {
			ln.Close()
			return err
		}
		hs := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
		serveSplit(hs, ln, tlsCfg, ca, "si class", p.out)
		hp := net.JoinHostPort(hosts[0], strconv.Itoa(ta.Port))
		r.base, r.stop = "http://"+hp, func() { hs.Close() }
		if tlsCfg != nil {
			r.base, r.caURL = "https://"+hp, "http://"+hp+"/ca"
		}
	default:
		return fmt.Errorf("share %q: want tunnel, https or local", how)
	}
	p.mu.Lock()
	p.run = r
	p.mu.Unlock()
	how2 := map[string]string{"tunnel": "a public HTTPS address (Cloudflare quick tunnel)", "https": "this network, over HTTPS", "local": "this machine only"}[how]
	fmt.Fprintf(p.out, "si class: %s runs on %s: %s/go/<code>\n", c.ID, how2, r.base)
	if r.caURL != "" {
		fmt.Fprintf(p.out, "  a learner's device trusts si's certificate once at %s\n", r.caURL)
	}
	room.View(func(c *sandbox.Class) { printSeats(p.out, c, r.base) })
	return nil
}

// stopClass stops the running class: the listener, then every box.
func (p *planner) stopClass() {
	p.mu.Lock()
	r := p.run
	p.run = nil
	p.mu.Unlock()
	if r == nil {
		return
	}
	fmt.Fprintf(p.out, "si class: stopping %s, removing its boxes\n", r.id)
	r.stop()
	r.room.Close()
	r.srv.CloseTerminals()
}

// handler is the planner: its page and JSON calls, keyed and loopback only.
func (p *planner) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", p.page)
	mux.HandleFunc("GET /api/state", p.state)
	mux.HandleFunc("GET /api/images", p.images)
	mux.HandleFunc("POST /api/pull", p.pull)
	mux.HandleFunc("POST /api/class", p.create)
	mux.HandleFunc("POST /api/class/{id}/seats", p.addSeats)
	mux.HandleFunc("POST /api/class/{id}/seat/{code}/remove", p.removeSeat)
	mux.HandleFunc("POST /api/class/{id}/delete", p.remove)
	mux.HandleFunc("POST /api/class/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Share string }
		json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req)
		p.reply(w, p.start(r.PathValue("id"), req.Share))
	})
	mux.HandleFunc("POST /api/stop", func(w http.ResponseWriter, r *http.Request) { p.stopClass(); p.reply(w, nil) })
	mux.HandleFunc("POST /api/seat/{code}/reset", func(w http.ResponseWriter, r *http.Request) {
		if run := p.current(); run != nil {
			run.room.Reset(sandbox.NormalCode(r.PathValue("code")))
		}
		p.reply(w, nil)
	})
	mux.HandleFunc("GET /sheet/{id}", p.sheet)
	// the live class view of the running class (internal/serve/class.go)
	mux.HandleFunc("/_si/class", p.classView)
	mux.HandleFunc("/_si/class/", p.classView)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() || !loopbackName(r.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if k := r.URL.Query().Get("k"); k != "" && subtle.ConstantTimeCompare([]byte(k), []byte(p.key)) == 1 {
			http.SetCookie(w, &http.Cookie{Name: "si_planner", Value: p.key, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			if r.URL.Path == "/" {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		} else if c, err := r.Cookie("si_planner"); err != nil || subtle.ConstantTimeCompare([]byte(c.Value), []byte(p.key)) != 1 {
			http.Error(w, "forbidden: open the planner link si class printed", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); r.Method == "POST" && o != "" && o != "http://"+r.Host {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		mux.ServeHTTP(w, r)
	})
}

func loopbackName(hostport string) bool {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		h = hostport
	}
	if strings.EqualFold(h, "localhost") {
		return true
	}
	ip := net.ParseIP(strings.Trim(h, "[]"))
	return ip != nil && ip.IsLoopback()
}

func (p *planner) current() *running {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.run
}

func (p *planner) reply(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	io.WriteString(w, `{"ok":true}`)
}

func (p *planner) page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, classPlannerHTML)
}

// classView is the running class's live class view, from its own server.
func (p *planner) classView(w http.ResponseWriter, r *http.Request) {
	run := p.current()
	if run == nil {
		http.Error(w, "no class is running", http.StatusNotFound)
		return
	}
	run.srv.Handler().ServeHTTP(w, r)
}

// courseChoice is a course the planner offers: a folder or a lab page with
// a sandbox:.
type courseChoice struct {
	Path  string `json:"path"`
	Image string `json:"image"`
}

// courses are the planner's course choices: the project itself, its
// top-level folders and its top-level pages, where they have a sandbox:.
func (p *planner) courses() []courseChoice {
	var out []courseChoice
	try := func(rel string) {
		if lb, err := findLabBox(filepath.Join(p.root, rel)); err == nil && lb != nil {
			out = append(out, courseChoice{filepath.ToSlash(rel), lb.spec.Image})
		}
	}
	if fileExists(filepath.Join(p.root, "_course.md")) {
		try(".")
	}
	entries, _ := os.ReadDir(p.root)
	for _, e := range entries {
		n := e.Name()
		switch {
		case strings.HasPrefix(n, ".") || strings.HasPrefix(n, "_") || n == "node_modules":
		case e.IsDir():
			try(n)
		case strings.HasSuffix(n, ".md"):
			try(n)
		}
	}
	return out
}

// state is GET /api/state: classes, courses and the running class's seats.
func (p *planner) state(w http.ResponseWriter, r *http.Request) {
	type seat struct {
		sandbox.Seat
		Label string             `json:"label"`
		Show  string             `json:"show"`
		Link  string             `json:"link,omitempty"`
		Box   *sandbox.SeatState `json:"box,omitempty"`
	}
	type class struct {
		*sandbox.Class
		Seats   []seat `json:"seats"`
		Running bool   `json:"running"`
	}
	out := struct {
		Root    string         `json:"root"`
		Classes []class        `json:"classes"`
		Courses []courseChoice `json:"courses"`
		Run     any            `json:"run"`
	}{Root: p.root, Classes: []class{}, Courses: p.courses()}
	run := p.current()
	var states map[string]sandbox.SeatState
	if run != nil {
		states = run.fleet.States()
		var used int64
		for range states {
			used += run.fleet.Spec.Memory
		}
		out.Run = map[string]any{"id": run.id, "how": run.how, "base": run.base, "ca": run.caURL, "budget": run.budget, "reserved": used,
			"image": run.fleet.Spec.Image, "memory": run.fleet.Spec.Memory, "cpus": run.fleet.Spec.CPUs, "unprivileged": !p.trusted}
	}
	for _, c := range sandbox.ListClasses(p.root) {
		live := run != nil && run.id == c.ID
		if live {
			run.room.View(func(rc *sandbox.Class) { c = rc })
		}
		cl := class{Class: c, Running: live, Seats: []seat{}}
		for _, s := range c.Seats {
			st := seat{Seat: s, Label: c.Label(s.Code), Show: sandbox.ShowCode(s.Code)}
			if live {
				st.Link = run.base + "/go/" + st.Show
				if b, ok := states[s.Code]; ok {
					st.Box = &b
				}
			}
			cl.Seats = append(cl.Seats, st)
		}
		out.Classes = append(out.Classes, cl)
	}
	writeJSONOut(w, out)
}

func writeJSONOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (p *planner) engine() (*sandbox.Engine, error) {
	sock, err := sandboxSocket(p.root)
	if err != nil {
		return nil, err
	}
	return sandbox.NewEngine(sock), nil
}

// images is GET /api/images: the sandbox host's images.
func (p *planner) images(w http.ResponseWriter, r *http.Request) {
	eng, err := p.engine()
	var list []sandbox.Image
	if err == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		list, err = eng.Images(ctx)
	}
	if err != nil {
		writeJSONOut(w, map[string]any{"error": "the sandbox host does not answer (si sandbox host start): " + err.Error(), "images": []any{}})
		return
	}
	type img struct {
		Tag  string `json:"tag"`
		Size int64  `json:"size"`
	}
	out := []img{}
	for _, i := range list {
		for _, t := range i.RepoTags {
			if t != "<none>:<none>" {
				out = append(out, img{t, i.Size})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	writeJSONOut(w, map[string]any{"images": out})
}

// pull is POST /api/pull {image}.
func (p *planner) pull(w http.ResponseWriter, r *http.Request) {
	var req struct{ Image string }
	json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req)
	img := strings.TrimSpace(req.Image)
	if img == "" || strings.ContainsAny(img, " \t") {
		p.reply(w, errors.New("want an image name"))
		return
	}
	eng, err := p.engine()
	if err == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Minute)
		defer cancel()
		err = eng.Pull(ctx, img, nil)
	}
	p.reply(w, err)
}

// create is POST /api/class: the planner's New class form.
func (p *planner) create(w http.ResponseWriter, r *http.Request) {
	var f classForm
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&f); err != nil {
		p.reply(w, errors.New("want the class form as JSON"))
		return
	}
	c, err := newClass(p.root, f)
	if err != nil {
		p.reply(w, err)
		return
	}
	writeJSONOut(w, map[string]string{"id": c.ID})
}

// edit changes class id (the running one under its lock) and saves it.
func (p *planner) edit(id string, fn func(*sandbox.Class) error) error {
	if run := p.current(); run != nil && run.id == id {
		var err error
		run.room.Update(func(c *sandbox.Class) {
			if err = fn(c); err == nil {
				err = sandbox.SaveClass(p.root, c)
			}
		})
		return err
	}
	c, err := sandbox.LoadClass(p.root, id)
	if err != nil {
		return err
	}
	if err := fn(c); err != nil {
		return err
	}
	return sandbox.SaveClass(p.root, c)
}

// addSeats is POST /api/class/{id}/seats {seats, roster}: seats on the day.
func (p *planner) addSeats(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Seats  int
		Roster string
	}
	json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req)
	p.reply(w, p.edit(r.PathValue("id"), func(c *sandbox.Class) error {
		seats, err := sandbox.ReadRoster(strings.NewReader(req.Roster))
		if err != nil {
			return err
		}
		if req.Seats > 0 {
			seats = append(seats, make([]sandbox.Seat, req.Seats)...)
		}
		return c.AddSeats(seats...)
	}))
}

// removeSeat is POST /api/class/{id}/seat/{code}/remove: the code stops
// working, the seat's box goes.
func (p *planner) removeSeat(w http.ResponseWriter, r *http.Request) {
	code := sandbox.NormalCode(r.PathValue("code"))
	err := p.edit(r.PathValue("id"), func(c *sandbox.Class) error {
		for i, s := range c.Seats {
			if s.Code == code {
				c.Seats = append(c.Seats[:i], c.Seats[i+1:]...)
				return nil
			}
		}
		return errors.New("no such seat")
	})
	if run := p.current(); err == nil && run != nil && run.id == r.PathValue("id") {
		run.room.Reset(code)
	}
	p.reply(w, err)
}

// remove is POST /api/class/{id}/delete.
func (p *planner) remove(w http.ResponseWriter, r *http.Request) {
	if run := p.current(); run != nil && run.id == r.PathValue("id") {
		p.reply(w, errors.New("stop the class first"))
		return
	}
	p.reply(w, sandbox.RemoveClass(p.root, r.PathValue("id")))
}

// sheet is GET /sheet/{id}: a printable page, a card per seat with its code
// and, while the class runs, its link as a QR code.
func (p *planner) sheet(w http.ResponseWriter, r *http.Request) {
	c, err := sandbox.LoadClass(p.root, r.PathValue("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	base := ""
	if run := p.current(); run != nil && run.id == c.ID {
		base = run.base
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>%s: seat codes</title><style>
body{font:14px/1.4 system-ui,sans-serif;margin:1.5rem;color:#111;background:#fff}h1{font-size:1.3rem;margin:0 0 .2rem}.sub{color:#555;margin:0 0 1rem}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(14rem,1fr));gap:.8rem}.card{border:1px dashed #999;border-radius:.6rem;padding:.8rem;break-inside:avoid}
.who{font-weight:600}.code{font:600 1.4rem ui-monospace,monospace;letter-spacing:.08em;margin:.3rem 0}.link{font:11px ui-monospace,monospace;word-break:break-all;color:#333}
.card svg{width:8rem;height:8rem;display:block;margin:.4rem 0}@media print{body{margin:.5cm}.noprint{display:none}}</style></head><body>
<h1>%s</h1><p class="sub">%s</p>`, html.EscapeString(c.Name), html.EscapeString(c.Name), sheetSub(base))
	b.WriteString(`<div class="grid">`)
	for _, s := range c.Seats {
		fmt.Fprintf(&b, `<div class="card"><div class="who">%s</div><div class="code">%s</div>`, html.EscapeString(c.Label(s.Code)), sandbox.ShowCode(s.Code))
		if base != "" {
			link := base + "/go/" + sandbox.ShowCode(s.Code)
			if q, err := qr.Encode([]byte(link), qr.Options{Level: qr.M}); err == nil {
				b.WriteString(q.SVG(qr.Style{FG: color.NRGBA{0, 0, 0, 255}, BG: color.NRGBA{255, 255, 255, 255}, Label: "Seat link"}, "8rem", "", ""))
			}
			fmt.Fprintf(&b, `<div class="link">%s</div>`, html.EscapeString(link))
		}
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div></body></html>`)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, b.String())
}

func sheetSub(base string) string {
	if base == "" {
		return "Open the course with your code once the class starts (the links and QR codes show here while it runs)."
	}
	return "Scan your code, or open " + html.EscapeString(base) + "/go and type it."
}
