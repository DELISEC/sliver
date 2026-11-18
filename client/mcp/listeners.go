package mcp

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
)

const (
	listJobsToolName          = "list_jobs"
	startMTLSListenerToolName = "start_mtls_listener"
	startHTTPSToolName        = "start_https_listener"
	startHTTPToolName         = "start_http_listener"
	startDNSToolName          = "start_dns_listener"
	startWGToolName           = "start_wg_listener"
	killJobToolName           = "kill_job"
)

// --- Args structs ---

type startMTLSArgs struct {
	Host string `json:"host" jsonschema:"description=Bind host (default: 0.0.0.0)"`
	Port uint32 `json:"port" jsonschema:"description=Bind port (default: 8888)"`
}

type startHTTPSArgs struct {
	Domain          string `json:"domain" jsonschema:"description=Domain for the HTTPS listener"`
	Host            string `json:"host" jsonschema:"description=Bind host (default: 0.0.0.0)"`
	Port            uint32 `json:"port" jsonschema:"description=Bind port (default: 443)"`
	Website         string `json:"website" jsonschema:"description=Website name to serve"`
	ACME            bool   `json:"acme" jsonschema:"description=Use ACME for TLS certificates"`
	EnforceOTP      bool   `json:"enforce_otp" jsonschema:"description=Enforce OTP authentication"`
	LongPollTimeout int64  `json:"long_poll_timeout" jsonschema:"description=Long poll timeout in seconds"`
	LongPollJitter  int64  `json:"long_poll_jitter" jsonschema:"description=Long poll jitter in seconds"`
	RandomizeJARM   bool   `json:"randomize_jarm" jsonschema:"description=Randomize JARM fingerprint"`
}

type startHTTPArgs struct {
	Domain          string `json:"domain" jsonschema:"description=Domain for the HTTP listener"`
	Host            string `json:"host" jsonschema:"description=Bind host (default: 0.0.0.0)"`
	Port            uint32 `json:"port" jsonschema:"description=Bind port (default: 80)"`
	Website         string `json:"website" jsonschema:"description=Website name to serve"`
	EnforceOTP      bool   `json:"enforce_otp" jsonschema:"description=Enforce OTP authentication"`
	LongPollTimeout int64  `json:"long_poll_timeout" jsonschema:"description=Long poll timeout in seconds"`
	LongPollJitter  int64  `json:"long_poll_jitter" jsonschema:"description=Long poll jitter in seconds"`
}

type startDNSArgs struct {
	Domains    []string `json:"domains" jsonschema:"description=DNS domains to listen on,required"`
	Canaries   bool     `json:"canaries" jsonschema:"description=Enable DNS canaries"`
	Host       string   `json:"host" jsonschema:"description=Bind host (default: 0.0.0.0)"`
	Port       uint32   `json:"port" jsonschema:"description=Bind port (default: 53)"`
	EnforceOTP bool     `json:"enforce_otp" jsonschema:"description=Enforce OTP authentication"`
}

type startWGArgs struct {
	Host    string `json:"host" jsonschema:"description=Bind host (default: 0.0.0.0)"`
	Port    uint32 `json:"port" jsonschema:"description=WireGuard port (default: 53)"`
	NPort   uint32 `json:"n_port" jsonschema:"description=Virtual network port"`
	KeyPort uint32 `json:"key_port" jsonschema:"description=Key exchange port"`
}

type killJobArgs struct {
	JobID uint32 `json:"job_id" jsonschema:"description=Job ID to kill,required"`
}

// --- Result structs ---

type jobSummary struct {
	ID          uint32   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Protocol    string   `json:"protocol"`
	Port        uint32   `json:"port"`
	Domains     []string `json:"domains,omitempty"`
}

type listJobsResult struct {
	Jobs  []jobSummary `json:"jobs"`
	Count int          `json:"count"`
}

type listenerResult struct {
	JobID uint32 `json:"job_id"`
	Type  string `json:"type"`
}

type killJobResult struct {
	JobID   uint32 `json:"job_id"`
	Success bool   `json:"success"`
}

// --- Handlers ---

func (s *SliverMCPServer) listJobsHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}
	s.logToolCall(listJobsToolName, "", "")

	resp, err := s.Rpc.GetJobs(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list jobs", err), nil
	}

	result := listJobsResult{
		Jobs: make([]jobSummary, 0, len(resp.GetActive())),
	}
	for _, job := range resp.GetActive() {
		if job == nil {
			continue
		}
		result.Jobs = append(result.Jobs, jobSummary{
			ID:          job.ID,
			Name:        job.Name,
			Description: job.Description,
			Protocol:    job.Protocol,
			Port:        job.Port,
			Domains:     job.Domains,
		})
	}
	result.Count = len(result.Jobs)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}

func (s *SliverMCPServer) startMTLSListenerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startMTLSArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(startMTLSListenerToolName, "", "")

	job, err := s.Rpc.StartMTLSListener(ctx, &clientpb.MTLSListenerReq{
		Host: args.Host,
		Port: args.Port,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to start MTLS listener", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(listenerResult{
		JobID: job.JobID,
		Type:  job.Type,
	}), nil
}

func (s *SliverMCPServer) startHTTPSListenerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startHTTPSArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(startHTTPSToolName, "", "")

	job, err := s.Rpc.StartHTTPSListener(ctx, &clientpb.HTTPListenerReq{
		Domain:          args.Domain,
		Host:            args.Host,
		Port:            args.Port,
		Secure:          true,
		Website:         args.Website,
		ACME:            args.ACME,
		EnforceOTP:      args.EnforceOTP,
		LongPollTimeout: args.LongPollTimeout,
		LongPollJitter:  args.LongPollJitter,
		RandomizeJARM:   args.RandomizeJARM,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to start HTTPS listener", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(listenerResult{
		JobID: job.JobID,
		Type:  job.Type,
	}), nil
}

func (s *SliverMCPServer) startHTTPListenerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startHTTPArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(startHTTPToolName, "", "")

	job, err := s.Rpc.StartHTTPListener(ctx, &clientpb.HTTPListenerReq{
		Domain:          args.Domain,
		Host:            args.Host,
		Port:            args.Port,
		Secure:          false,
		Website:         args.Website,
		EnforceOTP:      args.EnforceOTP,
		LongPollTimeout: args.LongPollTimeout,
		LongPollJitter:  args.LongPollJitter,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to start HTTP listener", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(listenerResult{
		JobID: job.JobID,
		Type:  job.Type,
	}), nil
}

func (s *SliverMCPServer) startDNSListenerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startDNSArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(startDNSToolName, "", "")

	job, err := s.Rpc.StartDNSListener(ctx, &clientpb.DNSListenerReq{
		Domains:    args.Domains,
		Canaries:   args.Canaries,
		Host:       args.Host,
		Port:       args.Port,
		EnforceOTP: args.EnforceOTP,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to start DNS listener", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(listenerResult{
		JobID: job.JobID,
		Type:  job.Type,
	}), nil
}

func (s *SliverMCPServer) startWGListenerHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args startWGArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(startWGToolName, "", "")

	job, err := s.Rpc.StartWGListener(ctx, &clientpb.WGListenerReq{
		Host:    args.Host,
		Port:    args.Port,
		NPort:   args.NPort,
		KeyPort: args.KeyPort,
	})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to start WireGuard listener", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(listenerResult{
		JobID: job.JobID,
		Type:  job.Type,
	}), nil
}

func (s *SliverMCPServer) killJobHandler(ctx context.Context, request mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	var args killJobArgs
	if err := request.BindArguments(&args); err != nil {
		return mcpapi.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}
	s.logToolCall(killJobToolName, "", "")

	resp, err := s.Rpc.KillJob(ctx, &clientpb.KillJobReq{ID: args.JobID})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to kill job", err), nil
	}

	return mcpapi.NewToolResultStructuredOnly(killJobResult{
		JobID:   resp.ID,
		Success: resp.Success,
	}), nil
}
