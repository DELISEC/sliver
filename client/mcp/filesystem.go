package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/encoders"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	lsToolName           = "fs_ls"
	cdToolName           = "fs_cd"
	catToolName          = "fs_cat"
	pwdToolName          = "fs_pwd"
	rmToolName           = "fs_rm"
	mvToolName           = "fs_mv"
	cpToolName           = "fs_cp"
	mkdirToolName        = "fs_mkdir"
	chmodToolName        = "fs_chmod"
	chownToolName        = "fs_chown"
	downloadToolName     = "download_file"
	uploadToolName       = "upload_file"
	grepToolName         = "fs_grep"
	chtimesToolName      = "fs_chtimes"
	mountToolName        = "fs_mount"
	memfilesListToolName = "fs_memfiles_list"
	memfilesAddToolName  = "fs_memfiles_add"
	memfilesRmToolName   = "fs_memfiles_rm"
)

type lsArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Path           string `json:"path,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type cdArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Path           string `json:"path,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type catArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Path           string `json:"path,omitempty"`
	MaxBytes       int64  `json:"max_bytes,omitempty"`
	MaxLines       int64  `json:"max_lines,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type pwdArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type rmArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Path           string `json:"path,omitempty"`
	Recursive      bool   `json:"recursive,omitempty"`
	Force          bool   `json:"force,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type mvArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Src            string `json:"src,omitempty"`
	Dst            string `json:"dst,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type cpArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Src            string `json:"src,omitempty"`
	Dst            string `json:"dst,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type mkdirArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Path           string `json:"path,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type chmodArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Path           string `json:"path,omitempty"`
	FileMode       string `json:"file_mode,omitempty"`
	Recursive      bool   `json:"recursive,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type chownArgs struct {
	SessionID      string `json:"session_id,omitempty"`
	BeaconID       string `json:"beacon_id,omitempty"`
	Path           string `json:"path,omitempty"`
	UID            string `json:"uid,omitempty"`
	GID            string `json:"gid,omitempty"`
	Recursive      bool   `json:"recursive,omitempty"`
	Wait           bool   `json:"wait,omitempty"`
	TimeoutSeconds int64  `json:"timeout_seconds,omitempty"`
}

type lsFileEntry struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	Mode    string `json:"mode"`
	Link    string `json:"link,omitempty"`
	UID     string `json:"uid,omitempty"`
	GID     string `json:"gid,omitempty"`
}

type lsResult struct {
	Path           string        `json:"path"`
	Exists         bool          `json:"exists"`
	Timezone       string        `json:"timezone"`
	TimezoneOffset int32         `json:"timezone_offset"`
	Files          []lsFileEntry `json:"files"`
}

type cdResult struct {
	Path string `json:"path"`
}

type catResult struct {
	Path          string `json:"path"`
	RequestedPath string `json:"requested_path"`
	Exists        bool   `json:"exists"`
	IsDir         bool   `json:"is_dir"`
	Encoder       string `json:"encoder,omitempty"`
	Decoded       bool   `json:"decoded"`
	ByteLen       int    `json:"byte_len"`
	DataBase64    string `json:"data_base64"`
	Text          string `json:"text,omitempty"`
}

type pwdResult struct {
	Path string `json:"path"`
}

type rmResult struct {
	Path string `json:"path"`
}

type mvResult struct {
	Src string `json:"src"`
	Dst string `json:"dst"`
}

type cpResult struct {
	Src          string `json:"src"`
	Dst          string `json:"dst"`
	BytesWritten int64  `json:"bytes_written"`
}

type mkdirResult struct {
	Path string `json:"path"`
}

type chmodResult struct {
	Path string `json:"path"`
}

type chownResult struct {
	Path string `json:"path"`
}

func (s *SliverMCPServer) lsHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args lsArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" {
		args.Path = "."
	}

	return s.handleLs(ctx, args)
}

func (s *SliverMCPServer) cdHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args cdArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" {
		args.Path = "."
	}

	return s.handleCd(ctx, args)
}

func (s *SliverMCPServer) catHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args catArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" {
		return mcpapi.NewToolResultError("path is required"), nil
	}

	return s.handleCat(ctx, args)
}

func (s *SliverMCPServer) pwdHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args pwdArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	return s.handlePwd(ctx, args)
}

func (s *SliverMCPServer) rmHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args rmArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" {
		return mcpapi.NewToolResultError("path is required"), nil
	}

	return s.handleRm(ctx, args)
}

func (s *SliverMCPServer) mvHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args mvArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Src == "" || args.Dst == "" {
		return mcpapi.NewToolResultError("src and dst are required"), nil
	}

	return s.handleMv(ctx, args)
}

func (s *SliverMCPServer) cpHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args cpArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Src == "" || args.Dst == "" {
		return mcpapi.NewToolResultError("src and dst are required"), nil
	}

	return s.handleCp(ctx, args)
}

func (s *SliverMCPServer) mkdirHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args mkdirArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" {
		return mcpapi.NewToolResultError("path is required"), nil
	}

	return s.handleMkdir(ctx, args)
}

func (s *SliverMCPServer) chmodHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args chmodArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" || args.FileMode == "" {
		return mcpapi.NewToolResultError("path and file_mode are required"), nil
	}

	return s.handleChmod(ctx, args)
}

func (s *SliverMCPServer) chownHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	var args chownArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.Path == "" || args.UID == "" || args.GID == "" {
		return mcpapi.NewToolResultError("path, uid, and gid are required"), nil
	}

	return s.handleChown(ctx, args)
}

func (s *SliverMCPServer) handleLs(ctx context.Context, args lsArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(lsToolName, args.SessionID, args.BeaconID, fmt.Sprintf("path=%q", args.Path))

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	lsResp, err := s.Rpc.Ls(ctx, &sliverpb.LsReq{
		Request: req,
		Path:    args.Path,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list directory", err), nil
	}

	if lsResp.Response != nil && lsResp.Response.Err != "" {
		return mcpapi.NewToolResultError(lsResp.Response.Err), nil
	}

	if isBeacon && lsResp.Response != nil && lsResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("ls", lsResp.Response.TaskID, lsResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Ls{}
		if err := s.waitForBeaconTaskResponse(ctx, lsResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await ls task", err), nil
		}
		lsResp = resolved
		if lsResp.Response != nil && lsResp.Response.Err != "" {
			return mcpapi.NewToolResultError(lsResp.Response.Err), nil
		}
	}

	result := lsResult{
		Path:           lsResp.Path,
		Exists:         lsResp.Exists,
		Timezone:       lsResp.Timezone,
		TimezoneOffset: lsResp.TimezoneOffset,
		Files:          make([]lsFileEntry, 0, len(lsResp.Files)),
	}

	for _, fileInfo := range lsResp.Files {
		if fileInfo == nil {
			continue
		}
		result.Files = append(result.Files, lsFileEntry{
			Name:    fileInfo.Name,
			IsDir:   fileInfo.IsDir,
			Size:    fileInfo.Size,
			ModTime: fileInfo.ModTime,
			Mode:    fileInfo.Mode,
			Link:    fileInfo.Link,
			UID:     fileInfo.Uid,
			GID:     fileInfo.Gid,
		})
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) handleCd(ctx context.Context, args cdArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(cdToolName, args.SessionID, args.BeaconID, fmt.Sprintf("path=%q", args.Path))

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	pwdResp, err := s.Rpc.Cd(ctx, &sliverpb.CdReq{
		Request: req,
		Path:    args.Path,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to change directory", err), nil
	}

	if pwdResp.Response != nil && pwdResp.Response.Err != "" {
		return mcpapi.NewToolResultError(pwdResp.Response.Err), nil
	}

	if isBeacon && pwdResp.Response != nil && pwdResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("cd", pwdResp.Response.TaskID, pwdResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Pwd{}
		if err := s.waitForBeaconTaskResponse(ctx, pwdResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await cd task", err), nil
		}
		pwdResp = resolved
		if pwdResp.Response != nil && pwdResp.Response.Err != "" {
			return mcpapi.NewToolResultError(pwdResp.Response.Err), nil
		}
	}

	return mcpapi.NewToolResultStructuredOnly(cdResult{Path: pwdResp.Path}), nil
}

func (s *SliverMCPServer) handleCat(ctx context.Context, args catArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	extras := []string{fmt.Sprintf("path=%q", args.Path)}
	if args.MaxBytes > 0 {
		extras = append(extras, fmt.Sprintf("max_bytes=%d", args.MaxBytes))
	}
	if args.MaxLines > 0 {
		extras = append(extras, fmt.Sprintf("max_lines=%d", args.MaxLines))
	}
	s.logToolCall(catToolName, args.SessionID, args.BeaconID, extras...)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	download, err := s.Rpc.Download(ctx, &sliverpb.DownloadReq{
		Request:          req,
		RestrictedToFile: true,
		Path:             args.Path,
		MaxBytes:         args.MaxBytes,
		MaxLines:         args.MaxLines,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to download file", err), nil
	}

	if download.Response != nil && download.Response.Err != "" {
		return mcpapi.NewToolResultError(download.Response.Err), nil
	}

	if isBeacon && download.Response != nil && download.Response.Async {
		if !args.Wait {
			return newAsyncResult("cat", download.Response.TaskID, download.Response.BeaconID), nil
		}
		resolved := &sliverpb.Download{}
		if err := s.waitForBeaconTaskResponse(ctx, download.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await cat task", err), nil
		}
		download = resolved
		if download.Response != nil && download.Response.Err != "" {
			return mcpapi.NewToolResultError(download.Response.Err), nil
		}
	}

	result, err := buildCatResult(download, args.Path)
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to decode file contents", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) handlePwd(ctx context.Context, args pwdArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(pwdToolName, args.SessionID, args.BeaconID)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	pwdResp, err := s.Rpc.Pwd(ctx, &sliverpb.PwdReq{Request: req})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get working directory", err), nil
	}

	if pwdResp.Response != nil && pwdResp.Response.Err != "" {
		return mcpapi.NewToolResultError(pwdResp.Response.Err), nil
	}

	if isBeacon && pwdResp.Response != nil && pwdResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("pwd", pwdResp.Response.TaskID, pwdResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Pwd{}
		if err := s.waitForBeaconTaskResponse(ctx, pwdResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await pwd task", err), nil
		}
		pwdResp = resolved
		if pwdResp.Response != nil && pwdResp.Response.Err != "" {
			return mcpapi.NewToolResultError(pwdResp.Response.Err), nil
		}
	}

	return mcpapi.NewToolResultStructuredOnly(pwdResult{Path: pwdResp.Path}), nil
}

func (s *SliverMCPServer) handleRm(ctx context.Context, args rmArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(
		rmToolName,
		args.SessionID,
		args.BeaconID,
		fmt.Sprintf("path=%q", args.Path),
		fmt.Sprintf("recursive=%t", args.Recursive),
		fmt.Sprintf("force=%t", args.Force),
	)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	rmResp, err := s.Rpc.Rm(ctx, &sliverpb.RmReq{
		Request:   req,
		Path:      args.Path,
		Recursive: args.Recursive,
		Force:     args.Force,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to remove path", err), nil
	}

	if rmResp.Response != nil && rmResp.Response.Err != "" {
		return mcpapi.NewToolResultError(rmResp.Response.Err), nil
	}

	if isBeacon && rmResp.Response != nil && rmResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("rm", rmResp.Response.TaskID, rmResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Rm{}
		if err := s.waitForBeaconTaskResponse(ctx, rmResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await rm task", err), nil
		}
		rmResp = resolved
		if rmResp.Response != nil && rmResp.Response.Err != "" {
			return mcpapi.NewToolResultError(rmResp.Response.Err), nil
		}
	}

	path := rmResp.Path
	if path == "" {
		path = args.Path
	}
	return mcpapi.NewToolResultStructuredOnly(rmResult{Path: path}), nil
}

func (s *SliverMCPServer) handleMv(ctx context.Context, args mvArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(
		mvToolName,
		args.SessionID,
		args.BeaconID,
		fmt.Sprintf("src=%q", args.Src),
		fmt.Sprintf("dst=%q", args.Dst),
	)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	mvResp, err := s.Rpc.Mv(ctx, &sliverpb.MvReq{
		Request: req,
		Src:     args.Src,
		Dst:     args.Dst,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to move path", err), nil
	}

	if mvResp.Response != nil && mvResp.Response.Err != "" {
		return mcpapi.NewToolResultError(mvResp.Response.Err), nil
	}

	if isBeacon && mvResp.Response != nil && mvResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("mv", mvResp.Response.TaskID, mvResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Mv{}
		if err := s.waitForBeaconTaskResponse(ctx, mvResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await mv task", err), nil
		}
		mvResp = resolved
		if mvResp.Response != nil && mvResp.Response.Err != "" {
			return mcpapi.NewToolResultError(mvResp.Response.Err), nil
		}
	}

	src := mvResp.Src
	if src == "" {
		src = args.Src
	}
	dst := mvResp.Dst
	if dst == "" {
		dst = args.Dst
	}
	return mcpapi.NewToolResultStructuredOnly(mvResult{Src: src, Dst: dst}), nil
}

func (s *SliverMCPServer) handleCp(ctx context.Context, args cpArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(
		cpToolName,
		args.SessionID,
		args.BeaconID,
		fmt.Sprintf("src=%q", args.Src),
		fmt.Sprintf("dst=%q", args.Dst),
	)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	cpResp, err := s.Rpc.Cp(ctx, &sliverpb.CpReq{
		Request: req,
		Src:     args.Src,
		Dst:     args.Dst,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to copy path", err), nil
	}

	if cpResp.Response != nil && cpResp.Response.Err != "" {
		return mcpapi.NewToolResultError(cpResp.Response.Err), nil
	}

	if isBeacon && cpResp.Response != nil && cpResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("cp", cpResp.Response.TaskID, cpResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Cp{}
		if err := s.waitForBeaconTaskResponse(ctx, cpResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await cp task", err), nil
		}
		cpResp = resolved
		if cpResp.Response != nil && cpResp.Response.Err != "" {
			return mcpapi.NewToolResultError(cpResp.Response.Err), nil
		}
	}

	src := cpResp.Src
	if src == "" {
		src = args.Src
	}
	dst := cpResp.Dst
	if dst == "" {
		dst = args.Dst
	}
	return mcpapi.NewToolResultStructuredOnly(cpResult{
		Src:          src,
		Dst:          dst,
		BytesWritten: cpResp.BytesWritten,
	}), nil
}

func (s *SliverMCPServer) handleMkdir(ctx context.Context, args mkdirArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(mkdirToolName, args.SessionID, args.BeaconID, fmt.Sprintf("path=%q", args.Path))

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	mkdirResp, err := s.Rpc.Mkdir(ctx, &sliverpb.MkdirReq{
		Request: req,
		Path:    args.Path,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to create directory", err), nil
	}

	if mkdirResp.Response != nil && mkdirResp.Response.Err != "" {
		return mcpapi.NewToolResultError(mkdirResp.Response.Err), nil
	}

	if isBeacon && mkdirResp.Response != nil && mkdirResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("mkdir", mkdirResp.Response.TaskID, mkdirResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Mkdir{}
		if err := s.waitForBeaconTaskResponse(ctx, mkdirResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await mkdir task", err), nil
		}
		mkdirResp = resolved
		if mkdirResp.Response != nil && mkdirResp.Response.Err != "" {
			return mcpapi.NewToolResultError(mkdirResp.Response.Err), nil
		}
	}

	path := mkdirResp.Path
	if path == "" {
		path = args.Path
	}
	return mcpapi.NewToolResultStructuredOnly(mkdirResult{Path: path}), nil
}

func (s *SliverMCPServer) handleChmod(ctx context.Context, args chmodArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(
		chmodToolName,
		args.SessionID,
		args.BeaconID,
		fmt.Sprintf("path=%q", args.Path),
		fmt.Sprintf("file_mode=%q", args.FileMode),
		fmt.Sprintf("recursive=%t", args.Recursive),
	)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	chmodResp, err := s.Rpc.Chmod(ctx, &sliverpb.ChmodReq{
		Request:   req,
		Path:      args.Path,
		FileMode:  args.FileMode,
		Recursive: args.Recursive,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to change mode", err), nil
	}

	if chmodResp.Response != nil && chmodResp.Response.Err != "" {
		return mcpapi.NewToolResultError(chmodResp.Response.Err), nil
	}

	if isBeacon && chmodResp.Response != nil && chmodResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("chmod", chmodResp.Response.TaskID, chmodResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Chmod{}
		if err := s.waitForBeaconTaskResponse(ctx, chmodResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await chmod task", err), nil
		}
		chmodResp = resolved
		if chmodResp.Response != nil && chmodResp.Response.Err != "" {
			return mcpapi.NewToolResultError(chmodResp.Response.Err), nil
		}
	}

	path := chmodResp.Path
	if path == "" {
		path = args.Path
	}
	return mcpapi.NewToolResultStructuredOnly(chmodResult{Path: path}), nil
}

func (s *SliverMCPServer) handleChown(ctx context.Context, args chownArgs) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(
		chownToolName,
		args.SessionID,
		args.BeaconID,
		fmt.Sprintf("path=%q", args.Path),
		fmt.Sprintf("uid=%q", args.UID),
		fmt.Sprintf("gid=%q", args.GID),
		fmt.Sprintf("recursive=%t", args.Recursive),
	)

	args.TimeoutSeconds = applyDefaultTimeout(args.Wait, args.TimeoutSeconds)
	ctx, cancel := withTimeout(ctx, args.TimeoutSeconds)
	if cancel != nil {
		defer cancel()
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}

	chownResp, err := s.Rpc.Chown(ctx, &sliverpb.ChownReq{
		Request:   req,
		Path:      args.Path,
		Uid:       args.UID,
		Gid:       args.GID,
		Recursive: args.Recursive,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to change owner", err), nil
	}

	if chownResp.Response != nil && chownResp.Response.Err != "" {
		return mcpapi.NewToolResultError(chownResp.Response.Err), nil
	}

	if isBeacon && chownResp.Response != nil && chownResp.Response.Async {
		if !args.Wait {
			return newAsyncResult("chown", chownResp.Response.TaskID, chownResp.Response.BeaconID), nil
		}
		resolved := &sliverpb.Chown{}
		if err := s.waitForBeaconTaskResponse(ctx, chownResp.Response.TaskID, resolved); err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to await chown task", err), nil
		}
		chownResp = resolved
		if chownResp.Response != nil && chownResp.Response.Err != "" {
			return mcpapi.NewToolResultError(chownResp.Response.Err), nil
		}
	}

	path := chownResp.Path
	if path == "" {
		path = args.Path
	}
	return mcpapi.NewToolResultStructuredOnly(chownResult{Path: path}), nil
}

func buildCatResult(download *sliverpb.Download, requestedPath string) (*catResult, error) {
	data := download.Data
	decoded := false
	if download.Encoder == "gzip" {
		decoded = true
		var err error
		data, err = new(encoders.Gzip).Decode(download.Data)
		if err != nil {
			return nil, err
		}
	}

	result := &catResult{
		Path:          download.Path,
		RequestedPath: requestedPath,
		Exists:        download.Exists,
		IsDir:         download.IsDir,
		Encoder:       download.Encoder,
		Decoded:       decoded,
		ByteLen:       len(data),
		DataBase64:    base64.StdEncoding.EncodeToString(data),
	}

	if len(data) == 0 {
		return result, nil
	}

	if utf8.Valid(data) {
		result.Text = string(data)
	}

	return result, nil
}

// --- Download/Upload args ---

type downloadFileArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	RemotePath     string `json:"remote_path" jsonschema:"description=Remote file path to download,required"`
	Recurse        bool   `json:"recurse" jsonschema:"description=Recursively download directory"`
	MaxBytes       int64  `json:"max_bytes" jsonschema:"description=Maximum bytes to download (0 for unlimited)"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type uploadFileArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	RemotePath     string `json:"remote_path" jsonschema:"description=Remote destination path,required"`
	DataBase64     string `json:"data_base64" jsonschema:"description=File content as base64 (use this OR local_path)"`
	LocalPath      string `json:"local_path" jsonschema:"description=Path to local file to upload (use this OR data_base64)"`
	IsIOC          bool   `json:"is_ioc" jsonschema:"description=Mark as IOC"`
	Overwrite      bool   `json:"overwrite" jsonschema:"description=Overwrite existing file"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

// --- Download/Upload results ---

type downloadFileResult struct {
	Path       string `json:"path"`
	Exists     bool   `json:"exists"`
	IsDir      bool   `json:"is_dir"`
	Size       int    `json:"size"`
	DataBase64 string `json:"data_base64"`
	Text       string `json:"text,omitempty"`
}

type uploadFileResult struct {
	Path         string `json:"path"`
	WrittenFiles int32  `json:"written_files"`
}

// --- Download/Upload handlers ---

func (s *SliverMCPServer) downloadFileHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args downloadFileArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(downloadToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Download(ctx, &sliverpb.DownloadReq{
		Path:     args.RemotePath,
		Recurse:  args.Recurse,
		MaxBytes: args.MaxBytes,
		Request:  req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("download failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("download_file", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.Download{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	if !resp.Exists {
		return mcpapi.NewToolResultError(fmt.Sprintf("remote path does not exist: %s", args.RemotePath)), nil
	}

	data := resp.Data
	if resp.Encoder == "gzip" {
		data, err = new(encoders.Gzip).Decode(resp.Data)
		if err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to decompress", err), nil
		}
	}

	result := downloadFileResult{
		Path:       resp.Path,
		Exists:     resp.Exists,
		IsDir:      resp.IsDir,
		Size:       len(data),
		DataBase64: base64.StdEncoding.EncodeToString(data),
	}

	if len(data) > 0 && utf8.Valid(data) {
		result.Text = string(data)
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) uploadFileHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args uploadFileArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	var data []byte
	if args.LocalPath != "" {
		var err error
		data, err = os.ReadFile(args.LocalPath)
		if err != nil {
			return mcpapi.NewToolResultErrorFromErr("failed to read local file", err), nil
		}
	} else if args.DataBase64 != "" {
		var err error
		data, err = base64.StdEncoding.DecodeString(args.DataBase64)
		if err != nil {
			return mcpapi.NewToolResultError("invalid base64 data: " + err.Error()), nil
		}
	} else {
		return mcpapi.NewToolResultError("either local_path or data_base64 is required"), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(uploadToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Upload(ctx, &sliverpb.UploadReq{
		Path:      args.RemotePath,
		Data:      data,
		IsIOC:     args.IsIOC,
		Overwrite: args.Overwrite,
		Request:   req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("upload failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("upload_file", taskID, args.BeaconID), nil
		}

		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}

		waited := &sliverpb.Upload{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(uploadFileResult{
		Path:         resp.Path,
		WrittenFiles: resp.WrittenFiles,
	}), nil
}

// --- Tier 2: Grep ---

type grepArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	SearchPattern  string `json:"search_pattern" jsonschema:"description=Regex search pattern,required"`
	Path           string `json:"path" jsonschema:"description=Path to search in,required"`
	Recursive      bool   `json:"recursive" jsonschema:"description=Search recursively"`
	LinesBefore    int32  `json:"lines_before" jsonschema:"description=Context lines before match"`
	LinesAfter     int32  `json:"lines_after" jsonschema:"description=Context lines after match"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type grepMatchResult struct {
	LineNumber  int64    `json:"line_number"`
	Line        string   `json:"line"`
	LinesBefore []string `json:"lines_before,omitempty"`
	LinesAfter  []string `json:"lines_after,omitempty"`
}

type grepFileResult struct {
	File     string            `json:"file"`
	IsBinary bool              `json:"is_binary"`
	Matches  []grepMatchResult `json:"matches"`
}

type grepResult struct {
	SearchPath string           `json:"search_path"`
	Files      []grepFileResult `json:"files"`
	FileCount  int              `json:"file_count"`
}

func (s *SliverMCPServer) grepHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args grepArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(grepToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Grep(ctx, &sliverpb.GrepReq{
		SearchPattern: args.SearchPattern,
		Path:          args.Path,
		Recursive:     args.Recursive,
		LinesBefore:   args.LinesBefore,
		LinesAfter:    args.LinesAfter,
		Request:       req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("grep failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("fs_grep", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Grep{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := grepResult{
		SearchPath: resp.SearchPathAbsolute,
		Files:      make([]grepFileResult, 0),
	}
	for file, fileResults := range resp.Results {
		if fileResults == nil {
			continue
		}
		fr := grepFileResult{
			File:     file,
			IsBinary: fileResults.IsBinary,
			Matches:  make([]grepMatchResult, 0, len(fileResults.FileResults)),
		}
		for _, m := range fileResults.FileResults {
			if m == nil {
				continue
			}
			fr.Matches = append(fr.Matches, grepMatchResult{
				LineNumber:  m.LineNumber,
				Line:        m.Line,
				LinesBefore: m.LinesBefore,
				LinesAfter:  m.LinesAfter,
			})
		}
		result.Files = append(result.Files, fr)
	}
	result.FileCount = len(result.Files)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// --- Tier 2: Chtimes ---

type chtimesArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Path           string `json:"path" jsonschema:"description=File path,required"`
	ATime          int64  `json:"atime" jsonschema:"description=Access time (Unix timestamp),required"`
	MTime          int64  `json:"mtime" jsonschema:"description=Modification time (Unix timestamp),required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type chtimesResult struct {
	Path string `json:"path"`
}

func (s *SliverMCPServer) chtimesHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args chtimesArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(chtimesToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Chtimes(ctx, &sliverpb.ChtimesReq{
		Path:    args.Path,
		ATime:   args.ATime,
		MTime:   args.MTime,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("chtimes failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("fs_chtimes", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Chtimes{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(chtimesResult{Path: resp.Path}), nil
}

// --- Tier 2: Mount ---

type mountArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type mountInfoResult struct {
	VolumeName   string `json:"volume_name"`
	VolumeType   string `json:"volume_type"`
	MountPoint   string `json:"mount_point"`
	Label        string `json:"label"`
	FileSystem   string `json:"file_system"`
	UsedSpace    uint64 `json:"used_space"`
	FreeSpace    uint64 `json:"free_space"`
	TotalSpace   uint64 `json:"total_space"`
	MountOptions string `json:"mount_options,omitempty"`
}

type mountResult struct {
	Mounts []mountInfoResult `json:"mounts"`
	Count  int               `json:"count"`
}

func (s *SliverMCPServer) mountHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args mountArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(mountToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Mount(ctx, &sliverpb.MountReq{Request: req})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("mount failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("fs_mount", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Mount{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := mountResult{
		Mounts: make([]mountInfoResult, 0, len(resp.Info)),
	}
	for _, m := range resp.Info {
		if m == nil {
			continue
		}
		result.Mounts = append(result.Mounts, mountInfoResult{
			VolumeName:   m.VolumeName,
			VolumeType:   m.VolumeType,
			MountPoint:   m.MountPoint,
			Label:        m.Label,
			FileSystem:   m.FileSystem,
			UsedSpace:    m.UsedSpace,
			FreeSpace:    m.FreeSpace,
			TotalSpace:   m.TotalSpace,
			MountOptions: m.MountOptions,
		})
	}
	result.Count = len(result.Mounts)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

// --- Tier 2: Memfiles ---

type memfilesListArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type memfilesAddArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type memfilesRmArgs struct {
	SessionID      string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID       string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	Fd             int64  `json:"fd" jsonschema:"description=File descriptor to remove,required"`
	Wait           bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type memfileResult struct {
	Fd int64 `json:"fd"`
}

func (s *SliverMCPServer) memfilesListHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args memfilesListArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(memfilesListToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.MemfilesList(ctx, &sliverpb.MemfilesListReq{Request: req})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("memfiles list failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("fs_memfiles_list", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Ls{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	result := lsResult{
		Path:   resp.Path,
		Exists: resp.Exists,
		Files:  make([]lsFileEntry, 0, len(resp.Files)),
	}
	for _, f := range resp.Files {
		if f == nil {
			continue
		}
		result.Files = append(result.Files, lsFileEntry{
			Name:    f.Name,
			IsDir:   f.IsDir,
			Size:    f.Size,
			ModTime: f.ModTime,
			Mode:    f.Mode,
		})
	}

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) memfilesAddHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args memfilesAddArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(memfilesAddToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.MemfilesAdd(ctx, &sliverpb.MemfilesAddReq{Request: req})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("memfiles add failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("fs_memfiles_add", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.MemfilesAdd{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(memfileResult{Fd: resp.Fd}), nil
}

func (s *SliverMCPServer) memfilesRmHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args memfilesRmArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(memfilesRmToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.MemfilesRm(ctx, &sliverpb.MemfilesRmReq{
		Fd:      args.Fd,
		Request: req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("memfiles rm failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("fs_memfiles_rm", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.MemfilesRm{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
		resp = waited
	}

	return mcpapi.NewToolResultStructuredOnly(memfileResult{Fd: resp.Fd}), nil
}
