// Command broker is the tiny message broker of examples/webdemo/broker.md, a
// test helper for si's web demos with a terminal: `serve` takes messages
// over HTTP and shows them on a console page, `send` and `load` send some,
// `reset` empties it. Standard library only; run it with go run ./broker.
package main

import (
	"flag"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: broker serve|send|load|reset [--port N] …")
		os.Exit(2)
	}
	fl := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	port := fl.Int("port", 7310, "the broker's port")
	count := fl.Int("count", 30, "load: how many messages")
	every := fl.Duration("every", 100*time.Millisecond, "load: the time between two messages")
	fl.Parse(os.Args[2:])
	base := "http://127.0.0.1:" + strconv.Itoa(*port)
	switch os.Args[1] {
	case "serve":
		serve(*port)
	case "send":
		text := strings.Join(fl.Args(), " ")
		if err := send(base, text); err != nil {
			fmt.Fprintln(os.Stderr, "send:", err)
			os.Exit(1)
		}
		fmt.Printf("sent 1 message: %q\n", text)
	case "load":
		fmt.Printf("sending %d messages, one every %s\n", *count, *every)
		for i := 1; i <= *count; i++ {
			if err := send(base, fmt.Sprintf("load %d", i)); err != nil {
				fmt.Fprintln(os.Stderr, "load:", err)
				os.Exit(1)
			}
			time.Sleep(*every)
		}
		fmt.Printf("sent %d messages\n", *count)
	case "reset":
		if _, err := http.Post(base+"/reset", "text/plain", nil); err != nil {
			fmt.Fprintln(os.Stderr, "reset:", err)
			os.Exit(1)
		}
		fmt.Println("the queue is empty")
	default:
		fmt.Fprintln(os.Stderr, "broker: serve, send, load or reset")
		os.Exit(2)
	}
}

func send(base, text string) error {
	res, err := http.PostForm(base+"/send", url.Values{"text": {text}})
	if err != nil {
		return err
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", res.Status)
	}
	return nil
}

func serve(port int) {
	var mu sync.Mutex
	var msgs []string
	mux := http.NewServeMux()
	mux.HandleFunc("POST /send", func(w http.ResponseWriter, r *http.Request) {
		text := r.FormValue("text")
		mu.Lock()
		msgs = append(msgs, text)
		n := len(msgs)
		mu.Unlock()
		fmt.Printf("received #%d: %s\n", n, text)
	})
	mux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		msgs = nil
		mu.Unlock()
		fmt.Println("queue emptied")
	})
	mux.HandleFunc("GET /count", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, countHTML(msgs))
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, strings.Replace(page, "<!--count-->", countHTML(msgs), 1))
	})
	srv := &http.Server{Addr: "127.0.0.1:" + strconv.Itoa(port), Handler: mux}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		fmt.Println("broker stopped")
		srv.Close()
	}()
	fmt.Printf("broker ready on 127.0.0.1:%d, console at http://127.0.0.1:%d/\n", port, port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "broker:", err)
		os.Exit(1)
	}
}

func countHTML(msgs []string) string {
	word := "messages"
	if len(msgs) == 1 {
		word = "message"
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<p class="n"><b>%d</b> %s waiting</p><ol>`, len(msgs), word)
	for i := len(msgs) - 1; i >= 0 && i >= len(msgs)-6; i-- {
		fmt.Fprintf(&b, "<li><span>#%d</span> %s</li>", i+1, html.EscapeString(msgs[i]))
	}
	b.WriteString("</ol>")
	return b.String()
}

const page = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Broker console</title>
<style>
:root{color-scheme:light;--bg:#f4f6fb;--fg:#1b2030;--muted:#5b6478;--card:#fff;--line:#dfe3ec;--accent:#3157d5}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);font:clamp(15px,1.2vw,19px)/1.5 system-ui,-apple-system,"Segoe UI",sans-serif}
header{display:flex;align-items:center;gap:.8em;padding:1em clamp(1em,4vw,3em);background:var(--card);border-bottom:1px solid var(--line)}
header b{font-size:1.2em}header span{color:var(--muted)}
main{max-width:48em;margin:2em auto;padding:0 1em}
section{background:var(--card);border:1px solid var(--line);border-radius:.8em;padding:1.4em 1.6em}
h1{margin:0 0 .4em;font-size:1.3em}
.n{margin:0 0 .6em;font-size:1.2em}.n b{font-size:2.6em;color:var(--accent);margin-right:.15em;font-variant-numeric:tabular-nums}
ol{list-style:none;margin:0;padding:0}li{padding:.45em 0;border-top:1px solid var(--line)}li span{color:var(--muted);margin-right:.6em;font-variant-numeric:tabular-nums}
</style></head><body>
<header><b>Broker console</b><span>queue: orders</span></header>
<main><section><h1>Queue: orders</h1><div id="q"><!--count--></div></section></main>
<script>
var q=document.getElementById('q'),last='';setInterval(function(){fetch('/count').then(function(r){return r.text()}).then(function(t){if(t!==last){last=t;q.innerHTML=t}}).catch(function(){})},300);
</script>
</body></html>`
