package deck

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Scenes: a ```scene fence in a slide or a document page. The first lines
// name the blocks (box Shop, row A, B, C, column …, grid …, box X at 30,60);
// every other line is one build step that moves things: a parcel flies, a
// token hops, a block morphs, splits, merges, swaps, opens, scatters, tiles
// assemble, a queue fills and a consumer takes from it, a message fans out,
// a route draws itself, a parcel drops into a box. ParseScene reads the lines,
// lays the blocks out (one row; a column for what a split or a fan-out
// reaches) and works out where every object is after every step, so the page
// can show any step at once (reduced motion, print, a #N.k jump) and
// web/scene.js only has to play the move from one step to the next. Units
// are the SVG's: the stage is SceneH tall and at least SceneW wide.

// SceneW and SceneH are the stage's least width and its height, in viewBox units.
const (
	SceneW = 320.0
	SceneH = 180.0
)

// SceneVerbs are the step verbs, for messages and docs.
var SceneVerbs = []string{"flies", "hops", "morphs", "splits", "merge", "swap", "assemble", "scatter", "queue", "takes", "fans out", "opens", "route", "drops into"}

// small objects: a step's subject may be one of these words without a line
// naming it; the plural (parcels merge) means every one on the stage
var sceneSmall = []string{"parcel", "token", "message"}

// Scene is one ```scene fence, read and laid out.
type Scene struct {
	Title  string
	W, H   float64
	Objs   []SceneObj
	Steps  []SceneStep
	States [][]SceneState // States[k]: every object after k steps
}

// SceneObj is one thing on the stage.
type SceneObj struct {
	Kind string // box, parcel, token, message, tile, piece, queue, trail, route, pin
	Name string // what steps call it (boxes, named small objects, queues)
	D    string // trail and route: the path
}

// SceneState is an object at one step: its centre, size, corner radius,
// turn, whether it shows, whether it is lit, its label and the label's place.
type SceneState struct {
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	W  float64 `json:"w,omitempty"`
	H  float64 `json:"h,omitempty"`
	R  float64 `json:"r,omitempty"`
	A  float64 `json:"a,omitempty"`
	V  bool    `json:"v,omitempty"`
	On bool    `json:"on,omitempty"`
	L  string  `json:"l,omitempty"`
	LY float64 `json:"ly,omitempty"`
}

// SceneStep is one step: the verb and what web/scene.js needs to play it.
type SceneStep struct {
	Line     int          `json:"-"`
	Text     string       `json:"-"`
	Verb     string       `json:"v"`
	Objs     []int        `json:"o,omitempty"`    // the objects that move
	Src      int          `json:"src"`            // the block it starts from (-1: none)
	Tgt      int          `json:"t"`              // the block it ends at (-1: none)
	From     []float64    `json:"from,omitempty"` // where the moving objects start
	Pts      [][]float64  `json:"pts,omitempty"`  // hops: the landing places
	Lights   []int        `json:"lights,omitempty"`
	Stages   []SceneState `json:"st,omitempty"` // morphs: each shape in turn
	Hide     []int        `json:"hide,omitempty"`
	Fill     []int        `json:"fill,omitempty"` // queue: messages that join
	Take     int          `json:"take,omitempty"` // queue: how many the consumer takes
	Lane     []int        `json:"lane,omitempty"` // queue: the line, front first, after joining
	Slots    []float64    `json:"slots,omitempty"`
	Stops    []float64    `json:"f,omitempty"` // route: where each pin drops, 0..1 of the way
	Straight bool         `json:"straight,omitempty"`
	Speed    float64      `json:"sp,omitempty"`
	Dur      int          `json:"d,omitempty"` // ms, the whole step
}

// a line as read, before names are looked up
type sceneLine struct {
	n     int
	text  string
	verb  string   // "" for a declaration
	subj  string   // the subject (flies, hops, morphs, …)
	names []string // the names after the verb
	count int      // queue: messages; takes: how many; assemble: columns
	rows  int      // assemble: rows
	take  string   // queue: the consumer named after ·
	opts  []string
}

type sceneBox struct {
	name, label string
	at          bool
	x, y        float64
	slot        int // layout slot (-1: none yet)
	stacked     bool
	group       string // column/grid group, or a split's targets
	hidden      bool   // appears later (assemble, opens)
	implicit    bool   // named by a step, not a line (opens, queue consumers)
	queue       bool
	qw          float64
	sw          float64 // a stacked block: its slot's width
}

var (
	sceneVerbRE = regexp.MustCompile(`(?i)^\s*(flies|fly|hops|hop|morphs|morph|splits|split|merges|merge|assembles|assemble|scatters|scatter|fans out|fan out|opens|open|drops into|drops in|drop into|drops|drop|takes|take)\b\s*(.*)$`)
	sceneAtRE   = regexp.MustCompile(`(?i)^(.*?)\s+at\s+(\d+(?:\.\d+)?)\s*%?\s*,\s*(\d+(?:\.\d+)?)\s*%?$`)
	sceneGridRE = regexp.MustCompile(`^(\d+)\s*[x×*]\s*(\d+)$`)
	sceneDurRE  = regexp.MustCompile(`^(\d+(?:\.\d+)?)(ms|s)$`)
	sceneTakeRE = regexp.MustCompile(`(?i)^(.+?)\s+takes?\s+(\d+)$`)
)

// sceneName strips quotes and spaces from a name as written.
func sceneName(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' && s[len(s)-1] == '"' || s[0] == '\'' && s[len(s)-1] == '\'') {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

func sceneList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = sceneName(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func sceneArrows(s string) []string {
	var out []string
	for _, p := range strings.Split(s, "→") {
		out = append(out, sceneName(p))
	}
	return out
}

// splitSubject cuts a line into its subject and the rest, a quoted subject
// kept whole.
func splitSubject(s string) (string, string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, `"`) {
		if i := strings.Index(s[1:], `"`); i >= 0 {
			return s[1 : i+1], s[i+2:]
		}
	}
	words := strings.Fields(s)
	for i := 1; i < len(words); i++ {
		if sceneVerbRE.MatchString(strings.Join(words[i:], " ")) {
			return strings.Join(words[:i], " "), strings.Join(words[i:], " ")
		}
	}
	return s, ""
}

func sceneSmallKind(word string) (kind string, plural bool) {
	w := strings.ToLower(word)
	for _, k := range sceneSmall {
		if w == k {
			return k, false
		}
		if w == k+"s" {
			return k, true
		}
	}
	return "", false
}

func boxWidth(label string) float64 {
	return math.Max(46, math.Min(124, float64(len([]rune(label)))*5.6+20))
}

func r1(x float64) float64 { return math.Round(x*10) / 10 }

// ParseScene reads a scene fence. Problems are si check errors, each with
// its line in the fence; the scene is still drawn as far as it can be.
func ParseScene(attrs map[string]string, body string) (*Scene, []string) {
	sc := &Scene{Title: attrs["title"], H: SceneH}
	var probs []string
	bad := func(n int, f string, a ...any) {
		probs = append(probs, fmt.Sprintf("scene line %d: ", n)+fmt.Sprintf(f, a...))
	}
	for k := range attrs {
		if k != "title" {
			probs = append(probs, fmt.Sprintf("scene: unknown option %s (use title=)", k))
		}
	}
	var boxes []*sceneBox
	byName := map[string]*sceneBox{}
	smallNamed := map[string]string{} // declared small objects: name -> kind
	addBox := func(n int, name, group string) *sceneBox {
		if b := byName[strings.ToLower(name)]; b != nil {
			if group != "" {
				b.group = group
			}
			return b
		}
		if _, ok := smallNamed[strings.ToLower(name)]; ok {
			bad(n, "%q is already a small object", name)
		}
		b := &sceneBox{name: name, label: name, slot: -1, group: group}
		boxes = append(boxes, b)
		byName[strings.ToLower(name)] = b
		return b
	}
	var lines []sceneLine
	groups := 0
	for i, raw := range strings.Split(body, "\n") {
		n := i + 1
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		t = strings.NewReplacer("<->", "↔", "->", "→", "=>", "→").Replace(t)
		parts := strings.Split(t, "·")
		head := strings.TrimSpace(parts[0])
		l := sceneLine{n: n, text: t}
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			if m := sceneTakeRE.FindStringSubmatch(p); m != nil {
				l.take = sceneName(m[1])
				l.count, _ = strconv.Atoi(m[2])
				continue
			}
			l.opts = append(l.opts, strings.Fields(strings.ReplaceAll(p, ",", " "))...)
		}
		first, rest, _ := strings.Cut(head, " ")
		rest = strings.TrimSpace(rest)
		switch strings.ToLower(first) {
		case "box", "boxes", "row":
			if m := sceneAtRE.FindStringSubmatch(rest); m != nil && strings.ToLower(first) == "box" {
				b := addBox(n, sceneName(m[1]), "")
				x, _ := strconv.ParseFloat(m[2], 64)
				y, _ := strconv.ParseFloat(m[3], 64)
				if x > 100 || y > 100 {
					bad(n, "at %s,%s: places are percent of the stage, 0 to 100", m[2], m[3])
				}
				b.at, b.x, b.y = true, x, y
				continue
			}
			names := sceneList(rest)
			if len(names) == 0 {
				bad(n, "%s needs a name (box Shop)", first)
			}
			for _, nm := range names {
				addBox(n, nm, "")
			}
			continue
		case "column", "grid":
			names := sceneList(rest)
			if len(names) == 0 {
				bad(n, "%s needs names (%s A, B, C)", first, first)
			}
			groups++
			g := fmt.Sprintf("%s%d", strings.ToLower(first), groups)
			for _, nm := range names {
				addBox(n, nm, g)
			}
			continue
		case "swap":
			a, b, ok := strings.Cut(rest, "↔")
			if !ok {
				bad(n, "write swap A ↔ B (or A <-> B)")
				continue
			}
			l.verb, l.names = "swap", []string{sceneName(a), sceneName(b)}
		case "queue":
			name := rest
			if f := strings.Fields(rest); len(f) > 1 {
				if c, err := strconv.Atoi(f[len(f)-1]); err == nil {
					l.rows = c // the messages that join (0: the default)
					name = strings.Join(f[:len(f)-1], " ")
				}
			}
			if sceneName(name) == "" {
				bad(n, "write queue NAME (queue orders · consumer takes 3)")
				continue
			}
			l.verb, l.subj = "queue", sceneName(name)
		case "route":
			l.verb, l.names = "route", sceneArrows(rest)
			if len(l.names) < 2 {
				bad(n, "a route goes through two or more blocks (route Depot → Stop 1 → Stop 2)")
				continue
			}
		default:
			subj, after := splitSubject(head)
			m := sceneVerbRE.FindStringSubmatch(after)
			if m == nil {
				if k, _ := sceneSmallKind(first); k != "" && !strings.Contains(rest, "→") && rest != "" {
					smallNamed[strings.ToLower(sceneName(rest))] = k
					continue
				}
				bad(n, "no verb in %q (use %s)", t, strings.Join(SceneVerbs, ", "))
				continue
			}
			l.subj = sceneName(subj)
			v, arg := strings.ToLower(m[1]), strings.TrimSpace(m[2])
			switch {
			case strings.HasPrefix(v, "fl"):
				l.verb = "flies"
				if strings.HasPrefix(arg, "→") {
					l.names = []string{"", sceneName(arg[len("→"):])}
				} else {
					l.names = sceneArrows(arg)
				}
				if len(l.names) != 2 {
					bad(n, "write %s flies A → B", l.subj)
					continue
				}
			case strings.HasPrefix(v, "hop"):
				l.verb, l.names = "hops", sceneArrows(arg)
				if len(l.names) < 2 {
					bad(n, "write %s hops A → B → C", l.subj)
					continue
				}
			case strings.HasPrefix(v, "morph"):
				l.verb = "morphs"
				for _, p := range sceneArrows(arg) {
					if p != "" {
						l.names = append(l.names, p)
					}
				}
				if len(l.names) == 0 {
					bad(n, "write %s morphs → Event circle → Message diamond", l.subj)
					continue
				}
			case strings.HasPrefix(v, "split"), strings.HasPrefix(v, "fan"), strings.HasPrefix(v, "open"):
				l.verb = map[byte]string{'s': "splits", 'f': "fans out", 'o': "opens"}[v[0]]
				l.names = sceneList(strings.TrimPrefix(strings.TrimSpace(arg), "→"))
				if len(l.names) == 0 {
					bad(n, "write %s %s → A, B, C", l.subj, l.verb)
					continue
				}
			case strings.HasPrefix(v, "merge"):
				l.verb = "merge"
				l.names = sceneList(strings.TrimPrefix(strings.TrimSpace(arg), "→"))
				if len(l.names) != 1 {
					bad(n, "write %s merge → Delivery (one block)", l.subj)
					continue
				}
			case strings.HasPrefix(v, "assemble"):
				l.verb = "assemble"
				g := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(arg), "→"))
				g = strings.TrimSpace(strings.TrimPrefix(g, "grid"))
				m := sceneGridRE.FindStringSubmatch(g)
				if m == nil {
					bad(n, "write %s assemble 3×3 (columns × rows)", l.subj)
					continue
				}
				l.count, _ = strconv.Atoi(m[1])
				l.rows, _ = strconv.Atoi(m[2])
				if l.count*l.rows < 1 || l.count*l.rows > 64 {
					bad(n, "assemble takes 1 to 64 tiles, not %d×%d", l.count, l.rows)
					continue
				}
			case strings.HasPrefix(v, "scatter"):
				l.verb = "scatter"
				if arg != "" {
					bad(n, "scatter takes nothing after it (Plan scatters)")
				}
			case strings.HasPrefix(v, "drop"):
				l.verb = "drops into"
				l.names = []string{sceneName(strings.TrimPrefix(strings.TrimSpace(arg), "→"))}
				if l.names[0] == "" {
					bad(n, "write %s drops into Locker", l.subj)
					continue
				}
			case strings.HasPrefix(v, "take"):
				l.verb = "takes"
				c, err := strconv.Atoi(strings.TrimSpace(arg))
				if err != nil || c < 1 {
					bad(n, "write %s takes 3 (a number)", l.subj)
					continue
				}
				l.count = c
			}
		}
		for _, o := range l.opts {
			lo := strings.ToLower(o)
			if lo != "arc" && lo != "straight" && lo != "slow" && lo != "fast" && !sceneDurRE.MatchString(lo) {
				bad(n, "unknown option %q (use arc, straight, slow, fast or a time: 2s, 800ms)", o)
			}
		}
		lines = append(lines, l)
	}

	// steps name these blocks without a line: the parts a block opens to
	// and a queue's consumer; a block that assembles appears with its tiles
	var queues []*sceneBox
	consumerOf := map[*sceneBox]*sceneBox{}
	lastQ := (*sceneBox)(nil)
	for _, l := range lines {
		switch l.verb {
		case "opens":
			for _, nm := range l.names {
				if byName[strings.ToLower(nm)] == nil {
					b := addBox(l.n, nm, "")
					b.implicit, b.hidden = true, true
				}
			}
		case "queue":
			q := byName[strings.ToLower(l.subj)]
			if q == nil {
				q = addBox(l.n, l.subj, "")
				q.queue, q.implicit = true, true
				queues = append(queues, q)
			}
			lastQ = q
			if l.take != "" {
				c := byName[strings.ToLower(l.take)]
				if c == nil {
					c = addBox(l.n, l.take, "")
					c.implicit = true
					if c.label == strings.ToLower(c.label) {
						c.label = strings.ToUpper(c.label[:1]) + c.label[1:]
					}
				}
				consumerOf[q] = c
			}
		case "takes":
			if lastQ != nil && consumerOf[lastQ] == nil {
				if c := byName[strings.ToLower(l.subj)]; c != nil {
					consumerOf[lastQ] = c
				}
			}
		case "assemble":
			if b := byName[strings.ToLower(l.subj)]; b != nil {
				b.hidden = true
			}
		case "splits", "fans out":
			groups++
			g := fmt.Sprintf("to%d", groups)
			for _, nm := range l.names {
				if b := byName[strings.ToLower(nm)]; b != nil && b.group == "" && !b.at {
					b.group = g
				}
			}
		}
	}
	for _, q := range queues {
		n := 0
		for _, l := range lines {
			if l.verb == "queue" && strings.EqualFold(l.subj, q.name) {
				if l.rows > 0 {
					n += l.rows
				} else if n == 0 {
					n += 5
				}
			}
		}
		q.qw = math.Max(110, float64(n)*30+20)
	}
	sc.layout(boxes, consumerOf)

	// objects: queues, then blocks, in that order (the boxes' order is the page's)
	s := &sceneSim{sc: sc, idx: map[string]int{}, small: map[string]int{}, box: map[int]*sceneBox{}}
	for _, b := range boxes {
		if b.queue {
			i := s.add(SceneObj{Kind: "queue", Name: b.name}, SceneState{X: b.x, Y: b.y, W: b.qw, H: 40, R: 20, V: true, L: b.label})
			s.idx[strings.ToLower(b.name)] = i
			s.box[i] = b
		}
	}
	for _, b := range boxes {
		if b.queue {
			continue
		}
		h, w := 30.0, boxWidth(b.label)
		if b.stacked {
			h, w = 26, math.Max(w, b.sw) // a column lines up
		}
		i := s.add(SceneObj{Kind: "box", Name: b.name}, SceneState{X: b.x, Y: b.y, W: w, H: h, R: 7, V: !b.hidden, L: b.label})
		s.idx[strings.ToLower(b.name)] = i
		s.box[i] = b
	}
	for nm, k := range smallNamed {
		s.small[nm] = s.add(SceneObj{Kind: k, Name: nm}, SceneState{})
	}
	sc.States = [][]SceneState{append([]SceneState(nil), s.cur...)}
	for _, l := range lines {
		if l.verb == "" {
			continue
		}
		s.line = l.n
		s.err = nil
		st := SceneStep{Line: l.n, Text: l.text, Verb: l.verb, Src: -1, Tgt: -1}
		for _, o := range l.opts {
			switch o = strings.ToLower(o); {
			case o == "straight":
				st.Straight = true
			case o == "slow":
				st.Speed = 1.6
			case o == "fast":
				st.Speed = 0.6
			case sceneDurRE.MatchString(o):
				m := sceneDurRE.FindStringSubmatch(o)
				v, _ := strconv.ParseFloat(m[1], 64)
				if m[2] == "s" {
					v *= 1000
				}
				st.Dur = int(v)
			}
		}
		for i := range s.cur { // a trail shows for the step that drew it
			if sc.Objs[i].Kind == "trail" {
				s.cur[i].V = false
			}
		}
		s.step(&st, l, consumerOf, lastQueue(lines, l))
		for _, e := range s.err {
			bad(l.n, "%s", e)
		}
		sc.Steps = append(sc.Steps, st)
		sc.States = append(sc.States, append([]SceneState(nil), s.cur...))
	}
	return sc, probs
}

func lastQueue(lines []sceneLine, upto sceneLine) string {
	q := ""
	for _, l := range lines {
		if l.n > upto.n {
			break
		}
		if l.verb == "queue" {
			q = l.subj
		}
	}
	return q
}

// layout places the blocks: by at x,y if given; otherwise in slots left to
// right, one block per slot, a column or grid group (and the blocks a split
// or a fan-out reaches) stacked in slots of their own, a queue's lane with
// its consumer in the slot after it.
func (sc *Scene) layout(boxes []*sceneBox, consumerOf map[*sceneBox]*sceneBox) {
	type slot struct {
		boxes []*sceneBox
		w     float64
	}
	var slots []*slot
	isConsumer := map[*sceneBox]bool{}
	for _, c := range consumerOf {
		isConsumer[c] = true
	}
	put := func(bs ...*sceneBox) {
		s := &slot{}
		for _, b := range bs {
			if b.slot >= 0 || b.at {
				continue
			}
			b.slot = len(slots)
			b.stacked = len(bs) > 1
			s.boxes = append(s.boxes, b)
			w := boxWidth(b.label)
			if b.queue {
				w = b.qw
			}
			s.w = math.Max(s.w, w)
		}
		if len(s.boxes) > 0 {
			slots = append(slots, s)
		}
	}
	for _, b := range boxes {
		if b.slot >= 0 || b.at || b.implicit && !b.queue || isConsumer[b] && !b.queue {
			continue
		}
		if b.group != "" {
			var g []*sceneBox
			for _, o := range boxes {
				if o.group == b.group && !o.at && o.slot < 0 {
					g = append(g, o)
				}
			}
			if strings.HasPrefix(b.group, "grid") {
				cols := int(math.Ceil(math.Sqrt(float64(len(g)))))
				rows := (len(g) + cols - 1) / cols
				for c := 0; c < cols; c++ {
					var col []*sceneBox
					for r := 0; r < rows && c*rows+r < len(g); r++ {
						col = append(col, g[c*rows+r])
					}
					put(col...)
					for _, x := range col {
						x.stacked = rows > 1
					}
				}
			} else {
				put(g...)
			}
			continue
		}
		put(b)
		if c := consumerOf[b]; c != nil {
			put(c)
		}
	}
	for _, b := range boxes { // consumers of no queue in a slot, and the rest
		if !b.implicit || isConsumer[b] {
			put(b)
		}
	}
	gap, margin := 20.0, 16.0
	total := 0.0
	for _, s := range slots {
		total += s.w
	}
	if len(slots) > 1 {
		total += gap * float64(len(slots)-1)
	}
	sc.W = math.Max(SceneW, total+2*margin)
	if len(slots) > 0 {
		spare := (sc.W - 2*margin - total) / float64(len(slots)+1) // spread out when there is room
		x := margin + spare
		for _, s := range slots {
			n := len(s.boxes)
			pitch := math.Min(50, 132/float64(n))
			for i, b := range s.boxes {
				b.x, b.sw = r1(x+s.w/2), s.w
				b.y = r1(104 + (float64(i)-float64(n-1)/2)*pitch)
			}
			x += s.w + gap + spare
		}
	}
	for _, b := range boxes {
		if b.at {
			b.x, b.y = r1(b.x*sc.W/100), r1(b.y*sc.H/100)
		}
	}
	// the parts a block opens to sit in a row inside it, opened
	for _, b := range boxes {
		if b.implicit && !b.queue && !isConsumer[b] && b.slot < 0 {
			b.x, b.y, b.slot = sc.W/2, 112, -2
		}
	}
}

// sceneSim works out every object's place step by step.
type sceneSim struct {
	sc     *Scene
	cur    []SceneState
	idx    map[string]int    // boxes and queues by lower-case name
	small  map[string]int    // small objects by lower-case name
	box    map[int]*sceneBox // the block behind an object
	lanes  map[int][]int     // a queue's line, front first
	filled map[int]bool      // queues that have had messages
	line   int
	err    []string
}

func (s *sceneSim) add(o SceneObj, st SceneState) int {
	s.sc.Objs = append(s.sc.Objs, o)
	s.cur = append(s.cur, st)
	hid := st
	hid.V = false
	for k := range s.sc.States { // earlier steps: not there yet
		s.sc.States[k] = append(s.sc.States[k], hid)
	}
	if o.Kind == "box" || o.Kind == "queue" {
		for k := range s.sc.States {
			s.sc.States[k][len(s.cur)-1].V = st.V
		}
	}
	return len(s.cur) - 1
}

func (s *sceneSim) fail(f string, a ...any) { s.err = append(s.err, fmt.Sprintf(f, a...)) }

func (s *sceneSim) names() string {
	var out []string
	for i, o := range s.sc.Objs {
		if o.Kind == "box" && s.box[i] != nil {
			out = append(out, s.box[i].name)
		}
	}
	if len(out) == 0 {
		return "none yet: name them first, box Shop"
	}
	return strings.Join(out, ", ")
}

// block finds a box by name, or says what there is.
func (s *sceneSim) block(name string) int {
	if i, ok := s.idx[strings.ToLower(name)]; ok && s.sc.Objs[i].Kind == "box" {
		return i
	}
	s.fail("unknown block %q (blocks: %s)", name, s.names())
	return -1
}

// thing finds a small object (a kind word makes one) or a block.
func (s *sceneSim) thing(name string) (int, bool) {
	ln := strings.ToLower(name)
	if i, ok := s.small[ln]; ok {
		return i, true
	}
	if k, plural := sceneSmallKind(name); k != "" && !plural {
		i := s.add(SceneObj{Kind: k, Name: ln}, SceneState{})
		s.small[ln] = i
		return i, true
	}
	if i, ok := s.idx[ln]; ok && s.sc.Objs[i].Kind == "box" {
		return i, false
	}
	s.fail("unknown name %q (blocks: %s; small objects: parcel, token, message)", name, s.names())
	return -1, false
}

// dock is where a small object rests at a block: on top of it, or at its
// left side when the block is one of a column.
func (s *sceneSim) dock(b int) []float64 {
	c := s.cur[b]
	if bx := s.box[b]; bx != nil && bx.stacked {
		return []float64{r1(c.X - c.W/2 - 13), c.Y}
	}
	return []float64{c.X, r1(c.Y - c.H/2 - 10)}
}

func (s *sceneSim) at(o int, p []float64) {
	s.cur[o].X, s.cur[o].Y, s.cur[o].V = p[0], p[1], true
}

func sceneShape(label, shape string, h float64) SceneState {
	switch shape {
	case "circle":
		d := h * 1.75
		return SceneState{W: d, H: d, R: d / 2, L: label}
	case "diamond":
		d := h * 1.5
		return SceneState{W: r1(d), H: r1(d), R: 4, A: 45, L: label}
	case "pill":
		return SceneState{W: boxWidth(label) + 8, H: h, R: h / 2, L: label}
	}
	return SceneState{W: boxWidth(label), H: h, R: 7, L: label}
}

func (s *sceneSim) step(st *SceneStep, l sceneLine, consumerOf map[*sceneBox]*sceneBox, qname string) {
	switch l.verb {
	case "flies":
		o, small := s.thing(l.subj)
		if o < 0 {
			return
		}
		if !small {
			s.fail("%q is a block; parcels, tokens and messages fly (box lines name blocks)", l.subj)
			return
		}
		from := []float64{s.cur[o].X, s.cur[o].Y}
		if l.names[0] != "" {
			if b := s.block(l.names[0]); b >= 0 {
				from = s.dock(b)
			}
		} else if !s.cur[o].V {
			s.fail("%s is not on the stage yet: write %s flies A → B", l.subj, l.subj)
		}
		b := s.block(l.names[1])
		if b < 0 {
			return
		}
		to := s.dock(b)
		d := fmt.Sprintf("M%g %g L%g %g", from[0], from[1], to[0], to[1])
		if !st.Straight {
			top := math.Max(8, math.Min(from[1], to[1])-math.Max(40, math.Abs(to[0]-from[0])*0.4))
			dx := to[0] - from[0]
			d = fmt.Sprintf("M%g %g C%g %g %g %g %g %g", from[0], from[1], r1(from[0]+dx*0.2), r1(top), r1(to[0]-dx*0.2), r1(top), to[0], to[1])
		}
		tr := s.add(SceneObj{Kind: "trail", D: d}, SceneState{})
		s.cur[tr].V = true
		st.Objs, st.From, st.Tgt = []int{o, tr}, from, b
		s.at(o, to)
	case "hops":
		o, small := s.thing(l.subj)
		if o < 0 {
			return
		}
		if !small {
			s.fail("%q is a block; parcels, tokens and messages hop", l.subj)
			return
		}
		st.Objs = []int{o}
		for _, nm := range l.names {
			b := s.block(nm)
			if b < 0 {
				return
			}
			st.Pts = append(st.Pts, s.dock(b))
			st.Lights = append(st.Lights, b)
		}
		for _, b := range st.Lights {
			s.cur[b].On = true
		}
		s.at(o, st.Pts[len(st.Pts)-1])
	case "morphs":
		b := s.block(l.subj)
		if b < 0 {
			return
		}
		c := s.cur[b]
		shape := "box"
		switch {
		case c.A == 45:
			shape = "diamond"
		case c.R >= c.W/2-0.5 && c.W == c.H:
			shape = "circle"
		}
		for _, p := range l.names {
			p = strings.NewReplacer("(", " ", ")", " ").Replace(p)
			f := strings.Fields(p)
			label := strings.Join(f, " ")
			if len(f) > 0 {
				switch w := strings.ToLower(f[len(f)-1]); w {
				case "box", "square", "rect", "circle", "diamond", "pill":
					shape = map[string]string{"square": "box", "rect": "box"}[w]
					if shape == "" {
						shape = w
					}
					label = strings.Join(f[:len(f)-1], " ")
				}
			}
			if label == "" {
				label = c.L
			}
			g := sceneShape(label, shape, s.cur[b].H)
			if shape != "box" && shape != "pill" {
				g = sceneShape(label, shape, 30)
			}
			st.Stages = append(st.Stages, g)
			s.idx[strings.ToLower(label)] = b // the new name finds it too
		}
		last := st.Stages[len(st.Stages)-1]
		s.cur[b].W, s.cur[b].H, s.cur[b].R, s.cur[b].A, s.cur[b].L = last.W, last.H, last.R, last.A, last.L
		st.Objs = []int{b}
	case "splits", "fans out":
		src, small := s.thing(l.subj)
		if src < 0 {
			return
		}
		kind := "parcel"
		if l.verb == "fans out" {
			kind = "message"
		}
		if small {
			kind = s.sc.Objs[src].Kind
			if !s.cur[src].V {
				s.fail("%s is not on the stage yet", l.subj)
			}
			st.From = []float64{s.cur[src].X, s.cur[src].Y}
			s.cur[src].V = false
			st.Hide = []int{src}
		} else {
			st.From = []float64{s.cur[src].X, s.cur[src].Y}
			st.Src = src
			d := s.dock(src) // what flew in goes into the block
			for i, o := range s.sc.Objs {
				if (o.Kind == "parcel" || o.Kind == "token" || o.Kind == "message") && s.cur[i].V && s.cur[i].X == d[0] && s.cur[i].Y == d[1] {
					s.cur[i].V = false
					st.Hide = append(st.Hide, i)
				}
			}
			if l.verb == "fans out" {
				s.cur[src].On = true
			}
		}
		if len(l.names) < 2 {
			s.fail("%s reaches two or more blocks", l.verb)
		}
		for _, nm := range l.names {
			b := s.block(nm)
			if b < 0 {
				continue
			}
			o := s.add(SceneObj{Kind: kind}, SceneState{X: st.From[0], Y: st.From[1]})
			s.at(o, s.dock(b))
			st.Objs = append(st.Objs, o)
		}
	case "merge":
		t := s.block(l.names[0])
		if t < 0 {
			return
		}
		var objs []int
		if k, plural := sceneSmallKind(l.subj); plural {
			for i, o := range s.sc.Objs {
				if o.Kind == k && s.cur[i].V {
					objs = append(objs, i)
				}
			}
			if len(objs) == 0 { // three of them, waiting at the left
				c := s.cur[t]
				for _, d := range [][]float64{{-200, -60}, {-210, 10}, {-160, 60}} {
					p := []float64{r1(math.Max(16, math.Min(s.sc.W-16, c.X+d[0]))), r1(math.Max(16, math.Min(s.sc.H-16, c.Y+d[1])))}
					o := s.add(SceneObj{Kind: k}, SceneState{X: p[0], Y: p[1], V: true})
					for j := range s.sc.States {
						s.sc.States[j][o].V = true
					}
					objs = append(objs, o)
				}
			}
		} else {
			for _, nm := range sceneList(l.subj) {
				if o, _ := s.thing(nm); o >= 0 {
					objs = append(objs, o)
				}
			}
		}
		for _, o := range objs {
			s.cur[o].X, s.cur[o].Y, s.cur[o].V = s.cur[t].X, s.cur[t].Y, false
		}
		s.cur[t].On, s.cur[t].V = true, true
		st.Objs, st.Tgt = objs, t
	case "swap":
		a, b := s.block(l.names[0]), s.block(l.names[1])
		if a < 0 || b < 0 {
			return
		}
		s.cur[a].X, s.cur[a].Y, s.cur[b].X, s.cur[b].Y = s.cur[b].X, s.cur[b].Y, s.cur[a].X, s.cur[a].Y
		st.Objs = []int{a, b}
	case "assemble":
		cx, cy := s.sc.W/2, 104.0
		t := -1
		if ls := strings.ToLower(l.subj); ls != "tiles" && ls != "tile" {
			if t = s.block(l.subj); t < 0 {
				return
			}
			cx, cy = s.cur[t].X, s.cur[t].Y
		}
		cols, rows := l.count, l.rows
		pitch := math.Min(36, math.Min(140/float64(rows), 220/float64(cols)))
		for i := 0; i < cols*rows; i++ {
			x := r1(cx + (float64(i%cols)-float64(cols-1)/2)*pitch)
			y := r1(cy + (float64(i/cols)-float64(rows-1)/2)*pitch)
			o := s.add(SceneObj{Kind: "tile"}, SceneState{X: x, Y: y, W: r1(pitch * 0.78), H: r1(pitch * 0.78), R: 4})
			s.cur[o].V = t < 0
			if i == (cols*rows)/2 {
				s.cur[o].On = true
			}
			st.Objs = append(st.Objs, o)
		}
		if t >= 0 {
			s.cur[t].V = true
			st.Tgt = t
		}
	case "scatter":
		ls := strings.ToLower(l.subj)
		if ls == "tiles" || ls == "tile" {
			for i, o := range s.sc.Objs {
				if o.Kind == "tile" && s.cur[i].V {
					st.Objs = append(st.Objs, i)
					s.cur[i].V = false
				}
			}
			if len(st.Objs) == 0 {
				s.fail("no tiles on the stage to scatter (tiles assemble 3×3 first)")
			}
			st.From = []float64{s.sc.W / 2, 104}
			return
		}
		b, small := s.thing(l.subj)
		if b < 0 {
			return
		}
		c := s.cur[b]
		st.From, st.Src = []float64{c.X, c.Y}, b
		s.cur[b].V = false
		if small {
			st.Objs = []int{b}
			return
		}
		pw, ph := c.W/4, c.H/4
		if ph < 8 {
			ph = c.H / 2
		}
		for y := c.Y - c.H/2 + ph/2; y < c.Y+c.H/2; y += ph {
			for x := c.X - c.W/2 + pw/2; x < c.X+c.W/2; x += pw {
				o := s.add(SceneObj{Kind: "piece"}, SceneState{X: r1(x), Y: r1(y), W: r1(pw - 1), H: r1(ph - 1), R: 2})
				st.Objs = append(st.Objs, o)
			}
		}
	case "queue", "takes":
		qi, ok := s.idx[strings.ToLower(qname)]
		if !ok || s.sc.Objs[qi].Kind != "queue" {
			s.fail("%s takes from a queue: write queue NAME first", l.subj)
			return
		}
		if s.lanes == nil {
			s.lanes, s.filled = map[int][]int{}, map[int]bool{}
		}
		q := s.cur[qi]
		right := q.X + q.W/2 - 20
		slot := func(k int) float64 { return r1(right - float64(k)*30) }
		lane := s.lanes[qi]
		if l.verb == "queue" {
			n := l.rows
			if n == 0 && !s.filled[qi] {
				n = 5
			}
			for i := 0; i < n; i++ {
				o := s.add(SceneObj{Kind: "message"}, SceneState{X: r1(q.X - q.W/2 - 16), Y: q.Y})
				lane = append(lane, o)
				st.Fill = append(st.Fill, o)
			}
		}
		st.Lane = append([]int(nil), lane...)
		for k := range lane {
			st.Slots = append(st.Slots, slot(k))
		}
		st.Src, st.From = qi, []float64{r1(q.X - q.W/2 - 16), q.Y}
		take := l.count
		cname := l.take
		if l.verb == "takes" {
			cname = l.subj
		}
		if take > 0 {
			c := s.block(cname)
			if c < 0 {
				return
			}
			if cb := consumerOf[s.box[qi]]; cb != nil && !strings.EqualFold(cb.name, cname) && l.verb == "takes" {
				s.fail("%s does not take from %s (its consumer is %s)", cname, qname, cb.name)
			}
			if take > len(lane) {
				s.fail("%s takes %d but the queue holds %d", cname, take, len(lane))
				take = len(lane)
			}
			st.Tgt, st.Take = c, take
			for _, o := range lane[:take] {
				s.cur[o].X, s.cur[o].Y, s.cur[o].V = s.cur[c].X, s.cur[c].Y, false
			}
			lane = lane[take:]
		}
		for k, o := range lane {
			s.cur[o].X, s.cur[o].Y, s.cur[o].V = slot(k), q.Y, true
		}
		s.lanes[qi] = lane
		s.filled[qi] = true
	case "opens":
		b := s.block(l.subj)
		if b < 0 {
			return
		}
		w, h := s.sc.W-60, 118.0
		s.cur[b].X, s.cur[b].Y, s.cur[b].W, s.cur[b].H, s.cur[b].LY, s.cur[b].A, s.cur[b].R = s.sc.W/2, 106, w, h, -h/2+13, 0, 8
		s.cur[b].V = true
		var kids []int
		for _, nm := range l.names {
			if k := s.block(nm); k >= 0 {
				kids = append(kids, k)
			}
		}
		n := float64(len(kids))
		pitch := math.Min(80, (w-20)/n)
		for i, k := range kids {
			s.cur[k].X = r1(s.sc.W/2 + (float64(i)-(n-1)/2)*pitch)
			s.cur[k].Y = 114
			s.cur[k].W = math.Min(s.cur[k].W, pitch-10)
			s.cur[k].V, s.cur[k].On = true, true
		}
		for i := range s.cur {
			if i != b && s.cur[i].V && !contains3(kids, i) {
				s.cur[i].V = false
				st.Hide = append(st.Hide, i)
			}
		}
		st.Objs, st.Src = kids, b
	case "route":
		var pts [][]float64
		var stops []int
		for _, nm := range l.names {
			b := s.block(nm)
			if b < 0 {
				return
			}
			pts = append(pts, []float64{s.cur[b].X, r1(s.cur[b].Y + s.cur[b].H/2 + 12)}) // the road runs under the blocks
			stops = append(stops, b)
		}
		d := fmt.Sprintf("M%g %g", pts[0][0], pts[0][1])
		total, lens := 0.0, []float64{}
		for i := 1; i < len(pts); i++ {
			d += fmt.Sprintf(" H%g V%g", pts[i][0], pts[i][1])
			total += math.Abs(pts[i][0]-pts[i-1][0]) + math.Abs(pts[i][1]-pts[i-1][1])
			lens = append(lens, total)
		}
		ro := s.add(SceneObj{Kind: "route", D: d}, SceneState{})
		s.cur[ro].V = true
		rider := s.add(SceneObj{Kind: "token"}, SceneState{X: pts[0][0], Y: pts[0][1]})
		s.at(rider, pts[len(pts)-1])
		st.Objs = []int{ro, rider}
		for i := range stops[1:] {
			p := s.add(SceneObj{Kind: "pin"}, SceneState{X: pts[i+1][0], Y: pts[i+1][1] - 4})
			s.cur[p].V = true
			st.Lights = append(st.Lights, p)
			f := 1.0
			if total > 0 {
				f = lens[i] / total
			}
			st.Stops = append(st.Stops, math.Round(f*1000)/1000)
		}
	case "drops into":
		o, small := s.thing(l.subj)
		if o < 0 {
			return
		}
		if !small {
			s.fail("%q is a block; parcels, tokens and messages drop", l.subj)
			return
		}
		b := s.block(l.names[0])
		if b < 0 {
			return
		}
		s.at(o, []float64{s.cur[b].X, s.cur[b].Y})
		s.cur[b].LY = r1(s.cur[b].H/2 + 8) // its name moves under it, as on a locker
		st.Objs, st.Tgt = []int{o}, b
	}
}

func contains3(xs []int, x int) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
