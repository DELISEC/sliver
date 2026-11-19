package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	runOverlordSuiteToolName   = "run_overlord_suite"
	listOverlordSuitesToolName = "list_overlord_suites"
)

// --- Suite manifest types ---

type overlordManifest struct {
	Version string                    `json:"version"`
	DistDir string                    `json:"dist_dir"`
	Suites  map[string]overlordSuite  `json:"suites"`
}

type overlordSuite struct {
	Description string   `json:"description"`
	Type        string   `json:"type"`
	BOFs        []string `json:"bofs"`
}

// --- Tool args ---

type runOverlordSuiteArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID,required"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Suite          string `json:"suite" jsonschema:"description=Suite name (e.g. recon\\, fullrecon\\, network\\, privesc),required"`
	OverlordDir    string `json:"overlord_dir" jsonschema:"description=Path to overlord repo root (uses server default if empty)"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon tasks"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Per-BOF timeout in seconds (default 60)"`
}

type listOverlordSuitesArgs struct {
	OverlordDir string `json:"overlord_dir" jsonschema:"description=Path to overlord repo root (uses server default if empty)"`
}

// --- Results ---

type suiteResult struct {
	Suite     string      `json:"suite"`
	SessionID string      `json:"session_id"`
	Total     int         `json:"total"`
	Succeeded int         `json:"succeeded"`
	Failed    int         `json:"failed"`
	Results   []bofResult `json:"results"`
}

type bofResult struct {
	BOF       string `json:"bof"`
	Success   bool   `json:"success"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	ElapsedMs int64  `json:"elapsed_ms"`
}

type suitesListResult struct {
	Suites []suiteInfo `json:"suites"`
	Count  int         `json:"count"`
}

type suiteInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	BOFCount    int    `json:"bof_count"`
}

// --- Helpers ---

func (s *SliverMCPServer) resolveOverlordDir(override string) string {
	if override != "" {
		return override
	}
	return s.OverlordDir
}

func loadOverlordManifest(overlordDir string) (*overlordManifest, error) {
	data, err := os.ReadFile(filepath.Join(overlordDir, "suites.json"))
	if err != nil {
		return nil, fmt.Errorf("cannot read suites.json: %w", err)
	}
	var m overlordManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid suites.json: %w", err)
	}
	if m.DistDir == "" {
		m.DistDir = "dist"
	}
	return &m, nil
}

// --- Handlers ---

func (s *SliverMCPServer) listOverlordSuitesHandler(_ context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args listOverlordSuitesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	dir := s.resolveOverlordDir(args.OverlordDir)
	if dir == "" {
		return mcpapi.NewToolResultError("overlord directory not configured — pass overlord_dir or set --overlord-dir"), nil
	}

	manifest, err := loadOverlordManifest(dir)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	var suites []suiteInfo
	for name, suite := range manifest.Suites {
		suites = append(suites, suiteInfo{
			Name:        name,
			Description: suite.Description,
			Type:        suite.Type,
			BOFCount:    len(suite.BOFs),
		})
	}

	return mcpapi.NewToolResultStructuredOnly(suitesListResult{
		Suites: suites,
		Count:  len(suites),
	}), nil
}

func (s *SliverMCPServer) runOverlordSuiteHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args runOverlordSuiteArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	dir := s.resolveOverlordDir(args.OverlordDir)
	if dir == "" {
		return mcpapi.NewToolResultError("overlord directory not configured — pass overlord_dir or set --overlord-dir"), nil
	}

	manifest, err := loadOverlordManifest(dir)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	suite, ok := manifest.Suites[args.Suite]
	if !ok {
		names := make([]string, 0, len(manifest.Suites))
		for k := range manifest.Suites {
			names = append(names, k)
		}
		return mcpapi.NewToolResultError(fmt.Sprintf("unknown suite %q — available: %v", args.Suite, names)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(runOverlordSuiteToolName, args.SessionID, args.BeaconID)

	perBOFTimeout := args.TimeoutSeconds
	if perBOFTimeout <= 0 {
		perBOFTimeout = 60
	}

	distDir := filepath.Join(dir, manifest.DistDir)
	var results []bofResult
	succeeded := 0

	for _, bofName := range suite.BOFs {
		bofPath := filepath.Join(distDir, bofName+".o")
		bofData, readErr := os.ReadFile(bofPath)
		if readErr != nil {
			results = append(results, bofResult{
				BOF:   bofName,
				Error: fmt.Sprintf("failed to read %s: %v", bofPath, readErr),
			})
			continue
		}

		start := time.Now()
		result, callErr := s.callBOF(ctx, req, isBeacon, bofData, nil, args.Wait, perBOFTimeout)
		elapsed := time.Since(start).Milliseconds()

		if callErr != nil {
			results = append(results, bofResult{
				BOF:       bofName,
				Error:     callErr.Error(),
				ElapsedMs: elapsed,
			})
			continue
		}

		if result.AsyncTaskID != "" {
			results = append(results, bofResult{
				BOF:       bofName,
				Success:   true,
				Output:    fmt.Sprintf("async task queued: %s", result.AsyncTaskID),
				ElapsedMs: elapsed,
			})
			succeeded++
			continue
		}

		if result.Output == "" && result.RespErr != "" {
			results = append(results, bofResult{
				BOF:       bofName,
				Error:     result.RespErr,
				ElapsedMs: elapsed,
			})
			continue
		}

		results = append(results, bofResult{
			BOF:       bofName,
			Success:   true,
			Output:    result.Output,
			ElapsedMs: elapsed,
		})
		succeeded++
	}

	return mcpapi.NewToolResultStructuredOnly(suiteResult{
		Suite:     args.Suite,
		SessionID: args.SessionID,
		Total:     len(suite.BOFs),
		Succeeded: succeeded,
		Failed:    len(suite.BOFs) - succeeded,
		Results:   results,
	}), nil
}

