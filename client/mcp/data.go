package mcp

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	lootAddToolName     = "loot_add"
	lootRmToolName      = "loot_rm"
	lootUpdateToolName  = "loot_update"
	lootContentToolName = "loot_content"
	lootAllToolName     = "loot_all"

	credsListToolName          = "creds_list"
	credsAddToolName           = "creds_add"
	credsRmToolName            = "creds_rm"
	credsUpdateToolName        = "creds_update"
	credByIDToolName           = "cred_by_id"
	credsByHashTypeToolName    = "creds_by_hash_type"
	credsSniffHashTypeToolName = "creds_sniff_hash_type"

	hostsListToolName = "hosts_list"
	hostInfoToolName  = "host_info"
	hostRmToolName    = "host_rm"
	hostIOCRmToolName = "host_ioc_rm"
)

// --- Loot Args/Results ---

type lootAddArgs struct {
	Name           string `json:"name" jsonschema:"description=Name for the loot entry,required"`
	FileType       int32  `json:"file_type" jsonschema:"description=File type (0=no_file 1=binary 2=text)"`
	OriginHostUUID string `json:"origin_host_uuid" jsonschema:"description=UUID of the host the loot came from"`
	FileName       string `json:"file_name" jsonschema:"description=File name for the loot data"`
	DataBase64     string `json:"data_base64" jsonschema:"description=File data as base64"`
}

type lootRmArgs struct {
	LootID string `json:"loot_id" jsonschema:"description=ID of loot entry to remove,required"`
}

type lootUpdateArgs struct {
	LootID   string `json:"loot_id" jsonschema:"description=ID of loot entry to update,required"`
	Name     string `json:"name" jsonschema:"description=Updated name"`
	FileType int32  `json:"file_type" jsonschema:"description=Updated file type (0=no_file 1=binary 2=text)"`
}

type lootContentArgs struct {
	LootID string `json:"loot_id" jsonschema:"description=ID of loot entry to get content for,required"`
}

type lootSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	FileType       int32  `json:"file_type"`
	OriginHostUUID string `json:"origin_host_uuid"`
	Size           int64  `json:"size"`
}

type lootContentResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	FileName   string `json:"file_name"`
	Size       int64  `json:"size"`
	DataBase64 string `json:"data_base64"`
}

type lootListResult struct {
	Loot  []lootSummary `json:"loot"`
	Count int           `json:"count"`
}

// --- Creds Args/Results ---

type credsAddArgs struct {
	Username       string `json:"username" jsonschema:"description=Username"`
	Plaintext      string `json:"plaintext" jsonschema:"description=Plaintext password"`
	Hash           string `json:"hash" jsonschema:"description=Hash value"`
	HashType       int32  `json:"hash_type" jsonschema:"description=Hash type (0=MD5 100=SHA1 1400=SHA256 1000=NTLM etc.)"`
	OriginHostUUID string `json:"origin_host_uuid" jsonschema:"description=UUID of originating host"`
	Collection     string `json:"collection" jsonschema:"description=Collection name"`
}

type credsRmArgs struct {
	CredID string `json:"cred_id" jsonschema:"description=Credential ID to remove,required"`
}

type credsUpdateArgs struct {
	CredID    string `json:"cred_id" jsonschema:"description=Credential ID to update,required"`
	Username  string `json:"username" jsonschema:"description=Updated username"`
	Plaintext string `json:"plaintext" jsonschema:"description=Updated plaintext password"`
	Hash      string `json:"hash" jsonschema:"description=Updated hash"`
	HashType  int32  `json:"hash_type" jsonschema:"description=Updated hash type"`
}

type credByIDArgs struct {
	CredID string `json:"cred_id" jsonschema:"description=Credential ID,required"`
}

type credsByHashTypeArgs struct {
	HashType int32 `json:"hash_type" jsonschema:"description=Hash type to filter by,required"`
}

type credsSniffHashTypeArgs struct {
	Hash string `json:"hash" jsonschema:"description=Hash string to identify the type of,required"`
}

type credSummary struct {
	ID             string `json:"id"`
	Username       string `json:"username"`
	Plaintext      string `json:"plaintext,omitempty"`
	Hash           string `json:"hash,omitempty"`
	HashType       int32  `json:"hash_type"`
	IsCracked      bool   `json:"is_cracked"`
	OriginHostUUID string `json:"origin_host_uuid"`
	Collection     string `json:"collection"`
}

type credsListResult struct {
	Credentials []credSummary `json:"credentials"`
	Count       int           `json:"count"`
}

// --- Host Args/Results ---

type hostInfoArgs struct {
	HostID string `json:"host_id" jsonschema:"description=Host ID to get info for,required"`
}

type hostRmArgs struct {
	HostID string `json:"host_id" jsonschema:"description=Host ID to remove,required"`
}

type hostIOCRmArgs struct {
	IOCID string `json:"ioc_id" jsonschema:"description=IOC ID to remove,required"`
}

type iocSummary struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	FileHash string `json:"file_hash"`
}

type hostSummary struct {
	ID           string       `json:"id"`
	Hostname     string       `json:"hostname"`
	HostUUID     string       `json:"host_uuid"`
	OSVersion    string       `json:"os_version"`
	Locale       string       `json:"locale"`
	FirstContact int64        `json:"first_contact"`
	IOCs         []iocSummary `json:"iocs,omitempty"`
}

type hostsListResult struct {
	Hosts []hostSummary `json:"hosts"`
	Count int           `json:"count"`
}

// --- Loot Handlers ---

func (s *SliverMCPServer) lootAddHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args lootAddArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(lootAddToolName, "", "")

	loot := &clientpb.Loot{
		Name:           args.Name,
		FileType:       clientpb.FileType(args.FileType),
		OriginHostUUID: args.OriginHostUUID,
	}

	if args.DataBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(args.DataBase64)
		if err != nil {
			return mcpapi.NewToolResultError(fmt.Sprintf("invalid base64 data: %v", err)), nil
		}
		loot.File = &commonpb.File{
			Name: args.FileName,
			Data: data,
		}
	}

	resp, err := s.Rpc.LootAdd(ctx, loot)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("loot_add failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(lootSummary{
		ID:             resp.ID,
		Name:           resp.Name,
		FileType:       int32(resp.FileType),
		OriginHostUUID: resp.OriginHostUUID,
		Size:           resp.Size,
	}), nil
}

func (s *SliverMCPServer) lootRmHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args lootRmArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(lootRmToolName, "", "")

	if _, err := s.Rpc.LootRm(ctx, &clientpb.Loot{ID: args.LootID}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("loot_rm failed", err), nil
	}

	type result struct {
		LootID  string `json:"loot_id"`
		Removed bool   `json:"removed"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{LootID: args.LootID, Removed: true}), nil
}

func (s *SliverMCPServer) lootUpdateHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args lootUpdateArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(lootUpdateToolName, "", "")

	resp, err := s.Rpc.LootUpdate(ctx, &clientpb.Loot{
		ID:       args.LootID,
		Name:     args.Name,
		FileType: clientpb.FileType(args.FileType),
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("loot_update failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(lootSummary{
		ID:             resp.ID,
		Name:           resp.Name,
		FileType:       int32(resp.FileType),
		OriginHostUUID: resp.OriginHostUUID,
		Size:           resp.Size,
	}), nil
}

func (s *SliverMCPServer) lootContentHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args lootContentArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(lootContentToolName, "", "")

	resp, err := s.Rpc.LootContent(ctx, &clientpb.Loot{ID: args.LootID})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("loot_content failed", err), nil
	}

	result := lootContentResult{
		ID:   resp.ID,
		Name: resp.Name,
		Size: resp.Size,
	}
	if resp.File != nil {
		result.FileName = resp.File.Name
		result.DataBase64 = base64.StdEncoding.EncodeToString(resp.File.Data)
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) lootAllHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(lootAllToolName, "", "")

	resp, err := s.Rpc.LootAll(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("loot_all failed", err), nil
	}

	result := lootListResult{
		Loot: make([]lootSummary, 0, len(resp.Loot)),
	}
	for _, l := range resp.Loot {
		if l == nil {
			continue
		}
		result.Loot = append(result.Loot, lootSummary{
			ID:             l.ID,
			Name:           l.Name,
			FileType:       int32(l.FileType),
			OriginHostUUID: l.OriginHostUUID,
			Size:           l.Size,
		})
	}
	result.Count = len(result.Loot)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// --- Creds Handlers ---

func (s *SliverMCPServer) credsListHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(credsListToolName, "", "")

	resp, err := s.Rpc.Creds(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("creds_list failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(buildCredsResult(resp)), nil
}

func (s *SliverMCPServer) credsAddHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args credsAddArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(credsAddToolName, "", "")

	if _, err := s.Rpc.CredsAdd(ctx, &clientpb.Credentials{
		Credentials: []*clientpb.Credential{
			{
				Username:       args.Username,
				Plaintext:      args.Plaintext,
				Hash:           args.Hash,
				HashType:       clientpb.HashType(args.HashType),
				OriginHostUUID: args.OriginHostUUID,
				Collection:     args.Collection,
			},
		},
	}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("creds_add failed", err), nil
	}

	type result struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{Success: true}), nil
}

func (s *SliverMCPServer) credsRmHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args credsRmArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(credsRmToolName, "", "")

	if _, err := s.Rpc.CredsRm(ctx, &clientpb.Credentials{
		Credentials: []*clientpb.Credential{{ID: args.CredID}},
	}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("creds_rm failed", err), nil
	}

	type result struct {
		CredID  string `json:"cred_id"`
		Removed bool   `json:"removed"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{CredID: args.CredID, Removed: true}), nil
}

func (s *SliverMCPServer) credsUpdateHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args credsUpdateArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(credsUpdateToolName, "", "")

	if _, err := s.Rpc.CredsUpdate(ctx, &clientpb.Credentials{
		Credentials: []*clientpb.Credential{
			{
				ID:        args.CredID,
				Username:  args.Username,
				Plaintext: args.Plaintext,
				Hash:      args.Hash,
				HashType:  clientpb.HashType(args.HashType),
			},
		},
	}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("creds_update failed", err), nil
	}

	type result struct {
		Success bool `json:"success"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{Success: true}), nil
}

func (s *SliverMCPServer) credByIDHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args credByIDArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(credByIDToolName, "", "")

	resp, err := s.Rpc.GetCredByID(ctx, &clientpb.Credential{ID: args.CredID})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("cred_by_id failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(credSummary{
		ID:             resp.ID,
		Username:       resp.Username,
		Plaintext:      resp.Plaintext,
		Hash:           resp.Hash,
		HashType:       int32(resp.HashType),
		IsCracked:      resp.IsCracked,
		OriginHostUUID: resp.OriginHostUUID,
		Collection:     resp.Collection,
	}), nil
}

func (s *SliverMCPServer) credsByHashTypeHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args credsByHashTypeArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(credsByHashTypeToolName, "", "")

	resp, err := s.Rpc.GetCredsByHashType(ctx, &clientpb.Credential{HashType: clientpb.HashType(args.HashType)})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("creds_by_hash_type failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(buildCredsResult(resp)), nil
}

func (s *SliverMCPServer) credsSniffHashTypeHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args credsSniffHashTypeArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(credsSniffHashTypeToolName, "", "")

	resp, err := s.Rpc.CredsSniffHashType(ctx, &clientpb.Credential{Hash: args.Hash})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("creds_sniff_hash_type failed", err), nil
	}

	type result struct {
		HashType int32  `json:"hash_type"`
		Hash     string `json:"hash"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{
		HashType: int32(resp.HashType),
		Hash:     resp.Hash,
	}), nil
}

// --- Host Handlers ---

func (s *SliverMCPServer) hostsListHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(hostsListToolName, "", "")

	resp, err := s.Rpc.Hosts(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("hosts_list failed", err), nil
	}

	result := hostsListResult{
		Hosts: make([]hostSummary, 0, len(resp.Hosts)),
	}
	for _, h := range resp.Hosts {
		if h == nil {
			continue
		}
		result.Hosts = append(result.Hosts, buildHostSummary(h))
	}
	result.Count = len(result.Hosts)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) hostInfoHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args hostInfoArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(hostInfoToolName, "", "")

	resp, err := s.Rpc.Host(ctx, &clientpb.Host{ID: args.HostID})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("host_info failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(buildHostSummary(resp)), nil
}

func (s *SliverMCPServer) hostRmHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args hostRmArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(hostRmToolName, "", "")

	if _, err := s.Rpc.HostRm(ctx, &clientpb.Host{ID: args.HostID}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("host_rm failed", err), nil
	}

	type result struct {
		HostID  string `json:"host_id"`
		Removed bool   `json:"removed"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{HostID: args.HostID, Removed: true}), nil
}

func (s *SliverMCPServer) hostIOCRmHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args hostIOCRmArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(hostIOCRmToolName, "", "")

	if _, err := s.Rpc.HostIOCRm(ctx, &clientpb.IOC{ID: args.IOCID}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("host_ioc_rm failed", err), nil
	}

	type result struct {
		IOCID   string `json:"ioc_id"`
		Removed bool   `json:"removed"`
	}
	return mcpapi.NewToolResultStructuredOnly(result{IOCID: args.IOCID, Removed: true}), nil
}

// --- Helpers ---

func buildCredsResult(resp *clientpb.Credentials) credsListResult {
	result := credsListResult{
		Credentials: make([]credSummary, 0, len(resp.Credentials)),
	}
	for _, c := range resp.Credentials {
		if c == nil {
			continue
		}
		result.Credentials = append(result.Credentials, credSummary{
			ID:             c.ID,
			Username:       c.Username,
			Plaintext:      c.Plaintext,
			Hash:           c.Hash,
			HashType:       int32(c.HashType),
			IsCracked:      c.IsCracked,
			OriginHostUUID: c.OriginHostUUID,
			Collection:     c.Collection,
		})
	}
	result.Count = len(result.Credentials)
	return result
}

func buildHostSummary(h *clientpb.Host) hostSummary {
	hs := hostSummary{
		ID:           h.ID,
		Hostname:     h.Hostname,
		HostUUID:     h.HostUUID,
		OSVersion:    h.OSVersion,
		Locale:       h.Locale,
		FirstContact: h.FirstContact,
	}
	for _, ioc := range h.IOCs {
		if ioc == nil {
			continue
		}
		hs.IOCs = append(hs.IOCs, iocSummary{
			ID:       ioc.ID,
			Path:     ioc.Path,
			FileHash: ioc.FileHash,
		})
	}
	return hs
}
