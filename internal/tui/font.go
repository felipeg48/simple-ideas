package tui

import "strings"

// The block-letter font of titles and big numbers: 5×5 pixel glyphs drawn
// with half blocks (▀ ▄ █), so one glyph is 3 text rows tall and 5 wide.
var glyphs = map[rune][5]string{
	'A': {".###.", "#...#", "#####", "#...#", "#...#"}, 'B': {"####.", "#...#", "####.", "#...#", "####."},
	'C': {".####", "#....", "#....", "#....", ".####"}, 'D': {"####.", "#...#", "#...#", "#...#", "####."},
	'E': {"#####", "#....", "####.", "#....", "#####"}, 'F': {"#####", "#....", "####.", "#....", "#...."},
	'G': {".####", "#....", "#..##", "#...#", ".###."}, 'H': {"#...#", "#...#", "#####", "#...#", "#...#"},
	'I': {"#####", "..#..", "..#..", "..#..", "#####"}, 'J': {"..###", "...#.", "...#.", "#..#.", ".##.."},
	'K': {"#...#", "#..#.", "###..", "#..#.", "#...#"}, 'L': {"#....", "#....", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#...#", "#...#"}, 'N': {"#...#", "##..#", "#.#.#", "#..##", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", ".###."}, 'P': {"####.", "#...#", "####.", "#....", "#...."},
	'Q': {".###.", "#...#", "#.#.#", "#..#.", ".##.#"}, 'R': {"####.", "#...#", "####.", "#..#.", "#...#"},
	'S': {".####", "#....", ".###.", "....#", "####."}, 'T': {"#####", "..#..", "..#..", "..#..", "..#.."},
	'U': {"#...#", "#...#", "#...#", "#...#", ".###."}, 'V': {"#...#", "#...#", "#...#", ".#.#.", "..#.."},
	'W': {"#...#", "#...#", "#.#.#", "##.##", "#...#"}, 'X': {"#...#", ".#.#.", "..#..", ".#.#.", "#...#"},
	'Y': {"#...#", ".#.#.", "..#..", "..#..", "..#.."}, 'Z': {"#####", "...#.", "..#..", ".#...", "#####"},
	'0': {".###.", "#..##", "#.#.#", "##..#", ".###."}, '1': {"..#..", ".##..", "..#..", "..#..", ".###."},
	'2': {".###.", "#...#", "..##.", ".#...", "#####"}, '3': {"####.", "....#", ".###.", "....#", "####."},
	'4': {"#..#.", "#..#.", "#####", "...#.", "...#."}, '5': {"#####", "#....", "####.", "....#", "####."},
	'6': {".###.", "#....", "####.", "#...#", ".###."}, '7': {"#####", "....#", "...#.", "..#..", "..#.."},
	'8': {".###.", "#...#", ".###.", "#...#", ".###."}, '9': {".###.", "#...#", ".####", "....#", ".###."},
	'.': {"...", "...", "...", "...", ".#."}, ',': {"...", "...", "...", ".#.", "#.."},
	':': {"...", ".#.", "...", ".#.", "..."}, '%': {"#...#", "...#.", "..#..", ".#...", "#...#"},
	'!': {".#.", ".#.", ".#.", "...", ".#."}, '?': {".###.", "#...#", "..##.", ".....", "..#.."},
	'-': {".....", ".....", "#####", ".....", "....."}, '+': {".....", "..#..", "#####", "..#..", "....."},
	'#': {".#.#.", "#####", ".#.#.", "#####", ".#.#."}, '&': {".##..", "#..#.", ".##.#", "#..#.", ".##.#"},
	'\'': {".#.", ".#.", "...", "...", "..."}, '/': {"....#", "...#.", "..#..", ".#...", "#...."},
	'×': {".....", "#...#", ".###.", "#...#", "....."}, '$': {".####", "#.#..", ".###.", "..#.#", "####."},
	'€': {".####", "#....", "####.", "#....", ".####"}, '(': {".#", "#.", "#.", "#.", ".#"}, ')': {"#.", ".#", ".#", ".#", "#."},
	' ': {"...", "...", "...", "...", "..."},
}

// canBig reports whether every rune of s has a glyph (after upper-casing).
func canBig(s string) bool {
	for _, r := range strings.ToUpper(s) {
		if _, ok := glyphs[r]; !ok {
			return false
		}
	}
	return true
}

// bigWidth is the width of s in block letters.
func bigWidth(s string) int {
	w := 0
	for _, r := range strings.ToUpper(s) {
		w += len(glyphs[r][0]) + 1
	}
	if w > 0 {
		w--
	}
	return w
}

// big draws s in block letters: three text rows.
func big(s string) []string {
	var rows [3]strings.Builder
	first := true
	for _, r := range strings.ToUpper(s) {
		g, ok := glyphs[r]
		if !ok {
			continue
		}
		if !first {
			for i := range rows {
				rows[i].WriteByte(' ')
			}
		}
		first = false
		for i := 0; i < 3; i++ {
			top, bot := g[i*2], ""
			if i*2+1 < 5 {
				bot = g[i*2+1]
			}
			for x := 0; x < len(top); x++ {
				t, b := top[x] == '#', x < len(bot) && bot[x] == '#'
				switch {
				case t && b:
					rows[i].WriteRune('█')
				case t:
					rows[i].WriteRune('▀')
				case b:
					rows[i].WriteRune('▄')
				default:
					rows[i].WriteByte(' ')
				}
			}
		}
	}
	return []string{rows[0].String(), rows[1].String(), rows[2].String()}
}

// bigLines breaks s into lines of block letters no wider than w (words are
// not split); ok is false when it cannot (a glyph is missing, or one word is
// wider than w), and the caller falls back to plain bold text.
func bigLines(s string, w int) ([]string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || !canBig(s) {
		return nil, false
	}
	var lines []string
	cur := ""
	for _, word := range strings.Fields(s) {
		if bigWidth(word) > w {
			return nil, false
		}
		try := strings.TrimSpace(cur + " " + word)
		if bigWidth(try) <= w {
			cur = try
			continue
		}
		lines = append(lines, cur)
		cur = word
	}
	lines = append(lines, cur)
	var out []string
	for i, l := range lines {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, big(l)...)
	}
	return out, true
}
