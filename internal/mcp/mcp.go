// Package mcp is a small Model Context Protocol server over stdio, standard
// library only: newline-delimited JSON-RPC 2.0. It speaks the stateless
// 2026-07-28 protocol (server/discover; the version in each request's _meta)
// and the earlier ones with their initialize handshake, which clients in use
// still send. It only offers tools; si's are in cmd/si/mcp.go.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"sync"
)

// Versions are the protocol versions served, newest first.
var Versions = []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"}

// Tool is one tool: its definition and what it does. Call returns the result
// (Text, optional Structured data and Images) or an error the agent sees as a
// tool error (isError), not a protocol error.
type Tool struct {
	Name, Title, Description string
	Input                    map[string]any // JSON Schema of the arguments
	ReadOnly                 bool
	Call                     func(args json.RawMessage) (Result, error)
}

// Result is a tool's answer.
type Result struct {
	Text       string
	Structured any
	Images     []Image
}

// Image is a picture in a result (base64 data).
type Image struct{ MIME, Data string }

// Server answers MCP requests with its tools.
type Server struct {
	Name, Version, Instructions string
	Tools                       []Tool
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Serve reads requests from r and writes responses to w until r ends. Each
// request is answered in its own goroutine (a build must not hold up a
// check); writes are serialised, one message per line.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	var mu sync.Mutex
	var wg sync.WaitGroup
	send := func(v any) {
		b, _ := json.Marshal(v) // json.Marshal never writes a raw newline
		mu.Lock()
		w.Write(append(b, '\n'))
		mu.Unlock()
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 32<<20)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			send(map[string]any{"jsonrpc": "2.0", "id": nil, "error": rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		if len(req.ID) == 0 { // a notification (initialized, cancelled): nothing to answer
			continue
		}
		wg.Add(1)
		go func(req request) {
			defer wg.Done()
			res, rerr := s.handle(req)
			msg := map[string]any{"jsonrpc": "2.0", "id": req.ID}
			if rerr != nil {
				msg["error"] = rerr
			} else {
				msg["result"] = res
			}
			send(msg)
		}(req)
	}
	wg.Wait()
	return sc.Err()
}

func (s *Server) serverInfo() map[string]any {
	return map[string]any{"name": s.Name, "version": s.Version}
}

func (s *Server) handle(req request) (any, *rpcError) {
	var p struct {
		Meta            map[string]any  `json:"_meta"`
		ProtocolVersion string          `json:"protocolVersion"` // initialize, before 2026-07-28
		Name            string          `json:"name"`
		Arguments       json.RawMessage `json:"arguments"`
	}
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
		}
	}
	if v, ok := p.Meta["io.modelcontextprotocol/protocolVersion"].(string); ok && !slices.Contains(Versions, v) {
		return nil, &rpcError{Code: -32022, Message: "Unsupported protocol version", Data: map[string]any{"supported": Versions, "requested": v}}
	}
	caps := map[string]any{"tools": map[string]any{}}
	meta := map[string]any{"io.modelcontextprotocol/serverInfo": s.serverInfo()}
	switch req.Method {
	case "initialize":
		v := p.ProtocolVersion
		if !slices.Contains(Versions, v) {
			v = Versions[1] // the newest version that still has a handshake
		}
		return map[string]any{"protocolVersion": v, "capabilities": caps, "serverInfo": s.serverInfo(), "instructions": s.Instructions}, nil
	case "server/discover":
		return map[string]any{"resultType": "complete", "supportedVersions": Versions, "capabilities": caps, "instructions": s.Instructions,
			"ttlMs": 3600000, "cacheScope": "public", "_meta": meta}, nil
	case "ping":
		return map[string]any{"resultType": "complete"}, nil
	case "tools/list":
		var list []map[string]any
		for _, t := range s.Tools {
			list = append(list, map[string]any{"name": t.Name, "title": t.Title, "description": t.Description, "inputSchema": t.Input,
				"annotations": map[string]any{"readOnlyHint": t.ReadOnly}})
		}
		return map[string]any{"resultType": "complete", "tools": list, "ttlMs": 3600000, "cacheScope": "public", "_meta": meta}, nil
	case "tools/call":
		for _, t := range s.Tools {
			if t.Name != p.Name {
				continue
			}
			res, err := t.Call(p.Arguments)
			if err != nil {
				return map[string]any{"resultType": "complete", "isError": true, "content": []any{map[string]any{"type": "text", "text": err.Error()}}, "_meta": meta}, nil
			}
			content := []any{map[string]any{"type": "text", "text": res.Text}}
			for _, im := range res.Images {
				content = append(content, map[string]any{"type": "image", "mimeType": im.MIME, "data": im.Data})
			}
			out := map[string]any{"resultType": "complete", "content": content, "_meta": meta}
			if res.Structured != nil {
				out["structuredContent"] = res.Structured
			}
			return out, nil
		}
		return nil, &rpcError{Code: -32602, Message: fmt.Sprintf("unknown tool %q", p.Name)}
	}
	return nil, &rpcError{Code: -32601, Message: "method not found: " + req.Method}
}
