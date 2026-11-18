package mcp

import (
	"context"
	"log"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	mcpapi "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	listSessionsAndBeaconsToolName = "list_sessions_and_beacons"
)

type sessionSummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Transport     string `json:"transport"`
	RemoteAddress string `json:"remote_address"`
	Hostname      string `json:"hostname"`
	Username      string `json:"username"`
	PID           int32  `json:"pid"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Locale        string `json:"locale"`
	LastCheckin   int64  `json:"last_checkin"`
	IsDead        bool   `json:"is_dead"`
	Integrity     string `json:"integrity"`
}

type beaconSummary struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Transport           string `json:"transport"`
	RemoteAddress       string `json:"remote_address"`
	Hostname            string `json:"hostname"`
	Username            string `json:"username"`
	PID                 int32  `json:"pid"`
	OS                  string `json:"os"`
	Arch                string `json:"arch"`
	Locale              string `json:"locale"`
	LastCheckin         int64  `json:"last_checkin"`
	NextCheckin         int64  `json:"next_checkin"`
	Interval            int64  `json:"interval"`
	Jitter              int64  `json:"jitter"`
	TasksCount          int64  `json:"tasks_count"`
	TasksCountCompleted int64  `json:"tasks_count_completed"`
	IsDead              bool   `json:"is_dead"`
	Integrity           string `json:"integrity"`
}

type listSessionsAndBeaconsResult struct {
	Sessions      []sessionSummary `json:"sessions"`
	SessionsCount int              `json:"sessions_count"`
	Beacons       []beaconSummary  `json:"beacons"`
	BeaconsCount  int              `json:"beacons_count"`
}

// SliverMCPServer wraps the MCP server with Sliver RPC access for handlers.
type SliverMCPServer struct {
	Rpc    rpcpb.SliverRPCClient
	server *mcpserver.MCPServer
	logger *log.Logger
}

func newServer(cfg Config, rpc rpcpb.SliverRPCClient, logger *log.Logger) *SliverMCPServer {
	base := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)

	listSessionsAndBeaconsTool := mcpapi.NewTool(
		listSessionsAndBeaconsToolName,
		mcpapi.WithDescription("List active sessions and beacons."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	lsTool := mcpapi.NewTool(
		lsToolName,
		mcpapi.WithDescription("List the contents of a remote directory."),
		mcpapi.WithInputSchema[lsArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	cdTool := mcpapi.NewTool(
		cdToolName,
		mcpapi.WithDescription("Change the current directory on a remote session or beacon."),
		mcpapi.WithInputSchema[cdArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(false),
	)
	catTool := mcpapi.NewTool(
		catToolName,
		mcpapi.WithDescription("Download and return the contents of a remote file."),
		mcpapi.WithInputSchema[catArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	pwdTool := mcpapi.NewTool(
		pwdToolName,
		mcpapi.WithDescription("Return the current working directory of a remote session or beacon."),
		mcpapi.WithInputSchema[pwdArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithDestructiveHintAnnotation(false),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	rmTool := mcpapi.NewTool(
		rmToolName,
		mcpapi.WithDescription("Remove a file or directory on the remote target."),
		mcpapi.WithInputSchema[rmArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	mvTool := mcpapi.NewTool(
		mvToolName,
		mcpapi.WithDescription("Move or rename a file or directory on the remote target."),
		mcpapi.WithInputSchema[mvArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	cpTool := mcpapi.NewTool(
		cpToolName,
		mcpapi.WithDescription("Copy a file on the remote target."),
		mcpapi.WithInputSchema[cpArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	mkdirTool := mcpapi.NewTool(
		mkdirToolName,
		mcpapi.WithDescription("Create a directory on the remote target."),
		mcpapi.WithInputSchema[mkdirArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	chmodTool := mcpapi.NewTool(
		chmodToolName,
		mcpapi.WithDescription("Change file or directory permissions on the remote target."),
		mcpapi.WithInputSchema[chmodArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	chownTool := mcpapi.NewTool(
		chownToolName,
		mcpapi.WithDescription("Change file or directory ownership on the remote target."),
		mcpapi.WithInputSchema[chownArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	// --- Tier 1: Meta tools ---
	serverVersionTool := mcpapi.NewTool(
		serverVersionToolName,
		mcpapi.WithDescription("Get the Sliver server version."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	listOperatorsTool := mcpapi.NewTool(
		listOperatorsToolName,
		mcpapi.WithDescription("List connected operators."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)

	// --- Tier 1: Session/Beacon management ---
	sessionInfoTool := mcpapi.NewTool(
		sessionInfoToolName,
		mcpapi.WithDescription("Get detailed info for a specific session."),
		mcpapi.WithInputSchema[sessionInfoArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	killSessionTool := mcpapi.NewTool(
		killSessionToolName,
		mcpapi.WithDescription("Kill a session, terminating the implant."),
		mcpapi.WithInputSchema[killSessionArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	beaconInfoTool := mcpapi.NewTool(
		beaconInfoToolName,
		mcpapi.WithDescription("Get detailed info for a specific beacon."),
		mcpapi.WithInputSchema[beaconInfoArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	killBeaconTool := mcpapi.NewTool(
		killBeaconToolName,
		mcpapi.WithDescription("Remove a beacon from the server."),
		mcpapi.WithInputSchema[killBeaconArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	listBeaconTasksTool := mcpapi.NewTool(
		listBeaconTasksToolName,
		mcpapi.WithDescription("List tasks for a specific beacon."),
		mcpapi.WithInputSchema[listBeaconTasksArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	cancelBeaconTaskTool := mcpapi.NewTool(
		cancelBeaconTaskToolName,
		mcpapi.WithDescription("Cancel a pending beacon task."),
		mcpapi.WithInputSchema[cancelBeaconTaskArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)

	// --- Tier 1: Listeners ---
	listJobsTool := mcpapi.NewTool(
		listJobsToolName,
		mcpapi.WithDescription("List active jobs (listeners)."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	startMTLSTool := mcpapi.NewTool(
		startMTLSListenerToolName,
		mcpapi.WithDescription("Start an mTLS listener."),
		mcpapi.WithInputSchema[startMTLSArgs](),
	)
	startHTTPSTool := mcpapi.NewTool(
		startHTTPSToolName,
		mcpapi.WithDescription("Start an HTTPS listener."),
		mcpapi.WithInputSchema[startHTTPSArgs](),
	)
	startHTTPTool := mcpapi.NewTool(
		startHTTPToolName,
		mcpapi.WithDescription("Start an HTTP listener."),
		mcpapi.WithInputSchema[startHTTPArgs](),
	)
	startDNSTool := mcpapi.NewTool(
		startDNSToolName,
		mcpapi.WithDescription("Start a DNS listener."),
		mcpapi.WithInputSchema[startDNSArgs](),
	)
	startWGTool := mcpapi.NewTool(
		startWGToolName,
		mcpapi.WithDescription("Start a WireGuard listener."),
		mcpapi.WithInputSchema[startWGArgs](),
	)
	killJobTool := mcpapi.NewTool(
		killJobToolName,
		mcpapi.WithDescription("Kill a job (stop a listener)."),
		mcpapi.WithInputSchema[killJobArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)

	// --- Tier 1: Execution ---
	executeTool := mcpapi.NewTool(
		executeToolName,
		mcpapi.WithDescription("Execute a command on the remote target."),
		mcpapi.WithInputSchema[executeArgs](),
	)

	// --- Tier 1: Filesystem additions ---
	downloadFileTool := mcpapi.NewTool(
		downloadToolName,
		mcpapi.WithDescription("Download a file from the remote target. Returns content as base64 and text if valid UTF-8."),
		mcpapi.WithInputSchema[downloadFileArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	uploadFileTool := mcpapi.NewTool(
		uploadToolName,
		mcpapi.WithDescription("Upload a file to the remote target. Data must be provided as base64."),
		mcpapi.WithInputSchema[uploadFileArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)

	// --- Tier 1: Processes ---
	listProcessesTool := mcpapi.NewTool(
		listProcessesToolName,
		mcpapi.WithDescription("List running processes on the remote target."),
		mcpapi.WithInputSchema[listProcessesArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)

	// --- Tier 1: Network ---
	networkInterfacesTool := mcpapi.NewTool(
		networkInterfacesToolName,
		mcpapi.WithDescription("List network interfaces on the remote target."),
		mcpapi.WithInputSchema[networkInterfacesArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	networkConnectionsTool := mcpapi.NewTool(
		networkConnectionsToolName,
		mcpapi.WithDescription("List network connections (netstat) on the remote target."),
		mcpapi.WithInputSchema[networkConnectionsArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)

	// --- Tier 1: Environment ---
	getEnvironmentTool := mcpapi.NewTool(
		getEnvironmentToolName,
		mcpapi.WithDescription("Get environment variables from the remote target."),
		mcpapi.WithInputSchema[getEnvironmentArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)

	// --- Tier 1: Registry ---
	registryReadTool := mcpapi.NewTool(
		registryReadToolName,
		mcpapi.WithDescription("Read a Windows registry value."),
		mcpapi.WithInputSchema[registryReadArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)

	// --- Tier 1: Screenshot ---
	screenshotTool := mcpapi.NewTool(
		screenshotToolName,
		mcpapi.WithDescription("Take a screenshot of the remote target's screen. Returns PNG as base64."),
		mcpapi.WithInputSchema[screenshotArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)

	srv := &SliverMCPServer{
		Rpc:    rpc,
		server: base,
		logger: logger,
	}

	// Tier 0: Existing tools
	srv.server.AddTool(listSessionsAndBeaconsTool, srv.listSessionsAndBeaconsHandler)
	srv.server.AddTool(lsTool, srv.lsHandler)
	srv.server.AddTool(cdTool, srv.cdHandler)
	srv.server.AddTool(catTool, srv.catHandler)
	srv.server.AddTool(pwdTool, srv.pwdHandler)
	srv.server.AddTool(rmTool, srv.rmHandler)
	srv.server.AddTool(mvTool, srv.mvHandler)
	srv.server.AddTool(cpTool, srv.cpHandler)
	srv.server.AddTool(mkdirTool, srv.mkdirHandler)
	srv.server.AddTool(chmodTool, srv.chmodHandler)
	srv.server.AddTool(chownTool, srv.chownHandler)

	// Tier 1: Meta
	srv.server.AddTool(serverVersionTool, srv.serverVersionHandler)
	srv.server.AddTool(listOperatorsTool, srv.listOperatorsHandler)

	// Tier 1: Sessions/Beacons
	srv.server.AddTool(sessionInfoTool, srv.sessionInfoHandler)
	srv.server.AddTool(killSessionTool, srv.killSessionHandler)
	srv.server.AddTool(beaconInfoTool, srv.beaconInfoHandler)
	srv.server.AddTool(killBeaconTool, srv.killBeaconHandler)
	srv.server.AddTool(listBeaconTasksTool, srv.listBeaconTasksHandler)
	srv.server.AddTool(cancelBeaconTaskTool, srv.cancelBeaconTaskHandler)

	// Tier 1: Listeners
	srv.server.AddTool(listJobsTool, srv.listJobsHandler)
	srv.server.AddTool(startMTLSTool, srv.startMTLSListenerHandler)
	srv.server.AddTool(startHTTPSTool, srv.startHTTPSListenerHandler)
	srv.server.AddTool(startHTTPTool, srv.startHTTPListenerHandler)
	srv.server.AddTool(startDNSTool, srv.startDNSListenerHandler)
	srv.server.AddTool(startWGTool, srv.startWGListenerHandler)
	srv.server.AddTool(killJobTool, srv.killJobHandler)

	// Tier 1: Execution
	srv.server.AddTool(executeTool, srv.executeHandler)

	// Tier 1: Filesystem additions
	srv.server.AddTool(downloadFileTool, srv.downloadFileHandler)
	srv.server.AddTool(uploadFileTool, srv.uploadFileHandler)

	// Tier 1: Processes
	srv.server.AddTool(listProcessesTool, srv.listProcessesHandler)

	// Tier 1: Network
	srv.server.AddTool(networkInterfacesTool, srv.networkInterfacesHandler)
	srv.server.AddTool(networkConnectionsTool, srv.networkConnectionsHandler)

	// Tier 1: Environment
	srv.server.AddTool(getEnvironmentTool, srv.getEnvironmentHandler)

	// Tier 1: Registry
	srv.server.AddTool(registryReadTool, srv.registryReadHandler)

	// Tier 1: Screenshot
	srv.server.AddTool(screenshotTool, srv.screenshotHandler)

	// ===== Tier 2: Core Operations =====

	// Tier 2: Filesystem
	grepTool := mcpapi.NewTool(grepToolName,
		mcpapi.WithDescription("Search file contents on the implant using regex patterns."),
		mcpapi.WithInputSchema[grepArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	chtimesTool := mcpapi.NewTool(chtimesToolName,
		mcpapi.WithDescription("Change file access and modification timestamps (timestomping)."),
		mcpapi.WithInputSchema[chtimesArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	mountTool := mcpapi.NewTool(mountToolName,
		mcpapi.WithDescription("List mounted filesystems/volumes on the implant."),
		mcpapi.WithInputSchema[mountArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	memfilesListTool := mcpapi.NewTool(memfilesListToolName,
		mcpapi.WithDescription("List in-memory files on the implant."),
		mcpapi.WithInputSchema[memfilesListArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	memfilesAddTool := mcpapi.NewTool(memfilesAddToolName,
		mcpapi.WithDescription("Create a new in-memory file on the implant."),
		mcpapi.WithInputSchema[memfilesAddArgs](),
	)
	memfilesRmTool := mcpapi.NewTool(memfilesRmToolName,
		mcpapi.WithDescription("Remove an in-memory file by file descriptor."),
		mcpapi.WithInputSchema[memfilesRmArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(grepTool, srv.grepHandler)
	srv.server.AddTool(chtimesTool, srv.chtimesHandler)
	srv.server.AddTool(mountTool, srv.mountHandler)
	srv.server.AddTool(memfilesListTool, srv.memfilesListHandler)
	srv.server.AddTool(memfilesAddTool, srv.memfilesAddHandler)
	srv.server.AddTool(memfilesRmTool, srv.memfilesRmHandler)

	// Tier 2: Processes
	terminateProcessTool := mcpapi.NewTool(terminateProcessToolName,
		mcpapi.WithDescription("Terminate a process by PID on the implant."),
		mcpapi.WithInputSchema[terminateProcessArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	processDumpTool := mcpapi.NewTool(processDumpToolName,
		mcpapi.WithDescription("Dump the memory of a process by PID (returns base64-encoded data)."),
		mcpapi.WithInputSchema[processDumpArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	srv.server.AddTool(terminateProcessTool, srv.terminateProcessHandler)
	srv.server.AddTool(processDumpTool, srv.processDumpHandler)

	// Tier 2: Environment
	setEnvironmentTool := mcpapi.NewTool(setEnvironmentToolName,
		mcpapi.WithDescription("Set an environment variable on the implant."),
		mcpapi.WithInputSchema[setEnvironmentArgs](),
	)
	unsetEnvironmentTool := mcpapi.NewTool(unsetEnvironmentToolName,
		mcpapi.WithDescription("Remove an environment variable on the implant."),
		mcpapi.WithInputSchema[unsetEnvironmentArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(setEnvironmentTool, srv.setEnvironmentHandler)
	srv.server.AddTool(unsetEnvironmentTool, srv.unsetEnvironmentHandler)

	// Tier 2: Execution
	executeAssemblyTool := mcpapi.NewTool(executeAssemblyToolName,
		mcpapi.WithDescription("Execute a .NET assembly in-memory on the implant (Windows)."),
		mcpapi.WithInputSchema[executeAssemblyArgs](),
	)
	sideloadTool := mcpapi.NewTool(sideloadToolName,
		mcpapi.WithDescription("Load and execute a shared library (DLL/dylib/so) in a sacrificial process."),
		mcpapi.WithInputSchema[sideloadArgs](),
	)
	spawnDllTool := mcpapi.NewTool(spawnDllToolName,
		mcpapi.WithDescription("Spawn a DLL in a sacrificial process and call an entry point (Windows)."),
		mcpapi.WithInputSchema[spawnDllArgs](),
	)
	runSSHCommandTool := mcpapi.NewTool(runSSHCommandToolName,
		mcpapi.WithDescription("Execute a command on a remote host via SSH from the implant."),
		mcpapi.WithInputSchema[runSSHCommandArgs](),
	)
	srv.server.AddTool(executeAssemblyTool, srv.executeAssemblyHandler)
	srv.server.AddTool(sideloadTool, srv.sideloadHandler)
	srv.server.AddTool(spawnDllTool, srv.spawnDllHandler)
	srv.server.AddTool(runSSHCommandTool, srv.runSSHCommandHandler)

	// Tier 2: Registry
	registryWriteTool := mcpapi.NewTool(registryWriteToolName,
		mcpapi.WithDescription("Write a value to the Windows registry."),
		mcpapi.WithInputSchema[registryWriteArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	registryCreateKeyTool := mcpapi.NewTool(registryCreateKeyToolName,
		mcpapi.WithDescription("Create a new registry key in the Windows registry."),
		mcpapi.WithInputSchema[registryCreateKeyArgs](),
	)
	registryDeleteKeyTool := mcpapi.NewTool(registryDeleteKeyToolName,
		mcpapi.WithDescription("Delete a registry key from the Windows registry."),
		mcpapi.WithInputSchema[registryDeleteKeyArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	registryListSubkeysTool := mcpapi.NewTool(registryListSubkeysToolName,
		mcpapi.WithDescription("List subkeys under a registry key."),
		mcpapi.WithInputSchema[registryListSubkeysArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	registryListValuesTool := mcpapi.NewTool(registryListValuesToolName,
		mcpapi.WithDescription("List value names under a registry key."),
		mcpapi.WithInputSchema[registryListValuesArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	registryReadHiveTool := mcpapi.NewTool(registryReadHiveToolName,
		mcpapi.WithDescription("Dump an entire registry hive (e.g. SAM, SYSTEM, SECURITY) as base64."),
		mcpapi.WithInputSchema[registryReadHiveArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	srv.server.AddTool(registryWriteTool, srv.registryWriteHandler)
	srv.server.AddTool(registryCreateKeyTool, srv.registryCreateKeyHandler)
	srv.server.AddTool(registryDeleteKeyTool, srv.registryDeleteKeyHandler)
	srv.server.AddTool(registryListSubkeysTool, srv.registryListSubkeysHandler)
	srv.server.AddTool(registryListValuesTool, srv.registryListValuesHandler)
	srv.server.AddTool(registryReadHiveTool, srv.registryReadHiveHandler)

	// Tier 2: Sessions
	reconfigureImplantTool := mcpapi.NewTool(reconfigureImplantToolName,
		mcpapi.WithDescription("Reconfigure implant settings (reconnect interval, beacon interval/jitter, C2 URI)."),
		mcpapi.WithInputSchema[reconfigureImplantArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	renameImplantTool := mcpapi.NewTool(renameImplantToolName,
		mcpapi.WithDescription("Rename a session or beacon implant."),
		mcpapi.WithInputSchema[renameImplantArgs](),
	)
	srv.server.AddTool(reconfigureImplantTool, srv.reconfigureImplantHandler)
	srv.server.AddTool(renameImplantTool, srv.renameImplantHandler)

	return srv
}

func (s *SliverMCPServer) listSessionsAndBeaconsHandler(ctx context.Context, _ mcpapi.CallToolRequest) (*mcpapi.CallToolResult, error) {
	if s.Rpc == nil {
		return mcpapi.NewToolResultError("rpc client not configured"), nil
	}

	s.logToolCall(listSessionsAndBeaconsToolName, "", "")

	sessionsResp, err := s.Rpc.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list sessions", err), nil
	}

	beaconsResp, err := s.Rpc.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return mcpapi.NewToolResultErrorFromErr("failed to list beacons", err), nil
	}

	result := listSessionsAndBeaconsResult{
		Sessions: make([]sessionSummary, 0, len(sessionsResp.GetSessions())),
		Beacons:  make([]beaconSummary, 0, len(beaconsResp.GetBeacons())),
	}

	for _, session := range sessionsResp.GetSessions() {
		if session == nil {
			continue
		}
		result.Sessions = append(result.Sessions, sessionSummary{
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
		})
	}

	for _, beacon := range beaconsResp.GetBeacons() {
		if beacon == nil {
			continue
		}
		result.Beacons = append(result.Beacons, beaconSummary{
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
		})
	}

	result.SessionsCount = len(result.Sessions)
	result.BeaconsCount = len(result.Beacons)

	return mcpapi.NewToolResultStructuredOnly(result), nil
}
