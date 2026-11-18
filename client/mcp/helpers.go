package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
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
