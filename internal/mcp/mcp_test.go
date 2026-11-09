package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func exchange(t *testing.T, s *Server, lines ...string) map[string]map[string]any {
	t.Helper()
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	got := map[string]map[string]any{}
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("not one JSON message per line: %q", l)
		}
		got[string(must(json.Marshal(m["id"])))] = m
	}
	return got
}

func must(b []byte, _ error) []byte { return b }

// Both kinds of client: the 2026-07-28 stateless one (server/discover, the
// version in _meta) and an earlier one (initialize, then initialized).
func TestProtocol(t *testing.T) {
	s := &Server{Name: "si", Version: "9.9.9", Instructions: "use the tools", Tools: []Tool{
		{Name: "echo", Input: map[string]any{"type": "object"}, ReadOnly: true, Call: func(a json.RawMessage) (Result, error) {
			return Result{Text: string(a), Structured: map[string]any{"ok": true}, Images: []Image{{MIME: "image/png", Data: "AAAA"}}}, nil
		}},
		{Name: "fail", Call: func(json.RawMessage) (Result, error) { return Result{}, errors.New("bad input") }},
	}}
	r := exchange(t, s,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"c","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{"x":1}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"fail","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"1900-01-01"}}}`,
		`{"jsonrpc":"2.0","id":7,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":8,"method":"ping"}`,
	)
	if len(r) != 8 {
		t.Fatalf("want 8 answers (the notification gets none), got %d", len(r))
	}
	res := func(id string) map[string]any { m, _ := r[id]["result"].(map[string]any); return m }
	if res("1")["protocolVersion"] != "2025-06-18" || res("1")["serverInfo"].(map[string]any)["version"] != "9.9.9" {
		t.Errorf("initialize: %v", r["1"])
	}
	if v := res("2")["supportedVersions"].([]any); v[0] != "2026-07-28" || res("2")["resultType"] != "complete" {
		t.Errorf("discover: %v", r["2"])
	}
	if tools := res("3")["tools"].([]any); len(tools) != 2 || tools[0].(map[string]any)["annotations"].(map[string]any)["readOnlyHint"] != true {
		t.Errorf("tools/list: %v", r["3"])
	}
	c := res("4")["content"].([]any)
	if len(c) != 2 || c[0].(map[string]any)["text"] != `{"x":1}` || c[1].(map[string]any)["type"] != "image" || res("4")["structuredContent"] == nil {
		t.Errorf("tools/call: %v", r["4"])
	}
	if res("5")["isError"] != true {
		t.Errorf("a tool's error is a result with isError: %v", r["5"])
	}
	if e := r["6"]["error"].(map[string]any); e["code"].(float64) != -32022 {
		t.Errorf("unsupported version: %v", r["6"])
	}
	if e := r["7"]["error"].(map[string]any); e["code"].(float64) != -32601 {
		t.Errorf("unknown method: %v", r["7"])
	}
	if res("8") == nil {
		t.Errorf("ping: %v", r["8"])
	}
}
