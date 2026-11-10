package themefile

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/felipeg48/simple-ideas/internal/assets"
)

// The W3C Design Tokens format (.tokens.json): every token sits in one "si"
// group by its name without --si- ("accent", "font-body"); a colour's dark
// value travels in $extensions["org.simple-ideas"].dark. Colours, font
// families, durations, numbers and rem/em dimensions map; other values
// (a shadow, a cqi length, a var()) are reported as unmapped, never guessed.

const dtcgExt = "org.simple-ideas"

type dtcgToken struct {
	Type       string         `json:"$type"`
	Value      any            `json:"$value"`
	Desc       string         `json:"$description,omitempty"`
	Extensions map[string]any `json:"$extensions,omitempty"`
}

// DTCG writes the theme's own tokens as Design Tokens JSON and lists the
// tokens it could not map.
func (f *File) DTCG() ([]byte, []string) {
	group := map[string]any{}
	if f.Name != "" {
		group["$description"] = f.Name + " (an si theme)"
	}
	var unmapped []string
	keys := map[string]bool{}
	for k := range f.Light {
		keys[k] = true
	}
	for k := range f.Dark {
		keys[k] = true
	}
	all := map[string]string{}
	for k := range keys {
		all[k] = ""
	}
	for _, k := range Order(all) {
		v, ok := f.Light[k]
		if !ok { // dark only: the light value comes from the extended theme
			if bl, _, has := assets.ThemeTokens(f.Extends); has {
				v = bl[k]
			}
		}
		t, ok := toDTCG(k, v)
		if !ok {
			unmapped = append(unmapped, k)
			continue
		}
		if d, has := f.Dark[k]; has {
			dt, ok := toDTCG(k, d)
			if !ok || dt.Type != t.Type {
				unmapped = append(unmapped, k+" (dark)")
			} else {
				t.Extensions = map[string]any{dtcgExt: map[string]any{"dark": dt.Value}}
			}
		}
		group[strings.TrimPrefix(k, "--si-")] = t
	}
	root := map[string]any{"si": group}
	if f.Extends != "" {
		root["$extensions"] = map[string]any{dtcgExt: map[string]any{"extends": f.Extends, "name": f.Name}}
	} else if f.Name != "" {
		root["$extensions"] = map[string]any{dtcgExt: map[string]any{"name": f.Name}}
	}
	b, _ := json.MarshalIndent(root, "", "  ")
	return append(b, '\n'), unmapped
}

func toDTCG(name, v string) (dtcgToken, bool) {
	m, ok := assets.Meta(name)
	v = strings.TrimSpace(v)
	if !ok || v == "" || strings.Contains(v, "var(") {
		return dtcgToken{}, false
	}
	switch m.Kind {
	case assets.KindColor:
		if !hexRE.MatchString(v) {
			return dtcgToken{}, false
		}
		return dtcgToken{Type: "color", Value: strings.ToLower(v)}, true
	case assets.KindFont:
		var fam []string
		for _, p := range split(v, ',') {
			fam = append(fam, strings.Trim(strings.TrimSpace(p), `"'`))
		}
		return dtcgToken{Type: "fontFamily", Value: fam}, true
	case assets.KindDuration:
		if strings.HasSuffix(v, "ms") {
			n, err := strconv.ParseFloat(strings.TrimSuffix(v, "ms"), 64)
			return dtcgToken{Type: "duration", Value: map[string]any{"value": n, "unit": "ms"}}, err == nil
		}
		n, err := strconv.ParseFloat(strings.TrimSuffix(v, "s"), 64)
		return dtcgToken{Type: "duration", Value: map[string]any{"value": n, "unit": "s"}}, err == nil && strings.HasSuffix(v, "s")
	case assets.KindNumber:
		n, err := strconv.ParseFloat(v, 64)
		return dtcgToken{Type: "number", Value: n}, err == nil
	case assets.KindLength:
		for _, u := range []string{"rem", "em"} { // DTCG dimensions are px or rem; em travels as a string
			if n, err := strconv.ParseFloat(strings.TrimSuffix(v, u), 64); err == nil && strings.HasSuffix(v, u) {
				if u == "rem" {
					return dtcgToken{Type: "dimension", Value: map[string]any{"value": n, "unit": "rem"}}, true
				}
				return dtcgToken{Type: "dimension", Value: v}, true
			}
		}
	}
	return dtcgToken{}, false
}

// FromDTCG reads Design Tokens JSON into a theme. A token maps when its path,
// without a leading "si", "light" or "dark" group, joined with "-" names an
// si token: "accent", "si.font-body", "dark.accent". A top-level "dark" group
// (or the org.simple-ideas dark extension) gives dark values. Every other
// token is listed as unmapped.
func FromDTCG(data []byte) (*File, []string, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, nil, fmt.Errorf("not Design Tokens JSON: %w", err)
	}
	f := New("", "")
	if ext, ok := root["$extensions"].(map[string]any); ok {
		if si, ok := ext[dtcgExt].(map[string]any); ok {
			f.Name, _ = si["name"].(string)
			f.Extends, _ = si["extends"].(string)
		}
	}
	var unmapped []string
	var walk func(path []string, node map[string]any, dark bool)
	walk = func(path []string, node map[string]any, dark bool) {
		if _, leaf := node["$value"]; leaf {
			p := path
			if len(p) > 0 && (p[0] == "si" || p[0] == "light" || p[0] == "dark") {
				p = p[1:]
			}
			if len(p) > 0 && p[0] == "si" {
				p = p[1:]
			}
			name := "--si-" + strings.Join(p, "-")
			v, ok := fromDTCGValue(node["$value"])
			if _, known := assets.Meta(name); !known || !ok || CheckValue(name, v) != nil {
				unmapped = append(unmapped, strings.Join(path, "."))
				return
			}
			if dark {
				f.Dark[name] = v
				return
			}
			f.Light[name] = v
			if ext, ok := node["$extensions"].(map[string]any); ok {
				if si, ok := ext[dtcgExt].(map[string]any); ok {
					if d, ok := fromDTCGValue(si["dark"]); ok && CheckValue(name, d) == nil {
						f.Dark[name] = d
					}
				}
			}
			return
		}
		keys := make([]string, 0, len(node))
		for k := range node {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if strings.HasPrefix(k, "$") {
				continue
			}
			if child, ok := node[k].(map[string]any); ok {
				walk(append(append([]string{}, path...), k), child, dark || (len(path) == 0 && k == "dark"))
			}
		}
	}
	walk(nil, root, false)
	return f, unmapped, nil
}

func fromDTCGValue(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		if strings.HasPrefix(x, "{") { // an alias: not followed
			return "", false
		}
		return x, true
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), true
	case []any:
		var fam []string
		for _, e := range x {
			s, ok := e.(string)
			if !ok {
				return "", false
			}
			if strings.ContainsAny(s, " ") && !strings.HasPrefix(s, "ui-") {
				s = `"` + s + `"`
			}
			fam = append(fam, s)
		}
		return strings.Join(fam, ", "), len(fam) > 0
	case map[string]any:
		if hex, ok := x["hex"].(string); ok { // a 2025 colour object
			return hex, true
		}
		n, ok1 := x["value"].(float64)
		u, ok2 := x["unit"].(string)
		if ok1 && ok2 {
			return strconv.FormatFloat(n, 'f', -1, 64) + u, true
		}
	}
	return "", false
}
