package main

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipeg48/simple-ideas/internal/mcp"
	"github.com/felipeg48/simple-ideas/internal/qr"
)

// The longest short remote link (any IPv4 address and port, the longest role,
// a 128-bit hex key) must make a QR code small enough for a terminal.
func TestRemoteLinkFitsAQR(t *testing.T) {
	u := "http://255.255.255.255:65535/clicker?k=" + strings.Repeat("f", 32)
	c, err := qr.Encode([]byte(u), qr.Options{Level: qr.M})
	if err != nil || c.Version > 5 {
		t.Fatalf("%d-byte remote link: %v (version %d)", len(u), err, c.Version)
	}
}

// si qr writes every format asked for, escapes a Wi-Fi network and refuses a
// picture the error correction could not carry, saying what fits.
func TestQRCommand(t *testing.T) {
	dir := t.TempDir()
	var out, errb strings.Builder
	if code := run([]string{"qr", "https://simple-ideas.org/", "--type", "png,jpg,svg,pdf", "-o", filepath.Join(dir, "site")}, &out, &errb); code != exitOK {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	for ext, magic := range map[string]string{"png": "\x89PNG", "jpg": "\xff\xd8", "svg": "<svg", "pdf": "%PDF-"} {
		b, err := os.ReadFile(filepath.Join(dir, "site."+ext))
		if err != nil || !strings.HasPrefix(string(b), magic) {
			t.Errorf("site.%s: %v", ext, err)
		}
	}
	text, err := qrPayload(nil, &options{qrWifi: "Office;Net", qrPassword: `a:b\c`, qrSecurity: "wpa"})
	if err != nil || text != `WIFI:T:WPA;S:Office\;Net;P:a\:b\\c;;` {
		t.Errorf("wifi payload %q %v", text, err)
	}
	logo := filepath.Join(dir, "logo.png")
	f, _ := os.Create(logo)
	png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 8, 8)))
	f.Close()
	errb.Reset()
	if code := run([]string{"qr", "https://simple-ideas.org/", "--logo", logo, "--logo-size", "45", "-o", filepath.Join(dir, "big")}, &out, &errb); code == exitOK || !strings.Contains(errb.String(), "at most") {
		t.Errorf("a 45 %% picture should be refused with the size that fits: %d %s", code, errb.String())
	}
}

func TestTailscaleName(t *testing.T) {
	name, err := tailscaleName([]byte(`{"BackendState":"Running","Self":{"DNSName":"felipes-mac.tail1234.ts.net."}}`))
	if err != nil || name != "felipes-mac.tail1234.ts.net" {
		t.Errorf("%q %v", name, err)
	}
	if _, err := tailscaleName([]byte(`{"BackendState":"Stopped","Self":{"DNSName":""}}`)); err == nil {
		t.Error("a stopped Tailscale must be refused")
	}
	if _, err := tailscaleName([]byte(`{"BackendState":"Running","Self":{"DNSName":"mac."}}`)); err == nil || !strings.Contains(err.Error(), "MagicDNS") {
		t.Errorf("no ts.net name should point at MagicDNS: %v", err)
	}
}

func TestCATrustCommand(t *testing.T) {
	args, _ := caTrustCommand("darwin", true, "/x/si-local-ca.pem", "si local CA (mac)")
	if len(args) == 0 || args[0] != "security" || args[1] != "add-trusted-cert" || args[len(args)-1] != "/x/si-local-ca.pem" || !strings.HasSuffix(args[len(args)-2], "login.keychain-db") {
		t.Errorf("macOS trust: %q", args)
	}
	if args, _ := caTrustCommand("darwin", false, "/x/si-local-ca.pem", "si local CA (mac)"); args[1] != "delete-certificate" || args[3] != "si local CA (mac)" {
		t.Errorf("macOS untrust: %q", args)
	}
	if args, manual := caTrustCommand("linux", true, "/x/si-local-ca.pem", ""); args != nil || !strings.Contains(manual, "update-ca-certificates") {
		t.Errorf("linux: %q %q", args, manual)
	}
}

// The MCP tools work inside the served folder only, return si check's
// problems as data, and look rules up by number, title word or text.
func TestMCPTools(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "talk.md"), []byte("---\ntitle: x\n---\n\n# X\n\n<!-- layout: nope -->\n\nhi\n"), 0o644)
	tools := map[string]func(json.RawMessage) (mcp.Result, error){}
	for _, tl := range mcpTools(root) {
		tools[tl.Name] = tl.Call
		// what a client validates: an object schema whose required, if any, is a list
		b, _ := json.Marshal(tl.Input)
		var sch map[string]any
		json.Unmarshal(b, &sch)
		if r, ok := sch["required"]; sch["type"] != "object" || ok && r == nil {
			t.Errorf("%s: inputSchema %s", tl.Name, b)
		}
		if _, ok := sch["required"].([]any); ok != (sch["required"] != nil) {
			t.Errorf("%s: required must be an array: %s", tl.Name, b)
		}
	}
	res, err := tools["si_check"](json.RawMessage(`{"path":"talk.md"}`))
	st, _ := res.Structured.(map[string]any)
	ps, _ := st["problems"].([]map[string]any)
	if err != nil || st["ok"] != false || len(ps) != 1 || ps[0]["file"] != "talk.md" || ps[0]["line"] != 7 || !strings.Contains(ps[0]["message"].(string), "layout") {
		t.Errorf("si_check: %v %+v", err, res.Structured)
	}
	if _, err := tools["si_check"](json.RawMessage(`{"path":"../../etc"}`)); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("a path outside the folder must be refused: %v", err)
	}
	if res, err := tools["si_rules"](json.RawMessage(`{"query":"<!-- qr:"}`)); err != nil || !strings.Contains(res.Text, "qr") {
		t.Errorf("si_rules query: %v", err)
	}
	if res, _ := tools["si_rules"](json.RawMessage(`{}`)); !strings.Contains(res.Text, "1. ") {
		t.Errorf("si_rules list: %q", res.Text)
	}
	if _, err := tools["si_new"](json.RawMessage(`{"name":"post","type":"article"}`)); err != nil {
		t.Errorf("si_new: %v", err)
	}
	if res, _ := tools["si_check"](json.RawMessage(`{"path":"post.md"}`)); res.Structured.(map[string]any)["ok"] != true {
		t.Errorf("a fresh starter should check clean: %s", res.Text)
	}
}

// Handouts carry no speaker text unless asked: the default is slides only;
// points and notes are named. With Chrome, a real build prints a PDF.
func TestHandout(t *testing.T) {
	var h handoutFlag
	for v, want := range map[string]string{"true": "deck", "slides": "slides", "grid": "slides", "lines": "lines", "points": "points", "notes": "notes"} {
		if err := h.Set(v); err != nil || string(h) != want {
			t.Errorf("--handout=%s → %q %v, want %q", v, h, err, want)
		}
	}
	if err := h.Set("teleprompter"); err == nil {
		t.Error("an unknown handout mode must be refused")
	}
	if findChrome() == "" {
		t.Skip("no Chrome for the PDF")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "talk.md")
	os.WriteFile(src, []byte("---\ntitle: Talk\nhandout-url: https://example.com/talk\n---\n\n# Talk\n\nNotes:\nSAY-THIS-ALOUD\n\nPoints:\n- POINT-FOR-ME\n\n---\n\n## Two\n\nText.\n"), 0o644)
	for _, mode := range []string{"slides", "notes"} {
		out := filepath.Join(dir, "out-"+mode)
		var o, e strings.Builder
		if code := run([]string{"build", src, "-o", out, "--handout=" + mode}, &o, &e); code != exitOK {
			t.Fatalf("%s: exit %d: %s", mode, code, e.String())
		}
		b, err := os.ReadFile(filepath.Join(out, "handouts", "talk.pdf"))
		if err != nil || !strings.HasPrefix(string(b), "%PDF-") {
			t.Fatalf("%s: no handout PDF: %v", mode, err)
		}
	}
}
