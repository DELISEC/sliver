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
	sessionInfoToolName        = "session_info"
	killSessionToolName        = "kill_session"
	beaconInfoToolName         = "beacon_info"
	killBeaconToolName         = "kill_beacon"
	listBeaconTasksToolName    = "list_beacon_tasks"
	cancelBeaconTaskToolName   = "cancel_beacon_task"
	reconfigureImplantToolName = "reconfigure_implant"
	renameImplantToolName      = "rename_implant"
)

// --- Args structs ---

type sessionInfoArgs struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID to get info for,required"`
}

type killSessionArgs struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID to kill,required"`
	Force     bool   `json:"force" jsonschema:"description=Force kill the session"`
}

type beaconInfoArgs struct {
	BeaconID string `json:"beacon_id" jsonschema:"description=Beacon ID to get info for,required"`
}

type killBeaconArgs struct {
	BeaconID string `json:"beacon_id" jsonschema:"description=Beacon ID to remove,required"`
}

type listBeaconTasksArgs struct {
	BeaconID string `json:"beacon_id" jsonschema:"description=Beacon ID to list tasks for,required"`
}

type cancelBeaconTaskArgs struct {
	TaskID string `json:"task_id" jsonschema:"description=Beacon task ID to cancel,required"`
}

// --- Result structs ---

type beaconTaskResult struct {
	ID          string `json:"id"`
	BeaconID    string `json:"beacon_id"`
	State       string `json:"state"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
	SentAt      int64  `json:"sent_at"`
	CompletedAt int64  `json:"completed_at"`
}

type listBeaconTasksResult struct {
	Tasks []beaconTaskResult `json:"tasks"`
	Count int                `json:"count"`
}

// --- Handlers ---

func (s *SliverMCPServer) sessionInfoHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args sessionInfoArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(sessionInfoToolName, args.SessionID, "")

	sessions, err := s.Rpc.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get sessions", err), nil
	}

	for _, session := range sessions.GetSessions() {
		if session == nil || session.ID != args.SessionID {
			continue
		}
		return mcpapi.NewToolResultStructuredOnly(sessionSummary{
			ID:            session.ID,
			Name:          session.Name,
			Transport:     session.Transport,
			RemoteAddress: session.RemoteAddress,
			Hostname:      session.Hostname,
			Username:      session.Username,
			PID:           session.PID,
			OS:            session.OS,
			Arch:          session.Arch,
			Locale:        session.Locale,
			LastCheckin:   session.LastCheckin,
			IsDead:        session.IsDead,
			Integrity:     session.Integrity,
		}), nil
	}

	return mcpapi.NewToolResultError("session not found: " + args.SessionID), nil
}

func (s *SliverMCPServer) killSessionHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args killSessionArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(killSessionToolName, args.SessionID, "")

	if _, err := s.Rpc.Kill(ctx, &sliverpb.KillReq{
		Force: args.Force,
		Request: &commonpb.Request{
			SessionID: args.SessionID,
		},
	}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to kill session", err), nil
	}

	type killResult struct {
		SessionID string `json:"session_id"`
		Killed    bool   `json:"killed"`
	}
	return mcpapi.NewToolResultStructuredOnly(killResult{
		SessionID: args.SessionID,
		Killed:    true,
	}), nil
}

func (s *SliverMCPServer) beaconInfoHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args beaconInfoArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(beaconInfoToolName, "", args.BeaconID)

	beacon, err := s.Rpc.GetBeacon(ctx, &clientpb.Beacon{ID: args.BeaconID})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to get beacon", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(beaconSummary{
		ID:                  beacon.ID,
		Name:                beacon.Name,
		Transport:           beacon.Transport,
		RemoteAddress:       beacon.RemoteAddress,
		Hostname:            beacon.Hostname,
		Username:            beacon.Username,
		PID:                 beacon.PID,
		OS:                  beacon.OS,
		Arch:                beacon.Arch,
		Locale:              beacon.Locale,
		LastCheckin:         beacon.LastCheckin,
		NextCheckin:         beacon.NextCheckin,
		Interval:            beacon.Interval,
		Jitter:              beacon.Jitter,
		TasksCount:          beacon.TasksCount,
		TasksCountCompleted: beacon.TasksCountCompleted,
		IsDead:              beacon.IsDead,
		Integrity:           beacon.Integrity,
	}), nil
}

func (s *SliverMCPServer) killBeaconHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args killBeaconArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(killBeaconToolName, "", args.BeaconID)

	if _, err := s.Rpc.RmBeacon(ctx, &clientpb.Beacon{ID: args.BeaconID}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to remove beacon", err), nil
	}

	type rmResult struct {
		BeaconID string `json:"beacon_id"`
		Removed  bool   `json:"removed"`
	}
	return mcpapi.NewToolResultStructuredOnly(rmResult{
		BeaconID: args.BeaconID,
		Removed:  true,
	}), nil
}

func (s *SliverMCPServer) listBeaconTasksHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args listBeaconTasksArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(listBeaconTasksToolName, "", args.BeaconID)

	resp, err := s.Rpc.GetBeaconTasks(ctx, &clientpb.Beacon{ID: args.BeaconID})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list beacon tasks", err), nil
	}

	result := listBeaconTasksResult{
		Tasks: make([]beaconTaskResult, 0, len(resp.GetTasks())),
	}
	for _, task := range resp.GetTasks() {
		if task == nil {
			continue
		}
		result.Tasks = append(result.Tasks, beaconTaskResult{
			ID:          task.ID,
			BeaconID:    task.BeaconID,
			State:       task.State,
			Description: task.Description,
			CreatedAt:   task.CreatedAt,
			SentAt:      task.SentAt,
			CompletedAt: task.CompletedAt,
		})
	}
	result.Count = len(result.Tasks)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) cancelBeaconTaskHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args cancelBeaconTaskArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(cancelBeaconTaskToolName, "", "")

	task, err := s.Rpc.CancelBeaconTask(ctx, &clientpb.BeaconTask{ID: args.TaskID})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to cancel beacon task", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(beaconTaskResult{
		ID:          task.ID,
		BeaconID:    task.BeaconID,
		State:       task.State,
		Description: task.Description,
		CreatedAt:   task.CreatedAt,
		SentAt:      task.SentAt,
		CompletedAt: task.CompletedAt,
	}), nil
}

// --- Tier 2: Reconfigure Implant ---

type reconfigureImplantArgs struct {
	SessionID         string `json:"session_id" jsonschema:"description=Session ID"`
	BeaconID          string `json:"beacon_id" jsonschema:"description=Beacon ID"`
	ReconnectInterval int64  `json:"reconnect_interval" jsonschema:"description=Reconnect interval in nanoseconds"`
	BeaconInterval    int64  `json:"beacon_interval" jsonschema:"description=Beacon check-in interval in nanoseconds"`
	BeaconJitter      int64  `json:"beacon_jitter" jsonschema:"description=Beacon jitter in nanoseconds"`
	C2URI             string `json:"c2_uri" jsonschema:"description=New C2 URI"`
	Wait              bool   `json:"wait" jsonschema:"description=Wait for beacon task to complete"`
	TimeoutSeconds    int64  `json:"timeout_seconds" jsonschema:"description=Timeout in seconds"`
}

type reconfigureResult struct {
	Success bool `json:"success"`
}

func (s *SliverMCPServer) reconfigureImplantHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args reconfigureImplantArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	req, isBeacon, err := s.buildCheckedRequest(ctx, args.SessionID, args.BeaconID, args.TimeoutSeconds)
	if err != nil {
		return mcpapi.NewToolResultError(err.Error()), nil
	}
	s.logToolCall(reconfigureImplantToolName, args.SessionID, args.BeaconID)

	resp, err := s.Rpc.Reconfigure(ctx, &sliverpb.ReconfigureReq{
		ReconnectInterval: args.ReconnectInterval,
		BeaconInterval:    args.BeaconInterval,
		BeaconJitter:      args.BeaconJitter,
		C2URI:             args.C2URI,
		Request:           req,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("reconfigure failed", err), nil
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !args.Wait {
			return newAsyncResult("reconfigure_implant", taskID, args.BeaconID), nil
		}
		timeout := applyDefaultTimeout(true, args.TimeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.Reconfigure{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return mcpapi.NewToolResultErrorFromErr("beacon task failed", err), nil
		}
	}

	return mcpapi.NewToolResultStructuredOnly(reconfigureResult{Success: true}), nil
}

// --- Tier 2: Rename Implant ---

type renameImplantArgs struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID to rename"`
	BeaconID  string `json:"beacon_id" jsonschema:"description=Beacon ID to rename"`
	Name      string `json:"name" jsonschema:"description=New name for the implant,required"`
}

type renameResult struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
}

func (s *SliverMCPServer) renameImplantHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args renameImplantArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	if args.SessionID == "" && args.BeaconID == "" {
		return mcpapi.NewToolResultError("session_id or beacon_id is required"), nil
	}
	s.logToolCall(renameImplantToolName, args.SessionID, args.BeaconID)

	if _, err := s.Rpc.Rename(ctx, &clientpb.RenameReq{
		SessionID: args.SessionID,
		BeaconID:  args.BeaconID,
		Name:      args.Name,
	}); err != nil {
		return mcpapi.NewToolResultErrorFromErr("rename failed", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(renameResult{
		Name:    args.Name,
		Success: true,
	}), nil
}
