package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	executeToolName = "execute"
)

// --- Args structs ---

type executeArgs struct {
	SessionID      string            `json:"session_id" jsonschema:"description=Session ID to execute on"`
	BeaconID       string            `json:"beacon_id" jsonschema:"description=Beacon ID to execute on"`
	Path           string            `json:"path" jsonschema:"description=Path to executable,required"`
	Args           []string          `json:"args" jsonschema:"description=Command arguments"`
	Output         bool              `json:"output" jsonschema:"description=Capture stdout/stderr (default: true)"`
	EnvInheritance bool             `json:"env_inheritance" jsonschema:"description=Inherit parent environment"`
	Env            map[string]string `json:"env" jsonschema:"description=Additional environment variables"`
	PPid           uint32            `json:"ppid" jsonschema:"description=Parent PID for spoofing (Windows)"`
	Wait           bool              `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64             `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Result structs ---

type executeResult struct {
	Status uint32 `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Pid    uint32 `json:"pid"`
}

// --- Handlers ---

func (s *SliverMCPServer) executeHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args executeArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(executeToolName, args.SessionID, args.BeaconID)

	output := true
	if request.GetArguments()["output"] != nil {
		output = args.Output
	}

	resp, err := s.Rpc.Execute(ctx, &sliverpb.ExecuteReq{
		Path:           args.Path,
		Args:           args.Args,
		Output:         output,
		EnvInheritance: args.EnvInheritance,
		Env:            args.Env,
		PPid:           args.PPid,
		Request:        req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("execute failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("execute", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.Execute{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(executeResult{
		Status: resp.Status,
		Stdout: string(resp.Stdout),
		Stderr: string(resp.Stderr),
		Pid:    resp.Pid,
	}), nil
}
