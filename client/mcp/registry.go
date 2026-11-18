package mcp

import (
	"context"
	"fmt"
	"encoding/base64"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	registryReadToolName = "registry_read"
)

// --- Args structs ---

type registryReadArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Hive           string `json:"hive" jsonschema:"description=Registry hive (e.g. HKCU HKLM HKCR HKU HKCC),required"`
	Path           string `json:"path" jsonschema:"description=Registry key path,required"`
	Key            string `json:"key" jsonschema:"description=Registry value name,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname for remote registry"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Result structs ---

type registryReadResult struct {
	Value  string `json:"value"`
	Binary string `json:"binary,omitempty"`
	Type   string `json:"type"`
}

// --- Handlers ---

func (s *SliverMCPServer) registryReadHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registryReadArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registryReadToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RegistryRead(ctx, &sliverpb.RegistryReadReq{
		Hive:     args.Hive,
		Path:     args.Path,
		Key:      args.Key,
		Hostname: args.Hostname,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to read registry", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("registry_read", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.RegistryRead{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := registryReadResult{
		Value: resp.Value,
		Type:  resp.Type.String(),
	}
	if len(resp.Binary) > 0 {
		result.Binary = base64.StdEncoding.EncodeToString(resp.Binary)
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
