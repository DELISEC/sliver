package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	pivotStartListenerToolName    = "pivot_start_listener"
	pivotStopListenerToolName     = "pivot_stop_listener"
	pivotSessionListenersToolName = "pivot_session_listeners"
	pivotGraphToolName            = "pivot_graph"
)

// --- Args ---

type pivotStartListenerArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Type           int32  `json:"type" jsonschema:"description=Pivot type (0=TCP 1=UDP 2=NamedPipe),required"`
	BindAddress    string `json:"bind_address" jsonschema:"description=Address to bind the pivot listener,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type pivotStopListenerArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ID             uint32 `json:"id" jsonschema:"description=Pivot listener ID to stop,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type pivotSessionListenersArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Results ---

type pivotPeer struct {
	PeerID        int64  `json:"peer_id"`
	RemoteAddress string `json:"remote_address"`
}

type pivotListenerResult struct {
	ID          uint32      `json:"id"`
	Type        int32       `json:"type"`
	BindAddress string      `json:"bind_address"`
	Pivots      []pivotPeer `json:"pivots,omitempty"`
}

type pivotListenersResult struct {
	Listeners []pivotListenerResult `json:"listeners"`
	Count     int                   `json:"count"`
}

type pivotGraphEntry struct {
	PeerID    int64             `json:"peer_id"`
	SessionID string            `json:"session_id,omitempty"`
	Name      string            `json:"name"`
	Children  []pivotGraphEntry `json:"children,omitempty"`
}

type pivotGraphResult struct {
	Children []pivotGraphEntry `json:"children"`
}

// --- Handlers ---

func (s *SliverMCPServer) pivotStartListenerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args pivotStartListenerArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(pivotStartListenerToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.PivotStartListener(ctx, &sliverpb.PivotStartListenerReq{
		Type:        sliverpb.PivotType(args.Type),
		BindAddress: args.BindAddress,
		Request:     req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("pivot_start_listener failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("pivot_start_listener", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.PivotListener{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := pivotListenerResult{
		ID:          resp.ID,
		Type:        int32(resp.Type),
		BindAddress: resp.BindAddress,
	}
	for _, p := range resp.Pivots {
		if p == nil {
			continue
		}
		result.Pivots = append(result.Pivots, pivotPeer{
			PeerID:        p.PeerID,
			RemoteAddress: p.RemoteAddress,
		})
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) pivotStopListenerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args pivotStopListenerArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(pivotStopListenerToolName, args.SessionID, args.BeaconID)

	if _, err := s.Rpc.PivotStopListener(ctx, &sliverpb.PivotStopListenerReq{
		ID:      args.ID,
		Request: req,
	}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("pivot_stop_listener failed", err), nil
	}

	if isBeacon {
		return newAsyncResult("pivot_stop_listener", "", args.BeaconID), nil
	}

	type stopResult struct {
		ID      uint32 `json:"id"`
		Stopped bool   `json:"stopped"`
	}
	return mcpapi.NewToolResultStructuredOnly(stopResult{ID: args.ID, Stopped: true}), nil
}

func (s *SliverMCPServer) pivotSessionListenersHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args pivotSessionListenersArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(pivotSessionListenersToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.PivotSessionListeners(ctx, &sliverpb.PivotListenersReq{
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("pivot_session_listeners failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("pivot_session_listeners", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.PivotListeners{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := pivotListenersResult{Listeners: make([]pivotListenerResult, 0, len(resp.Listeners))}
	for _, l := range resp.Listeners {
		if l == nil {
			continue
		}
		item := pivotListenerResult{
			ID:          l.ID,
			Type:        int32(l.Type),
			BindAddress: l.BindAddress,
		}
		for _, p := range l.Pivots {
			if p == nil {
				continue
			}
			item.Pivots = append(item.Pivots, pivotPeer{
				PeerID:        p.PeerID,
				RemoteAddress: p.RemoteAddress,
			})
		}
		result.Listeners = append(result.Listeners, item)
	}
	result.Count = len(result.Listeners)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func buildPivotGraphEntries(entries []*clientpb.PivotGraphEntry) []pivotGraphEntry {
	result := make([]pivotGraphEntry, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		item := pivotGraphEntry{
			PeerID: e.PeerID,
			Name:   e.Name,
		}
		if e.Session != nil {
			item.SessionID = e.Session.ID
		}
		if len(e.Children) > 0 {
			item.Children = buildPivotGraphEntries(e.Children)
		}
		result = append(result, item)
	}
	return result
}

func (s *SliverMCPServer) pivotGraphHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(pivotGraphToolName, "", "")

	resp, err := s.Rpc.PivotGraph(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("pivot_graph failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(pivotGraphResult{
		Children: buildPivotGraphEntries(resp.Children),
	}), nil
}
