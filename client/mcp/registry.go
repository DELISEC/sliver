package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	registryReadToolName        = "registry_read"
	registryWriteToolName       = "registry_write"
	registryCreateKeyToolName   = "registry_create_key"
	registryDeleteKeyToolName   = "registry_delete_key"
	registryListSubkeysToolName = "registry_list_subkeys"
	registryListValuesToolName  = "registry_list_values"
	registryReadHiveToolName    = "registry_read_hive"
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

// --- Tier 2: Registry Write ---

type registryWriteArgs struct {
	SessionID       string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID        string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Hive            string `json:"hive" jsonschema:"description=Registry hive (HKCU HKLM etc),required"`
	Path            string `json:"path" jsonschema:"description=Registry key path,required"`
	Key             string `json:"key" jsonschema:"description=Registry value name,required"`
	Hostname        string `json:"hostname" jsonschema:"description=Target hostname for remote registry"`
	StringValue     string `json:"string_value" jsonschema:"description=String value to write"`
	ByteValueBase64 string `json:"byte_value_base64" jsonschema:"description=Binary value as base64"`
	DWordValue      uint32 `json:"dword_value" jsonschema:"description=DWORD value"`
	QWordValue      uint64 `json:"qword_value" jsonschema:"description=QWORD value"`
	RegType         uint32 `json:"reg_type" jsonschema:"description=Registry value type (1=SZ 2=EXPAND_SZ 3=BINARY 4=DWORD 11=QWORD),required"`
	Wait            bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds  int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type registryWriteResult struct {
	Success bool `json:"success"`
}

func (s *SliverMCPServer) registryWriteHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registryWriteArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	var byteValue []byte
	if args.ByteValueBase64 != "" {
		var decErr error
		byteValue, decErr = base64.StdEncoding.DecodeString(args.ByteValueBase64)
		if decErr != nil {
			return mcpapi.NewToolResultError("invalid base64 byte value: " + decErr.Error()), nil
		}
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registryWriteToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RegistryWrite(ctx, &sliverpb.RegistryWriteReq{
		Hive:        args.Hive,
		Path:        args.Path,
		Key:         args.Key,
		Hostname:    args.Hostname,
		StringValue: args.StringValue,
		ByteValue:   byteValue,
		DWordValue:  args.DWordValue,
		QWordValue:  args.QWordValue,
		Type:        args.RegType,
		Request:     req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("registry write failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("registry_write", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RegistryWrite{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	return mcpapi.NewToolResultStructuredOnly(registryWriteResult{Success: true}), nil
}

// --- Tier 2: Registry Create Key ---

type registryCreateKeyArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Hive           string `json:"hive" jsonschema:"description=Registry hive,required"`
	Path           string `json:"path" jsonschema:"description=Registry key path,required"`
	Key            string `json:"key" jsonschema:"description=Subkey name to create,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type registryCreateKeyResult struct {
	Success bool `json:"success"`
}

func (s *SliverMCPServer) registryCreateKeyHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registryCreateKeyArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registryCreateKeyToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RegistryCreateKey(ctx, &sliverpb.RegistryCreateKeyReq{
		Hive:     args.Hive,
		Path:     args.Path,
		Key:      args.Key,
		Hostname: args.Hostname,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("registry create key failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("registry_create_key", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RegistryCreateKey{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	return mcpapi.NewToolResultStructuredOnly(registryCreateKeyResult{Success: true}), nil
}

// --- Tier 2: Registry Delete Key ---

type registryDeleteKeyArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Hive           string `json:"hive" jsonschema:"description=Registry hive,required"`
	Path           string `json:"path" jsonschema:"description=Registry key path,required"`
	Key            string `json:"key" jsonschema:"description=Subkey name to delete,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type registryDeleteKeyResult struct {
	Success bool `json:"success"`
}

func (s *SliverMCPServer) registryDeleteKeyHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registryDeleteKeyArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registryDeleteKeyToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RegistryDeleteKey(ctx, &sliverpb.RegistryDeleteKeyReq{
		Hive:     args.Hive,
		Path:     args.Path,
		Key:      args.Key,
		Hostname: args.Hostname,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("registry delete key failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("registry_delete_key", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RegistryDeleteKey{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	return mcpapi.NewToolResultStructuredOnly(registryDeleteKeyResult{Success: true}), nil
}

// --- Tier 2: Registry List Subkeys ---

type registryListSubkeysArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Hive           string `json:"hive" jsonschema:"description=Registry hive,required"`
	Path           string `json:"path" jsonschema:"description=Registry key path,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type registryListSubkeysResult struct {
	Subkeys []string `json:"subkeys"`
	Count   int      `json:"count"`
}

func (s *SliverMCPServer) registryListSubkeysHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registryListSubkeysArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registryListSubkeysToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RegistryListSubKeys(ctx, &sliverpb.RegistrySubKeyListReq{
		Hive:     args.Hive,
		Path:     args.Path,
		Hostname: args.Hostname,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("registry list subkeys failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("registry_list_subkeys", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RegistrySubKeyList{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(registryListSubkeysResult{
		Subkeys: resp.Subkeys,
		Count:   len(resp.Subkeys),
	}), nil
}

// --- Tier 2: Registry List Values ---

type registryListValuesArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Hive           string `json:"hive" jsonschema:"description=Registry hive,required"`
	Path           string `json:"path" jsonschema:"description=Registry key path,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type registryListValuesResult struct {
	ValueNames []string `json:"value_names"`
	Count      int      `json:"count"`
}

func (s *SliverMCPServer) registryListValuesHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registryListValuesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registryListValuesToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RegistryListValues(ctx, &sliverpb.RegistryListValuesReq{
		Hive:     args.Hive,
		Path:     args.Path,
		Hostname: args.Hostname,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("registry list values failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("registry_list_values", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RegistryValuesList{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(registryListValuesResult{
		ValueNames: resp.ValueNames,
		Count:      len(resp.ValueNames),
	}), nil
}

// --- Tier 2: Registry Read Hive ---

type registryReadHiveArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	RootHive       string `json:"root_hive" jsonschema:"description=Root hive name,required"`
	RequestedHive  string `json:"requested_hive" jsonschema:"description=Requested hive path,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type registryReadHiveResult struct {
	DataBase64 string `json:"data_base64"`
	Size       int    `json:"size"`
	Encoder    string `json:"encoder,omitempty"`
}

func (s *SliverMCPServer) registryReadHiveHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args registryReadHiveArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(registryReadHiveToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RegistryReadHive(ctx, &sliverpb.RegistryReadHiveReq{
		RootHive:      args.RootHive,
		RequestedHive: args.RequestedHive,
		Request:       req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("registry read hive failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("registry_read_hive", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RegistryReadHive{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(registryReadHiveResult{
		DataBase64: base64.StdEncoding.EncodeToString(resp.Data),
		Size:       len(resp.Data),
		Encoder:    resp.Encoder,
	}), nil
}
