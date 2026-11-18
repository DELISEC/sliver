//go:build integration

package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/transport"
)

const (
	testConfigPath = "local.cfg"
	testListenAddr = "127.0.0.1:18443"
	testTransport  = TransportHTTP
)

func loadTestConfig(t *testing.T) *assets.ClientConfig {
	t.Helper()
	configPath := os.Getenv("SLIVER_TEST_CONFIG")
	if configPath == "" {
		configs := assets.GetConfigs()
		for _, cfg := range configs {
			if cfg.Operator == "local" {
				return cfg
			}
		}
		t.Skip("no local operator config found; set SLIVER_TEST_CONFIG to a config file path")
	}
	cfg, err := assets.ReadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to read config %s: %v", configPath, err)
	}
	return cfg
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// mcpClient holds state for a single MCP HTTP session.
type mcpClient struct {
	endpoint  string
	authToken string
	sessionID string
	nextID    int
}

func newMCPClient(endpoint, authToken string) *mcpClient {
	return &mcpClient{
		endpoint:  endpoint,
		authToken: authToken,
		nextID:    1,
	}
}

func (c *mcpClient) call(t *testing.T, method string, params interface{}) json.RawMessage {
	t.Helper()
	c.nextID++
	body, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(AuthHeaderName(), c.authToken)
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s %s: %v", method, c.endpoint, err)
	}
	defer resp.Body.Close()

	// Capture session ID from response
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		c.sessionID = sid
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP %d on %s: %s", resp.StatusCode, method, string(respBody))
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		t.Fatalf("unmarshal response: %v\nbody: %s", err, string(respBody))
	}
	if rpcResp.Error != nil {
		t.Fatalf("RPC error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result
}

// callTool is a shorthand for tools/call.
func (c *mcpClient) callTool(t *testing.T, name string, args map[string]interface{}) json.RawMessage {
	t.Helper()
	if args == nil {
		args = map[string]interface{}{}
	}
	return c.call(t, "tools/call", map[string]interface{}{
		"name":      name,
		"arguments": args,
	})
}

func (c *mcpClient) initialize(t *testing.T) {
	t.Helper()
	result := c.call(t, "initialize", map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":   map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "integration-test",
			"version": "0.0.1",
		},
	})
	t.Logf("initialize: %s", string(result))
	t.Logf("session ID: %s", c.sessionID)

	// Send initialized notification (required by MCP protocol)
	c.nextID++
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	req, _ := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(AuthHeaderName(), c.authToken)
	if c.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", c.sessionID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("initialized notification failed: %v", err)
	} else {
		resp.Body.Close()
		t.Logf("initialized notification: HTTP %d", resp.StatusCode)
	}
}

func TestIntegration(t *testing.T) {
	clientConfig := loadTestConfig(t)

	t.Log("connecting to Sliver gRPC...")
	rpc, conn, err := transport.MTLSConnect(clientConfig)
	if err != nil {
		t.Fatalf("gRPC connect failed: %v", err)
	}
	defer conn.Close()
	t.Log("gRPC connected")

	cfg := Config{
		Transport:     testTransport,
		ListenAddress: testListenAddr,
		ServerName:    "sliver-mcp-test",
		ServerVersion: "0.0.0-test",
	}

	if err := Start(cfg, rpc); err != nil {
		t.Fatalf("MCP start failed: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		Stop(ctx)
	}()

	status := GetStatus()
	if !status.Running {
		t.Fatal("MCP server not running after Start")
	}
	token := status.AuthToken
	endpoint := fmt.Sprintf("http://%s/mcp", testListenAddr)
	t.Logf("MCP running at %s (token: %s...)", endpoint, token[:8])

	// Give transport a moment to bind
	time.Sleep(200 * time.Millisecond)

	client := newMCPClient(endpoint, token)

	// --- Initialize the MCP session ---
	t.Run("initialize", func(t *testing.T) {
		client.initialize(t)
		if client.sessionID == "" {
			t.Log("warning: no session ID received (server may be stateless)")
		}
	})

	// --- Test: tools/list ---
	t.Run("tools_list", func(t *testing.T) {
		result := client.call(t, "tools/list", nil)

		var toolsResp struct {
			Tools []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"tools"`
		}
		if err := json.Unmarshal(result, &toolsResp); err != nil {
			t.Fatalf("unmarshal tools: %v", err)
		}
		t.Logf("tools/list returned %d tools", len(toolsResp.Tools))
		if len(toolsResp.Tools) < 131 {
			t.Errorf("expected >= 131 tools, got %d", len(toolsResp.Tools))
		}
		for _, tool := range toolsResp.Tools {
			if tool.Description == "" {
				t.Errorf("tool %q has empty description", tool.Name)
			}
		}
	})

	// --- Server-side tools (no session needed) ---
	t.Run("list_sessions_and_beacons", func(t *testing.T) {
		result := client.callTool(t, "list_sessions_and_beacons", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("server_version", func(t *testing.T) {
		result := client.callTool(t, "server_version", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("list_operators", func(t *testing.T) {
		result := client.callTool(t, "list_operators", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("list_jobs", func(t *testing.T) {
		result := client.callTool(t, "list_jobs", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("list_implant_builds", func(t *testing.T) {
		result := client.callTool(t, "list_implant_builds", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("list_implant_profiles", func(t *testing.T) {
		result := client.callTool(t, "list_implant_profiles", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("get_compiler", func(t *testing.T) {
		result := client.callTool(t, "get_compiler", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("list_canaries", func(t *testing.T) {
		result := client.callTool(t, "list_canaries", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("list_websites", func(t *testing.T) {
		result := client.callTool(t, "list_websites", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("hosts_list", func(t *testing.T) {
		result := client.callTool(t, "hosts_list", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("loot_all", func(t *testing.T) {
		result := client.callTool(t, "loot_all", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("creds_list", func(t *testing.T) {
		result := client.callTool(t, "creds_list", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	t.Run("list_shellcode_encoders", func(t *testing.T) {
		result := client.callTool(t, "list_shellcode_encoders", nil)
		t.Logf("result: %s", truncate(string(result), 500))
	})

	// --- Error handling ---
	t.Run("session_info_missing", func(t *testing.T) {
		result := client.callTool(t, "session_info", map[string]interface{}{
			"session_id": "00000000-0000-0000-0000-000000000000",
		})
		var callResp struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}
		if err := json.Unmarshal(result, &callResp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !callResp.IsError {
			t.Error("expected error for nonexistent session")
		}
		t.Logf("session_info error (expected): %s", truncate(string(result), 300))
	})

	// --- Listener lifecycle ---
	t.Run("start_and_kill_mtls_listener", func(t *testing.T) {
		result := client.callTool(t, "start_mtls_listener", map[string]interface{}{
			"host": "127.0.0.1",
			"port": 44444,
		})
		t.Logf("start_mtls_listener: %s", truncate(string(result), 300))

		var startResp struct {
			Content []struct {
				Type string          `json:"type"`
				Text json.RawMessage `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}
		if err := json.Unmarshal(result, &startResp); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if startResp.IsError {
			t.Fatalf("start_mtls_listener error: %s", string(result))
		}

		if len(startResp.Content) > 0 {
			var textObj struct {
				JobID uint32 `json:"job_id"`
			}
			if err := json.Unmarshal([]byte(unquoteJSON(startResp.Content[0].Text)), &textObj); err == nil && textObj.JobID > 0 {
				killResult := client.callTool(t, "kill_job", map[string]interface{}{
					"job_id": textObj.JobID,
				})
				t.Logf("kill_job %d: %s", textObj.JobID, truncate(string(killResult), 300))
			}
		}
	})

	// --- Data round-trips ---
	t.Run("loot_roundtrip", func(t *testing.T) {
		addResult := client.callTool(t, "loot_add", map[string]interface{}{
			"name":      "mcp-test-loot",
			"loot_type": 1,
			"file_type": 0,
			"data":      "dGVzdCBkYXRh",
		})
		t.Logf("loot_add: %s", truncate(string(addResult), 300))

		var addResp struct {
			Content []struct {
				Text json.RawMessage `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}
		if err := json.Unmarshal(addResult, &addResp); err != nil || addResp.IsError {
			t.Skipf("loot_add failed: %s", truncate(string(addResult), 200))
		}

		listResult := client.callTool(t, "loot_all", nil)
		t.Logf("loot_list after add: %s", truncate(string(listResult), 400))
	})

	t.Run("creds_roundtrip", func(t *testing.T) {
		addResult := client.callTool(t, "creds_add", map[string]interface{}{
			"collection": "mcp-test",
			"username":   "testuser",
			"plaintext":  "testpassword",
			"hash_type":  0,
		})
		t.Logf("creds_add: %s", truncate(string(addResult), 300))
	})
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func unquoteJSON(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return string(raw)
	}
	return s
}
