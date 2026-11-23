package deck

import (
	"fmt"
	"regexp"
	"strings"
)

// ```sequence: participants (named as they appear, or by participants:) and
// messages, one per step: `A → B: text` a call, `A --> B: text` a reply,
// `· 380 ms` a timing badge, `note A: text` a note on a lifeline.

var seqMsgRE = regexp.MustCompile(`^(.+?)\s*(-->|->|→|⇢)\s*(.+?)\s*:\s*(.*)$`)
var seqNoteRE = regexp.MustCompile(`(?i)^note\s+(?:over\s+)?(.+?)\s*:\s*(.*)$`)

const (
	sPad, sHeadH, sRow = 12.0, 36.0, 46.0
)

func parseSequence(attrs map[string]string, body string) (*Diagram, []string) {
	d := &Diagram{Kind: "sequence", Title: attrs["title"]}
	sW := dgWidth(attrs, 1000)
	row := sRow
	if sW < 800 { // a narrow (portrait) layout: its text is larger, give it room
		row = 70
	}
	var probs []string
	var parts []string
	idx := func(name string) int {
		name = strings.TrimSpace(name)
		for i, p := range parts {
			if strings.EqualFold(p, name) {
				return i
			}
		}
		parts = append(parts, name)
		return len(parts) - 1
	}
	type msg struct {
		from, to    int
		text, badge string
		reply, note bool
	}
	var msgs []msg
	for _, l := range diagramLines(body) {
		low := strings.ToLower(l)
		switch {
		case strings.HasPrefix(low, "participants:"):
			for _, p := range strings.Split(l[len("participants:"):], ",") {
				if p = strings.TrimSpace(p); p != "" {
					idx(p)
				}
			}
		case seqNoteRE.MatchString(l):
			m := seqNoteRE.FindStringSubmatch(l)
			i := idx(m[1])
			msgs = append(msgs, msg{from: i, to: i, text: m[2], note: true})
		case seqMsgRE.MatchString(l):
			m := seqMsgRE.FindStringSubmatch(l)
			f := dotFields(m[4])
			text, badge := "", ""
			if len(f) > 0 {
				text = f[0]
			}
			if len(f) > 1 {
				badge = strings.Join(f[1:], " · ")
			}
			msgs = append(msgs, msg{from: idx(m[1]), to: idx(m[3]), text: text, badge: badge, reply: m[2] == "-->"})
		default:
			probs = append(probs, fmt.Sprintf("sequence %q: write A → B: what is sent (--> for a reply), or note A: text", l))
		}
	}
	if len(parts) == 0 {
		return &Diagram{Kind: "sequence", W: 400, H: 60, States: [][]DState{{}}}, append(probs, "sequence: no messages (Browser → Shop: POST /orders)")
	}
	top := sPad
	if d.Title != "" {
		top += 22
	}
	colW := (sW - 2*sPad) / float64(len(parts))
	px := func(i int) float64 { return sPad + colW*(float64(i)+.5) }
	headW := min(colW-16, 170)
	firstY := top + sHeadH + 34
	d.W, d.H = sW, firstY+float64(len(msgs))*row+sPad
	var st0 []DState
	add := func(e DEl, s DState) { d.Els = append(d.Els, e); st0 = append(st0, s) }
	for i := range parts {
		add(DEl{K: "line", C: "si-dg-life", D: fmt.Sprintf("M0 0V%g", d.H-sPad-top-sHeadH)}, DState{X: px(i), Y: top + sHeadH, O: 1, P: 1})
	}
	headAt := len(d.Els)
	for i, p := range parts {
		add(DEl{K: "box", C: "si-dg-actor", R: 8}, DState{X: px(i) - headW/2, Y: top, W: headW, H: sHeadH, O: 1, T: p})
	}
	msgAt := len(d.Els)
	for k, m := range msgs {
		y := firstY + float64(k)*row
		a, b := px(m.from), px(m.to)
		switch {
		case m.note:
			w := min(colW*.9, 260)
			add(DEl{K: "box", C: "si-dg-note", R: 4}, DState{X: a - w/2, Y: y - 16, W: w, H: 28, T: m.text})
		case m.from == m.to:
			add(DEl{K: "line", C: "si-dg-msg is-self", D: "M0 0h46v18h-46M9 13l-9 5l9 5", TX: 54, TY: 9}, DState{X: a, Y: y - 6, T: m.text, U: m.badge})
		default:
			L := b - a
			path := fmt.Sprintf("M0 0H%g M%g -5L%g 0L%g 5", L, L-9, L, L-9)
			if L < 0 {
				path = fmt.Sprintf("M0 0H%g M%g -5L%g 0L%g 5", L, L+9, L, L+9)
			}
			cls := "si-dg-msg"
			if m.reply {
				cls += " is-reply"
			}
			add(DEl{K: "line", C: cls, D: path, TX: L / 2, TY: -9}, DState{X: a, Y: y, T: m.text, U: m.badge})
		}
		label := m.text
		if label == "" {
			label = parts[m.from] + " → " + parts[m.to]
		}
		d.Steps = append(d.Steps, label)
	}
	if d.Title != "" {
		add(DEl{K: "text", C: "si-dg-title"}, DState{X: sPad, Y: sPad + 8, O: 1, T: strings.ToUpper(d.Title)})
	}
	// state k: the first k messages shown, the parties of the k-th lit
	for k := 0; k <= len(msgs); k++ {
		st := dsCopy(st0)
		for j := 0; j < k; j++ {
			st[msgAt+j].O, st[msgAt+j].P = 1, 1
		}
		if k > 0 {
			m := msgs[k-1]
			st[headAt+m.from].C = "is-on"
			st[headAt+m.to].C = "is-on"
			st[msgAt+k-1].C = "is-on"
		}
		d.States = append(d.States, st)
	}
	d.In = dsCopy(d.States[0])
	for i := range parts {
		d.In[headAt+i].O, d.In[headAt+i].Y = 0, d.In[headAt+i].Y-10
		d.In[i].P = 0
	}
	return d, probs
}
