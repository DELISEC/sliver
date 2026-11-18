package mcp

import (
	"log"
	"os"
	"testing"
)

func TestNewServerRegistersAllTools(t *testing.T) {
	cfg := Config{
		ServerName:    "sliver-mcp-test",
		ServerVersion: "0.0.0-test",
	}
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	srv := newServer(cfg, nil, logger)
	if srv == nil {
		t.Fatal("newServer returned nil")
	}
	if srv.server == nil {
		t.Fatal("newServer did not create underlying MCPServer")
	}

	tools := srv.server.ListTools()
	count := len(tools)

	const expectedMinTools = 131
	if count < expectedMinTools {
		t.Errorf("expected at least %d registered tools, got %d", expectedMinTools, count)
		// Print missing for debugging
		t.Log("Registered tools:")
		for name := range tools {
			t.Logf("  - %s", name)
		}
	}
	t.Logf("Total registered tools: %d", count)
}

func TestAllToolsHaveHandlers(t *testing.T) {
	cfg := Config{
		ServerName:    "sliver-mcp-test",
		ServerVersion: "0.0.0-test",
	}
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	srv := newServer(cfg, nil, logger)
	tools := srv.server.ListTools()

	for name, tool := range tools {
		if tool == nil {
			t.Errorf("tool %q has nil ServerTool entry", name)
			continue
		}
		if tool.Handler == nil {
			t.Errorf("tool %q is registered but has no handler", name)
		}
	}
}

func TestAllToolsHaveDescriptions(t *testing.T) {
	cfg := Config{
		ServerName:    "sliver-mcp-test",
		ServerVersion: "0.0.0-test",
	}
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	srv := newServer(cfg, nil, logger)
	tools := srv.server.ListTools()

	for name, tool := range tools {
		if tool == nil {
			continue
		}
		desc := tool.Tool.Description
		if desc == "" {
			t.Errorf("tool %q has no description", name)
		}
	}
}

func TestNoDuplicateToolNames(t *testing.T) {
	cfg := Config{
		ServerName:    "sliver-mcp-test",
		ServerVersion: "0.0.0-test",
	}
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	srv := newServer(cfg, nil, logger)
	tools := srv.server.ListTools()

	seen := make(map[string]bool, len(tools))
	for name := range tools {
		if seen[name] {
			t.Errorf("duplicate tool name: %q", name)
		}
		seen[name] = true
	}
}
