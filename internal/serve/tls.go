package serve

// HTTPS for si play --remote (--https). A phone only grants a page the screen
// wake lock (and a reader only trusts a link) in a secure context, and a
// phone reaches the laptop by a private address that no public certificate
// authority will sign. So si keeps a small certificate authority of its own:
// created once, its key readable by the user only, its certificate limited
// by name constraints to private and Tailscale addresses (it can never vouch
// for a public site, even if its key leaked). Each run signs a short-lived
// certificate for the machine's current addresses. The phone installs the
// authority's certificate once, from the remote port over plain HTTP (/ca);
// the same port speaks TLS to everything else (SplitTLS).

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/felipeg48/simple-ideas/internal/home"
	"html"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalCA is si's certificate authority for --https.
type LocalCA struct {
	Cert    *x509.Certificate
	DER     []byte
	Dir     string
	Created bool // made by this run
	key     *ecdsa.PrivateKey
}

// CADir is where the authority lives: $SI_CA_DIR, else ~/.si/ca.
func CADir() (string, error) {
	if d := os.Getenv("SI_CA_DIR"); d != "" {
		return d, nil
	}
	return home.Path("ca", home.Config), nil
}

// privateNets are the only addresses the authority may vouch for.
var privateNets = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16"}

// LoadCA reads the authority from dir, creating it the first time.
func LoadCA(dir string) (*LocalCA, error) {
	certPath, keyPath := filepath.Join(dir, "si-local-ca.pem"), filepath.Join(dir, "si-local-ca-key.pem")
	if cb, err := os.ReadFile(certPath); err == nil {
		kb, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("si's certificate authority: %s is there but its key is not: %v", certPath, err)
		}
		return parseCA(dir, cb, kb)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	host, _ := os.Hostname()
	var ranges []*net.IPNet
	for _, c := range privateNets {
		_, n, _ := net.ParseCIDR(c)
		ranges = append(ranges, n)
	}
	tmpl := &x509.Certificate{
		SerialNumber:                serial(),
		Subject:                     pkix.Name{CommonName: "si local CA (" + host + ")", Organization: []string{"simple-ideas, si play --remote --https"}},
		NotBefore:                   time.Now().Add(-time.Hour),
		NotAfter:                    time.Now().AddDate(10, 0, 0),
		KeyUsage:                    x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid:       true,
		IsCA:                        true,
		MaxPathLenZero:              true,
		PermittedIPRanges:           ranges,
		PermittedDNSDomains:         []string{".local"}, // mDNS names (mac.local); nothing public
		PermittedDNSDomainsCritical: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	kder, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder}), 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644); err != nil {
		return nil, err
	}
	ca, err := parseCA(dir, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder}))
	if ca != nil {
		ca.Created = true
	}
	return ca, err
}

func parseCA(dir string, certPEM, keyPEM []byte) (*LocalCA, error) {
	cb, _ := pem.Decode(certPEM)
	kb, _ := pem.Decode(keyPEM)
	if cb == nil || kb == nil {
		return nil, fmt.Errorf("si's certificate authority in %s is damaged (delete the folder to make a new one)", dir)
	}
	cert, err := x509.ParseCertificate(cb.Bytes)
	if err != nil {
		return nil, err
	}
	key, err := x509.ParseECPrivateKey(kb.Bytes)
	if err != nil {
		return nil, err
	}
	return &LocalCA{Cert: cert, DER: cb.Bytes, Dir: dir, key: key}, nil
}

func serial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 126))
	return n
}

// Fingerprint is the SHA-256 of the authority's certificate, as a phone's
// settings show it (AA:BB:…), to compare before trusting it.
func (ca *LocalCA) Fingerprint() string {
	sum := sha256.Sum256(ca.DER)
	parts := make([]string, len(sum))
	for i, b := range sum {
		parts[i] = fmt.Sprintf("%02X", b)
	}
	return strings.Join(parts, ":")
}

// Leaf signs a certificate for this run: the given addresses (and names),
// valid 7 days, the shape Apple asks of a server certificate (names in the
// SAN, serverAuth, well under 825 days).
func (ca *LocalCA) Leaf(hosts []string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: "si play --remote"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(0, 0, 7),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.Cert, &key.PublicKey, ca.key)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der, ca.DER}, PrivateKey: key}, nil
}

// SplitTLS serves TLS and plain HTTP on one port: a connection whose first
// byte opens a TLS handshake (0x16) goes to the TLS listener, any other to
// the plain one.
func SplitTLS(ln net.Listener, cfg *tls.Config) (tlsLn, plainLn net.Listener) {
	t := &chanListener{addr: ln.Addr(), c: make(chan net.Conn), done: make(chan struct{}), close: ln.Close}
	p := &chanListener{addr: ln.Addr(), c: make(chan net.Conn), done: t.done, close: ln.Close, once: &t.closeOnce}
	t.once = &t.closeOnce
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				t.Close()
				return
			}
			go func() {
				conn.SetReadDeadline(time.Now().Add(10 * time.Second))
				r := bufio.NewReader(conn)
				b, err := r.Peek(1)
				if err != nil {
					conn.Close()
					return
				}
				conn.SetReadDeadline(time.Time{})
				pc := &peekedConn{Conn: conn, r: r}
				to, c := p, net.Conn(pc)
				if b[0] == 0x16 {
					to, c = t, tls.Server(pc, cfg)
				}
				select {
				case to.c <- c:
				case <-t.done:
					conn.Close()
				}
			}()
		}
	}()
	return t, p
}

type peekedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *peekedConn) Read(p []byte) (int, error) { return c.r.Read(p) }

type chanListener struct {
	addr      net.Addr
	c         chan net.Conn
	done      chan struct{}
	close     func() error
	once      *sync.Once
	closeOnce sync.Once
}

func (l *chanListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.c:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *chanListener) Close() error {
	l.once.Do(func() { close(l.done); l.close() })
	return nil
}

func (l *chanListener) Addr() net.Addr { return l.addr }

// CAHandler is the plain-HTTP side of the remote port: /ca explains how to
// trust si's authority on a phone, /ca.crt is its certificate, and anything
// else moves to HTTPS. ca is nil with --https=tailscale (nothing to install).
func CAHandler(ca *LocalCA) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case ca != nil && r.URL.Path == "/ca.crt":
			w.Header().Set("Content-Type", "application/x-x509-ca-cert")
			w.Header().Set("Content-Disposition", `attachment; filename="si-local-ca.crt"`)
			w.Write(ca.DER)
		case ca != nil && (r.URL.Path == "/ca" || r.URL.Path == "/ca/"):
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, caPage, html.EscapeString(ca.Cert.Subject.CommonName), ca.Fingerprint())
		default:
			u := *r.URL
			u.Scheme, u.Host = "https", r.Host
			http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
		}
	})
}

const caPage = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Trust si on this phone</title>
<style>body{font:17px/1.5 system-ui,sans-serif;max-width:34rem;margin:0 auto;padding:1.2rem;color:#1a1a1a;background:#fafafa}h1{font-size:1.4rem}a.btn{display:block;text-align:center;padding:.9em;border-radius:.6em;background:#2f5bd3;color:#fff;font-weight:600;text-decoration:none;margin:1.2em 0}code{font-size:.78em;word-break:break-all}li{margin:.4em 0}@media (prefers-color-scheme:dark){body{color:#eee;background:#16171b}}</style></head><body>
<h1>Trust si on this phone, once</h1>
<p>The slides, teleprompter and clicker come over HTTPS from the laptop. This phone trusts them after it trusts <b>%s</b>, a certificate authority that only exists on that laptop and can only vouch for local addresses.</p>
<a class="btn" href="/ca.crt">Download the certificate</a>
<p><b>iPhone / iPad</b></p><ol><li>Allow the download, then open <b>Settings</b>: <b>Profile Downloaded</b> is at the top. Tap <b>Install</b>.</li><li>Then <b>Settings › General › About › Certificate Trust Settings</b>: turn on full trust for it.</li></ol>
<p><b>Android</b></p><ol><li>Open the downloaded file, or <b>Settings › Security › Encryption &amp; credentials › Install a certificate › CA certificate</b>.</li></ol>
<p>Before you trust it, its SHA-256 fingerprint in the phone's details must read the same as on the laptop's terminal:</p><p><code>%s</code></p>
<p>Then go back and scan the code again. To remove it later: Settings › General › VPN &amp; Device Management (iPhone), or Trusted credentials › User (Android).</p>
</body></html>`
