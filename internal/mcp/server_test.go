package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func echoTool() Tool {
	return Tool{
		Name:        "echo",
		Description: "echoes its input",
		InputSchema: ObjectSchema(map[string]Property{
			"text": {Type: "string", Description: "text to echo"},
		}, []string{"text"}),
		Handler: func(args json.RawMessage) (interface{}, error) {
			var p struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return map[string]string{"echoed": p.Text}, nil
		},
	}
}

func failTool() Tool {
	return Tool{
		Name:        "fail",
		Description: "always errors",
		InputSchema: ObjectSchema(nil, nil),
		Handler: func(json.RawMessage) (interface{}, error) {
			return nil, errTest
		},
	}
}

var errTest = errString("boom")

type errString string

func (e errString) Error() string { return string(e) }

func newTestServer() *Server {
	return &Server{Name: "test", Version: "0.0.0", Tools: []Tool{echoTool(), failTool()}}
}

// decodeLines splits w's output into one decoded JSON object per line.
func decodeLines(t *testing.T, out string) []map[string]interface{} {
	t.Helper()
	var lines []map[string]interface{}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l == "" {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("response line not valid JSON: %v — %q", err, l)
		}
		lines = append(lines, m)
	}
	return lines
}

func TestServer_Initialize(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	if len(lines) != 1 {
		t.Fatalf("expected 1 response, got %d", len(lines))
	}
	result, ok := lines[0]["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result object, got %v", lines[0])
	}
	if result["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion = %v, want %v", result["protocolVersion"], protocolVersion)
	}
	serverInfo, _ := result["serverInfo"].(map[string]interface{})
	if serverInfo["name"] != "test" {
		t.Errorf("serverInfo.name = %v, want test", serverInfo["name"])
	}
}

func TestServer_ToolsList(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	result := lines[0]["result"].(map[string]interface{})
	tools := result["tools"].([]interface{})
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	first := tools[0].(map[string]interface{})
	if first["name"] != "echo" {
		t.Errorf("tools[0].name = %v, want echo", first["name"])
	}
	if _, ok := first["inputSchema"].(map[string]interface{}); !ok {
		t.Errorf("inputSchema not an object: %v", first["inputSchema"])
	}
}

func TestServer_ToolsCall_Success(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hi"}}}` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	result := lines[0]["result"].(map[string]interface{})
	if result["isError"] != false {
		t.Errorf("isError = %v, want false", result["isError"])
	}
	content := result["content"].([]interface{})[0].(map[string]interface{})
	if !strings.Contains(content["text"].(string), "hi") {
		t.Errorf("expected echoed text in result, got %v", content["text"])
	}
}

func TestServer_ToolsCall_HandlerError(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fail","arguments":{}}}` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	result := lines[0]["result"].(map[string]interface{})
	if result["isError"] != true {
		t.Errorf("isError = %v, want true — handler errors must not become protocol errors", result["isError"])
	}
	content := result["content"].([]interface{})[0].(map[string]interface{})
	if content["text"] != "boom" {
		t.Errorf("content text = %v, want boom", content["text"])
	}
}

func TestServer_ToolsCall_UnknownTool(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope","arguments":{}}}` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	rpcErr, ok := lines[0]["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected a protocol-level error, got %v", lines[0])
	}
	if rpcErr["code"] != float64(-32602) {
		t.Errorf("error code = %v, want -32602", rpcErr["code"])
	}
}

func TestServer_UnknownMethod(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"bogus"}` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	rpcErr := lines[0]["error"].(map[string]interface{})
	if rpcErr["code"] != float64(-32601) {
		t.Errorf("error code = %v, want -32601", rpcErr["code"])
	}
}

func TestServer_ParseError(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader(`not json at all` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	rpcErr := lines[0]["error"].(map[string]interface{})
	if rpcErr["code"] != float64(-32700) {
		t.Errorf("error code = %v, want -32700", rpcErr["code"])
	}
}

func TestServer_NotificationProducesNoResponse(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	// No "id" field — this is a notification per JSON-RPC 2.0.
	in := strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "" {
		t.Errorf("expected no output for a notification, got %q", got)
	}
}

func TestServer_BlankLinesSkipped(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	in := strings.NewReader("\n\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n\n")

	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	lines := decodeLines(t, out.String())
	if len(lines) != 1 {
		t.Fatalf("expected 1 response, got %d", len(lines))
	}
}

func TestServer_PrintSchemas(t *testing.T) {
	s := newTestServer()
	var out bytes.Buffer
	if err := s.PrintSchemas(&out); err != nil {
		t.Fatalf("PrintSchemas: %v", err)
	}

	var tools []map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &tools); err != nil {
		t.Fatalf("output not a valid JSON array: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tool schemas, got %d", len(tools))
	}
}

func TestObjectSchema_NoProperties(t *testing.T) {
	raw := ObjectSchema(nil, nil)
	var s map[string]interface{}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if s["type"] != "object" {
		t.Errorf("type = %v, want object", s["type"])
	}
	if _, hasRequired := s["required"]; hasRequired {
		t.Error("required should be omitted when empty")
	}
}

func TestObjectSchema_ArrayProperty(t *testing.T) {
	raw := ObjectSchema(map[string]Property{
		"keys": {Type: "array", Items: "string", Description: "some keys"},
	}, []string{"keys"})

	var s map[string]interface{}
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	props := s["properties"].(map[string]interface{})
	keys := props["keys"].(map[string]interface{})
	if keys["type"] != "array" {
		t.Errorf("keys.type = %v, want array", keys["type"])
	}
	items := keys["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("keys.items.type = %v, want string", items["type"])
	}
}
