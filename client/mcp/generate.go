package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	generateImplantToolName       = "generate_implant"
	regenerateImplantToolName     = "regenerate_implant"
	generateStageToolName         = "generate_stage"
	listImplantBuildsToolName     = "list_implant_builds"
	deleteImplantBuildToolName    = "delete_implant_build"
	listImplantProfilesToolName   = "list_implant_profiles"
	saveImplantProfileToolName    = "save_implant_profile"
	deleteImplantProfileToolName  = "delete_implant_profile"
	getCompilerToolName           = "get_compiler"
	listCanariesToolName          = "list_canaries"
	shellcodeRDIToolName          = "shellcode_rdi"
	shellcodeEncodeToolName       = "shellcode_encode"
	listShellcodeEncodersToolName = "list_shellcode_encoders"
)

// --- Args ---

type generateImplantArgs struct {
	Name                string   `json:"name" jsonschema:"description=Name for the implant"`
	OS                  string   `json:"os" jsonschema:"description=Target OS (windows/linux/darwin),required"`
	Arch                string   `json:"arch" jsonschema:"description=Target architecture (amd64/386/arm64),required"`
	IsBeacon            bool     `json:"is_beacon" jsonschema:"description=Generate as beacon (vs session)"`
	BeaconInterval      int64    `json:"beacon_interval" jsonschema:"description=Beacon interval in nanoseconds"`
	BeaconJitter        int64    `json:"beacon_jitter" jsonschema:"description=Beacon jitter in nanoseconds"`
	C2URLs              []string `json:"c2_urls" jsonschema:"description=C2 URLs (e.g. mtls://host:port https://host),required"`
	Format              int32    `json:"format" jsonschema:"description=Output format (0=SHARED_LIB 1=SHELLCODE 2=EXECUTABLE 3=SERVICE)"`
	Debug               bool     `json:"debug" jsonschema:"description=Enable debug mode"`
	Evasion             bool     `json:"evasion" jsonschema:"description=Enable evasion features"`
	ObfuscateSymbols    bool     `json:"obfuscate_symbols" jsonschema:"description=Obfuscate Go symbols"`
	IncludeMTLS         bool     `json:"include_mtls" jsonschema:"description=Include mTLS transport"`
	IncludeHTTP         bool     `json:"include_http" jsonschema:"description=Include HTTP transport"`
	IncludeWG           bool     `json:"include_wg" jsonschema:"description=Include WireGuard transport"`
	IncludeDNS          bool     `json:"include_dns" jsonschema:"description=Include DNS transport"`
	TemplateName        string   `json:"template_name" jsonschema:"description=Implant template name"`
	ReconnectInterval   int64    `json:"reconnect_interval" jsonschema:"description=Reconnect interval in nanoseconds"`
	MaxConnectionErrors uint32   `json:"max_connection_errors" jsonschema:"description=Max connection errors before giving up"`
	LimitHostname       string   `json:"limit_hostname" jsonschema:"description=Only execute on matching hostname"`
	LimitUsername       string   `json:"limit_username" jsonschema:"description=Only execute as matching username"`
	LimitFileExists     string   `json:"limit_file_exists" jsonschema:"description=Only execute if file exists"`
	OutputDir           string   `json:"output_dir" jsonschema:"description=Directory to save output file (default /tmp)"`
}

type regenerateImplantArgs struct {
	ImplantName string `json:"implant_name" jsonschema:"description=Name of the implant to regenerate,required"`
}

type generateStageArgs struct {
	Profile       string `json:"profile" jsonschema:"description=Profile name to use,required"`
	Name          string `json:"name" jsonschema:"description=Name for the stage"`
	AESEncryptKey string `json:"aes_encrypt_key" jsonschema:"description=AES encryption key"`
	AESEncryptIv  string `json:"aes_encrypt_iv" jsonschema:"description=AES encryption IV"`
	RC4EncryptKey string `json:"rc4_encrypt_key" jsonschema:"description=RC4 encryption key"`
	PrependSize   bool   `json:"prepend_size" jsonschema:"description=Prepend size to payload"`
	CompressF     string `json:"compress_format" jsonschema:"description=Compression format"`
	Compress      string `json:"compress" jsonschema:"description=Compression method"`
}

type deleteImplantBuildArgs struct {
	Name string `json:"name" jsonschema:"description=Implant build name to delete,required"`
}

type saveImplantProfileArgs struct {
	Name             string   `json:"name" jsonschema:"description=Profile name,required"`
	OS               string   `json:"os" jsonschema:"description=Target OS,required"`
	Arch             string   `json:"arch" jsonschema:"description=Target architecture,required"`
	IsBeacon         bool     `json:"is_beacon" jsonschema:"description=Beacon mode"`
	BeaconInterval   int64    `json:"beacon_interval" jsonschema:"description=Beacon interval"`
	BeaconJitter     int64    `json:"beacon_jitter" jsonschema:"description=Beacon jitter"`
	C2URLs           []string `json:"c2_urls" jsonschema:"description=C2 URLs"`
	Format           int32    `json:"format" jsonschema:"description=Output format"`
	Debug            bool     `json:"debug" jsonschema:"description=Debug mode"`
	Evasion          bool     `json:"evasion" jsonschema:"description=Evasion features"`
	ObfuscateSymbols bool     `json:"obfuscate_symbols" jsonschema:"description=Obfuscate symbols"`
}

type deleteImplantProfileArgs struct {
	Name string `json:"name" jsonschema:"description=Profile name to delete,required"`
}

type shellcodeRDIArgs struct {
	Data         string `json:"data" jsonschema:"description=Base64-encoded DLL data,required"`
	FunctionName string `json:"function_name" jsonschema:"description=Function to call"`
	Arguments    string `json:"arguments" jsonschema:"description=Arguments to the function"`
}

type shellcodeEncodeArgs struct {
	Encoder      int32  `json:"encoder" jsonschema:"description=Encoder type (0=NONE 1=SHIKATA_GA_NAI 2=XOR 3=XOR_DYNAMIC),required"`
	Architecture string `json:"architecture" jsonschema:"description=Target architecture"`
	Iterations   uint32 `json:"iterations" jsonschema:"description=Encoding iterations"`
	BadChars     string `json:"bad_chars" jsonschema:"description=Base64-encoded bad characters"`
	Data         string `json:"data" jsonschema:"description=Base64-encoded shellcode,required"`
}

// --- Results ---

type generateResult struct {
	FileName    string `json:"file_name"`
	ImplantName string `json:"implant_name"`
	Size        int    `json:"size"`
	OutputPath  string `json:"output_path"`
}

type implantBuildItem struct {
	Name   string `json:"name"`
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Format string `json:"format"`
	Staged bool   `json:"staged"`
}

type implantBuildsResult struct {
	Builds []implantBuildItem `json:"builds"`
	Count  int                `json:"count"`
}

type implantProfileItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

type implantProfilesResult struct {
	Profiles []implantProfileItem `json:"profiles"`
	Count    int                  `json:"count"`
}

type compilerResult struct {
	GOOS    string           `json:"goos"`
	GOARCH  string           `json:"goarch"`
	Targets []compilerTarget `json:"targets"`
}

type compilerTarget struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	Format string `json:"format"`
}

type canaryItem struct {
	ID             string `json:"id"`
	ImplantName    string `json:"implant_name"`
	Domain         string `json:"domain"`
	Triggered      bool   `json:"triggered"`
	FirstTriggered string `json:"first_triggered"`
	LatestTrigger  string `json:"latest_trigger"`
	Count          uint32 `json:"count"`
}

type canariesResult struct {
	Canaries []canaryItem `json:"canaries"`
	Count    int          `json:"count"`
}

type shellcodeResult struct {
	Size int    `json:"size"`
	Data string `json:"data"`
}

// --- Helpers ---

func buildC2List(urls []string) []*clientpb.ImplantC2 {
	c2s := make([]*clientpb.ImplantC2, 0, len(urls))
	for i, u := range urls {
		c2s = append(c2s, &clientpb.ImplantC2{
			Priority: uint32(i),
			URL:      u,
		})
	}
	return c2s
}

func buildImplantConfig(os, arch string, isBeacon bool, beaconInterval, beaconJitter int64,
	c2URLs []string, format int32, debug, evasion, obfuscate bool,
	includeMTLS, includeHTTP, includeWG, includeDNS bool,
	templateName string, reconnectInterval int64, maxConnErrors uint32,
	limitHostname, limitUsername, limitFileExists string) *clientpb.ImplantConfig {

	cfg := &clientpb.ImplantConfig{
		GOOS:                os,
		GOARCH:              arch,
		IsBeacon:            isBeacon,
		BeaconInterval:      beaconInterval,
		BeaconJitter:        beaconJitter,
		C2:                  buildC2List(c2URLs),
		Format:              clientpb.OutputFormat(format),
		Debug:               debug,
		Evasion:             evasion,
		ObfuscateSymbols:    obfuscate,
		IncludeMTLS:         includeMTLS,
		IncludeHTTP:         includeHTTP,
		IncludeWG:           includeWG,
		IncludeDNS:          includeDNS,
		TemplateName:        templateName,
		ReconnectInterval:   reconnectInterval,
		MaxConnectionErrors: maxConnErrors,
		LimitHostname:       limitHostname,
		LimitUsername:       limitUsername,
		LimitFileExists:     limitFileExists,
		HTTPC2ConfigName:    "default",
	}
	return cfg
}

// --- Handlers ---

func (s *SliverMCPServer) generateImplantHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args generateImplantArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(generateImplantToolName, "", "")

	cfg := buildImplantConfig(args.OS, args.Arch, args.IsBeacon,
		args.BeaconInterval, args.BeaconJitter, args.C2URLs,
		args.Format, args.Debug, args.Evasion, args.ObfuscateSymbols,
		args.IncludeMTLS, args.IncludeHTTP, args.IncludeWG, args.IncludeDNS,
		args.TemplateName, args.ReconnectInterval, args.MaxConnectionErrors,
		args.LimitHostname, args.LimitUsername, args.LimitFileExists)

	resp, err := s.Rpc.Generate(ctx, &clientpb.GenerateReq{
		Config: cfg,
		Name:   args.Name,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("generate_implant failed", err), nil
	}

	result := generateResult{
		ImplantName: resp.ImplantName,
	}
	if resp.File != nil {
		result.FileName = resp.File.Name
		result.Size = len(resp.File.Data)

		outDir := args.OutputDir
		if outDir == "" {
			outDir = os.TempDir()
		}
		outPath := filepath.Join(outDir, resp.File.Name)
		if err := os.WriteFile(outPath, resp.File.Data, 0o600); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to write output file", err), nil
		}
		result.OutputPath = outPath
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) regenerateImplantHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args regenerateImplantArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(regenerateImplantToolName, "", "")

	resp, err := s.Rpc.Regenerate(ctx, &clientpb.RegenerateReq{
		ImplantName: args.ImplantName,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("regenerate_implant failed", err), nil
	}

	result := generateResult{ImplantName: resp.ImplantName}
	if resp.File != nil {
		result.FileName = resp.File.Name
		result.Size = len(resp.File.Data)

		outPath := filepath.Join(os.TempDir(), resp.File.Name)
		if err := os.WriteFile(outPath, resp.File.Data, 0o600); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to write output file", err), nil
		}
		result.OutputPath = outPath
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) generateStageHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args generateStageArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(generateStageToolName, "", "")

	resp, err := s.Rpc.GenerateStage(ctx, &clientpb.GenerateStageReq{
		Profile:       args.Profile,
		Name:          args.Name,
		AESEncryptKey: args.AESEncryptKey,
		AESEncryptIv:  args.AESEncryptIv,
		RC4EncryptKey: args.RC4EncryptKey,
		PrependSize:   args.PrependSize,
		CompressF:     args.CompressF,
		Compress:      args.Compress,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("generate_stage failed", err), nil
	}

	result := generateResult{}
	if resp.File != nil {
		result.FileName = resp.File.Name
		result.Size = len(resp.File.Data)

		outPath := filepath.Join(os.TempDir(), resp.File.Name)
		if err := os.WriteFile(outPath, resp.File.Data, 0o600); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to write output file", err), nil
		}
		result.OutputPath = outPath
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) listImplantBuildsHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(listImplantBuildsToolName, "", "")

	resp, err := s.Rpc.ImplantBuilds(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_implant_builds failed", err), nil
	}

	result := implantBuildsResult{Builds: make([]implantBuildItem, 0)}
	for name, cfg := range resp.Configs {
		item := implantBuildItem{Name: name}
		if cfg != nil {
			item.OS = cfg.GOOS
			item.Arch = cfg.GOARCH
			item.Format = cfg.Format.String()
		}
		if resp.Staged != nil {
			item.Staged = resp.Staged[name]
		}
		result.Builds = append(result.Builds, item)
	}
	result.Count = len(result.Builds)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) deleteImplantBuildHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args deleteImplantBuildArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(deleteImplantBuildToolName, "", "")

	if _, err := s.Rpc.DeleteImplantBuild(ctx, &clientpb.DeleteReq{Name: args.Name}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("delete_implant_build failed", err), nil
	}

	type delResult struct {
		Name    string `json:"name"`
		Deleted bool   `json:"deleted"`
	}
	return mcpapi.NewToolResultStructuredOnly(delResult{Name: args.Name, Deleted: true}), nil
}

func (s *SliverMCPServer) listImplantProfilesHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(listImplantProfilesToolName, "", "")

	resp, err := s.Rpc.ImplantProfiles(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_implant_profiles failed", err), nil
	}

	result := implantProfilesResult{Profiles: make([]implantProfileItem, 0, len(resp.Profiles))}
	for _, p := range resp.Profiles {
		if p == nil {
			continue
		}
		item := implantProfileItem{ID: p.ID, Name: p.Name}
		if p.Config != nil {
			item.OS = p.Config.GOOS
			item.Arch = p.Config.GOARCH
		}
		result.Profiles = append(result.Profiles, item)
	}
	result.Count = len(result.Profiles)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) saveImplantProfileHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args saveImplantProfileArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(saveImplantProfileToolName, "", "")

	cfg := &clientpb.ImplantConfig{
		GOOS:             args.OS,
		GOARCH:           args.Arch,
		IsBeacon:         args.IsBeacon,
		BeaconInterval:   args.BeaconInterval,
		BeaconJitter:     args.BeaconJitter,
		C2:               buildC2List(args.C2URLs),
		Format:           clientpb.OutputFormat(args.Format),
		Debug:            args.Debug,
		Evasion:          args.Evasion,
		ObfuscateSymbols: args.ObfuscateSymbols,
	}

	resp, err := s.Rpc.SaveImplantProfile(ctx, &clientpb.ImplantProfile{
		Name:   args.Name,
		Config: cfg,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("save_implant_profile failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(implantProfileItem{
		ID:   resp.ID,
		Name: resp.Name,
	}), nil
}

func (s *SliverMCPServer) deleteImplantProfileHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args deleteImplantProfileArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(deleteImplantProfileToolName, "", "")

	if _, err := s.Rpc.DeleteImplantProfile(ctx, &clientpb.DeleteReq{Name: args.Name}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("delete_implant_profile failed", err), nil
	}

	type delResult struct {
		Name    string `json:"name"`
		Deleted bool   `json:"deleted"`
	}
	return mcpapi.NewToolResultStructuredOnly(delResult{Name: args.Name, Deleted: true}), nil
}

func (s *SliverMCPServer) getCompilerHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(getCompilerToolName, "", "")

	resp, err := s.Rpc.GetCompiler(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("get_compiler failed", err), nil
	}

	result := compilerResult{
		GOOS:    resp.GOOS,
		GOARCH:  resp.GOARCH,
		Targets: make([]compilerTarget, 0, len(resp.Targets)),
	}
	for _, t := range resp.Targets {
		if t == nil {
			continue
		}
		result.Targets = append(result.Targets, compilerTarget{
			GOOS:   t.GOOS,
			GOARCH: t.GOARCH,
			Format: t.Format.String(),
		})
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) listCanariesHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(listCanariesToolName, "", "")

	resp, err := s.Rpc.Canaries(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_canaries failed", err), nil
	}

	result := canariesResult{Canaries: make([]canaryItem, 0, len(resp.Canaries))}
	for _, c := range resp.Canaries {
		if c == nil {
			continue
		}
		result.Canaries = append(result.Canaries, canaryItem{
			ID:             c.ID,
			ImplantName:    c.ImplantName,
			Domain:         c.Domain,
			Triggered:      c.Triggered,
			FirstTriggered: c.FirstTriggered,
			LatestTrigger:  c.LatestTrigger,
			Count:          c.Count,
		})
	}
	result.Count = len(result.Canaries)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) shellcodeRDIHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args shellcodeRDIArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(shellcodeRDIToolName, "", "")

	data, err := base64.StdEncoding.DecodeString(args.Data)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 data: %v", err)), nil
	}

	resp, err := s.Rpc.ShellcodeRDI(ctx, &clientpb.ShellcodeRDIReq{
		Data:         data,
		FunctionName: args.FunctionName,
		Arguments:    args.Arguments,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("shellcode_rdi failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(shellcodeResult{
		Size: len(resp.Data),
		Data: base64.StdEncoding.EncodeToString(resp.Data),
	}), nil
}

func (s *SliverMCPServer) shellcodeEncodeHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args shellcodeEncodeArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(shellcodeEncodeToolName, "", "")

	data, err := base64.StdEncoding.DecodeString(args.Data)
	if err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 data: %v", err)), nil
	}

	req := &clientpb.ShellcodeEncodeReq{
		Encoder:      clientpb.ShellcodeEncoder(args.Encoder),
		Architecture: args.Architecture,
		Iterations:   args.Iterations,
		Data:         data,
	}
	if args.BadChars != "" {
		bc, err := base64.StdEncoding.DecodeString(args.BadChars)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 bad_chars: %v", err)), nil
		}
		req.BadChars = bc
	}

	resp, err := s.Rpc.ShellcodeEncoder(ctx, req)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("shellcode_encode failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(shellcodeResult{
		Size: len(resp.Data),
		Data: base64.StdEncoding.EncodeToString(resp.Data),
	}), nil
}

func (s *SliverMCPServer) listShellcodeEncodersHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(listShellcodeEncodersToolName, "", "")

	resp, err := s.Rpc.ShellcodeEncoderMap(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("list_shellcode_encoders failed", err), nil
	}

	type encoderEntry struct {
		Name string `json:"name"`
	}
	type encoderMapResult struct {
		Encoders []encoderEntry `json:"encoders"`
		Count    int            `json:"count"`
	}

	result := encoderMapResult{Encoders: make([]encoderEntry, 0)}
	for name := range resp.Encoders {
		result.Encoders = append(result.Encoders, encoderEntry{Name: name})
	}
	result.Count = len(result.Encoders)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
