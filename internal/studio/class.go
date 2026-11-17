package studio

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/qr"
	"github.com/felipeg48/simple-ideas/internal/serve"
)

// The Class view (RULES.md §28, docs/studio.md): Start the class on a course
// folder runs what `si serve <course>/ --class` runs, in this process: the
// course served on its own port, its pages sending their events to the class
// collection, the dashboard (/_si/class) shown in the Studio. "On my network"
// listens on every interface, so learners' devices can open the course (and
// only the course: its own folder, read-only, as si serve does); otherwise it
// stays on this machine. Stop (or quitting the Studio) ends it; the events
// stay in <course>/.si/class/<day>.jsonl.

type classRun struct {
	srv  *serve.Server
	hs   *http.Server
	stop chan struct{}
	port int
	lan  bool
	key  string
}

type classes struct {
	mu   sync.Mutex
	runs map[string]*classRun // by the course's _course.md (workspace path)
}

func (c *classes) closeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, r := range c.runs {
		r.close()
		delete(c.runs, k)
	}
}

func (r *classRun) close() {
	close(r.stop)
	r.hs.Close()
}

// classState is what the Class view shows.
type classState struct {
	Course    string `json:"course"`
	Running   bool   `json:"running"`
	LAN       bool   `json:"lan,omitempty"`
	Dashboard string `json:"dashboard,omitempty"` // for the Studio's frame (this machine, no key)
	Learners  string `json:"learners,omitempty"`  // the course for learners' devices
	Teacher   string `json:"teacher,omitempty"`   // the dashboard on a tablet: keyed, keep it private
	QR        string `json:"qr,omitempty"`        // SVG of Learners
	Note      string `json:"note,omitempty"`
}

func (a *api) classState(course string, r *classRun) classState {
	st := classState{Course: course}
	if r == nil {
		return st
	}
	port := strconv.Itoa(r.port)
	st.Running, st.LAN = true, r.lan
	st.Dashboard = "http://127.0.0.1:" + port + "/_si/class"
	st.Learners = "http://127.0.0.1:" + port + "/"
	if r.lan {
		if ips := serve.LANIPs(); len(ips) > 0 {
			host := net.JoinHostPort(ips[0], port)
			st.Learners = "http://" + host + "/"
			st.Teacher = "http://" + host + "/_si/class?k=" + r.key
		} else {
			st.Note = "no network address found: learners on this machine only"
		}
	} else {
		st.Note = "this machine only: start it on your network for learners' devices"
	}
	if c, err := qr.Encode([]byte(st.Learners), qr.Options{Level: qr.M}); err == nil {
		st.QR = c.SVG(qr.Style{Label: "Course link"}, "", "", "")
	}
	return st
}

// courseDir checks that rel is a course (its _course.md, or the folder
// holding one) and returns the _course.md's workspace path and folder.
func (a *api) courseDir(w http.ResponseWriter, rel string) (string, string, bool) {
	if filepath.Base(rel) != deck.CourseFile {
		rel = filepath.ToSlash(filepath.Join(rel, deck.CourseFile))
	}
	abs, ok := a.resolve(w, rel)
	if !ok {
		return "", "", false
	}
	if g, err := deck.LoadGuide(filepath.Dir(abs)); err != nil || !g.IsCourse() {
		fail(w, http.StatusNotFound, "not a course (a folder with _course.md)")
		return "", "", false
	}
	return rel, filepath.Dir(abs), true
}

// class is GET class?course=…: the class of a course, running or not.
func (a *api) class(w http.ResponseWriter, r *http.Request) {
	rel, _, ok := a.courseDir(w, r.URL.Query().Get("course"))
	if !ok {
		return
	}
	a.cls.mu.Lock()
	run := a.cls.runs[rel]
	a.cls.mu.Unlock()
	writeJSON(w, http.StatusOK, a.classState(rel, run))
}

// classStart is POST class {course, lan, stop}: start (or attach to) the
// class of a course, or stop it.
func (a *api) classStart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Course    string `json:"course"`
		LAN, Stop bool
	}
	if !decode(w, r, &in) {
		return
	}
	rel, dir, ok := a.courseDir(w, in.Course)
	if !ok {
		return
	}
	a.cls.mu.Lock()
	defer a.cls.mu.Unlock()
	run := a.cls.runs[rel]
	if in.Stop || run != nil && run.lan != in.LAN {
		if run != nil {
			run.close()
			delete(a.cls.runs, rel)
		}
		run = nil
		if in.Stop {
			writeJSON(w, http.StatusOK, a.classState(rel, nil))
			return
		}
	}
	if run == nil {
		var err error
		if run, err = a.startClass(dir, in.LAN); err != nil {
			fail(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.cls.runs[rel] = run
	}
	writeJSON(w, http.StatusOK, a.classState(rel, run))
}

func (a *api) startClass(dir string, lan bool) (*classRun, error) {
	srv, err := serve.New(dir, a.css, a.theme, "")
	if err != nil {
		return nil, err
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, err
	}
	srv.Class, srv.ClassKey = true, hex.EncodeToString(raw[:])
	addr := "127.0.0.1:0"
	if lan {
		addr = "0.0.0.0:0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	run := &classRun{srv: srv, stop: make(chan struct{}), port: ln.Addr().(*net.TCPAddr).Port, lan: lan, key: srv.ClassKey,
		hs: &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}}
	go srv.Watch(run.stop)
	go run.hs.Serve(ln)
	return run, nil
}
