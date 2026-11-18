package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	getSystemToolName         = "get_system"
	runAsToolName             = "run_as"
	impersonateToolName       = "impersonate"
	revToSelfToolName         = "rev_to_self"
	makeTokenToolName         = "make_token"
	currentTokenOwnerToolName = "current_token_owner"
	getPrivsToolName          = "get_privs"
)

// --- Args ---

type getSystemArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	HostingProcess string `json:"hosting_process" jsonschema:"description=Process to migrate into (e.g. svchost.exe)"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type runAsArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Username       string `json:"username" jsonschema:"description=Username to run as,required"`
	ProcessName    string `json:"process_name" jsonschema:"description=Process to execute,required"`
	Args           string `json:"args" jsonschema:"description=Arguments for the process"`
	Domain         string `json:"domain" jsonschema:"description=Domain for the user"`
	Password       string `json:"password" jsonschema:"description=Password for the user"`
	HideWindow     bool   `json:"hide_window" jsonschema:"description=Hide the process window"`
	NetOnly        bool   `json:"net_only" jsonschema:"description=Network-only logon"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type impersonateArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Username       string `json:"username" jsonschema:"description=Username to impersonate,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type revToSelfArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type makeTokenArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Username       string `json:"username" jsonschema:"description=Username,required"`
	Password       string `json:"password" jsonschema:"description=Password,required"`
	Domain         string `json:"domain" jsonschema:"description=Domain"`
	LogonType      uint32 `json:"logon_type" jsonschema:"description=Windows logon type (default 9 = LOGON32_LOGON_NEW_CREDENTIALS)"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type currentTokenOwnerArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type getPrivsArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Results ---

type privilegeEntry struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Enabled          bool   `json:"enabled"`
	EnabledByDefault bool   `json:"enabled_by_default"`
	Removed          bool   `json:"removed"`
	UsedForAccess    bool   `json:"used_for_access"`
}

type getPrivsResult struct {
	Privileges       []privilegeEntry `json:"privileges"`
	Count            int              `json:"count"`
	ProcessIntegrity string           `json:"process_integrity"`
	ProcessName      string           `json:"process_name"`
}

type tokenOwnerResult struct {
	Output  string `json:"output"`
	Success bool   `json:"success"`
}

type runAsResult struct {
	Output  string `json:"output"`
	Success bool   `json:"success"`
}

// --- Handlers ---

func (s *SliverMCPServer) getSystemHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args getSystemArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(getSystemToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.GetSystem(ctx, &clientpb.GetSystemReq{
		HostingProcess: args.HostingProcess,
		Request:        req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("get_system failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("get_system", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.GetSystem{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{Success: true}), nil
}

func (s *SliverMCPServer) runAsHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args runAsArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(runAsToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RunAs(ctx, &sliverpb.RunAsReq{
		Username:    args.Username,
		ProcessName: args.ProcessName,
		Args:        args.Args,
		Domain:      args.Domain,
		Password:    args.Password,
		HideWindow:  args.HideWindow,
		NetOnly:     args.NetOnly,
		Request:     req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("run_as failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("run_as", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RunAs{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		return mcpapi.NewToolResultStructuredOnly(runAsResult{Output: waited.Output, Success: true}), nil
	}

	return mcpapi.NewToolResultStructuredOnly(runAsResult{Output: resp.Output, Success: true}), nil
}

func (s *SliverMCPServer) impersonateHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args impersonateArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(impersonateToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Impersonate(ctx, &sliverpb.ImpersonateReq{
		Username: args.Username,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("impersonate failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("impersonate", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Impersonate{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{Success: true}), nil
}

func (s *SliverMCPServer) revToSelfHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args revToSelfArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(revToSelfToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RevToSelf(ctx, &sliverpb.RevToSelfReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("rev_to_self failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("rev_to_self", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RevToSelf{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{Success: true}), nil
}

func (s *SliverMCPServer) makeTokenHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args makeTokenArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(makeTokenToolName, args.SessionID, args.BeaconID)

	logonType := args.LogonType
	if logonType == 0 {
		logonType = 9
	}

	resp, err := s.Rpc.MakeToken(ctx, &sliverpb.MakeTokenReq{
		Username:  args.Username,
		Password:  args.Password,
		Domain:    args.Domain,
		LogonType: logonType,
		Request:   req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("make_token failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("make_token", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.MakeToken{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{Success: true}), nil
}

func (s *SliverMCPServer) currentTokenOwnerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args currentTokenOwnerArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(currentTokenOwnerToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.CurrentTokenOwner(ctx, &sliverpb.CurrentTokenOwnerReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("current_token_owner failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("current_token_owner", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.CurrentTokenOwner{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		return mcpapi.NewToolResultStructuredOnly(tokenOwnerResult{Output: waited.Output, Success: true}), nil
	}

	return mcpapi.NewToolResultStructuredOnly(tokenOwnerResult{Output: resp.Output, Success: true}), nil
}

func (s *SliverMCPServer) getPrivsHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args getPrivsArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(getPrivsToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.GetPrivs(ctx, &sliverpb.GetPrivsReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("get_privs failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("get_privs", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.GetPrivs{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := getPrivsResult{
		Privileges:       make([]privilegeEntry, 0, len(resp.PrivInfo)),
		ProcessIntegrity: resp.ProcessIntegrity,
		ProcessName:      resp.ProcessName,
	}
	for _, p := range resp.PrivInfo {
		if p == nil {
			continue
		}
		result.Privileges = append(result.Privileges, privilegeEntry{
			Name:             p.Name,
			Description:      p.Description,
			Enabled:          p.Enabled,
			EnabledByDefault: p.EnabledByDefault,
			Removed:          p.Removed,
			UsedForAccess:    p.UsedForAccess,
		})
	}
	result.Count = len(result.Privileges)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
