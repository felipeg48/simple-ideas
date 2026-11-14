package studio

import (
	"path/filepath"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// New… → Course writes the course si new --type course writes, with the
// name as its title, and opens its _course.md.
func TestStarterCourse(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "web101")
	created, open, _, err := WriteStarter("course", abs, "Web for beginners", root)
	if err != nil {
		t.Fatal(err)
	}
	if open != filepath.Join(abs, deck.CourseFile) || len(created) < 5 {
		t.Fatalf("open %s, created %d", open, len(created))
	}
	g, err := deck.LoadGuide(abs)
	if err != nil || !g.IsCourse() || g.Meta.Title != "Web for beginners" || len(g.Problems) != 0 {
		t.Fatalf("course: %v %+v", err, g)
	}
}
