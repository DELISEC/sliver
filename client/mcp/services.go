package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	listServicesToolName       = "list_services"
	serviceDetailToolName      = "service_detail"
	startServiceByNameToolName = "start_service_by_name"
	startServiceToolName       = "start_service"
	stopServiceToolName        = "stop_service"
	removeServiceToolName      = "remove_service"
)

// --- Args ---

type listServicesArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname (optional for remote service enumeration)"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type serviceDetailArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ServiceName    string `json:"service_name" jsonschema:"description=Name of the service,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type startServiceByNameArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ServiceName    string `json:"service_name" jsonschema:"description=Name of the service to start,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type startServiceArgs struct {
	SessionID          string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID           string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ServiceName        string `json:"service_name" jsonschema:"description=Name for the new service,required"`
	ServiceDescription string `json:"service_description" jsonschema:"description=Service description"`
	BinPath            string `json:"bin_path" jsonschema:"description=Binary path for the service,required"`
	Hostname           string `json:"hostname" jsonschema:"description=Target hostname"`
	Arguments          string `json:"arguments" jsonschema:"description=Arguments for the service binary"`
	Wait               bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds     int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type stopServiceArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ServiceName    string `json:"service_name" jsonschema:"description=Name of the service to stop,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type removeServiceArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ServiceName    string `json:"service_name" jsonschema:"description=Name of the service to remove,required"`
	Hostname       string `json:"hostname" jsonschema:"description=Target hostname"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Results ---

type serviceDetailResult struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      uint32 `json:"status"`
	StartupType uint32 `json:"startup_type"`
	BinPath     string `json:"bin_path"`
	Account     string `json:"account"`
}

type listServicesResult struct {
	Services []serviceDetailResult `json:"services"`
	Count    int                   `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) listServicesHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args listServicesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(listServicesToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Services(ctx, &sliverpb.ServicesReq{
		Hostname: args.Hostname,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_services failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("list_services", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Services{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	if resp.Error != "" {
		return mcpapi.NewToolResultError("service error: " + resp.Error), nil
	}

	result := listServicesResult{
		Services: make([]serviceDetailResult, 0, len(resp.Details)),
	}
	for _, svc := range resp.Details {
		if svc == nil {
			continue
		}
		result.Services = append(result.Services, serviceDetailResult{
			Name:        svc.Name,
			DisplayName: svc.DisplayName,
			Description: svc.Description,
			Status:      svc.Status,
			StartupType: svc.StartupType,
			BinPath:     svc.BinPath,
			Account:     svc.Account,
		})
	}
	result.Count = len(result.Services)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) serviceDetailHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args serviceDetailArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(serviceDetailToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.ServiceDetail(ctx, &sliverpb.ServiceDetailReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{
			ServiceName: args.ServiceName,
			Hostname:    args.Hostname,
		},
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("service_detail failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("service_detail", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.ServiceDetail{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	if resp.Detail == nil {
		return mcpapi.NewToolResultError("service not found: " + resp.Message), nil
	}

	return mcpapi.NewToolResultStructuredOnly(serviceDetailResult{
		Name:        resp.Detail.Name,
		DisplayName: resp.Detail.DisplayName,
		Description: resp.Detail.Description,
		Status:      resp.Detail.Status,
		StartupType: resp.Detail.StartupType,
		BinPath:     resp.Detail.BinPath,
		Account:     resp.Detail.Account,
	}), nil
}

func (s *SliverMCPServer) startServiceByNameHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startServiceByNameArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(startServiceByNameToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.StartServiceByName(ctx, &sliverpb.StartServiceByNameReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{
			ServiceName: args.ServiceName,
			Hostname:    args.Hostname,
		},
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("start_service_by_name failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("start_service_by_name", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.ServiceInfo{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		ServiceName string `json:"service_name"`
		Success     bool   `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{ServiceName: args.ServiceName, Success: true}), nil
}

func (s *SliverMCPServer) startServiceHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startServiceArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(startServiceToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.StartService(ctx, &sliverpb.StartServiceReq{
		ServiceName:        args.ServiceName,
		ServiceDescription: args.ServiceDescription,
		BinPath:            args.BinPath,
		Hostname:           args.Hostname,
		Arguments:          args.Arguments,
		Request:            req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("start_service failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("start_service", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.ServiceInfo{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		ServiceName string `json:"service_name"`
		Success     bool   `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{ServiceName: args.ServiceName, Success: true}), nil
}

func (s *SliverMCPServer) stopServiceHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args stopServiceArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(stopServiceToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.StopService(ctx, &sliverpb.StopServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{
			ServiceName: args.ServiceName,
			Hostname:    args.Hostname,
		},
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("stop_service failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("stop_service", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.ServiceInfo{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		ServiceName string `json:"service_name"`
		Success     bool   `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{ServiceName: args.ServiceName, Success: true}), nil
}

func (s *SliverMCPServer) removeServiceHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args removeServiceArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(removeServiceToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.RemoveService(ctx, &sliverpb.RemoveServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{
			ServiceName: args.ServiceName,
			Hostname:    args.Hostname,
		},
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("remove_service failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		if !args.Wait {
			return newAsyncResult("remove_service", resp.Response.TaskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.ServiceInfo{}
		if err := s.waitForBeaconTaskResponse(tctx, resp.Response.TaskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	type result struct {
		ServiceName string `json:"service_name"`
		Success     bool   `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{ServiceName: args.ServiceName, Success: true}), nil
}
