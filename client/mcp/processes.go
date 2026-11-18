package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	listProcessesToolName    = "list_processes"
	terminateProcessToolName = "terminate_process"
	processDumpToolName      = "process_dump"
)

// --- Args structs ---

type listProcessesArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Result structs ---

type processResult struct {
	Pid          int32    `json:"pid"`
	Ppid         int32    `json:"ppid"`
	Executable   string   `json:"executable"`
	Owner        string   `json:"owner"`
	Architecture string   `json:"architecture"`
	CmdLine      []string `json:"cmd_line,omitempty"`
}

type listProcessesResult struct {
	Processes []processResult `json:"processes"`
	Count     int             `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) listProcessesHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args listProcessesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(listProcessesToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Ps(ctx, &sliverpb.PsReq{Request: req})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list processes", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("list_processes", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.Ps{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := listProcessesResult{
		Processes: make([]processResult, 0, len(resp.Processes)),
	}
	for _, p := range resp.Processes {
		if p == nil {
			continue
		}
		result.Processes = append(result.Processes, processResult{
			Pid:          p.Pid,
			Ppid:         p.Ppid,
			Executable:   p.Executable,
			Owner:        p.Owner,
			Architecture: p.Architecture,
			CmdLine:      p.CmdLine,
		})
	}
	result.Count = len(result.Processes)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// --- Tier 2: Terminate Process ---

type terminateProcessArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Pid            int32  `json:"pid" jsonschema:"description=Process ID to terminate,required"`
	Force          bool   `json:"force" jsonschema:"description=Force kill the process"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type terminateResult struct {
	Pid int32 `json:"pid"`
}

func (s *SliverMCPServer) terminateProcessHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args terminateProcessArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(terminateProcessToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Terminate(ctx, &sliverpb.TerminateReq{
		Pid:     args.Pid,
		Force:   args.Force,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("terminate failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("terminate_process", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.Terminate{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(terminateResult{
		Pid: resp.Pid,
	}), nil
}

// --- Tier 2: Process Dump ---

type processDumpArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Pid            int32  `json:"pid" jsonschema:"description=Process ID to dump,required"`
	Timeout        int32  `json:"timeout" jsonschema:"description=Dump timeout in seconds"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type processDumpResult struct {
	Size int    `json:"size"`
	Data string `json:"data"`
}

func (s *SliverMCPServer) processDumpHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args processDumpArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(processDumpToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.ProcessDump(ctx, &sliverpb.ProcessDumpReq{
		Pid:     args.Pid,
		Timeout: args.Timeout,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("process dump failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("process_dump", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.ProcessDump{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(processDumpResult{
		Size: len(resp.Data),
		Data: base64.StdEncoding.EncodeToString(resp.Data),
	}), nil
}
