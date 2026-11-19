package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/protobuf/proto"
)

const (
	defaultWaitTimeoutSeconds = int64(60)
	pollInterval              = 1 * time.Second
)

type asyncTaskResult struct {
	Async     bool   `json:"async"`
	Operation string `json:"operation"`
	TaskID    string `json:"task_id"`
	BeaconID  string `json:"beacon_id,omitempty"`
	State     string `json:"state,omitempty"`
}

func buildRequest(sessionID, beaconID string, timeoutSeconds int64) (*commonpb.Request, bool, error) {
	if sessionID == "" && beaconID == "" {
		return nil, false, fmt.Errorf("session_id or beacon_id is required")
	}
	if sessionID != "" && beaconID != "" {
		return nil, false, fmt.Errorf("provide only one of session_id or beacon_id")
	}

	req := &commonpb.Request{}
	if timeoutSeconds > 0 {
		req.Timeout = (int64(time.Second) * timeoutSeconds) - 1
	}

	if sessionID != "" {
		req.SessionID = sessionID
		req.Async = false
		return req, false, nil
	}

	req.BeaconID = beaconID
	req.Async = true
	return req, true, nil
}

func (s *SliverMCPServer) buildCheckedRequest(ctx context.Context, sessionID, beaconID string, timeoutSeconds int64) (*commonpb.Request, bool, error) {
	req, isBeacon, err := buildRequest(sessionID, beaconID, timeoutSeconds)
	if err != nil {
		return nil, false, err
	}
	if isBeacon {
		if err := s.checkBeaconAlive(ctx, beaconID); err != nil {
			return nil, false, err
		}
	}
	return req, isBeacon, nil
}

func (s *SliverMCPServer) checkBeaconAlive(ctx context.Context, beaconID string) error {
	beacon, err := s.Rpc.GetBeacon(ctx, &clientpb.Beacon{ID: beaconID})
	if err != nil {
		return fmt.Errorf("beacon %s not found: %w", beaconID, err)
	}

	lastSeen := time.Unix(beacon.LastCheckin, 0)
	ago := time.Since(lastSeen).Truncate(time.Second)

	if beacon.IsDead {
		return fmt.Errorf("beacon %s (%s@%s) is dead — last check-in %s ago at %s. Use list_sessions_and_beacons to find a live beacon",
			beaconID, beacon.Username, beacon.Hostname, ago, lastSeen.Format(time.RFC3339))
	}

	// Beacon is stale if it missed 3+ check-in windows (interval + jitter)
	maxWindow := time.Duration(beacon.Interval + beacon.Jitter)
	if maxWindow > 0 && time.Since(lastSeen) > 3*maxWindow {
		return fmt.Errorf("beacon %s (%s@%s) appears stale — last check-in %s ago (expected every %s). The beacon may have died without the server marking it dead. Use list_sessions_and_beacons to verify or kill_beacon to clean up",
			beaconID, beacon.Username, beacon.Hostname, ago, maxWindow.Truncate(time.Second))
	}

	return nil
}

func applyDefaultTimeout(wait bool, timeoutSeconds int64) int64 {
	if wait && timeoutSeconds <= 0 {
		return defaultWaitTimeoutSeconds
	}
	return timeoutSeconds
}

func withTimeout(ctx context.Context, timeoutSeconds int64) (context.Context, context.CancelFunc) {
	if timeoutSeconds <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
}

func (s *SliverMCPServer) waitForBeaconTaskResponse(ctx context.Context, taskID string, out proto.Message) error {
	task, err := s.waitForBeaconTask(ctx, taskID)
	if err != nil {
		return err
	}

	state := strings.ToLower(task.State)
	if state != "completed" {
		if state == "" {
			state = "unknown"
		}
		return fmt.Errorf("beacon task %s %s", task.ID, state)
	}
	if len(task.Response) == 0 {
		return fmt.Errorf("beacon task %s returned empty response", task.ID)
	}
	return proto.Unmarshal(task.Response, out)
}

func (s *SliverMCPServer) waitForBeaconTask(ctx context.Context, taskID string) (*clientpb.BeaconTask, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		task, err := s.Rpc.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		if err != nil {
			return nil, err
		}
		state := strings.ToLower(task.State)
		if state == "completed" || state == "failed" || state == "canceled" {
			return task, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// callBOFResult holds the outcome of a single BOF execution.
type callBOFResult struct {
	Output      string
	RespErr     string
	AsyncTaskID string // non-empty only when beacon + !wait
}

// callBOF is the single code path for executing a BOF via CallExtension.
// Both execute_bof, run_bof_module, and run_overlord_suite delegate here.
func (s *SliverMCPServer) callBOF(ctx context.Context, req *commonpb.Request, isBeacon bool, bofData, bofArgs []byte, wait bool, timeoutSeconds int64) (*callBOFResult, error) {
	resp, err := s.Rpc.CallExtension(ctx, &sliverpb.CallExtensionReq{
		IsBOF:          true,
		BOFData:        bofData,
		Export:         "go",
		Args:           bofArgs,
		WantBOFOutputs: true,
		Request:        req,
	})
	if err != nil {
		return nil, err
	}

	if isBeacon && resp.Response != nil && resp.Response.Async {
		taskID := resp.Response.TaskID
		if !wait {
			return &callBOFResult{AsyncTaskID: taskID}, nil
		}
		timeout := applyDefaultTimeout(true, timeoutSeconds)
		tctx, cancel := withTimeout(ctx, timeout)
		if cancel != nil {
			defer cancel()
		}
		waited := &sliverpb.CallExtension{}
		if err := s.waitForBeaconTaskResponse(tctx, taskID, waited); err != nil {
			return nil, err
		}
		resp = waited
	}

	output := bofOutputString(resp)
	var respErr string
	if resp.Response != nil {
		respErr = resp.Response.Err
	}
	return &callBOFResult{Output: output, RespErr: respErr}, nil
}

func newAsyncResult(operation, taskID, beaconID string) *mcpapi.CallToolResult {
	state := "pending"
	if taskID == "" {
		state = "unknown"
	}
	result := asyncTaskResult{
		Async:     true,
		Operation: operation,
		TaskID:    taskID,
		BeaconID:  beaconID,
		State:     state,
	}
	return mcpapi.NewToolResultStructuredOnly(result)
}
