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
	Rpc           rpcpb.SliverRPCClient
	server        *mcpserver.MCPServer
	logger        *log.Logger
	BOFModulesDir string
	OverlordDir   string
	ZerglingPath  string
}

func newServer(cfg Config, rpc rpcpb.SliverRPCClient, logger *log.Logger) *SliverMCPServer {
	base := mcpserver.NewMCPServer(
		cfg.ServerName,
		cfg.ServerVersion,
		mcpserver.WithToolCapabilities(false),
	)

	listSessionsAndBeaconsTool := mcpapi.NewTool(
		listSessionsAndBeaconsToolName,
		mcpapi.WithDescription("List active sessions and beacons. Sessions are interactive (real-time). Beacons check in periodically (interval + jitter) — check last_checkin/next_checkin/is_dead before tasking a beacon. Dead beacons will not pick up tasks. When using wait:true with a beacon, set timeout_seconds > interval + jitter (e.g. 180s for a 60s/30s beacon)."),
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
		mcpapi.WithDescription("Execute a command on the remote target. For beacons: the command queues until the next check-in. Use wait:true with timeout_seconds > beacon interval+jitter to wait for results. Rejects dead beacons — use list_sessions_and_beacons to verify liveness first."),
		mcpapi.WithInputSchema[executeArgs](),
	)

	// --- Tier 1: Filesystem additions ---
	downloadFileTool := mcpapi.NewTool(
		downloadToolName,
		mcpapi.WithDescription("Download a file from the remote target. Saves the file to the operator's local disk (default: ./<filename>) and returns the local path. Use local_path to specify a different save location. Small UTF-8 files include a text preview."),
		mcpapi.WithInputSchema[downloadFileArgs](),
		mcpapi.WithReadOnlyHintAnnotation(false),
	)
	uploadFileTool := mcpapi.NewTool(
		uploadToolName,
		mcpapi.WithDescription("Upload a file to the remote target. Provide data_base64 (base64-encoded content) OR local_path (path to a file on the operator machine — preferred for large files like implants). Rejects dead beacons. For beacons, set wait:true with timeout_seconds > interval+jitter."),
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
		Rpc:           rpc,
		server:        base,
		logger:        logger,
		BOFModulesDir: cfg.BOFModulesDir,
		OverlordDir:   cfg.OverlordDir,
		ZerglingPath:  cfg.ZerglingPath,
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

	// ===== Tier 3: Privilege Escalation & Services =====

	// Tier 3: Privilege
	getSystemTool := mcpapi.NewTool(getSystemToolName,
		mcpapi.WithDescription("Attempt to get SYSTEM privileges on Windows via named pipe impersonation."),
		mcpapi.WithInputSchema[getSystemArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	runAsTool := mcpapi.NewTool(runAsToolName,
		mcpapi.WithDescription("Execute a process as another user (Windows runas)."),
		mcpapi.WithInputSchema[runAsArgs](),
	)
	impersonateTool := mcpapi.NewTool(impersonateToolName,
		mcpapi.WithDescription("Impersonate a logged-in user's token (Windows)."),
		mcpapi.WithInputSchema[impersonateArgs](),
	)
	revToSelfTool := mcpapi.NewTool(revToSelfToolName,
		mcpapi.WithDescription("Revert token to original user after impersonation (Windows)."),
		mcpapi.WithInputSchema[revToSelfArgs](),
	)
	makeTokenTool := mcpapi.NewTool(makeTokenToolName,
		mcpapi.WithDescription("Create a Windows logon token with credentials."),
		mcpapi.WithInputSchema[makeTokenArgs](),
	)
	currentTokenOwnerTool := mcpapi.NewTool(currentTokenOwnerToolName,
		mcpapi.WithDescription("Get the owner of the current process token (Windows)."),
		mcpapi.WithInputSchema[currentTokenOwnerArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	getPrivsTool := mcpapi.NewTool(getPrivsToolName,
		mcpapi.WithDescription("List current Windows token privileges and process integrity."),
		mcpapi.WithInputSchema[getPrivsArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	srv.server.AddTool(getSystemTool, srv.getSystemHandler)
	srv.server.AddTool(runAsTool, srv.runAsHandler)
	srv.server.AddTool(impersonateTool, srv.impersonateHandler)
	srv.server.AddTool(revToSelfTool, srv.revToSelfHandler)
	srv.server.AddTool(makeTokenTool, srv.makeTokenHandler)
	srv.server.AddTool(currentTokenOwnerTool, srv.currentTokenOwnerHandler)
	srv.server.AddTool(getPrivsTool, srv.getPrivsHandler)

	// Tier 3: Services
	listServicesTool := mcpapi.NewTool(listServicesToolName,
		mcpapi.WithDescription("List Windows services on the target."),
		mcpapi.WithInputSchema[listServicesArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	serviceDetailTool := mcpapi.NewTool(serviceDetailToolName,
		mcpapi.WithDescription("Get detailed info about a specific Windows service."),
		mcpapi.WithInputSchema[serviceDetailArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	startServiceByNameTool := mcpapi.NewTool(startServiceByNameToolName,
		mcpapi.WithDescription("Start an existing Windows service by name."),
		mcpapi.WithInputSchema[startServiceByNameArgs](),
	)
	startServiceTool := mcpapi.NewTool(startServiceToolName,
		mcpapi.WithDescription("Create and start a new Windows service with a custom binary."),
		mcpapi.WithInputSchema[startServiceArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	stopServiceTool := mcpapi.NewTool(stopServiceToolName,
		mcpapi.WithDescription("Stop a running Windows service."),
		mcpapi.WithInputSchema[stopServiceArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	removeServiceTool := mcpapi.NewTool(removeServiceToolName,
		mcpapi.WithDescription("Remove a Windows service from the system."),
		mcpapi.WithInputSchema[removeServiceArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(listServicesTool, srv.listServicesHandler)
	srv.server.AddTool(serviceDetailTool, srv.serviceDetailHandler)
	srv.server.AddTool(startServiceByNameTool, srv.startServiceByNameHandler)
	srv.server.AddTool(startServiceTool, srv.startServiceHandler)
	srv.server.AddTool(stopServiceTool, srv.stopServiceHandler)
	srv.server.AddTool(removeServiceTool, srv.removeServiceHandler)

	// ===== Tier 4: Data Management =====

	// Tier 4: Loot
	lootAddTool := mcpapi.NewTool(lootAddToolName,
		mcpapi.WithDescription("Add a loot entry (file/data captured from a target)."),
		mcpapi.WithInputSchema[lootAddArgs](),
	)
	lootRmTool := mcpapi.NewTool(lootRmToolName,
		mcpapi.WithDescription("Remove a loot entry."),
		mcpapi.WithInputSchema[lootRmArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	lootUpdateTool := mcpapi.NewTool(lootUpdateToolName,
		mcpapi.WithDescription("Update a loot entry's metadata."),
		mcpapi.WithInputSchema[lootUpdateArgs](),
	)
	lootContentTool := mcpapi.NewTool(lootContentToolName,
		mcpapi.WithDescription("Get the content/data of a loot entry (returns base64)."),
		mcpapi.WithInputSchema[lootContentArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	lootAllTool := mcpapi.NewTool(lootAllToolName,
		mcpapi.WithDescription("List all loot entries."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	srv.server.AddTool(lootAddTool, srv.lootAddHandler)
	srv.server.AddTool(lootRmTool, srv.lootRmHandler)
	srv.server.AddTool(lootUpdateTool, srv.lootUpdateHandler)
	srv.server.AddTool(lootContentTool, srv.lootContentHandler)
	srv.server.AddTool(lootAllTool, srv.lootAllHandler)

	// Tier 4: Credentials
	credsListTool := mcpapi.NewTool(credsListToolName,
		mcpapi.WithDescription("List all stored credentials."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	credsAddTool := mcpapi.NewTool(credsAddToolName,
		mcpapi.WithDescription("Add a credential to the credential store."),
		mcpapi.WithInputSchema[credsAddArgs](),
	)
	credsRmTool := mcpapi.NewTool(credsRmToolName,
		mcpapi.WithDescription("Remove a credential from the store."),
		mcpapi.WithInputSchema[credsRmArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	credsUpdateTool := mcpapi.NewTool(credsUpdateToolName,
		mcpapi.WithDescription("Update a credential in the store."),
		mcpapi.WithInputSchema[credsUpdateArgs](),
	)
	credByIDTool := mcpapi.NewTool(credByIDToolName,
		mcpapi.WithDescription("Get a specific credential by ID."),
		mcpapi.WithInputSchema[credByIDArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	credsByHashTypeTool := mcpapi.NewTool(credsByHashTypeToolName,
		mcpapi.WithDescription("Get credentials filtered by hash type."),
		mcpapi.WithInputSchema[credsByHashTypeArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	credsSniffHashTypeTool := mcpapi.NewTool(credsSniffHashTypeToolName,
		mcpapi.WithDescription("Identify the hash type of a given hash string."),
		mcpapi.WithInputSchema[credsSniffHashTypeArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	srv.server.AddTool(credsListTool, srv.credsListHandler)
	srv.server.AddTool(credsAddTool, srv.credsAddHandler)
	srv.server.AddTool(credsRmTool, srv.credsRmHandler)
	srv.server.AddTool(credsUpdateTool, srv.credsUpdateHandler)
	srv.server.AddTool(credByIDTool, srv.credByIDHandler)
	srv.server.AddTool(credsByHashTypeTool, srv.credsByHashTypeHandler)
	srv.server.AddTool(credsSniffHashTypeTool, srv.credsSniffHashTypeHandler)

	// Tier 4: Hosts
	hostsListTool := mcpapi.NewTool(hostsListToolName,
		mcpapi.WithDescription("List all known hosts."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	hostInfoTool := mcpapi.NewTool(hostInfoToolName,
		mcpapi.WithDescription("Get detailed info about a specific host."),
		mcpapi.WithInputSchema[hostInfoArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	hostRmTool := mcpapi.NewTool(hostRmToolName,
		mcpapi.WithDescription("Remove a host record from the server."),
		mcpapi.WithInputSchema[hostRmArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	hostIOCRmTool := mcpapi.NewTool(hostIOCRmToolName,
		mcpapi.WithDescription("Remove an IOC record from a host."),
		mcpapi.WithInputSchema[hostIOCRmArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(hostsListTool, srv.hostsListHandler)
	srv.server.AddTool(hostInfoTool, srv.hostInfoHandler)
	srv.server.AddTool(hostRmTool, srv.hostRmHandler)
	srv.server.AddTool(hostIOCRmTool, srv.hostIOCRmHandler)

	// Tier 4: Listener additions
	restartJobsTool := mcpapi.NewTool(restartJobsToolName,
		mcpapi.WithDescription("Restart one or more jobs by their IDs."),
		mcpapi.WithInputSchema[restartJobsArgs](),
	)
	startTCPStagerTool := mcpapi.NewTool(startTCPStagerToolName,
		mcpapi.WithDescription("Start a TCP stager listener."),
		mcpapi.WithInputSchema[startTCPStagerArgs](),
	)
	srv.server.AddTool(restartJobsTool, srv.restartJobsHandler)
	srv.server.AddTool(startTCPStagerTool, srv.startTCPStagerHandler)

	// ===== Tier 5: Networking & Pivots =====

	// Tier 5: Pivots
	pivotStartListenerTool := mcpapi.NewTool(pivotStartListenerToolName,
		mcpapi.WithDescription("Start a pivot listener on the implant (TCP/UDP/NamedPipe)."),
		mcpapi.WithInputSchema[pivotStartListenerArgs](),
	)
	pivotStopListenerTool := mcpapi.NewTool(pivotStopListenerToolName,
		mcpapi.WithDescription("Stop a pivot listener."),
		mcpapi.WithInputSchema[pivotStopListenerArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	pivotSessionListenersTool := mcpapi.NewTool(pivotSessionListenersToolName,
		mcpapi.WithDescription("List pivot listeners on a session."),
		mcpapi.WithInputSchema[pivotSessionListenersArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	pivotGraphTool := mcpapi.NewTool(pivotGraphToolName,
		mcpapi.WithDescription("Get the pivot graph showing all pivot connections."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	srv.server.AddTool(pivotStartListenerTool, srv.pivotStartListenerHandler)
	srv.server.AddTool(pivotStopListenerTool, srv.pivotStopListenerHandler)
	srv.server.AddTool(pivotSessionListenersTool, srv.pivotSessionListenersHandler)
	srv.server.AddTool(pivotGraphTool, srv.pivotGraphHandler)

	// Tier 5: WireGuard networking
	wgStartPortForwardTool := mcpapi.NewTool(wgStartPortForwardToolName,
		mcpapi.WithDescription("Start a WireGuard TCP port forward on the implant."),
		mcpapi.WithInputSchema[wgStartPortForwardArgs](),
	)
	wgStopPortForwardTool := mcpapi.NewTool(wgStopPortForwardToolName,
		mcpapi.WithDescription("Stop a WireGuard port forward."),
		mcpapi.WithInputSchema[wgStopPortForwardArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	wgStartSocksTool := mcpapi.NewTool(wgStartSocksToolName,
		mcpapi.WithDescription("Start a WireGuard SOCKS proxy on the implant."),
		mcpapi.WithInputSchema[wgStartSocksArgs](),
	)
	wgStopSocksTool := mcpapi.NewTool(wgStopSocksToolName,
		mcpapi.WithDescription("Stop a WireGuard SOCKS proxy."),
		mcpapi.WithInputSchema[wgStopSocksArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	wgListForwardersTool := mcpapi.NewTool(wgListForwardersToolName,
		mcpapi.WithDescription("List active WireGuard TCP forwarders."),
		mcpapi.WithInputSchema[wgListForwardersArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	wgListSocksServersTool := mcpapi.NewTool(wgListSocksServersToolName,
		mcpapi.WithDescription("List active WireGuard SOCKS servers."),
		mcpapi.WithInputSchema[wgListSocksServersArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	srv.server.AddTool(wgStartPortForwardTool, srv.wgStartPortForwardHandler)
	srv.server.AddTool(wgStopPortForwardTool, srv.wgStopPortForwardHandler)
	srv.server.AddTool(wgStartSocksTool, srv.wgStartSocksHandler)
	srv.server.AddTool(wgStopSocksTool, srv.wgStopSocksHandler)
	srv.server.AddTool(wgListForwardersTool, srv.wgListForwardersHandler)
	srv.server.AddTool(wgListSocksServersTool, srv.wgListSocksServersHandler)

	// Tier 5: Reverse Port Forwarding
	startRportFwdTool := mcpapi.NewTool(startRportFwdToolName,
		mcpapi.WithDescription("Start a reverse port forward listener on the implant."),
		mcpapi.WithInputSchema[startRportFwdArgs](),
	)
	listRportFwdTool := mcpapi.NewTool(listRportFwdToolName,
		mcpapi.WithDescription("List active reverse port forward listeners."),
		mcpapi.WithInputSchema[listRportFwdArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	stopRportFwdTool := mcpapi.NewTool(stopRportFwdToolName,
		mcpapi.WithDescription("Stop a reverse port forward listener."),
		mcpapi.WithInputSchema[stopRportFwdArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(startRportFwdTool, srv.startRportFwdHandler)
	srv.server.AddTool(listRportFwdTool, srv.listRportFwdHandler)
	srv.server.AddTool(stopRportFwdTool, srv.stopRportFwdHandler)

	// Tier 5: SOCKS proxy (unary, session-level)
	createSocksTool := mcpapi.NewTool(createSocksToolName,
		mcpapi.WithDescription("Create a SOCKS5 proxy through a session."),
		mcpapi.WithInputSchema[createSocksArgs](),
	)
	closeSocksTool := mcpapi.NewTool(closeSocksToolName,
		mcpapi.WithDescription("Close a SOCKS5 proxy."),
		mcpapi.WithInputSchema[closeSocksArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(createSocksTool, srv.createSocksHandler)
	srv.server.AddTool(closeSocksTool, srv.closeSocksHandler)

	// ===== Tier 6: Generation & Websites =====

	// Tier 6: Implant generation
	generateImplantTool := mcpapi.NewTool(generateImplantToolName,
		mcpapi.WithDescription("Generate a new Sliver implant. Returns the implant file as base64."),
		mcpapi.WithInputSchema[generateImplantArgs](),
	)
	regenerateImplantTool := mcpapi.NewTool(regenerateImplantToolName,
		mcpapi.WithDescription("Regenerate an implant by its name."),
		mcpapi.WithInputSchema[regenerateImplantArgs](),
	)
	generateStageTool := mcpapi.NewTool(generateStageToolName,
		mcpapi.WithDescription("Generate a stage (shellcode) for an implant profile."),
		mcpapi.WithInputSchema[generateStageArgs](),
	)
	listImplantBuildsTool := mcpapi.NewTool(listImplantBuildsToolName,
		mcpapi.WithDescription("List all implant builds."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	deleteImplantBuildTool := mcpapi.NewTool(deleteImplantBuildToolName,
		mcpapi.WithDescription("Delete an implant build by name."),
		mcpapi.WithInputSchema[deleteImplantBuildArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	listImplantProfilesTool := mcpapi.NewTool(listImplantProfilesToolName,
		mcpapi.WithDescription("List all implant profiles."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	saveImplantProfileTool := mcpapi.NewTool(saveImplantProfileToolName,
		mcpapi.WithDescription("Save a new implant profile."),
		mcpapi.WithInputSchema[saveImplantProfileArgs](),
	)
	deleteImplantProfileTool := mcpapi.NewTool(deleteImplantProfileToolName,
		mcpapi.WithDescription("Delete an implant profile by name."),
		mcpapi.WithInputSchema[deleteImplantProfileArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	getCompilerTool := mcpapi.NewTool(getCompilerToolName,
		mcpapi.WithDescription("Get compiler information (cross-compilers, supported targets)."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	listCanariesTool := mcpapi.NewTool(listCanariesToolName,
		mcpapi.WithDescription("List DNS canaries."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	srv.server.AddTool(generateImplantTool, srv.generateImplantHandler)
	srv.server.AddTool(regenerateImplantTool, srv.regenerateImplantHandler)
	srv.server.AddTool(generateStageTool, srv.generateStageHandler)
	srv.server.AddTool(listImplantBuildsTool, srv.listImplantBuildsHandler)
	srv.server.AddTool(deleteImplantBuildTool, srv.deleteImplantBuildHandler)
	srv.server.AddTool(listImplantProfilesTool, srv.listImplantProfilesHandler)
	srv.server.AddTool(saveImplantProfileTool, srv.saveImplantProfileHandler)
	srv.server.AddTool(deleteImplantProfileTool, srv.deleteImplantProfileHandler)
	srv.server.AddTool(getCompilerTool, srv.getCompilerHandler)
	srv.server.AddTool(listCanariesTool, srv.listCanariesHandler)

	// Tier 6: Websites
	listWebsitesTool := mcpapi.NewTool(listWebsitesToolName,
		mcpapi.WithDescription("List all hosted websites."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	websiteInfoTool := mcpapi.NewTool(websiteInfoToolName,
		mcpapi.WithDescription("Get detailed info about a hosted website."),
		mcpapi.WithInputSchema[websiteInfoArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	websiteRemoveTool := mcpapi.NewTool(websiteRemoveToolName,
		mcpapi.WithDescription("Remove a hosted website."),
		mcpapi.WithInputSchema[websiteRemoveArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	websiteAddContentTool := mcpapi.NewTool(websiteAddContentToolName,
		mcpapi.WithDescription("Add content to a hosted website."),
		mcpapi.WithInputSchema[websiteAddContentArgs](),
	)
	websiteUpdateContentTool := mcpapi.NewTool(websiteUpdateContentToolName,
		mcpapi.WithDescription("Update content on a hosted website."),
		mcpapi.WithInputSchema[websiteUpdateContentArgs](),
	)
	websiteRemoveContentTool := mcpapi.NewTool(websiteRemoveContentToolName,
		mcpapi.WithDescription("Remove content from a hosted website."),
		mcpapi.WithInputSchema[websiteRemoveContentArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(listWebsitesTool, srv.listWebsitesHandler)
	srv.server.AddTool(websiteInfoTool, srv.websiteInfoHandler)
	srv.server.AddTool(websiteRemoveTool, srv.websiteRemoveHandler)
	srv.server.AddTool(websiteAddContentTool, srv.websiteAddContentHandler)
	srv.server.AddTool(websiteUpdateContentTool, srv.websiteUpdateContentHandler)
	srv.server.AddTool(websiteRemoveContentTool, srv.websiteRemoveContentHandler)

	// ===== Tier 7: Extensions & BOF Modules =====

	callExtensionTool := mcpapi.NewTool(callExtensionToolName,
		mcpapi.WithDescription("Call a loaded Sliver extension."),
		mcpapi.WithInputSchema[callExtensionArgs](),
	)
	listExtensionsTool := mcpapi.NewTool(listExtensionsToolName,
		mcpapi.WithDescription("List loaded extensions on the implant."),
		mcpapi.WithInputSchema[listExtensionsArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
	)
	registerExtensionTool := mcpapi.NewTool(registerExtensionToolName,
		mcpapi.WithDescription("Register (upload) an extension to the implant."),
		mcpapi.WithInputSchema[registerExtensionArgs](),
	)
	executeBOFTool := mcpapi.NewTool(executeBOFToolName,
		mcpapi.WithDescription("Execute a Beacon Object File (BOF) on the implant."),
		mcpapi.WithInputSchema[executeBOFArgs](),
	)
	listBOFModulesTool := mcpapi.NewTool(listBOFModulesToolName,
		mcpapi.WithDescription("List available BOF modules from the modules directory."),
		mcpapi.WithInputSchema[listBOFModulesArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	runBOFModuleTool := mcpapi.NewTool(runBOFModuleToolName,
		mcpapi.WithDescription("Run a BOF module by path (e.g. recon/whoami). Validates args against manifest."),
		mcpapi.WithInputSchema[runBOFModuleArgs](),
	)
	srv.server.AddTool(callExtensionTool, srv.callExtensionHandler)
	srv.server.AddTool(listExtensionsTool, srv.listExtensionsHandler)
	srv.server.AddTool(registerExtensionTool, srv.registerExtensionHandler)
	srv.server.AddTool(executeBOFTool, srv.executeBOFHandler)
	srv.server.AddTool(listBOFModulesTool, srv.listBOFModulesHandler)
	srv.server.AddTool(runBOFModuleTool, srv.runBOFModuleHandler)

	// Overlord suite tools
	listOverlordSuitesTool := mcpapi.NewTool(listOverlordSuitesToolName,
		mcpapi.WithDescription("List available Overlord BOF suites (recon, network, privesc, fullrecon, etc.)."),
		mcpapi.WithInputSchema[listOverlordSuitesArgs](),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	runOverlordSuiteTool := mcpapi.NewTool(runOverlordSuiteToolName,
		mcpapi.WithDescription("Run an Overlord BOF suite against a session/beacon. Executes all BOFs in the suite sequentially and returns structured output from each. Use list_overlord_suites to see available suites."),
		mcpapi.WithInputSchema[runOverlordSuiteArgs](),
	)
	srv.server.AddTool(listOverlordSuitesTool, srv.listOverlordSuitesHandler)
	srv.server.AddTool(runOverlordSuiteTool, srv.runOverlordSuiteHandler)

	// ===== Tier 8: Advanced =====

	// Tier 8: Shellcode tools
	shellcodeRDITool := mcpapi.NewTool(shellcodeRDIToolName,
		mcpapi.WithDescription("Convert a DLL to position-independent shellcode (sRDI)."),
		mcpapi.WithInputSchema[shellcodeRDIArgs](),
	)
	shellcodeEncodeTool := mcpapi.NewTool(shellcodeEncodeToolName,
		mcpapi.WithDescription("Encode shellcode with a specified encoder."),
		mcpapi.WithInputSchema[shellcodeEncodeArgs](),
	)
	listShellcodeEncodersTool := mcpapi.NewTool(listShellcodeEncodersToolName,
		mcpapi.WithDescription("List available shellcode encoders."),
		mcpapi.WithReadOnlyHintAnnotation(true),
		mcpapi.WithIdempotentHintAnnotation(true),
	)
	srv.server.AddTool(shellcodeRDITool, srv.shellcodeRDIHandler)
	srv.server.AddTool(shellcodeEncodeTool, srv.shellcodeEncodeHandler)
	srv.server.AddTool(listShellcodeEncodersTool, srv.listShellcodeEncodersHandler)

	// Tier 8: Execution additions
	backdoorTool := mcpapi.NewTool(backdoorToolName,
		mcpapi.WithDescription("Backdoor a binary on the target with a Sliver shellcode payload."),
		mcpapi.WithInputSchema[backdoorArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	hijackDllTool := mcpapi.NewTool(hijackDllToolName,
		mcpapi.WithDescription("Hijack a DLL on the target for persistence."),
		mcpapi.WithInputSchema[hijackDllArgs](),
		mcpapi.WithDestructiveHintAnnotation(true),
	)
	srv.server.AddTool(backdoorTool, srv.backdoorHandler)
	srv.server.AddTool(hijackDllTool, srv.hijackDllHandler)

	// Tier 8: Zergling integration
	generatePackedImplantTool := mcpapi.NewTool(generatePackedImplantToolName,
		mcpapi.WithDescription("Generate Sliver shellcode, pack it with zergling (Rust shellcode packer), and save the exe to disk. Returns the local output_path — use upload_file with local_path to deliver it. IMPORTANT: the 'method' field controls injection behavior. create_thread (default) runs shellcode in the loader's own process — the implant dies if the loader exits (e.g. SSH session closes, parent killed). Use early_bird or process_hollow to inject into a new process so the implant survives the loader exiting. All zergling manifest options (syscalls_indirect, etw_patch, amsi_patch, sleep_mask, evasion_noaccess, etc.) are exposed as individual fields, or pass a full manifest via manifest_json."),
		mcpapi.WithInputSchema[generatePackedImplantArgs](),
	)
	srv.server.AddTool(generatePackedImplantTool, srv.generatePackedImplantHandler)

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
