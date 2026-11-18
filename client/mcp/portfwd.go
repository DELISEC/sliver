package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	startRportFwdToolName      = "start_rport_fwd"
	listRportFwdToolName       = "list_rport_fwd"
	stopRportFwdToolName       = "stop_rport_fwd"
	wgStartPortForwardToolName = "wg_start_port_forward"
	wgStopPortForwardToolName  = "wg_stop_port_forward"
	wgStartSocksToolName       = "wg_start_socks"
	wgStopSocksToolName        = "wg_stop_socks"
	wgListForwardersToolName   = "wg_list_forwarders"
	wgListSocksServersToolName = "wg_list_socks_servers"
	createSocksToolName        = "create_socks"
	closeSocksToolName         = "close_socks"
)

// --- Args ---

type startRportFwdArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	BindAddress    string `json:"bind_address" jsonschema:"description=Remote bind address,required"`
	BindPort       uint32 `json:"bind_port" jsonschema:"description=Remote bind port,required"`
	ForwardAddress string `json:"forward_address" jsonschema:"description=Local forward address,required"`
	ForwardPort    uint32 `json:"forward_port" jsonschema:"description=Local forward port,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type listRportFwdArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type stopRportFwdArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ID             uint32 `json:"id" jsonschema:"description=Reverse port forward listener ID,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type wgStartPortForwardArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	LocalPort      int32  `json:"local_port" jsonschema:"description=Local port to forward,required"`
	RemoteAddress  string `json:"remote_address" jsonschema:"description=Remote address to forward to,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type wgStopPortForwardArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ID             int32  `json:"id" jsonschema:"description=Forwarder ID to stop,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type wgStartSocksArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Port           int32  `json:"port" jsonschema:"description=SOCKS port,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type wgStopSocksArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ID             int32  `json:"id" jsonschema:"description=SOCKS server ID to stop,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type wgListForwardersArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type wgListSocksServersArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Results ---

type rportFwdItem struct {
	ID             uint32 `json:"id"`
	BindAddress    string `json:"bind_address"`
	BindPort       uint32 `json:"bind_port"`
	ForwardAddress string `json:"forward_address"`
	ForwardPort    uint32 `json:"forward_port"`
}

type rportFwdListResult struct {
	Listeners []rportFwdItem `json:"listeners"`
	Count     int            `json:"count"`
}

type wgForwarderItem struct {
	ID         int32  `json:"id"`
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
}

type wgForwardersResult struct {
	Forwarders []wgForwarderItem `json:"forwarders"`
	Count      int               `json:"count"`
}

type wgSocksItem struct {
	ID        int32  `json:"id"`
	LocalAddr string `json:"local_addr"`
}

type wgSocksResult struct {
	Servers []wgSocksItem `json:"servers"`
	Count   int           `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) startRportFwdHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startRportFwdArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(startRportFwdToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.StartRportFwdListener(ctx, &sliverpb.RportFwdStartListenerReq{
		BindAddress:    fmt.Sprintf("%s:%d", args.BindAddress, args.BindPort),
		ForwardAddress: fmt.Sprintf("%s:%d", args.ForwardAddress, args.ForwardPort),
		Request:        req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("start_rport_fwd failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("start_rport_fwd", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RportFwdListener{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(rportFwdItem{
		ID:             resp.ID,
		BindAddress:    resp.BindAddress,
		BindPort:       resp.BindPort,
		ForwardAddress: resp.ForwardAddress,
		ForwardPort:    resp.ForwardPort,
	}), nil
}

func (s *SliverMCPServer) listRportFwdHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args listRportFwdArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(listRportFwdToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.GetRportFwdListeners(ctx, &sliverpb.RportFwdListenersReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_rport_fwd failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("list_rport_fwd", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RportFwdListeners{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := rportFwdListResult{Listeners: make([]rportFwdItem, 0, len(resp.Listeners))}
	for _, l := range resp.Listeners {
		if l == nil {
			continue
		}
		result.Listeners = append(result.Listeners, rportFwdItem{
			ID:             l.ID,
			BindAddress:    l.BindAddress,
			BindPort:       l.BindPort,
			ForwardAddress: l.ForwardAddress,
			ForwardPort:    l.ForwardPort,
		})
	}
	result.Count = len(result.Listeners)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) stopRportFwdHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args stopRportFwdArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(stopRportFwdToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.StopRportFwdListener(ctx, &sliverpb.RportFwdStopListenerReq{
		ID:      args.ID,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("stop_rport_fwd failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("stop_rport_fwd", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.RportFwdListener{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		_ = waited
	}

	type stopResult struct {
		ID      uint32 `json:"id"`
		Stopped bool   `json:"stopped"`
	}
	return mcpapi.NewToolResultStructuredOnly(stopResult{ID: args.ID, Stopped: true}), nil
}

func (s *SliverMCPServer) wgStartPortForwardHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args wgStartPortForwardArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(wgStartPortForwardToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.WGStartPortForward(ctx, &sliverpb.WGPortForwardStartReq{
		LocalPort:     args.LocalPort,
		RemoteAddress: args.RemoteAddress,
		Request:       req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("wg_start_port_forward failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("wg_start_port_forward", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.WGPortForward{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := wgForwarderItem{}
	if resp.Forwarder != nil {
		result.ID = resp.Forwarder.ID
		result.LocalAddr = resp.Forwarder.LocalAddr
		result.RemoteAddr = resp.Forwarder.RemoteAddr
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) wgStopPortForwardHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args wgStopPortForwardArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(wgStopPortForwardToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.WGStopPortForward(ctx, &sliverpb.WGPortForwardStopReq{
		ID:      args.ID,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("wg_stop_port_forward failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("wg_stop_port_forward", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.WGPortForward{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		_ = waited
	}

	type stopResult struct {
		ID      int32 `json:"id"`
		Stopped bool  `json:"stopped"`
	}
	return mcpapi.NewToolResultStructuredOnly(stopResult{ID: args.ID, Stopped: true}), nil
}

func (s *SliverMCPServer) wgStartSocksHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args wgStartSocksArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(wgStartSocksToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.WGStartSocks(ctx, &sliverpb.WGSocksStartReq{
		Port:    args.Port,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("wg_start_socks failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("wg_start_socks", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.WGSocks{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := wgSocksItem{}
	if resp.Server != nil {
		result.ID = resp.Server.ID
		result.LocalAddr = resp.Server.LocalAddr
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) wgStopSocksHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args wgStopSocksArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(wgStopSocksToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.WGStopSocks(ctx, &sliverpb.WGSocksStopReq{
		ID:      args.ID,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("wg_stop_socks failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("wg_stop_socks", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.WGSocks{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		_ = waited
	}

	type stopResult struct {
		ID      int32 `json:"id"`
		Stopped bool  `json:"stopped"`
	}
	return mcpapi.NewToolResultStructuredOnly(stopResult{ID: args.ID, Stopped: true}), nil
}

func (s *SliverMCPServer) wgListForwardersHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args wgListForwardersArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(wgListForwardersToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.WGListForwarders(ctx, &sliverpb.WGTCPForwardersReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("wg_list_forwarders failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("wg_list_forwarders", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.WGTCPForwarders{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := wgForwardersResult{Forwarders: make([]wgForwarderItem, 0, len(resp.Forwarders))}
	for _, f := range resp.Forwarders {
		if f == nil {
			continue
		}
		result.Forwarders = append(result.Forwarders, wgForwarderItem{
			ID:         f.ID,
			LocalAddr:  f.LocalAddr,
			RemoteAddr: f.RemoteAddr,
		})
	}
	result.Count = len(result.Forwarders)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) wgListSocksServersHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args wgListSocksServersArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(wgListSocksServersToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.WGListSocksServers(ctx, &sliverpb.WGSocksServersReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("wg_list_socks_servers failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("wg_list_socks_servers", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.WGSocksServers{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := wgSocksResult{Servers: make([]wgSocksItem, 0, len(resp.Servers))}
	for _, s := range resp.Servers {
		if s == nil {
			continue
		}
		result.Servers = append(result.Servers, wgSocksItem{
			ID:        s.ID,
			LocalAddr: s.LocalAddr,
		})
	}
	result.Count = len(result.Servers)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// --- SOCKS proxy (unary) ---

type createSocksArgs struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID,required"`
	TunnelID  uint64 `json:"tunnel_id" jsonschema:"description=Tunnel ID"`
}

type closeSocksArgs struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID,required"`
	TunnelID  uint64 `json:"tunnel_id" jsonschema:"description=Tunnel ID"`
}

type socksResult struct {
	TunnelID  uint64 `json:"tunnel_id"`
	SessionID string `json:"session_id"`
}

func (s *SliverMCPServer) createSocksHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args createSocksArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(createSocksToolName, args.SessionID, "")

	resp, err := s.Rpc.CreateSocks(ctx, &sliverpb.Socks{
		SessionID: args.SessionID,
		TunnelID:  args.TunnelID,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("create_socks failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(socksResult{
		TunnelID:  resp.TunnelID,
		SessionID: resp.SessionID,
	}), nil
}

func (s *SliverMCPServer) closeSocksHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args closeSocksArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(closeSocksToolName, args.SessionID, "")

	if _, err := s.Rpc.CloseSocks(ctx, &sliverpb.Socks{
		SessionID: args.SessionID,
		TunnelID:  args.TunnelID,
	}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("close_socks failed", err), nil
	}

	type closeResult struct {
		Closed bool `json:"closed"`
	}
	return mcpapi.NewToolResultStructuredOnly(closeResult{Closed: true}), nil
}
