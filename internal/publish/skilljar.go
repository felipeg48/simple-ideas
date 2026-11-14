package publish

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

// SkillJar publishes a course through Skilljar's REST API
// (https://api.skilljar.com/docs/, spec at /docs/schema.yml): HTTP Basic
// with the API key as the user name and no password. Lessons are one flat,
// ordered list per course; a module is a lesson of type SECTION. The API
// takes no file uploads: a video (an asset) and a SCORM zip (a web package)
// are fetched by Skilljar from a content_url, so the files must be online
// first (si publish --base-url).
type SkillJar struct{ API, Key string }

// NewSkillJar is a client for the real Skilljar API.
func NewSkillJar(key string) *SkillJar {
	return &SkillJar{API: "https://api.skilljar.com/v1", Key: key}
}

// SJCourse is a course as Skilljar gets it.
type SJCourse struct {
	Title       string
	Description string // short_description
	LongHTML    string // long_description_html
	Duration    string // manual_course_time, shown on the course ("4h"; Skilljar keeps 15 characters)
	Lessons     []SJLesson
}

// SJLesson is one lesson, in course order.
type SJLesson struct {
	Key     string // stable across publishes (the lesson's path, "module-1"): the ids file's key
	Title   string
	Type    string // SECTION, HTML, ASSET (a video), WEB_PACKAGE (a SCORM zip)
	HTML    string // HTML: content_html
	URL     string // ASSET, WEB_PACKAGE: where Skilljar fetches the file
	Hash    string // ASSET, WEB_PACKAGE: the file's SHA-256; the same hash reuses the uploaded one
	Seconds int    // time_seconds
}

// SJIDs are the ids of an earlier publish (.si/publish/skilljar.json beside
// the course), so a publish updates in place.
type SJIDs struct {
	Course   string            `json:"course,omitempty"`
	Lessons  map[string]string `json:"lessons,omitempty"`  // lesson key -> lesson id
	Contents map[string]string `json:"contents,omitempty"` // lesson key -> "hash id" of its asset or web package
}

// LoadSJIDs reads the ids file; a missing file is a first publish.
func LoadSJIDs(path string) (*SJIDs, error) {
	ids := &SJIDs{}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ids, nil
	}
	if err != nil {
		return nil, err
	}
	return ids, json.Unmarshal(raw, ids)
}

// Save writes the ids file.
func (ids *SJIDs) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(ids, "", "  ")
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

func (s *SkillJar) req(ctx context.Context, method, path string, v any, dry io.Writer, fake string) (map[string]any, error) {
	var req *http.Request
	var body []byte
	var err error
	if v != nil {
		req, body, err = jsonReq(method, s.API+path, v)
	} else {
		req, err = http.NewRequest(method, s.API+path, nil)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.Key+":")))
	out, _, err := do(ctx, req, body, dry, fake)
	if err != nil {
		return nil, err
	}
	var r map[string]any
	if len(out) > 0 {
		json.Unmarshal(out, &r)
	}
	return r, nil
}

func idOf(r map[string]any) string {
	switch v := r["id"].(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	}
	return ""
}

// Publish creates the course or updates the one in ids: the course fields,
// every lesson in order (created, or patched in place), the videos and SCORM
// packages whose file changed (Skilljar fetches them from their URL), and
// deletes the lessons of the earlier publish that are gone. ids is updated
// as it goes; the caller saves it. It returns the course id.
func (s *SkillJar) Publish(ctx context.Context, c SJCourse, ids *SJIDs, dry io.Writer) (string, error) {
	if ids.Lessons == nil {
		ids.Lessons = map[string]string{}
	}
	if ids.Contents == nil {
		ids.Contents = map[string]string{}
	}
	course := map[string]any{"title": c.Title, "short_description": c.Description, "enforce_sequential_navigation": false}
	if c.LongHTML != "" {
		course["long_description_html"] = c.LongHTML
	}
	if d := c.Duration; d != "" {
		if len(d) > 15 {
			d = d[:15]
		}
		course["manual_course_time"], course["display_manual_course_time"] = d, true
	}
	if ids.Course == "" {
		r, err := s.req(ctx, "POST", "/courses", course, dry, `{"id":"<new course id>"}`)
		if err != nil {
			return "", err
		}
		if ids.Course = idOf(r); ids.Course == "" {
			return "", fmt.Errorf("skilljar: no course id in the answer")
		}
	} else if _, err := s.req(ctx, "PATCH", "/courses/"+ids.Course, course, dry, `{}`); err != nil {
		return "", err
	}
	keep := map[string]bool{}
	for i, l := range c.Lessons {
		keep[l.Key] = true
		lesson := map[string]any{"title": l.Title, "order": i + 1, "type": l.Type}
		if l.Seconds > 0 {
			lesson["time_seconds"] = l.Seconds
		}
		switch l.Type {
		case "HTML":
			lesson["content_html"] = l.HTML
		case "ASSET", "WEB_PACKAGE":
			id := ""
			var prevHash string
			if v := ids.Contents[l.Key]; v != "" {
				fmt.Sscan(v, &prevHash, &id)
			}
			if prevHash != l.Hash || id == "" { // a new or changed file: Skilljar fetches it again
				path, body, fake := "/assets", map[string]any{"content_url": l.URL, "asset": map[string]any{"name": l.Title, "aspect_ratio": "16:9", "sync_completion": true}}, `{"id":"<new asset id>"}`
				if l.Type == "WEB_PACKAGE" {
					path, body, fake = "/web-packages", map[string]any{"content_url": l.URL, "web_package": map[string]any{"title": l.Title, "redirect_on_completion": true, "sync_completion": true}}, `{"id":"<new web package id>"}`
				}
				r, err := s.req(ctx, "POST", path, body, dry, fake)
				if err != nil {
					return ids.Course, err
				}
				id = idOf(r)
				ids.Contents[l.Key] = l.Hash + " " + id
			}
			if l.Type == "ASSET" {
				lesson["content_asset_id"] = id
			} else {
				lesson["content_web_package_id"] = id
			}
		}
		if id := ids.Lessons[l.Key]; id != "" {
			if _, err := s.req(ctx, "PATCH", "/lessons/"+id, lesson, dry, `{}`); err != nil {
				return ids.Course, err
			}
			continue
		}
		lesson["course_id"] = ids.Course
		r, err := s.req(ctx, "POST", "/lessons", lesson, dry, `{"id":"<new lesson id>"}`)
		if err != nil {
			return ids.Course, err
		}
		ids.Lessons[l.Key] = idOf(r)
	}
	var gone []string
	for key := range ids.Lessons {
		if !keep[key] {
			gone = append(gone, key)
		}
	}
	sort.Strings(gone)
	for _, key := range gone {
		id := ids.Lessons[key]
		if _, err := s.req(ctx, "DELETE", "/lessons/"+id, nil, dry, ``); err != nil {
			return ids.Course, err
		}
		delete(ids.Lessons, key)
		delete(ids.Contents, key)
	}
	return ids.Course, nil
}
