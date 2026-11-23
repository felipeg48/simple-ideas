// Package sandbox gives a lab its own Docker box: a container, Docker inside
// it, made from the lab's image on a sandbox host (a Colima VM with sysbox, so
// the box runs unprivileged), with shells and checks run inside it by exec.
// Standard library only: the Docker Engine API is spoken over its socket.
// Spec: docs/superpowers/specs/2026-11-22-sandbox-labs.md.
package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/felipeg48/simple-ideas/internal/deck"
)

// apiVersion is the Engine API the client asks for (Docker 20.10 and later).
const apiVersion = "/v1.41"

// Engine talks to one Docker Engine over its unix socket.
type Engine struct {
	Socket string // the socket path
	hc     *http.Client
}

// NewEngine is a client for the engine listening on socket.
func NewEngine(socket string) *Engine {
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, "unix", socket)
	}}
	return &Engine{Socket: socket, hc: &http.Client{Transport: tr}}
}

// apiError is the engine's {"message": …} answer to a failed call.
type apiError struct {
	Status  int
	Message string
}

func (e *apiError) Error() string { return fmt.Sprintf("docker engine: %d %s", e.Status, e.Message) }

// notFound reports a 404 from the engine.
func notFound(err error) bool {
	var ae *apiError
	return errors.As(err, &ae) && ae.Status == http.StatusNotFound
}

// call sends method path with body (a value to encode as JSON, or nil) and
// decodes a JSON answer into out (nil: drop it).
func (e *Engine) call(ctx context.Context, method, path string, body, out any) error {
	resp, err := e.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// do sends a call and returns the response of a 2xx answer.
func (e *Engine) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var rd io.Reader
	switch b := body.(type) {
	case nil:
	case io.Reader:
		rd = b
	default:
		buf, err := json.Marshal(b)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+apiVersion+path, rd)
	if err != nil {
		return nil, err
	}
	switch body.(type) {
	case nil:
	case io.Reader:
		req.Header.Set("Content-Type", "application/x-tar")
	default:
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := e.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker engine at %s: %w", e.Socket, err)
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		var m struct{ Message string }
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		if json.Unmarshal(b, &m) != nil || m.Message == "" {
			m.Message = strings.TrimSpace(string(b))
		}
		return nil, &apiError{resp.StatusCode, m.Message}
	}
	return resp, nil
}

// Info is what the sandbox needs from GET /info.
type Info struct {
	ServerVersion string
	MemTotal      int64
	NCPU          int
	Runtimes      map[string]any
	OSType        string
}

// Info asks the engine about itself.
func (e *Engine) Info(ctx context.Context) (Info, error) {
	var i Info
	err := e.call(ctx, "GET", "/info", nil, &i)
	return i, err
}

// HasImage reports whether ref is in the engine's image store.
func (e *Engine) HasImage(ctx context.Context, ref string) (bool, error) {
	err := e.call(ctx, "GET", "/images/"+url.PathEscape(ref)+"/json", nil, nil)
	if notFound(err) {
		return false, nil
	}
	return err == nil, err
}

// Pull fetches ref (name:tag or name@digest); progress gets a line per status.
func (e *Engine) Pull(ctx context.Context, ref string, progress func(string)) error {
	name, tag := deck.SplitImageRef(ref)
	q := url.Values{"fromImage": {name}}
	if tag != "" {
		q.Set("tag", tag)
	}
	resp, err := e.do(ctx, "POST", "/images/create?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	for {
		var m struct{ Status, Error, ID string }
		if err := dec.Decode(&m); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		if m.Error != "" {
			return fmt.Errorf("pull %s: %s", ref, m.Error)
		}
		if progress != nil && m.Status != "" && m.ID == "" {
			progress(m.Status)
		}
	}
}

// CreateNetwork makes a bridge network with labels and returns its id.
func (e *Engine) CreateNetwork(ctx context.Context, name string, labels map[string]string) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := e.call(ctx, "POST", "/networks/create", map[string]any{"Name": name, "Driver": "bridge", "Labels": labels}, &out)
	return out.ID, err
}

// RemoveNetwork deletes a network (gone already is fine).
func (e *Engine) RemoveNetwork(ctx context.Context, id string) error {
	if err := e.call(ctx, "DELETE", "/networks/"+url.PathEscape(id), nil, nil); err != nil && !notFound(err) {
		return err
	}
	return nil
}

// PruneNetworks deletes the unused networks that carry label (k=v).
func (e *Engine) PruneNetworks(ctx context.Context, label string) error {
	f, _ := json.Marshal(map[string][]string{"label": {label}})
	return e.call(ctx, "POST", "/networks/prune?filters="+url.QueryEscape(string(f)), nil, nil)
}

// CreateContainer makes a container from config (the Engine's create body)
// and returns its id.
func (e *Engine) CreateContainer(ctx context.Context, name string, config map[string]any) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := e.call(ctx, "POST", "/containers/create?name="+url.QueryEscape(name), config, &out)
	return out.ID, err
}

// StartContainer starts a created container.
func (e *Engine) StartContainer(ctx context.Context, id string) error {
	return e.call(ctx, "POST", "/containers/"+url.PathEscape(id)+"/start", nil, nil)
}

// RemoveContainer stops and deletes a container and its anonymous volumes
// (gone already is fine).
func (e *Engine) RemoveContainer(ctx context.Context, id string) error {
	err := e.call(ctx, "DELETE", "/containers/"+url.PathEscape(id)+"?force=1&v=1", nil, nil)
	if err != nil && !notFound(err) {
		return err
	}
	return nil
}

// Container is what the sandbox reads back from an inspect.
type Container struct {
	ID    string `json:"Id"`
	State struct {
		Running  bool
		Status   string
		ExitCode int
	}
	NetworkSettings struct {
		Ports map[string][]struct{ HostIp, HostPort string }
	}
}

// Inspect reads a container's state and its published ports.
func (e *Engine) Inspect(ctx context.Context, id string) (Container, error) {
	var c Container
	err := e.call(ctx, "GET", "/containers/"+url.PathEscape(id)+"/json", nil, &c)
	return c, err
}

// Listed is one container of List.
type Listed struct {
	ID     string `json:"Id"`
	Names  []string
	Labels map[string]string
}

// List returns every container (running or not) that carries label (k=v).
func (e *Engine) List(ctx context.Context, label string) ([]Listed, error) {
	f, _ := json.Marshal(map[string][]string{"label": {label}})
	var out []Listed
	err := e.call(ctx, "GET", "/containers/json?all=1&filters="+url.QueryEscape(string(f)), nil, &out)
	return out, err
}

// ExecConfig is the part of an exec the sandbox sets.
type ExecConfig struct {
	Cmd        []string
	Tty        bool
	Stdin      bool
	Env        []string
	WorkingDir string
}

// ExecCreate prepares cmd in a running container and returns the exec id.
func (e *Engine) ExecCreate(ctx context.Context, id string, c ExecConfig) (string, error) {
	var out struct {
		ID string `json:"Id"`
	}
	err := e.call(ctx, "POST", "/containers/"+url.PathEscape(id)+"/exec", map[string]any{
		"AttachStdin": c.Stdin, "AttachStdout": true, "AttachStderr": true,
		"Tty": c.Tty, "Cmd": c.Cmd, "Env": c.Env, "WorkingDir": c.WorkingDir,
	}, &out)
	return out.ID, err
}

// ExecRun runs an exec without a terminal and returns what it wrote (stdout
// and stderr in order, up to max bytes) and its exit code.
func (e *Engine) ExecRun(ctx context.Context, execID string, max int) (string, int, error) {
	resp, err := e.do(ctx, "POST", "/exec/"+url.PathEscape(execID)+"/start", map[string]any{"Detach": false, "Tty": false})
	if err != nil {
		return "", -1, err
	}
	out, rerr := demux(resp.Body, max)
	resp.Body.Close()
	if rerr != nil && ctx.Err() == nil {
		return out, -1, rerr
	}
	if ctx.Err() != nil {
		return out, -1, ctx.Err()
	}
	code, err := e.ExecExit(context.Background(), execID)
	return out, code, err
}

// demux reads Docker's multiplexed stream (8-byte frame headers: stream,
// 0, 0, 0, then a big-endian length) into one string, keeping max bytes.
func demux(r io.Reader, max int) (string, error) {
	var b []byte
	hdr := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, hdr); err == io.EOF {
			return string(b), nil
		} else if err != nil {
			return string(b), err
		}
		n := int64(binary.BigEndian.Uint32(hdr[4:]))
		chunk, err := io.ReadAll(io.LimitReader(r, n))
		if room := max - len(b); room > 0 {
			b = append(b, chunk[:min(room, len(chunk))]...)
		}
		if err != nil {
			return string(b), err
		}
	}
}

// ExecExit is an ended exec's exit code.
func (e *Engine) ExecExit(ctx context.Context, execID string) (int, error) {
	var st struct {
		Running  bool
		ExitCode int
	}
	if err := e.call(ctx, "GET", "/exec/"+url.PathEscape(execID)+"/json", nil, &st); err != nil {
		return -1, err
	}
	if st.Running {
		return -1, errors.New("exec still running")
	}
	return st.ExitCode, nil
}

// ExecResize sets an exec terminal's size.
func (e *Engine) ExecResize(ctx context.Context, execID string, cols, rows int) error {
	return e.call(ctx, "POST", fmt.Sprintf("/exec/%s/resize?w=%d&h=%d", url.PathEscape(execID), cols, rows), nil, nil)
}

// ExecAttach starts an exec with a terminal and returns the raw two-way
// stream: the engine upgrades the connection, so what is written is the
// program's input and what is read its terminal output.
func (e *Engine) ExecAttach(ctx context.Context, execID string) (io.ReadWriteCloser, error) {
	return e.execStream(ctx, execID, true)
}

// execStream starts an exec on an upgraded connection (with or without a
// terminal: without one, what is read is the multiplexed stream).
func (e *Engine) execStream(ctx context.Context, execID string, tty bool) (*hijacked, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", e.Socket)
	if err != nil {
		return nil, fmt.Errorf("docker engine at %s: %w", e.Socket, err)
	}
	body := fmt.Sprintf(`{"Detach":false,"Tty":%v}`, tty)
	req := "POST " + apiVersion + "/exec/" + url.PathEscape(execID) + "/start HTTP/1.1\r\nHost: docker\r\n" +
		"Content-Type: application/json\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n" +
		fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body)) + body
	if dl, ok := ctx.Deadline(); ok {
		conn.SetDeadline(dl)
	}
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols && resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		conn.Close()
		return nil, &apiError{resp.StatusCode, strings.TrimSpace(string(b))}
	}
	conn.SetDeadline(time.Time{})
	return &hijacked{Conn: conn, r: br}, nil
}

// hijacked is an upgraded connection whose first bytes may sit in the reader
// that parsed the response.
type hijacked struct {
	net.Conn
	r *bufio.Reader
}

func (h *hijacked) Read(p []byte) (int, error) { return h.r.Read(p) }

// CloseWrite ends the program's input, the stream still open to read.
func (h *hijacked) CloseWrite() error {
	if cw, ok := h.Conn.(interface{ CloseWrite() error }); ok {
		return cw.CloseWrite()
	}
	return nil
}

// ExecFeed starts an exec without a terminal, writes in to its input, and
// returns its output (up to max bytes) and exit code.
func (e *Engine) ExecFeed(ctx context.Context, execID string, in io.Reader, max int) (string, int, error) {
	h, err := e.execStream(ctx, execID, false)
	if err != nil {
		return "", -1, err
	}
	defer h.Close()
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(h, in)
		h.CloseWrite()
		done <- err
	}()
	out, rerr := demux(h, max)
	if err := <-done; err != nil {
		return out, -1, err
	}
	if rerr != nil {
		return out, -1, rerr
	}
	code, err := e.ExecExit(ctx, execID)
	return out, code, err
}
