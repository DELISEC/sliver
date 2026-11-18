package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	serverVersionToolName = "server_version"
	listOperatorsToolName = "list_operators"
)

// --- Result structs ---

type versionResult struct {
	Major      int32  `json:"major"`
	Minor      int32  `json:"minor"`
	Patch      int32  `json:"patch"`
	Commit     string `json:"commit"`
	Dirty      bool   `json:"dirty"`
	CompiledAt int64  `json:"compiled_at"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
}

type operatorResult struct {
	Online bool   `json:"online"`
	Name   string `json:"name"`
}

type listOperatorsResult struct {
	Operators []operatorResult `json:"operators"`
	Count     int              `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) serverVersionHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(serverVersionToolName, "", "")

	ver, err := s.Rpc.GetVersion(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get version", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(versionResult{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}), nil
}

func (s *SliverMCPServer) listOperatorsHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(listOperatorsToolName, "", "")

	resp, err := s.Rpc.GetOperators(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list operators", err), nil
	}

	result := listOperatorsResult{
		Operators: make([]operatorResult, 0, len(resp.GetOperators())),
	}
	for _, op := range resp.GetOperators() {
		if op == nil {
			continue
		}
		result.Operators = append(result.Operators, operatorResult{
			Online: op.Online,
			Name:   op.Name,
		})
	}
	result.Count = len(result.Operators)

	if result.Count == 0 {
		return mcpapi.NewToolResultError(fmt.Sprintf("no operators connected")), nil
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
