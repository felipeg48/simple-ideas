package sandbox

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// Classes (RULES §37, si class): a class is a course served from this
// machine to learners, each on a seat with a code; a seat's box is made from
// the course's sandbox: when its learner first opens the course. The class
// file is .si/classes/<id>.json under the folder si class runs in.

// ClassDir is where a project's classes are kept.
func ClassDir(root string) string { return filepath.Join(root, ".si", "classes") }

// codeAlphabet has no 0/O, 1/I/L or U: a code read aloud or from paper is
// typed right.
const codeAlphabet = "23456789ABCDEFGHJKMNPQRSTVWXYZ"

// codeLen is a code's length: 30^8 ≈ 6.6e11, guessed only by trying for
// years at the rate /go allows.
const codeLen = 8

// MaxSeats caps a class.
const MaxSeats = 500

var classIDRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,47}$`)

// Class is one planned class.
type Class struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Course  string    `json:"course"`          // the course folder (or a lab page), relative to the project
	Image   string    `json:"image,omitempty"` // instead of the course's sandbox: image
	Memory  int64     `json:"memory,omitempty"`
	CPUs    float64   `json:"cpus,omitempty"`
	TTL     string    `json:"ttl,omitempty"`  // a box lives at most this long (2h; "" = while the class runs)
	Idle    string    `json:"idle,omitempty"` // a box no page has used this long is removed (30m; "" = never)
	Expires string    `json:"expires,omitempty"`
	Created time.Time `json:"created"`
	Seats   []Seat    `json:"seats"`
}

// Seat is one learner's place.
type Seat struct {
	Code   string    `json:"code"`
	Name   string    `json:"name,omitempty"`
	Email  string    `json:"email,omitempty"`
	Joined time.Time `json:"joined,omitzero"`
}

// Label is the seat as a person: the name, else "Seat N".
func (c *Class) Label(code string) string {
	for i, s := range c.Seats {
		if s.Code == code {
			if s.Name != "" {
				return s.Name
			}
			return fmt.Sprintf("Seat %d", i+1)
		}
	}
	return code
}

// Seat finds a seat by its code (any case, dashes and spaces ignored).
func (c *Class) Seat(code string) (*Seat, bool) {
	code = NormalCode(code)
	for i := range c.Seats {
		if c.Seats[i].Code == code {
			return &c.Seats[i], true
		}
	}
	return nil, false
}

// Expired reports whether the class's last day is over.
func (c *Class) Expired(now time.Time) bool {
	if c.Expires == "" {
		return false
	}
	t, err := time.ParseInLocation("2006-01-02", c.Expires, time.Local)
	return err == nil && !now.Before(t.AddDate(0, 0, 1))
}

// NormalCode is a typed code as stored: upper case, no dashes or spaces.
func NormalCode(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' {
			return -1
		}
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		return r
	}, s)
}

// ShowCode is a code as printed: ABCD-EFGH.
func ShowCode(c string) string {
	if len(c) == codeLen {
		return c[:4] + "-" + c[4:]
	}
	return c
}

// NewCode is a random seat code.
func NewCode() string {
	b := make([]byte, codeLen)
	n := big.NewInt(int64(len(codeAlphabet)))
	for i := range b {
		k, _ := rand.Int(rand.Reader, n)
		b[i] = codeAlphabet[k.Int64()]
	}
	return string(b)
}

// AddSeats adds seats (their codes made here when empty, or checked) and
// returns how many the class has.
func (c *Class) AddSeats(seats ...Seat) error {
	used := map[string]bool{}
	for _, s := range c.Seats {
		used[s.Code] = true
	}
	if len(c.Seats)+len(seats) > MaxSeats {
		return fmt.Errorf("a class has at most %d seats", MaxSeats)
	}
	for _, s := range seats {
		s.Code = NormalCode(s.Code)
		switch {
		case s.Code == "":
			for s.Code = NewCode(); used[s.Code]; s.Code = NewCode() {
			}
		case len(s.Code) != codeLen || strings.Trim(s.Code, codeAlphabet) != "":
			return fmt.Errorf("seat code %q: want %d of %s", s.Code, codeLen, codeAlphabet)
		case used[s.Code]:
			return fmt.Errorf("seat code %s twice", s.Code)
		}
		used[s.Code] = true
		s.Name, s.Email = clip(s.Name, 80), clip(s.Email, 120)
		c.Seats = append(c.Seats, s)
	}
	return nil
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		s = s[:n]
	}
	return s
}

// ReadRoster reads a roster: CSV rows name,email[,code], a header row
// (name,email,…) skipped.
func ReadRoster(r io.Reader) ([]Seat, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord, cr.TrimLeadingSpace = -1, true
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("roster: %v", err)
	}
	var out []Seat
	for i, row := range rows {
		if i == 0 && len(row) > 0 && strings.EqualFold(strings.TrimSpace(row[0]), "name") {
			continue
		}
		var s Seat
		for j, v := range row {
			switch j {
			case 0:
				s.Name = v
			case 1:
				s.Email = v
			case 2:
				s.Code = v
			}
		}
		if strings.TrimSpace(s.Name+s.Email+s.Code) == "" {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// Check says what is wrong with the class, nil when nothing.
func (c *Class) Check() error {
	switch {
	case !classIDRE.MatchString(c.ID):
		return fmt.Errorf("class id %q: lower-case letters, digits and dashes", c.ID)
	case c.Course == "":
		return errors.New("a class needs a course")
	}
	for _, d := range []string{c.TTL, c.Idle} {
		if d != "" {
			if v, err := time.ParseDuration(d); err != nil || v < time.Minute {
				return fmt.Errorf("%q: want a duration of a minute or more (90m, 2h)", d)
			}
		}
	}
	if c.Expires != "" {
		if _, err := time.Parse("2006-01-02", c.Expires); err != nil {
			return fmt.Errorf("expires %q: want YYYY-MM-DD", c.Expires)
		}
	}
	return nil
}

// ClassID is an id made from a name: lower case, dashes, a date when empty.
func ClassID(name string, now time.Time) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	id := strings.Trim(b.String(), "-")
	if len(id) > 40 {
		id = strings.Trim(id[:40], "-")
	}
	if id == "" {
		id = "class-" + now.Format("2006-01-02")
	}
	return id
}

// SaveClass writes the class file (0600: it holds every seat's code),
// atomically.
func SaveClass(root string, c *Class) error {
	if err := c.Check(); err != nil {
		return err
	}
	dir := ClassDir(root)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".class-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, c.ID+".json"))
}

// LoadClass reads a class file.
func LoadClass(root, id string) (*Class, error) {
	if !classIDRE.MatchString(id) {
		return nil, fmt.Errorf("no class %q", id)
	}
	b, err := os.ReadFile(filepath.Join(ClassDir(root), id+".json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no class %q (si class list shows them)", id)
	}
	if err != nil {
		return nil, err
	}
	var c Class
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("class %s: %v", id, err)
	}
	return &c, nil
}

// ListClasses reads every class of the project, newest first.
func ListClasses(root string) []*Class {
	files, _ := filepath.Glob(filepath.Join(ClassDir(root), "*.json"))
	var out []*Class
	for _, f := range files {
		if c, err := LoadClass(root, strings.TrimSuffix(filepath.Base(f), ".json")); err == nil {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

// RemoveClass deletes a class file.
func RemoveClass(root, id string) error {
	if !classIDRE.MatchString(id) {
		return fmt.Errorf("no class %q", id)
	}
	return os.Remove(filepath.Join(ClassDir(root), id+".json"))
}

// ErrBudget: a new box would take the class past its memory budget.
var ErrBudget = errors.New("the sandbox host's memory budget is used up: no box for another seat now (reset or stop a seat, or start the class with a larger --budget)")

// Fleet runs the seats' boxes of one class: a box per seat, made on first
// use, admitted against a memory budget, removed by Reap past the class's
// ttl: or idle:, and all removed by Close.
type Fleet struct {
	Eng    *Engine
	Spec   deck.Sandbox
	Dir    string // the course folder (sandbox-files: is relative to it)
	Class  string
	Opt    Options
	Budget int64 // bytes of box memory the class may use; 0: no cap
	TTL    time.Duration
	Idle   time.Duration

	mu    sync.Mutex
	seats map[string]*seatBox
}

type seatBox struct {
	box      *Box
	err      error
	ready    chan struct{} // closed when Up ends
	started  time.Time
	seen     time.Time
	inflight int // requests open now (a terminal stream counts while it is open)
	memory   int64
	steps    []string
}

// SeatState is what the planner shows of a seat's box.
type SeatState struct {
	Code    string    `json:"code"`
	State   string    `json:"state"` // starting, ready, failed
	Error   string    `json:"error,omitempty"`
	Step    string    `json:"step,omitempty"`
	Box     string    `json:"box,omitempty"`
	Started time.Time `json:"started"`
	Seen    time.Time `json:"seen"`
	Memory  int64     `json:"memory"`
}

// Get is the seat's box, made now when it has none (or its last one
// failed), waiting at most until ctx ends for it to be ready.
func (f *Fleet) Get(ctx context.Context, code string) (*Box, error) {
	f.mu.Lock()
	if f.seats == nil {
		f.seats = map[string]*seatBox{}
	}
	sb := f.seats[code]
	if sb != nil {
		select {
		case <-sb.ready:
			if sb.err != nil { // try again
				delete(f.seats, code)
				sb = nil
			}
		default:
		}
	}
	if sb == nil {
		if f.Budget > 0 && int64(len(f.seats)+1)*max(f.Spec.Memory, 1) > f.Budget {
			f.mu.Unlock()
			return nil, ErrBudget
		}
		now := time.Now()
		sb = &seatBox{ready: make(chan struct{}), started: now, seen: now}
		f.seats[code] = sb
		go f.up(code, sb)
	}
	f.mu.Unlock()
	select {
	case <-sb.ready:
		return sb.box, sb.err
	case <-ctx.Done():
		return nil, fmt.Errorf("your sandbox is still starting (%s); try again in a moment", f.step(sb))
	}
}

func (f *Fleet) step(sb *seatBox) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(sb.steps) == 0 {
		return "waiting"
	}
	return sb.steps[len(sb.steps)-1]
}

// up makes a seat's box.
func (f *Fleet) up(code string, sb *seatBox) {
	o := f.Opt
	o.Labels = map[string]string{"si.class": f.Class, "si.seat": code}
	o.Step = func(s string) {
		f.mu.Lock()
		sb.steps = append(sb.steps, s)
		f.mu.Unlock()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	b, err := Up(ctx, f.Eng, f.Spec, f.Dir, o)
	f.mu.Lock()
	sb.box, sb.err = b, err
	gone := f.seats[code] != sb // reset or closed while it started
	f.mu.Unlock()
	close(sb.ready)
	if gone && b != nil {
		b.Down()
	}
}

// Has reports whether the seat has a box (ready or starting).
func (f *Fleet) Has(code string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.seats[code]
	return ok
}

// Begin marks a request of the seat as open; End closes it. A seat with an
// open request is never idle.
func (f *Fleet) Begin(code string) { f.mark(code, 1) }

// End: see Begin.
func (f *Fleet) End(code string) { f.mark(code, -1) }

func (f *Fleet) mark(code string, d int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sb := f.seats[code]; sb != nil {
		sb.inflight += d
		sb.seen = time.Now()
	}
}

// Reset removes the seat's box (sandbox-cleanup: first); the next use makes
// a fresh one from the image.
func (f *Fleet) Reset(code string) {
	f.mu.Lock()
	sb := f.seats[code]
	delete(f.seats, code)
	f.mu.Unlock()
	if sb == nil {
		return
	}
	select {
	case <-sb.ready:
		if sb.box != nil {
			sb.box.Down()
		}
	default: // still starting: up removes it when it ends
	}
}

// Reap removes the boxes past the TTL, or idle (no open request) past Idle.
func (f *Fleet) Reap(now time.Time) (removed []string) {
	f.mu.Lock()
	for code, sb := range f.seats {
		select {
		case <-sb.ready:
		default:
			continue
		}
		if f.TTL > 0 && now.Sub(sb.started) >= f.TTL || f.Idle > 0 && sb.inflight <= 0 && now.Sub(sb.seen) >= f.Idle {
			removed = append(removed, code)
		}
	}
	f.mu.Unlock()
	for _, code := range removed {
		f.Reset(code)
	}
	return removed
}

// Measure reads each ready box's memory use (one stats call per box).
func (f *Fleet) Measure(ctx context.Context) {
	f.mu.Lock()
	boxes := map[*seatBox]*Box{}
	for _, sb := range f.seats {
		if sb.box != nil {
			boxes[sb] = sb.box
		}
	}
	f.mu.Unlock()
	for sb, b := range boxes {
		if m, err := f.Eng.Memory(ctx, b.ID); err == nil {
			f.mu.Lock()
			sb.memory = m
			f.mu.Unlock()
		}
	}
}

// States are the seats' boxes now.
func (f *Fleet) States() map[string]SeatState {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]SeatState{}
	for code, sb := range f.seats {
		st := SeatState{Code: code, State: "starting", Started: sb.started, Seen: sb.seen, Memory: sb.memory}
		if len(sb.steps) > 0 {
			st.Step = sb.steps[len(sb.steps)-1]
		}
		select {
		case <-sb.ready:
			switch {
			case sb.err != nil:
				st.State, st.Error = "failed", sb.err.Error()
			case sb.box != nil:
				st.State, st.Box = "ready", sb.box.Name
			}
		default:
		}
		out[code] = st
	}
	return out
}

// Close removes every box of the class.
func (f *Fleet) Close() {
	f.mu.Lock()
	codes := make([]string, 0, len(f.seats))
	for c := range f.seats {
		codes = append(codes, c)
	}
	f.mu.Unlock()
	var wg sync.WaitGroup
	for _, c := range codes {
		wg.Add(1)
		go func() { defer wg.Done(); f.Reset(c) }()
	}
	wg.Wait()
}
