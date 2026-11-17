package ui

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// Stats is what the si play monitor shows, gathered once a second.
type Stats struct {
	Target                     string // what is served: examples/
	Up                         time.Duration
	CPU                        float64   // percent of one core, since the last sample
	Mem                        uint64    // bytes: the Go runtime's memory from the OS (MemStats.Sys)
	CPUHist, MemHist           []float64 // recent samples, oldest first (sparklines)
	Shells, Pages, Files       int       // drawer shells running, pages open (live reload), files watched
	Open, Phone                string    // links: this machine, the network ("" without --addr 0.0.0.0)
	Slide, Slides, Step, Steps int       // the last position a deck sent (1-based); Slides 0: none yet
	Log                        []LogLine // recent reloads and warnings, oldest first
	Keys                       string    // the footer: q quit · m hide · o open
}

// LogLine is one recent event: a reload, a warning.
type LogLine struct {
	At   time.Time
	St   Status
	Text string
}

// Sampler turns CPU time and memory into Stats samples.
type Sampler struct {
	last     time.Time
	lastCPU  time.Duration
	cpu, mem []float64
}

// Sample fills s.CPU, s.Mem and the histories (at most n samples).
// Memory is runtime.MemStats.Sys, the memory Go holds from the OS: process
// RSS costs a cgo call on macOS, and Sys is close to it for a Go server.
func (m *Sampler) Sample(s *Stats, now time.Time, n int) {
	cpu := CPUTime()
	if !m.last.IsZero() && now.After(m.last) {
		s.CPU = 100 * float64(cpu-m.lastCPU) / float64(now.Sub(m.last))
	}
	m.last, m.lastCPU = now, cpu
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	s.Mem = ms.Sys
	m.cpu = append(m.cpu, s.CPU)
	m.mem = append(m.mem, float64(ms.Sys))
	if len(m.cpu) > n {
		m.cpu, m.mem = m.cpu[len(m.cpu)-n:], m.mem[len(m.mem)-n:]
	}
	s.CPUHist, s.MemHist = m.cpu, m.mem
}

// memScale is the memory a full bar stands for.
const memScale = 256 << 20

func bar(frac float64, n int) string {
	k := int(frac*float64(n) + 0.5)
	k = max(0, min(k, n))
	return strings.Repeat("█", k) + strings.Repeat("░", n-k)
}

// spark draws values as ▁…█ between 80% of the smallest and 125% of the largest
// (at least top): a steady line sits mid-height, a climb shows.
func spark(vals []float64, top float64, n int) string {
	if len(vals) > n {
		vals = vals[len(vals)-n:]
	}
	lo := 0.0
	if len(vals) > 0 {
		lo = vals[0]
	}
	for _, v := range vals {
		top, lo = max(top, v), min(lo, v)
	}
	lo, top = lo*0.8, top*1.25
	var b strings.Builder
	for _, v := range vals {
		b.WriteRune([]rune("▁▂▃▄▅▆▇█")[max(0, min(7, int((v-lo)/(top-lo)*7+0.5)))])
	}
	if b.Len() == 0 {
		return "▁"
	}
	return b.String()
}

func clock(d time.Duration) string {
	d = d.Round(time.Second)
	return fmt.Sprintf("%02d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

func mb(n uint64) string { return fmt.Sprintf("%d MB", n>>20) }

func count(n int, one string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %ss", n, one)
}

func pages(n int) string { return count(n, "page") + " open" }

// Monitor draws the monitor screen for s in look v: a pure function of s.
func Monitor(v View, s Stats, color bool) string {
	p := &Printer{v: v, color: color}
	var lines []string
	add := func(l ...string) { lines = append(lines, l...) }
	logText := func(l LogLine) string {
		t := []rune(l.Text)
		if len(t) > 58 { // one line, inside the steps box too
			t = append(t[:57], '…')
		}
		if l.St == OK {
			return string(t)
		}
		return p.c(statusColor(l.St), string(t))
	}
	switch v {
	case Badges:
		b := func(col, s string) string { return badge(color, col, " "+s+" ") }
		add(" "+b(magenta, "si play")+" "+p.c(dim, s.Target+" · up "+clock(s.Up)), "",
			" "+b(blue, "CPU")+" "+p.c(blue, spark(s.CPUHist, 10, 12))+" "+p.c(bold, fmt.Sprintf("%.1f%%", s.CPU))+
				"     "+b(cyan, "MEM")+" "+p.c(cyan, spark(s.MemHist, 1, 12))+" "+p.c(bold, mb(s.Mem))+
				"     "+b(yellow, "SHELLS")+" "+p.c(bold, fmt.Sprint(s.Shells)), "",
			" "+b(cyan, "OPEN ")+" "+p.value(Info, s.Open))
		if s.Phone != "" {
			add(" " + b(cyan, "PHONE") + " " + p.value(Info, s.Phone))
		}
		add("", " "+b(green, strings.ToUpper(count(s.Pages, "viewer")))+"  "+pages(s.Pages)+p.c(dim, " · "+count(s.Files, "file")+" watched"))
		if s.Slides > 0 {
			add(" " + b(magenta, fmt.Sprintf("SLIDE %d/%d", s.Slide, s.Slides)) + "  " + p.c(dim, fmt.Sprintf("step %d/%d", s.Step, s.Steps)))
		}
		if len(s.Log) > 0 {
			add("")
		}
		for _, l := range s.Log {
			tag := map[Status]string{OK: "RELOAD", Warn: " WARN ", Fail: " FAIL ", Info: " INFO "}[l.St]
			add(" " + b(statusColor(l.St), tag) + " " + p.c(dim, l.At.Format("15:04:05")) + " " + logText(l))
		}
		add("", p.c(dim, " "+s.Keys))
	case Checklist:
		add("🎬  "+p.c(bold, "si play")+"  "+p.c(dim, s.Target+" · running "+fmt.Sprint(int(s.Up.Minutes()))+" min"), "",
			"🔥  CPU     "+p.c(green, "▕"+bar(s.CPU/100, 16)+"▏")+" "+p.c(bold, fmt.Sprintf("%.1f%%", s.CPU)),
			"🧠  Memory  "+p.c(cyan, "▕"+bar(float64(s.Mem)/memScale, 16)+"▏")+" "+p.c(bold, mb(s.Mem)),
			"💻  Shells  "+p.c(bold, fmt.Sprint(s.Shells))+" running", "",
			"🌐  "+p.value(Info, s.Open))
		if s.Phone != "" {
			add("📱  " + p.value(Info, s.Phone))
		}
		add("", fmt.Sprintf("👥  %d watching, %s watched", s.Pages, count(s.Files, "file")))
		if s.Slides > 0 {
			add("🎞️   Slide " + p.c(bold, fmt.Sprintf("%d of %d", s.Slide, s.Slides)) + p.c(dim, fmt.Sprintf("  step %d of %d", s.Step, s.Steps)))
		}
		if len(s.Log) > 0 {
			add("")
		}
		for _, l := range s.Log {
			add([...]string{"🔄", "⚠️ ", "❌", "👉"}[l.St] + "  " + p.c(dim, l.At.Format("15:04:05")) + "  " + logText(l))
		}
		add("", p.c(dim, s.Keys))
	case Steps:
		const w = 74 // inside the box
		row := func(s string) string { return p.c(dim, "│") + Pad("  "+s, w) + p.c(dim, "│") }
		two := func(l, r string) string { return row(Pad(l, 36) + r) }
		head := " " + p.c(bold, "si play") + " " + p.c(dim, "· "+s.Target+" · up "+clock(s.Up)+" ")
		add(p.c(dim, "╭─")+head+p.c(dim, strings.Repeat("─", max(0, w-1-Width(head)))+"╮"), row(""),
			two(p.c(dim, "SERVER"), p.c(dim, "PRESENTING")))
		var right []string
		if s.Slides > 0 {
			right = append(right, "slide  "+p.c(bold, fmt.Sprint(s.Slide))+p.c(dim, fmt.Sprintf("/%d", s.Slides)),
				"step   "+fmt.Sprint(s.Step)+p.c(dim, fmt.Sprintf("/%d", s.Steps)))
		}
		right = append(right, "viewers "+p.c(green, strings.Repeat("●", min(s.Pages, 5)))+" "+pages(s.Pages))
		left := []string{
			"cpu    " + p.c(green, spark(s.CPUHist, 10, 10)) + " " + p.c(bold, fmt.Sprintf("%.1f%%", s.CPU)),
			"memory " + p.c(cyan, spark(s.MemHist, 1, 10)) + " " + p.c(bold, mb(s.Mem)),
			"shells " + p.c(bold, fmt.Sprint(s.Shells)),
			fmt.Sprintf("files  %d watched", s.Files),
		}
		for i := range max(len(left), len(right)) {
			l, r := "", ""
			if i < len(left) {
				l = left[i]
			}
			if i < len(right) {
				r = right[i]
			}
			add(two(l, r))
		}
		add(row(""), row(p.c(dim, "LINKS")), row(p.value(Info, s.Open)))
		if s.Phone != "" {
			add(row(p.value(Info, s.Phone) + "  " + p.c(dim, "phone")))
		}
		if len(s.Log) > 0 {
			add(row(""), row(p.c(dim, "LOG")))
		}
		for _, l := range s.Log {
			add(row(p.c(dim, l.At.Format("15:04:05")) + " " + p.mark(l.St) + " " + logText(l)))
		}
		foot := " " + s.Keys + " "
		add(p.c(dim, "╰─"+foot+strings.Repeat("─", max(0, w-1-Width(foot)))+"╯"))
	default:
		head := p.c(bold, "si play") + "  " + p.c(dim, s.Target+" · up "+clock(s.Up))
		add(Pad(head, 64)+p.c(dim, s.Keys), "",
			p.c(dim, "  cpu   ")+p.c(green, bar(s.CPU/100, 16))+"  "+Pad(p.c(bold, fmt.Sprintf("%.1f%%", s.CPU)), 9)+
				p.c(dim, "memory ")+p.c(cyan, bar(float64(s.Mem)/memScale, 16))+"  "+p.c(bold, mb(s.Mem)),
			p.c(dim, "  shells")+"  "+Pad(fmt.Sprintf("%d running", s.Shells), 25)+p.c(dim, "files ")+fmt.Sprintf(" %d watched", s.Files), "",
			p.c(dim, "  open    ")+p.value(Info, s.Open))
		if s.Phone != "" {
			add(p.c(dim, "  phone   ") + p.value(Info, s.Phone))
		}
		add("", p.c(dim, "  viewers ")+p.c(green, "●")+" "+pages(s.Pages))
		if s.Slides > 0 {
			add(p.c(dim, "  slide   ") + p.c(bold, fmt.Sprint(s.Slide)) + p.c(dim, fmt.Sprintf(" / %d", s.Slides)) + "   " + p.c(dim, fmt.Sprintf("step %d/%d", s.Step, s.Steps)))
		}
		if len(s.Log) > 0 {
			add("", p.c(dim, "  recent"))
		}
		for _, l := range s.Log {
			add(p.c(dim, "  "+l.At.Format("15:04:05")) + "  " + p.mark(l.St) + " " + logText(l))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}
