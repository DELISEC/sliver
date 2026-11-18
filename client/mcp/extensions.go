package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	callExtensionToolName     = "call_extension"
	listExtensionsToolName    = "list_extensions"
	registerExtensionToolName = "register_extension"
	executeBOFToolName        = "execute_bof"
	listBOFModulesToolName    = "list_bof_modules"
	runBOFModuleToolName      = "run_bof_module"
)

// --- Args ---

type callExtensionArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Name           string `json:"name" jsonschema:"description=Extension name,required"`
	Export         string `json:"export" jsonschema:"description=Export function to call"`
	Args           string `json:"args" jsonschema:"description=Base64-encoded arguments"`
	ServerStore    bool   `json:"server_store" jsonschema:"description=Store output on server"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type listExtensionsArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type registerExtensionArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Name           string `json:"name" jsonschema:"description=Extension name,required"`
	Data           string `json:"data" jsonschema:"description=Base64-encoded extension data,required"`
	OS             string `json:"os" jsonschema:"description=Target OS (windows/linux/darwin),required"`
	Init           string `json:"init" jsonschema:"description=Init function name"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type executeBOFArgs struct {
	SessionID      string          `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string          `json:"beacon_id" jsonschema:"description=Beacon ID"`
	BOFData        string          `json:"bof_data" jsonschema:"description=Base64-encoded BOF .o file,required"`
	Arguments      []bofArgument   `json:"arguments" jsonschema:"description=BOF arguments"`
	Wait           bool            `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64           `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type bofArgument struct {
	Type  string `json:"type" jsonschema:"description=Argument type (string/wstring/int/short/data),required"`
	Value string `json:"value" jsonschema:"description=Argument value (base64 for data type),required"`
}

type listBOFModulesArgs struct {
	ModulesDir string `json:"modules_dir" jsonschema:"description=Path to BOF modules directory"`
}

type runBOFModuleArgs struct {
	SessionID      string        `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string        `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ModuleName     string        `json:"module_name" jsonschema:"description=BOF module name,required"`
	ModulesDir     string        `json:"modules_dir" jsonschema:"description=Path to BOF modules directory"`
	Arguments      []bofArgument `json:"arguments" jsonschema:"description=BOF arguments"`
	Wait           bool          `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64         `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- BOF Module manifest ---

type bofManifest struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	BOFFile     string        `json:"bof_file"`
	Arguments   []bofArgSpec  `json:"arguments"`
}

type bofArgSpec struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// --- Results ---

type extensionResult struct {
	Output string `json:"output"`
}

type listExtensionsResult struct {
	Names []string `json:"names"`
	Count int      `json:"count"`
}

type bofModuleItem struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Arguments   []bofArgSpec `json:"arguments,omitempty"`
}

type bofModulesResult struct {
	Modules []bofModuleItem `json:"modules"`
	Count   int             `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) callExtensionHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args callExtensionArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(callExtensionToolName, args.SessionID, args.BeaconID)

	extReq := &sliverpb.CallExtensionReq{
		Name:        args.Name,
		Export:      args.Export,
		ServerStore: args.ServerStore,
		Request:     req,
	}
	if args.Args != "" {
		argBytes, err := base64.StdEncoding.DecodeString(args.Args)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 args: %v", err)), nil
		}
		extReq.Args = argBytes
	}

	resp, err := s.Rpc.CallExtension(ctx, extReq)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("call_extension failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("call_extension", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.CallExtension{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(extensionResult{
		Output: base64.StdEncoding.EncodeToString(resp.Output),
	}), nil
}

func (s *SliverMCPServer) listExtensionsHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args listExtensionsArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(listExtensionsToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.ListExtensions(ctx, &sliverpb.ListExtensionsReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_extensions failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("list_extensions", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.ListExtensions{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(listExtensionsResult{
		Names: resp.Names,
		Count: len(resp.Names),
	}), nil
}

func (s *SliverMCPServer) registerExtensionHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registerExtensionArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registerExtensionToolName, args.SessionID, args.BeaconID)

	data, err := base64.StdEncoding.DecodeString(args.Data)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 data: %v", err)), nil
	}

	resp, err := s.Rpc.RegisterExtension(ctx, &sliverpb.RegisterExtensionReq{
		Name:    args.Name,
		Data:    data,
		OS:      args.OS,
		Init:    args.Init,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("register_extension failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("register_extension", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RegisterExtension{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type regResult struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(regResult{Success: true}), nil
}

func packBOFArgs(arguments []bofArgument) ([]byte, error) {
	buf := core.BOFArgsBuffer{}
	for _, arg := range arguments {
		switch arg.Type {
		case "string":
			if err := buf.AddString(arg.Value); err != nil {
				return nil, fmt.Errorf("AddString failed: %w", err)
			}
		case "wstring":
			if err := buf.AddWString(arg.Value); err != nil {
				return nil, fmt.Errorf("AddWString failed: %w", err)
			}
		case "int":
			var v uint32
			if _, err := fmt.Sscanf(arg.Value, "%d", &v); err != nil {
				return nil, fmt.Errorf("invalid int value %q: %w", arg.Value, err)
			}
			if err := buf.AddInt(v); err != nil {
				return nil, fmt.Errorf("AddInt failed: %w", err)
			}
		case "short":
			var v uint16
			if _, err := fmt.Sscanf(arg.Value, "%d", &v); err != nil {
				return nil, fmt.Errorf("invalid short value %q: %w", arg.Value, err)
			}
			if err := buf.AddShort(v); err != nil {
				return nil, fmt.Errorf("AddShort failed: %w", err)
			}
		case "data":
			d, err := base64.StdEncoding.DecodeString(arg.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid base64 data: %w", err)
			}
			if err := buf.AddData(d); err != nil {
				return nil, fmt.Errorf("AddData failed: %w", err)
			}
		default:
			return nil, fmt.Errorf("unknown argument type: %s", arg.Type)
		}
	}
	return buf.GetBuffer()
}

func (s *SliverMCPServer) executeBOFHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args executeBOFArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(executeBOFToolName, args.SessionID, args.BeaconID)

	bofData, err := base64.StdEncoding.DecodeString(args.BOFData)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 bof_data: %v", err)), nil
	}

	var packedArgs []byte
	if len(args.Arguments) > 0 {
		packedArgs, err = packBOFArgs(args.Arguments)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("failed to pack BOF args: %v", err)), nil
		}
	}

	resp, err := s.Rpc.CallExtension(ctx, &sliverpb.CallExtensionReq{
		IsBOF:          true,
		BOFData:        bofData,
		Export:         "go",
		Args:           packedArgs,
		WantBOFOutputs: true,
		Request:        req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("execute_bof failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("execute_bof", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.CallExtension{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(extensionResult{
		Output: base64.StdEncoding.EncodeToString(resp.Output),
	}), nil
}

func (s *SliverMCPServer) listBOFModulesHandler(_ context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args listBOFModulesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	dir := args.ModulesDir
	if dir == "" {
		dir = s.BOFModulesDir
	}
	if dir == "" {
		return mcpapi.NewToolResultError("bof modules directory not configured"), nil
	}
	s.logToolCall(listBOFModulesToolName, "", "")

	entries, err := os.ReadDir(dir)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("failed to read modules dir: %v", err)), nil
	}

	result := bofModulesResult{Modules: make([]bofModuleItem, 0)}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(dir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		var manifest bofManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}
		result.Modules = append(result.Modules, bofModuleItem{
			Name:        manifest.Name,
			Description: manifest.Description,
			Arguments:   manifest.Arguments,
		})
	}
	result.Count = len(result.Modules)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) runBOFModuleHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args runBOFModuleArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	dir := args.ModulesDir
	if dir == "" {
		dir = s.BOFModulesDir
	}
	if dir == "" {
		return mcpapi.NewToolResultError("bof modules directory not configured"), nil
	}

	manifestPath := filepath.Join(dir, args.ModuleName, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("module %q not found: %v", args.ModuleName, err)), nil
	}
	var manifest bofManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid manifest: %v", err)), nil
	}

	bofPath := filepath.Join(dir, args.ModuleName, manifest.BOFFile)
	bofBytes, err := os.ReadFile(bofPath)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("failed to read BOF file: %v", err)), nil
	}

	bofB64 := base64.StdEncoding.EncodeToString(bofBytes)

	inner := executeBOFArgs{
		SessionID:      args.SessionID,
		BeaconID:       args.BeaconID,
		BOFData:        bofB64,
		Arguments:      args.Arguments,
		Wait:           args.Wait,
		TimeoutSeconds: args.TimeoutSeconds,
	}
	innerReq := mcpapi.CallToolRequest{}
	innerReq.Params.Name = executeBOFToolName

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(runBOFModuleToolName, args.SessionID, args.BeaconID)

	var packedArgs []byte
	if len(inner.Arguments) > 0 {
		packedArgs, err = packBOFArgs(inner.Arguments)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("failed to pack BOF args: %v", err)), nil
		}
	}

	resp, err := s.Rpc.CallExtension(ctx, &sliverpb.CallExtensionReq{
		IsBOF:          true,
		BOFData:        bofBytes,
		Export:         "go",
		Args:           packedArgs,
		WantBOFOutputs: true,
		Request:        req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("run_bof_module failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("run_bof_module", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.CallExtension{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(extensionResult{
		Output: base64.StdEncoding.EncodeToString(resp.Output),
	}), nil
}
