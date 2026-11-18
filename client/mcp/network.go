package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	networkInterfacesToolName  = "network_interfaces"
	networkConnectionsToolName = "network_connections"
)

// --- Args structs ---

type networkInterfacesArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type networkConnectionsArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	TCP            bool   `json:"tcp" jsonschema:"description=Include TCP connections (default: true)"`
	UDP            bool   `json:"udp" jsonschema:"description=Include UDP connections (default: true)"`
	IP4            bool   `json:"ip4" jsonschema:"description=Include IPv4 (default: true)"`
	IP6            bool   `json:"ip6" jsonschema:"description=Include IPv6"`
	Listening      bool   `json:"listening" jsonschema:"description=Only show listening sockets"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Result structs ---

type netInterfaceResult struct {
	Index       int32    `json:"index"`
	Name        string   `json:"name"`
	MAC         string   `json:"mac"`
	IPAddresses []string `json:"ip_addresses"`
}

type networkInterfacesResult struct {
	Interfaces []netInterfaceResult `json:"interfaces"`
	Count      int                  `json:"count"`
}

type sockAddrResult struct {
	IP   string `json:"ip"`
	Port uint32 `json:"port"`
}

type sockTabEntryResult struct {
	LocalAddr  sockAddrResult `json:"local_addr"`
	RemoteAddr sockAddrResult `json:"remote_addr"`
	State      string         `json:"state"`
	UID        uint32         `json:"uid"`
	Protocol   string         `json:"protocol"`
	PID        int32          `json:"pid,omitempty"`
	Executable string         `json:"executable,omitempty"`
}

type networkConnectionsResult struct {
	Entries []sockTabEntryResult `json:"entries"`
	Count   int                  `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) networkInterfacesHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args networkInterfacesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(networkInterfacesToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Ifconfig(ctx, &sliverpb.IfconfigReq{Request: req})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get network interfaces", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("network_interfaces", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.Ifconfig{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := networkInterfacesResult{
		Interfaces: make([]netInterfaceResult, 0, len(resp.NetInterfaces)),
	}
	for _, iface := range resp.NetInterfaces {
		if iface == nil {
			continue
		}
		result.Interfaces = append(result.Interfaces, netInterfaceResult{
			Index:       iface.Index,
			Name:        iface.Name,
			MAC:         iface.MAC,
			IPAddresses: iface.IPAddresses,
		})
	}
	result.Count = len(result.Interfaces)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) networkConnectionsHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args networkConnectionsArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := buildRequest(args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(networkConnectionsToolName, args.SessionID, args.BeaconID)

	tcp := true
	udp := true
	ip4 := true
	if request.GetArguments()["tcp"] != nil {
		tcp = args.TCP
	}
	if request.GetArguments()["udp"] != nil {
		udp = args.UDP
	}
	if request.GetArguments()["ip4"] != nil {
		ip4 = args.IP4
	}

	resp, err := s.Rpc.Netstat(ctx, &sliverpb.NetstatReq{
		TCP:       tcp,
		UDP:       udp,
		IP4:       ip4,
		IP6:       args.IP6,
		Listening: args.Listening,
		Request:   req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get network connections", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("network_connections", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.Netstat{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := networkConnectionsResult{
		Entries: make([]sockTabEntryResult, 0, len(resp.Entries)),
	}
	for _, entry := range resp.Entries {
		if entry == nil {
			continue
		}
		e := sockTabEntryResult{
			State:    entry.SkState,
			UID:      entry.UID,
			Protocol: entry.Protocol,
		}
		if entry.LocalAddr != nil {
			e.LocalAddr = sockAddrResult{IP: entry.LocalAddr.Ip, Port: entry.LocalAddr.Port}
		}
		if entry.RemoteAddr != nil {
			e.RemoteAddr = sockAddrResult{IP: entry.RemoteAddr.Ip, Port: entry.RemoteAddr.Port}
		}
		if entry.Process != nil {
			e.PID = entry.Process.Pid
			e.Executable = entry.Process.Executable
		}
		result.Entries = append(result.Entries, e)
	}
	result.Count = len(result.Entries)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
