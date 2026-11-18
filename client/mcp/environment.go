package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	getEnvironmentToolName = "get_environment"
)

// --- Args structs ---

type getEnvironmentArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Name           string `json:"name" jsonschema:"description=Filter by environment variable name (empty for all)"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Result structs ---

type envVarResult struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type getEnvironmentResult struct {
	Variables []envVarResult `json:"variables"`
	Count     int            `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) getEnvironmentHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args getEnvironmentArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(getEnvironmentToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.GetEnv(ctx, &sliverpb.EnvReq{
		Name:    args.Name,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get environment", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("get_environment", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.EnvInfo{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := getEnvironmentResult{
		Variables: make([]envVarResult, 0, len(resp.Variables)),
	}
	for _, v := range resp.Variables {
		if v == nil {
			continue
		}
		result.Variables = append(result.Variables, envVarResult{
			Key:   v.Key,
			Value: v.Value,
		})
	}
	result.Count = len(result.Variables)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
