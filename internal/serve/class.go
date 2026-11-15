package serve

import (
	"bufio"
	"bytes"
	"crypto/subtle"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/config"
	"github.com/felipeg48/simple-ideas/internal/deck"
	"github.com/felipeg48/simple-ideas/internal/render"
	"github.com/felipeg48/simple-ideas/internal/webhook"
)

// The live class view (si serve course/ --class; RULES.md §28, spec
// docs/superpowers/specs/2026-11-14-results-webhooks.md part 4). A course
// page served this way sends every result event, as JSON, to
// POST /_si/class/event (beside any webhook it has); the server keeps them in
// memory and appends them to .si/class/<date>.jsonl, and the instructor's
// dashboard (/_si/class: loopback, or the run's key) shows learners by
// lessons, who is stuck, per-lesson and per-question summaries, live over
// SSE, and exports CSV. Only what an event carries is kept: the random
// learner id, the name a learner typed, never an address or a browser.

const (
	classMaxEvents     = 200000 // in memory and on disk per run
	classMaxPerLearner = 5000
	classMaxBody       = 16 << 10
	classStuckDefault  = 10 // minutes
)

var (
	learnerRE  = regexp.MustCompile(`^l-[0-9a-f]{12}$`)
	questionRE = regexp.MustCompile(`^q[0-9]{1,3}$`)
	eventIDRE  = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
	dayRE      = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

//go:embed class.html
var classHTML string

// classQuestion is one quiz question of an attempt: right or not, never the answer.
type classQuestion struct {
	ID      string `json:"id"`
	Correct bool   `json:"correct"`
}

// classData is an event's data, the fields si's events carry, checked.
type classData struct {
	Score     int             `json:"score,omitempty"`
	Pass      bool            `json:"pass,omitempty"`
	PassMark  int             `json:"passMark,omitempty"`
	Attempt   int             `json:"attempt,omitempty"`
	Questions []classQuestion `json:"questions,omitempty"`
	Step      int             `json:"step,omitempty"`
	Title     string          `json:"title,omitempty"`
	Exit      int             `json:"exit,omitempty"`
	Percent   int             `json:"percent,omitempty"`
	Seconds   int             `json:"seconds,omitempty"`
}

// classEvent is one stored event (a line of .si/class/<date>.jsonl).
type classEvent struct {
	ID         string    `json:"id"`
	Event      string    `json:"event"`
	At         time.Time `json:"at"` // when the server got it (a learner's clock may be wrong)
	Course     string    `json:"course"`
	Lesson     string    `json:"lesson,omitempty"`
	Learner    string    `json:"learner"`
	Name       string    `json:"name,omitempty"`
	Instructor bool      `json:"instructor,omitempty"`
	Data       classData `json:"data"`
}

// parseClassEvent reads a page's JSON event (schema version 1) for course,
// keeping only the fields the class view uses, each checked and bounded.
func parseClassEvent(body []byte, course string, now time.Time) (classEvent, error) {
	var in struct {
		Version  int
		Event    string
		ID       string
		Audience string
		Lesson   *struct{ ID string }
		Learner  struct{ ID, Name string }
		Data     struct {
			Score, PassMark, Attempt, Step, Exit, Percent, Seconds float64
			Pass                                                   bool
			Title                                                  string
			Questions                                              []classQuestion
		}
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return classEvent{}, errors.New("not a JSON event")
	}
	switch {
	case in.Version != 1:
		return classEvent{}, errors.New("want version 1")
	case !slices.Contains(webhook.Events, in.Event):
		return classEvent{}, fmt.Errorf("unknown event %q", in.Event)
	case !eventIDRE.MatchString(in.ID):
		return classEvent{}, errors.New("bad id")
	case !learnerRE.MatchString(in.Learner.ID):
		return classEvent{}, errors.New("bad learner id")
	}
	num := func(f float64, lo, hi int) int {
		if math.IsNaN(f) {
			return lo
		}
		return max(lo, min(hi, int(math.Round(f))))
	}
	ev := classEvent{ID: in.ID, Event: in.Event, At: now, Course: course, Learner: in.Learner.ID,
		Name: clean(in.Learner.Name, 80, false), Instructor: in.Audience == "instructor"}
	if in.Lesson != nil && in.Event != "course.completed" {
		l := path.Clean("/" + clean(in.Lesson.ID, 300, false))[1:]
		if !strings.HasSuffix(l, ".html") {
			return classEvent{}, errors.New("bad lesson id")
		}
		ev.Lesson = l
	}
	d := in.Data
	ev.Data = classData{Score: num(d.Score, 0, 100), Pass: d.Pass, PassMark: num(d.PassMark, 0, 100), Attempt: num(d.Attempt, 0, 10000),
		Step: num(d.Step, 0, 999), Title: clean(d.Title, 200, false), Exit: num(d.Exit, -1, 255), Percent: num(d.Percent, 0, 100), Seconds: num(d.Seconds, 0, 1<<30)}
	for _, q := range d.Questions {
		if len(ev.Data.Questions) < 200 && questionRE.MatchString(q.ID) {
			ev.Data.Questions = append(ev.Data.Questions, q)
		}
	}
	return ev, nil
}

// classRoom holds a run's events and tells the dashboards about new ones.
type classRoom struct {
	mu     sync.Mutex
	dir    string // .si/class
	events []classEvent
	seen   map[string]bool // event ids: the offline queue may send one twice
	per    map[string]int  // learner -> events
	subs   map[chan struct{}]bool
}

// newClassRoom starts from today's file, so a restarted server keeps the class.
func newClassRoom(dir string, now time.Time) *classRoom {
	c := &classRoom{dir: dir, seen: map[string]bool{}, per: map[string]int{}, subs: map[chan struct{}]bool{}}
	f, err := os.Open(filepath.Join(dir, now.Format("2006-01-02")+".jsonl"))
	if err != nil {
		return c
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		var ev classEvent
		if json.Unmarshal(sc.Bytes(), &ev) == nil && ev.ID != "" && !c.seen[ev.ID] && len(c.events) < classMaxEvents {
			c.keep(ev)
		}
	}
	return c
}

func (c *classRoom) keep(ev classEvent) {
	c.events = append(c.events, ev)
	c.seen[ev.ID] = true
	c.per[ev.Learner]++
}

// errClassFull: the run (or this learner) has sent as much as it may.
var errClassFull = errors.New("the class view is full")

// add keeps ev and appends it to .si/class/<date>.jsonl; false for a
// duplicate id.
// ponytail: opens the file per event; fine for a classroom's pace.
func (c *classRoom) add(ev classEvent) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seen[ev.ID] {
		return false, nil
	}
	if len(c.events) >= classMaxEvents || c.per[ev.Learner] >= classMaxPerLearner {
		return false, errClassFull
	}
	line, _ := json.Marshal(ev)
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(filepath.Join(c.dir, ev.At.Format("2006-01-02")+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	_, err = f.Write(append(line, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return false, err
	}
	c.keep(ev)
	for ch := range c.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return true, nil
}

func (c *classRoom) list() []classEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.events)
}

func (c *classRoom) subscribe() (chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	c.mu.Lock()
	c.subs[ch] = true
	c.mu.Unlock()
	return ch, func() {
		c.mu.Lock()
		delete(c.subs, ch)
		c.mu.Unlock()
	}
}

// room is the run's classRoom, made on first use.
func (s *Server) room() *classRoom {
	s.classMu.Lock()
	defer s.classMu.Unlock()
	if s.class == nil {
		s.class = newClassRoom(filepath.Join(s.dir, ".si", "class"), time.Now())
	}
	return s.class
}

// ClassFiles are the .si/class files this run wrote to, for the exit message.
func (s *Server) ClassFiles() []string {
	s.classMu.Lock()
	c := s.class
	s.classMu.Unlock()
	if c == nil {
		return nil
	}
	var out []string
	for _, ev := range c.list() {
		p := filepath.Join(c.dir, ev.At.Format("2006-01-02")+".jsonl")
		if !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// classLesson is a column of the grid.
type classLesson struct {
	ID     string `json:"id"` // the page path inside the course, as events name it
	Title  string `json:"title"`
	Kind   string `json:"kind"`
	Module string `json:"module"`
	// Questions are a quiz lesson's question titles, in order (q1 is the
	// first), read from its source, so the view names them.
	Questions []string `json:"questions,omitempty"`
}

type classCourse struct {
	ID      string        `json:"id"` // the course folder's name
	Title   string        `json:"title"`
	Theme   string        `json:"-"`
	Lessons []classLesson `json:"lessons"`
}

// classCourses are the courses served: the folder itself, or its subfolders.
func (s *Server) classCourses() []classCourse {
	dirs := []string{s.dir}
	if es, err := os.ReadDir(s.dir); err == nil {
		for _, e := range es {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !strings.HasPrefix(e.Name(), "_") {
				dirs = append(dirs, filepath.Join(s.dir, e.Name()))
			}
		}
	}
	var out []classCourse
	for _, d := range dirs {
		g, err := deck.LoadGuide(d)
		if err != nil || !g.IsCourse() {
			continue
		}
		g = g.ForAudience(s.Instructor)
		c := classCourse{ID: filepath.Base(g.Dir), Title: g.Meta.Title, Theme: g.Meta.Theme}
		for _, m := range g.Items {
			for _, it := range m.Children {
				if it.File == "" {
					continue
				}
				r, _ := filepath.Rel(g.Dir, it.File)
				c.Lessons = append(c.Lessons, classLesson{ID: filepath.ToSlash(strings.TrimSuffix(r, filepath.Ext(r)) + ".html"),
					Title: it.Title, Kind: it.Kind, Module: m.Title, Questions: quizTitles(it.File, it.Kind)})
			}
		}
		out = append(out, c)
	}
	return out
}

// quizTitles are the question titles of a quiz lesson (type: quiz), as
// plain text, in the order doc.js numbers them; nil for any other lesson.
func quizTitles(file, kind string) []string {
	if kind != "quiz" {
		return nil
	}
	d, err := deck.Parse(file)
	if err != nil || len(d.Slides) == 0 {
		return nil
	}
	_, qs, _ := deck.ParseQuiz(d.Slides[0].Body)
	var out []string
	for _, q := range qs {
		out = append(out, strings.NewReplacer("**", "", "__", "", "`", "", "*", "").Replace(q.Title))
	}
	return out
}

// coursePage is the course a page (a URL path) belongs to, or nil.
func (s *Server) coursePage(page string) *deck.Guide {
	p := path.Clean("/" + page)
	file := filepath.Join(s.dir, filepath.FromSlash(p))
	if md := s.source(strings.TrimSuffix(p, ".html")); md != "" {
		file = md
	}
	if g := deck.FindGuide(file, s.dir); g != nil && g.IsCourse() {
		return g
	}
	return nil
}

/* ---------- the view: learners by lessons, who is stuck, summaries ---------- */

type classCheck struct {
	Step  int    `json:"step"`
	Title string `json:"title,omitempty"`
	Pass  bool   `json:"pass"`  // the last try passed
	Fails int    `json:"fails"` // failed tries since the last pass
	Tries int    `json:"tries"`
	last  time.Time
}

type classQuizCell struct {
	Score     int  `json:"score"` // the last attempt
	Best      int  `json:"best"`
	Attempts  int  `json:"attempts"`
	Pass      bool `json:"pass"` // any attempt passed
	questions []classQuestion
}

type classCell struct {
	Done   bool           `json:"done,omitempty"`
	Quiz   *classQuizCell `json:"quiz,omitempty"`
	Checks []*classCheck  `json:"checks,omitempty"`
	Demo   int            `json:"demo,omitempty"` // the most watched, percent
	Last   time.Time      `json:"last"`
}

type classLearner struct {
	ID         string                `json:"id"`
	Name       string                `json:"name,omitempty"`
	Label      string                `json:"label"`
	Instructor bool                  `json:"instructor,omitempty"`
	First      time.Time             `json:"first"`
	Last       time.Time             `json:"last"`
	Lesson     string                `json:"lesson,omitempty"` // the lesson of the last event
	Done       int                   `json:"done"`
	Completed  bool                  `json:"completed,omitempty"`
	Cells      map[string]*classCell `json:"cells"`
	progress   time.Time
}

type classStuck struct {
	Learner string    `json:"learner"`
	Label   string    `json:"label"`
	Lesson  string    `json:"lesson"`
	Title   string    `json:"title"`
	Reason  string    `json:"reason"`
	Since   time.Time `json:"since"`
}

type classLessonSum struct {
	classLesson
	Done       int `json:"done"`
	Started    int `json:"started"`
	QuizTakers int `json:"quizTakers,omitempty"`
	QuizPassed int `json:"quizPassed,omitempty"`
	QuizAvg    int `json:"quizAvg,omitempty"` // of each learner's best
	Failing    int `json:"failing,omitempty"` // learners whose last try of a check failed
	DemoAvg    int `json:"demoAvg,omitempty"`
}

type classQuizQ struct {
	ID       string `json:"id"`
	Text     string `json:"text,omitempty"` // the question's title, from the quiz source
	Wrong    int    `json:"wrong"`
	Answered int    `json:"answered"`
}

type classQuiz struct {
	Lesson    string       `json:"lesson"`
	Title     string       `json:"title"`
	Takers    int          `json:"takers"`
	Questions []classQuizQ `json:"questions"`
}

type classView struct {
	Course     classCourse      `json:"course"`
	Courses    []classCourse    `json:"courses"`
	Learners   []*classLearner  `json:"learners"`
	Stuck      []classStuck     `json:"stuck"`
	Lessons    []classLessonSum `json:"lessons"`
	Quizzes    []classQuiz      `json:"quizzes"`
	StuckAfter int              `json:"stuckAfter"` // minutes
	Events     int              `json:"events"`
	Now        time.Time        `json:"now"`
	Day        string           `json:"day"`  // the day shown: today (live), or a saved one
	Days       []string         `json:"days"` // the days with a .si/class/<day>.jsonl, newest first, today always
}

// learnerLabel is how a learner is shown: the typed name, else "Learner 3f9a".
func learnerLabel(id, name string) string {
	if name != "" {
		return name
	}
	return "Learner " + strings.TrimPrefix(id, "l-")[:4]
}

// classAggregate is the view of one course from the run's events.
func classAggregate(course classCourse, events []classEvent, now time.Time, stuckAfter time.Duration) classView {
	v := classView{Course: course, StuckAfter: int(stuckAfter / time.Minute), Now: now, // lists as [], never null
		Learners: []*classLearner{}, Stuck: []classStuck{}, Lessons: []classLessonSum{}, Quizzes: []classQuiz{}}
	by := map[string]*classLearner{}
	for _, ev := range events {
		if ev.Course != course.ID {
			continue
		}
		v.Events++
		l := by[ev.Learner]
		if l == nil {
			l = &classLearner{ID: ev.Learner, First: ev.At, Cells: map[string]*classCell{}, progress: ev.At}
			by[ev.Learner] = l
			v.Learners = append(v.Learners, l)
		}
		if ev.Name != "" {
			l.Name = ev.Name
		}
		l.Instructor = l.Instructor || ev.Instructor
		l.Last = ev.At
		moved := false
		if ev.Event == "course.completed" {
			l.Completed, moved = true, true
		}
		if ev.Lesson != "" {
			l.Lesson = ev.Lesson
			c := l.Cells[ev.Lesson]
			if c == nil {
				c = &classCell{}
				l.Cells[ev.Lesson] = c
			}
			c.Last = ev.At
			d := ev.Data
			switch ev.Event {
			case "lesson.completed":
				c.Done, moved = true, true
			case "quiz.submitted":
				if c.Quiz == nil {
					c.Quiz = &classQuizCell{}
				}
				q := c.Quiz
				q.Attempts++
				q.Score, q.Best, q.Pass, q.questions = d.Score, max(q.Best, d.Score), q.Pass || d.Pass, d.Questions
				moved = moved || d.Pass
			case "lab.step.checked":
				var k *classCheck
				for _, x := range c.Checks {
					if x.Step == d.Step {
						k = x
					}
				}
				if k == nil {
					k = &classCheck{Step: d.Step}
					c.Checks = append(c.Checks, k)
					slices.SortFunc(c.Checks, func(a, b *classCheck) int { return a.Step - b.Step })
				}
				if d.Title != "" {
					k.Title = d.Title
				}
				k.Tries++
				k.Pass, k.last = d.Pass, ev.At
				if d.Pass {
					k.Fails, moved = 0, true
				} else {
					k.Fails++
				}
			case "demo.watched":
				c.Demo = max(c.Demo, d.Percent)
				moved = moved || d.Percent >= 90
			}
		}
		if moved {
			l.progress = ev.At
		}
	}
	titles := map[string]string{}
	for _, ls := range course.Lessons {
		titles[ls.ID] = ls.Title
	}
	for _, l := range v.Learners {
		l.Label = learnerLabel(l.ID, l.Name)
		for _, ls := range course.Lessons {
			if c := l.Cells[ls.ID]; c != nil && c.Done {
				l.Done++
			}
		}
		// failing the same check twice in a row
		for _, ls := range course.Lessons {
			c := l.Cells[ls.ID]
			if c == nil {
				continue
			}
			for _, k := range c.Checks {
				if k.Fails >= 2 {
					what := "check " + strconv.Itoa(k.Step)
					if k.Title != "" {
						what += " (“" + k.Title + "”)"
					}
					v.Stuck = append(v.Stuck, classStuck{Learner: l.ID, Label: l.Label, Lesson: ls.ID, Title: ls.Title, Since: k.last,
						Reason: fmt.Sprintf("failed %s %d times in a row", what, k.Fails)})
				}
			}
		}
		// no progress for a while on the lesson of the last event
		if c := l.Cells[l.Lesson]; !l.Completed && l.Lesson != "" && (c == nil || !c.Done) && now.Sub(l.progress) >= stuckAfter {
			t := titles[l.Lesson]
			if t == "" {
				t = l.Lesson
			}
			v.Stuck = append(v.Stuck, classStuck{Learner: l.ID, Label: l.Label, Lesson: l.Lesson, Title: t, Since: l.progress,
				Reason: fmt.Sprintf("no progress for %d min", int(now.Sub(l.progress)/time.Minute))})
		}
	}
	slices.SortStableFunc(v.Learners, func(a, b *classLearner) int {
		return strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label))
	})
	slices.SortStableFunc(v.Stuck, func(a, b classStuck) int { return a.Since.Compare(b.Since) })
	for _, ls := range course.Lessons {
		sum := classLessonSum{classLesson: ls}
		var best, demo, demoN int
		wrong := map[string]*classQuizQ{}
		var qids []string
		for _, l := range v.Learners {
			c := l.Cells[ls.ID]
			if c == nil {
				continue
			}
			sum.Started++
			if c.Done {
				sum.Done++
			}
			if q := c.Quiz; q != nil {
				sum.QuizTakers++
				best += q.Best
				if q.Pass {
					sum.QuizPassed++
				}
				for _, a := range q.questions {
					w := wrong[a.ID]
					if w == nil {
						w = &classQuizQ{ID: a.ID}
						wrong[a.ID] = w
						qids = append(qids, a.ID)
					}
					w.Answered++
					if !a.Correct {
						w.Wrong++
					}
				}
			}
			for _, k := range c.Checks {
				if !k.Pass {
					sum.Failing++
					break
				}
			}
			if c.Demo > 0 {
				demo += c.Demo
				demoN++
			}
		}
		if sum.QuizTakers > 0 {
			sum.QuizAvg = int(math.Round(float64(best) / float64(sum.QuizTakers)))
			slices.SortFunc(qids, func(a, b string) int {
				x, _ := strconv.Atoi(a[1:])
				y, _ := strconv.Atoi(b[1:])
				return x - y
			})
			qz := classQuiz{Lesson: ls.ID, Title: ls.Title, Takers: sum.QuizTakers}
			for _, id := range qids {
				q := *wrong[id]
				if n, _ := strconv.Atoi(id[1:]); n >= 1 && n <= len(ls.Questions) {
					q.Text = ls.Questions[n-1]
				}
				qz.Questions = append(qz.Questions, q)
			}
			v.Quizzes = append(v.Quizzes, qz)
		}
		if demoN > 0 {
			sum.DemoAvg = int(math.Round(float64(demo) / float64(demoN)))
		}
		v.Lessons = append(v.Lessons, sum)
	}
	return v
}

// csvCell keeps a spreadsheet from reading a typed name as a formula.
func csvCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

// classCSV is the view as rows: one per learner and lesson.
func classCSV(v classView) []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Write([]string{"course", "learner", "name", "module", "lesson", "kind", "done", "quiz_score", "quiz_best", "quiz_attempts", "quiz_passed",
		"checks_passed", "checks_failing", "demo_percent", "last_activity"})
	for _, l := range v.Learners {
		for _, ls := range v.Course.Lessons {
			row := []string{v.Course.ID, l.ID, csvCell(l.Name), csvCell(ls.Module), csvCell(ls.Title), ls.Kind, "no", "", "", "", "", "", "", "", ""}
			if c := l.Cells[ls.ID]; c != nil {
				if c.Done {
					row[6] = "yes"
				}
				if q := c.Quiz; q != nil {
					row[7], row[8], row[9], row[10] = strconv.Itoa(q.Score), strconv.Itoa(q.Best), strconv.Itoa(q.Attempts), strconv.FormatBool(q.Pass)
				}
				if len(c.Checks) > 0 {
					ok, bad := 0, 0
					for _, k := range c.Checks {
						if k.Pass {
							ok++
						} else {
							bad++
						}
					}
					row[11], row[12] = strconv.Itoa(ok), strconv.Itoa(bad)
				}
				if c.Demo > 0 {
					row[13] = strconv.Itoa(c.Demo)
				}
				row[14] = c.Last.UTC().Format(time.RFC3339)
			}
			w.Write(row)
		}
	}
	w.Flush()
	return b.Bytes()
}

/* ---------- the routes ---------- */

func (s *Server) classRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /_si/class/event", s.classEvent) // learners' pages, on any address the course is served on
	mux.HandleFunc("GET /_si/class", s.classAuth(s.classPage))
	mux.HandleFunc("GET /_si/class/data", s.classAuth(s.classData))
	mux.HandleFunc("GET /_si/class/events", s.classAuth(s.classEvents))
	mux.HandleFunc("GET /_si/class/csv", s.classAuth(s.classCSVFile))
}

// classAuth lets the dashboard through on this machine's loopback (the
// address and the Host both), elsewhere only with the run's key (?k=).
func (s *Server) classAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		local := ip != nil && ip.IsLoopback() && loopbackHost(r.Host)
		k := r.URL.Query().Get("k")
		if !local && (s.ClassKey == "" || subtle.ConstantTimeCompare([]byte(k), []byte(s.ClassKey)) != 1) {
			http.Error(w, "forbidden: open the class link si printed", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		h(w, r)
	}
}

// classEvent is POST /_si/class/event?page=…: a course page's JSON event.
func (s *Server) classEvent(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	g := s.coursePage(r.URL.Query().Get("page"))
	if g == nil {
		http.Error(w, "not a course page", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, classMaxBody+1))
	if err != nil || len(body) > classMaxBody {
		http.Error(w, "too big", http.StatusRequestEntityTooLarge)
		return
	}
	ev, err := parseClassEvent(body, filepath.Base(g.Dir), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	added, err := s.room().add(ev)
	switch {
	case errors.Is(err, errClassFull):
		http.Error(w, err.Error(), http.StatusTooManyRequests)
	case err != nil:
		log.Printf("si: class: %v", err)
		http.Error(w, "cannot keep it", http.StatusInternalServerError)
	case !added:
		w.WriteHeader(http.StatusNoContent) // a duplicate: already kept
	default:
		w.WriteHeader(http.StatusAccepted)
	}
}

// classPick is the course a request asks for (?course=), else the first.
func (s *Server) classPick(r *http.Request) (classCourse, []classCourse, bool) {
	cs := s.classCourses()
	if len(cs) == 0 {
		return classCourse{}, nil, false
	}
	want := r.URL.Query().Get("course")
	for _, c := range cs {
		if c.ID == want {
			return c, cs, true
		}
	}
	return cs[0], cs, true
}

func (s *Server) classView(r *http.Request) (classView, bool) {
	c, cs, ok := s.classPick(r)
	if !ok {
		return classView{}, false
	}
	n, err := strconv.Atoi(r.URL.Query().Get("stuck"))
	if err != nil || n < 1 || n > 24*60 {
		n = classStuckDefault
	}
	today := time.Now().Format("2006-01-02")
	day := r.URL.Query().Get("day")
	events, now := s.room().list(), time.Now()
	if day != "" && day != today {
		t, err := time.ParseInLocation("2006-01-02", day, time.Local)
		if !dayRE.MatchString(day) || err != nil {
			return classView{}, false
		}
		// a saved day: read from its file, "now" its last event, so stuck reads as at the close
		events, now = newClassRoom(s.room().dir, t).list(), t
		if len(events) > 0 {
			now = events[len(events)-1].At
		}
	} else {
		day = today
	}
	v := classAggregate(c, events, now, time.Duration(n)*time.Minute)
	v.Courses, v.Day, v.Days = cs, day, classDays(s.room().dir, today)
	return v, true
}

// classDays are the days with saved events (.si/class/<day>.jsonl), newest
// first, today always.
func classDays(dir, today string) []string {
	days := []string{today}
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	for _, f := range files {
		if d := strings.TrimSuffix(filepath.Base(f), ".jsonl"); dayRE.MatchString(d) && d != today {
			days = append(days, d)
		}
	}
	slices.SortFunc(days, func(a, b string) int { return strings.Compare(b, a) })
	return days
}

func (s *Server) classData(w http.ResponseWriter, r *http.Request) {
	v, ok := s.classView(r)
	if !ok {
		http.Error(w, "no course here (a folder with _course.md), or no such day", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) classCSVFile(w http.ResponseWriter, r *http.Request) {
	v, ok := s.classView(r)
	if !ok {
		http.Error(w, "no course here (a folder with _course.md)", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-class-%s.csv"`, v.Course.ID, v.Day))
	w.Write(classCSV(v))
}

// classEvents is the dashboard's SSE stream: "changed" on every new event.
func (s *Server) classEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	sseHeaders(w)
	ch, done := s.room().subscribe()
	defer done()
	fmt.Fprint(w, "retry: 2000\n: connected\n\n")
	fl.Flush()
	tick := time.NewTicker(25 * time.Second) // keeps proxies from closing a quiet stream
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprint(w, "event: changed\ndata: 1\n\n")
		case <-tick.C:
			fmt.Fprint(w, ": ping\n\n")
		}
		fl.Flush()
	}
}

func (s *Server) classPage(w http.ResponseWriter, r *http.Request) {
	c, _, ok := s.classPick(r)
	if !ok {
		http.Error(w, "no course here (a folder with _course.md)", http.StatusNotFound)
		return
	}
	cfg := config.Load(s.Conf, filepath.Join(s.dir, "si.conf"))
	theme := s.Theme
	if theme == "" {
		theme = c.Theme
	}
	if theme == "" {
		theme = cfg.Theme
	}
	_, css, err := render.ThemeLayer(theme, s.dir)
	if err != nil {
		log.Printf("si: class: %v", err)
	}
	page := strings.Replace(classHTML, "/*SI-THEME*/", strings.ReplaceAll(css, "</", `<\/`), 1)
	page = strings.ReplaceAll(page, "{{TITLE}}", html.EscapeString(c.Title))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, page)
}
