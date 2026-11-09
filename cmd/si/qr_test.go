package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
