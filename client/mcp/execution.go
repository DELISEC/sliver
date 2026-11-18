package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	executeToolName         = "execute"
	executeAssemblyToolName = "execute_assembly"
	sideloadToolName        = "sideload"
	spawnDllToolName        = "spawn_dll"
	runSSHCommandToolName   = "run_ssh_command"
	backdoorToolName        = "backdoor"
	hijackDllToolName       = "hijack_dll"
)

// --- Args structs ---

type executeArgs struct {
	SessionID      string            `json:"session_id" jsonschema:"description=Session ID to execute on"`
	BeaconID       string            `json:"beacon_id" jsonschema:"description=Beacon ID to execute on"`
	Path           string            `json:"path" jsonschema:"description=Path to executable,required"`
	Args           []string          `json:"args" jsonschema:"description=Command arguments"`
	Output         bool              `json:"output" jsonschema:"description=Capture stdout/stderr (default: true)"`
	EnvInheritance bool              `json:"env_inheritance" jsonschema:"description=Inherit parent environment"`
	Env            map[string]string `json:"env" jsonschema:"description=Additional environment variables"`
	PPid           uint32            `json:"ppid" jsonschema:"description=Parent PID for spoofing (Windows)"`
	Wait           bool              `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64             `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type executeAssemblyArgs struct {
	SessionID      string   `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string   `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Assembly       string   `json:"assembly" jsonschema:"description=Base64-encoded .NET assembly,required"`
	Arguments      []string `json:"arguments" jsonschema:"description=Arguments to pass to the assembly"`
	Process        string   `json:"process" jsonschema:"description=Host process for execution"`
	IsDLL          bool     `json:"is_dll" jsonschema:"description=Assembly is a DLL"`
	Arch           string   `json:"arch" jsonschema:"description=Target architecture (x86/x64)"`
	ClassName      string   `json:"class_name" jsonschema:"description=Class name for DLL"`
	Method         string   `json:"method" jsonschema:"description=Method name for DLL"`
	AppDomain      string   `json:"app_domain" jsonschema:"description=AppDomain name"`
	PPid           uint32   `json:"ppid" jsonschema:"description=Parent PID for spoofing"`
	InProcess      bool     `json:"in_process" jsonschema:"description=Run in-process"`
	Runtime        string   `json:"runtime" jsonschema:"description=CLR runtime version"`
	AmsiBypass     bool     `json:"amsi_bypass" jsonschema:"description=Bypass AMSI"`
	EtwBypass      bool     `json:"etw_bypass" jsonschema:"description=Bypass ETW"`
	Wait           bool     `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64    `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type sideloadArgs struct {
	SessionID      string   `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string   `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Data           string   `json:"data" jsonschema:"description=Base64-encoded shared library,required"`
	ProcessName    string   `json:"process_name" jsonschema:"description=Host process name"`
	Args           []string `json:"args" jsonschema:"description=Arguments"`
	EntryPoint     string   `json:"entry_point" jsonschema:"description=Entry point function name"`
	Kill           bool     `json:"kill" jsonschema:"description=Kill host process after execution"`
	IsDLL          bool     `json:"is_dll" jsonschema:"description=Data is a DLL"`
	IsUnicode      bool     `json:"is_unicode" jsonschema:"description=Use unicode strings"`
	PPid           uint32   `json:"ppid" jsonschema:"description=Parent PID for spoofing"`
	Wait           bool     `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64    `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type spawnDllArgs struct {
	SessionID      string   `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string   `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Data           string   `json:"data" jsonschema:"description=Base64-encoded DLL,required"`
	ProcessName    string   `json:"process_name" jsonschema:"description=Host process name"`
	Args           []string `json:"args" jsonschema:"description=Arguments"`
	EntryPoint     string   `json:"entry_point" jsonschema:"description=DLL entry point"`
	Kill           bool     `json:"kill" jsonschema:"description=Kill host process after execution"`
	PPid           uint32   `json:"ppid" jsonschema:"description=Parent PID for spoofing"`
	Wait           bool     `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64    `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type runSSHCommandArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Username       string `json:"username" jsonschema:"description=SSH username,required"`
	Hostname       string `json:"hostname" jsonschema:"description=SSH hostname,required"`
	Port           uint32 `json:"port" jsonschema:"description=SSH port (default 22)"`
	Command        string `json:"command" jsonschema:"description=Command to execute,required"`
	Password       string `json:"password" jsonschema:"description=SSH password"`
	PrivKey        string `json:"priv_key" jsonschema:"description=Base64-encoded SSH private key"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type backdoorArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	FilePath       string `json:"file_path" jsonschema:"description=Path to binary to backdoor,required"`
	ProfileName    string `json:"profile_name" jsonschema:"description=Implant profile name,required"`
	Name           string `json:"name" jsonschema:"description=Name for the backdoored binary"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type hijackDllArgs struct {
	SessionID        string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID         string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ReferenceDLLPath string `json:"reference_dll_path" jsonschema:"description=Path to reference DLL,required"`
	TargetLocation   string `json:"target_location" jsonschema:"description=Target location for hijacked DLL,required"`
	ReferenceDLL     string `json:"reference_dll" jsonschema:"description=Base64-encoded reference DLL bytes"`
	TargetDLL        string `json:"target_dll" jsonschema:"description=Base64-encoded target DLL bytes"`
	ProfileName      string `json:"profile_name" jsonschema:"description=Implant profile name,required"`
	Name             string `json:"name" jsonschema:"description=Name for the implant"`
	Wait             bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds   int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Result structs ---

type executeResult struct {
	Status uint32 `json:"status"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Pid    uint32 `json:"pid"`
}

type assemblyResult struct {
	Output string `json:"output"`
}

type sideloadResult struct {
	Result string `json:"result"`
}

type sshCommandResult struct {
	StdOut string `json:"stdout"`
	StdErr string `json:"stderr"`
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

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
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

func (s *SliverMCPServer) executeAssemblyHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args executeAssemblyArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(executeAssemblyToolName, args.SessionID, args.BeaconID)

	asmBytes, err := base64.StdEncoding.DecodeString(args.Assembly)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 assembly: %v", err)), nil
	}

	resp, err := s.Rpc.ExecuteAssembly(ctx, &sliverpb.ExecuteAssemblyReq{
		Assembly:   asmBytes,
		Arguments:  args.Arguments,
		Process:    args.Process,
		IsDLL:      args.IsDLL,
		Arch:       args.Arch,
		ClassName:  args.ClassName,
		Method:     args.Method,
		AppDomain:  args.AppDomain,
		PPid:       args.PPid,
		InProcess:  args.InProcess,
		Runtime:    args.Runtime,
		AmsiBypass: args.AmsiBypass,
		EtwBypass:  args.EtwBypass,
		Request:    req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("execute_assembly failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("execute_assembly", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.ExecuteAssembly{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(assemblyResult{Output: string(resp.Output)}), nil
}

func (s *SliverMCPServer) sideloadHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args sideloadArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(sideloadToolName, args.SessionID, args.BeaconID)

	data, err := base64.StdEncoding.DecodeString(args.Data)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 data: %v", err)), nil
	}

	resp, err := s.Rpc.Sideload(ctx, &sliverpb.SideloadReq{
		Data:        data,
		ProcessName: args.ProcessName,
		Args:        args.Args,
		EntryPoint:  args.EntryPoint,
		Kill:        args.Kill,
		IsDLL:       args.IsDLL,
		IsUnicode:   args.IsUnicode,
		PPid:        args.PPid,
		Request:     req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("sideload failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("sideload", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Sideload{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(sideloadResult{Result: resp.Result}), nil
}

func (s *SliverMCPServer) spawnDllHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args spawnDllArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(spawnDllToolName, args.SessionID, args.BeaconID)

	data, err := base64.StdEncoding.DecodeString(args.Data)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 data: %v", err)), nil
	}

	resp, err := s.Rpc.SpawnDll(ctx, &sliverpb.InvokeSpawnDllReq{
		Data:        data,
		ProcessName: args.ProcessName,
		Args:        args.Args,
		EntryPoint:  args.EntryPoint,
		Kill:        args.Kill,
		PPid:        args.PPid,
		Request:     req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("spawn_dll failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("spawn_dll", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.SpawnDll{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(sideloadResult{Result: resp.Result}), nil
}

func (s *SliverMCPServer) runSSHCommandHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args runSSHCommandArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(runSSHCommandToolName, args.SessionID, args.BeaconID)

	port := args.Port
	if port == 0 {
		port = 22
	}

	sshReq := &sliverpb.SSHCommandReq{
		Username: args.Username,
		Hostname: args.Hostname,
		Port:     port,
		Command:  args.Command,
		Password: args.Password,
		Request:  req,
	}
	if args.PrivKey != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(args.PrivKey)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 private key: %v", err)), nil
		}
		sshReq.PrivKey = keyBytes
	}

	resp, err := s.Rpc.RunSSHCommand(ctx, sshReq)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("ssh command failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("run_ssh_command", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.SSHCommand{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(sshCommandResult{
		StdOut: resp.StdOut,
		StdErr: resp.StdErr,
	}), nil
}

func (s *SliverMCPServer) backdoorHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args backdoorArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(backdoorToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Backdoor(ctx, &clientpb.BackdoorReq{
		FilePath:    args.FilePath,
		ProfileName: args.ProfileName,
		Name:        args.Name,
		Request:     req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("backdoor failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("backdoor", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &clientpb.Backdoor{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type backdoorResult struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(backdoorResult{Success: true}), nil
}

func (s *SliverMCPServer) hijackDllHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args hijackDllArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(hijackDllToolName, args.SessionID, args.BeaconID)

	hijackReq := &clientpb.DllHijackReq{
		ReferenceDLLPath: args.ReferenceDLLPath,
		TargetLocation:   args.TargetLocation,
		ProfileName:      args.ProfileName,
		Name:             args.Name,
		Request:          req,
	}
	if args.ReferenceDLL != "" {
		refBytes, err := base64.StdEncoding.DecodeString(args.ReferenceDLL)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 reference_dll: %v", err)), nil
		}
		hijackReq.ReferenceDLL = refBytes
	}
	if args.TargetDLL != "" {
		tgtBytes, err := base64.StdEncoding.DecodeString(args.TargetDLL)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 target_dll: %v", err)), nil
		}
		hijackReq.TargetDLL = tgtBytes
	}

	resp, err := s.Rpc.HijackDLL(ctx, hijackReq)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("hijack_dll failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("hijack_dll", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &clientpb.DllHijack{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type hijackResult struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(hijackResult{Success: true}), nil
}
