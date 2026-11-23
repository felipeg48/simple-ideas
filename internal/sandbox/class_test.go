package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

func TestSeatCodes(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		c := NewCode()
		if len(c) != codeLen || strings.Trim(c, codeAlphabet) != "" {
			t.Fatalf("code %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 199 {
		t.Errorf("codes repeat: %d of 200 distinct", len(seen))
	}
	if NormalCode("ab2c-d3ef ") != "AB2CD3EF" || ShowCode("AB2CD3EF") != "AB2C-D3EF" {
		t.Error("normal/show")
	}
}

func TestClassSeats(t *testing.T) {
	c := &Class{ID: "x", Course: "lab"}
	seats, err := ReadRoster(strings.NewReader("name,email\nAda Lovelace, ada@example.com\n\nAlan,alan@example.com,abcd-efgh\n"))
	if err != nil || len(seats) != 2 {
		t.Fatalf("roster %v %v", seats, err)
	}
	if err := c.AddSeats(append(seats, Seat{}, Seat{})...); err != nil {
		t.Fatal(err)
	}
	if len(c.Seats) != 4 || c.Seats[1].Code != "ABCDEFGH" || c.Seats[0].Email != "ada@example.com" {
		t.Errorf("seats %+v", c.Seats)
	}
	if s, ok := c.Seat("abcd efgh"); !ok || s.Name != "Alan" {
		t.Error("seat by typed code")
	}
	if c.Label(c.Seats[2].Code) != "Seat 3" || c.Label("ABCDEFGH") != "Alan" {
		t.Error("labels")
	}
	if err := c.AddSeats(Seat{Code: "ABCD-EFGH"}); err == nil {
		t.Error("a code twice")
	}
	if err := c.AddSeats(Seat{Code: "OOOO1111"}); err == nil {
		t.Error("a code outside the alphabet")
	}
	if err := c.AddSeats(make([]Seat, MaxSeats)...); err == nil {
		t.Error("past the seat cap")
	}
}

func TestClassFile(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.Local)
	c := &Class{ID: ClassID("Docker basics, December!", now), Name: "Docker basics", Course: "lab", TTL: "3h", Expires: "2026-12-05", Created: now}
	if c.ID != "docker-basics-december" {
		t.Errorf("id %q", c.ID)
	}
	c.AddSeats(Seat{Name: "Ada"})
	if err := SaveClass(root, c); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(ClassDir(root), c.ID+".json"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("class file %v %v (want 0600: it holds the codes)", fi, err)
	}
	got, err := LoadClass(root, c.ID)
	if err != nil || got.Seats[0].Code != c.Seats[0].Code || got.TTL != "3h" {
		t.Errorf("load %+v %v", got, err)
	}
	if l := ListClasses(root); len(l) != 1 {
		t.Errorf("list %d", len(l))
	}
	if _, err := LoadClass(root, "../etc"); err == nil {
		t.Error("a path as an id")
	}
	if !c.Expired(time.Date(2026, 12, 6, 0, 0, 1, 0, time.Local)) || c.Expired(time.Date(2026, 12, 5, 23, 0, 0, 0, time.Local)) {
		t.Error("expires at the end of the last day")
	}
	for _, bad := range []*Class{{ID: "Bad Id", Course: "x"}, {ID: "x"}, {ID: "x", Course: "c", TTL: "5s"}, {ID: "x", Course: "c", Expires: "05/12/2026"}} {
		if bad.Check() == nil {
			t.Errorf("no problem with %+v", bad)
		}
	}
	if ClassID("!!!", now) != "class-2026-11-23" {
		t.Error("empty name")
	}
	if err := RemoveClass(root, c.ID); err != nil || len(ListClasses(root)) != 0 {
		t.Error("remove")
	}
}

func TestFleetBudgetAndReap(t *testing.T) {
	f := &Fleet{Eng: fakeEngine(t), Spec: deck.Sandbox{Image: "have:1", Memory: 2 << 30}, Budget: 3 << 30}
	// one seat fits the budget (its box starts against the fake engine and
	// fails there); the second does not
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := f.Get(ctx, "AAAAAAAA"); err == nil || errors.Is(err, ErrBudget) {
		t.Fatalf("first seat: %v", err)
	}
	f.mu.Lock()
	f.seats["AAAAAAAA"] = &seatBox{ready: closed(), started: time.Now().Add(-3 * time.Hour), seen: time.Now()}
	f.mu.Unlock()
	if _, err := f.Get(ctx, "BBBBBBBB"); !errors.Is(err, ErrBudget) {
		t.Fatalf("over budget: %v", err)
	}
	if st := f.States()["AAAAAAAA"]; st.State != "starting" {
		t.Errorf("state %+v", st)
	}
	f.TTL = 2 * time.Hour
	if got := f.Reap(time.Now()); len(got) != 1 || f.Has("AAAAAAAA") {
		t.Errorf("ttl reap %v", got)
	}
	f.TTL, f.Idle = 0, 30*time.Minute
	f.seats["CCCCCCCC"] = &seatBox{ready: closed(), started: time.Now(), seen: time.Now().Add(-time.Hour), inflight: 1}
	if got := f.Reap(time.Now()); len(got) != 0 {
		t.Error("a seat with an open terminal stream is not idle")
	}
	f.End("CCCCCCCC")
	f.seats["CCCCCCCC"].seen = time.Now().Add(-time.Hour)
	if got := f.Reap(time.Now()); len(got) != 1 {
		t.Errorf("idle reap %v", got)
	}
}

func closed() chan struct{} { c := make(chan struct{}); close(c); return c }
