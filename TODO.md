# MCP Full Coverage — TODO

Branch: `feature/mcp-full-coverage`
Goal: Expand built-in MCP server to ~128 tools, accessible over HTTP to any multiplayer operator.

## Tier 0 — Refactor (prerequisite)
- [x] Extract shared helpers from `filesystem.go` into `helpers.go`
  - `buildRequest()`, `applyDefaultTimeout()`, `withTimeout()`
  - `waitForBeaconTask()`, `waitForBeaconTaskResponse()`, `newAsyncResult()`
  - `asyncTaskResult` struct, `defaultWaitTimeoutSeconds`, `pollInterval` consts
- [x] Verify existing 11 tools compile: `go build ./client/...`

## Tier 1 — Python sliver-mcp Parity (22 new → 33 total)
- [x] `server_version` — meta.go — GetVersion
- [x] `list_operators` — meta.go — GetOperators
- [x] `session_info` — sessions.go — GetSessions (filter by ID)
- [x] `kill_session` — sessions.go — Kill
- [x] `beacon_info` — sessions.go — GetBeacon
- [x] `kill_beacon` — sessions.go — RmBeacon
- [x] `list_beacon_tasks` — sessions.go — GetBeaconTasks
- [x] `cancel_beacon_task` — sessions.go — CancelBeaconTask
- [x] `list_jobs` — listeners.go — GetJobs
- [x] `start_mtls_listener` — listeners.go — StartMTLSListener
- [x] `start_https_listener` — listeners.go — StartHTTPSListener
- [x] `start_http_listener` — listeners.go — StartHTTPListener
- [x] `start_dns_listener` — listeners.go — StartDNSListener
- [x] `start_wg_listener` — listeners.go — StartWGListener
- [x] `kill_job` — listeners.go — KillJob
- [x] `execute` — execution.go — Execute (smart shell detection)
- [x] `download_file` — filesystem.go — Download
- [x] `upload_file` — filesystem.go — Upload
- [x] `list_processes` — processes.go — Ps
- [x] `network_interfaces` — network.go — Ifconfig
- [x] `network_connections` — network.go — Netstat
- [x] `get_environment` — environment.go — GetEnv
- [x] `registry_read` — registry.go — RegistryRead
- [x] `screenshot` — screenshot.go — Screenshot

## Tier 2 — Core Operations (22 new → 55 total)
- [x] `fs_grep` — filesystem.go — Grep
- [x] `fs_chtimes` — filesystem.go — Chtimes
- [x] `fs_mount` — filesystem.go — Mount
- [x] `fs_memfiles_list` — filesystem.go — MemfilesList
- [x] `fs_memfiles_add` — filesystem.go — MemfilesAdd
- [x] `fs_memfiles_rm` — filesystem.go — MemfilesRm
- [x] `terminate_process` — processes.go — Terminate
- [x] `process_dump` — processes.go — ProcessDump
- [x] `set_environment` — environment.go — SetEnv
- [x] `unset_environment` — environment.go — UnsetEnv
- [x] `execute_assembly` — execution.go — ExecuteAssembly
- [x] `sideload` — execution.go — Sideload
- [x] `spawn_dll` — execution.go — SpawnDll
- [x] `run_ssh_command` — execution.go — RunSSHCommand
- [x] `registry_write` — registry.go — RegistryWrite
- [x] `registry_create_key` — registry.go — RegistryCreateKey
- [x] `registry_delete_key` — registry.go — RegistryDeleteKey
- [x] `registry_list_subkeys` — registry.go — RegistryListSubKeys
- [x] `registry_list_values` — registry.go — RegistryListValues
- [x] `registry_read_hive` — registry.go — RegistryReadHive
- [x] `reconfigure_implant` — sessions.go — Reconfigure
- [x] `rename_implant` — sessions.go — Rename

## Tier 3 — Privilege and Services (13 new → 68 total)
- [x] `run_as` — privilege.go — RunAs
- [x] `impersonate` — privilege.go — Impersonate
- [x] `rev_to_self` — privilege.go — RevToSelf
- [x] `get_system` — privilege.go — GetSystem
- [x] `make_token` — privilege.go — MakeToken
- [x] `current_token_owner` — privilege.go — CurrentTokenOwner
- [x] `get_privs` — privilege.go — GetPrivs
- [x] `list_services` — services.go — Services
- [x] `service_detail` — services.go — ServiceDetail
- [x] `start_service_by_name` — services.go — StartServiceByName
- [x] `start_service` — services.go — StartService
- [x] `stop_service` — services.go — StopService
- [x] `remove_service` — services.go — RemoveService

## Tier 4 — Data Management (17 new → 85 total)
- [x] `loot_add` — loot.go — LootAdd
- [x] `loot_rm` — loot.go — LootRm
- [x] `loot_update` — loot.go — LootUpdate
- [x] `loot_content` — loot.go — LootContent
- [x] `loot_list` — loot.go — LootAll
- [x] `creds_list` — credentials.go — Creds
- [x] `creds_add` — credentials.go — CredsAdd
- [x] `creds_rm` — credentials.go — CredsRm
- [x] `creds_update` — credentials.go — CredsUpdate
- [x] `creds_get_by_id` — credentials.go — GetCredByID
- [x] `creds_get_by_hash_type` — credentials.go — GetCredsByHashType
- [x] `hosts_list` — hosts.go — Hosts
- [x] `host_info` — hosts.go — Host
- [x] `host_rm` — hosts.go — HostRm
- [x] `host_ioc_rm` — hosts.go — HostIOCRm
- [x] `restart_jobs` — listeners.go — RestartJobs
- [x] `start_tcp_stager` — listeners.go — StartTCPStagerListener

## Tier 5 — Networking and Pivots (13 new → 98 total)
- [x] `pivot_start_listener` — pivots.go — PivotStartListener
- [x] `pivot_stop_listener` — pivots.go — PivotStopListener
- [x] `pivot_session_listeners` — pivots.go — PivotSessionListeners
- [x] `pivot_graph` — pivots.go — PivotGraph
- [x] `start_rport_fwd` — portfwd.go — StartRportFwdListener
- [x] `list_rport_fwd` — portfwd.go — GetRportFwdListeners
- [x] `stop_rport_fwd` — portfwd.go — StopRportFwdListener
- [x] `wg_start_port_forward` — portfwd.go — WGStartPortForward
- [x] `wg_stop_port_forward` — portfwd.go — WGStopPortForward
- [x] `wg_start_socks` — portfwd.go — WGStartSocks
- [x] `wg_stop_socks` — portfwd.go — WGStopSocks
- [x] `wg_list_forwarders` — portfwd.go — WGListForwarders
- [x] `wg_list_socks_servers` — portfwd.go — WGListSocksServers

## Tier 6 — Generation and Websites (16 new → 114 total)
- [x] `generate_implant` — generate.go — Generate
- [x] `generate_stage` — generate.go — GenerateStage
- [x] `regenerate_implant` — generate.go — Regenerate
- [x] `list_implant_builds` — generate.go — ImplantBuilds
- [x] `delete_implant_build` — generate.go — DeleteImplantBuild
- [x] `list_implant_profiles` — generate.go — ImplantProfiles
- [x] `save_implant_profile` — generate.go — SaveImplantProfile
- [x] `delete_implant_profile` — generate.go — DeleteImplantProfile
- [x] `get_compiler` — generate.go — GetCompiler
- [x] `list_canaries` — generate.go — Canaries
- [x] `list_websites` — websites.go — Websites
- [x] `website_info` — websites.go — Website
- [x] `website_remove` — websites.go — WebsiteRemove
- [x] `website_add_content` — websites.go — WebsiteAddContent
- [x] `website_update_content` — websites.go — WebsiteUpdateContent
- [x] `website_remove_content` — websites.go — WebsiteRemoveContent

## Tier 7 — Extensions and BOF Modules (6 new → 120 total)
- [x] `call_extension` — extensions.go — CallExtension
- [x] `list_extensions` — extensions.go — ListExtensions
- [x] `register_extension` — extensions.go — RegisterExtension
- [x] `execute_bof` — extensions.go — CallExtension (IsBOF=true, BOFArgsBuffer)
- [x] `list_bof_modules` — extensions.go — Scan BOF_MODULES_DIR, read manifests
- [x] `run_bof_module` — extensions.go — Resolve module, validate args, delegate to execute_bof

## Tier 8 — Advanced (8 new → 128 total)
- [x] `generate_packed_implant` — integrations.go — Generate + zergling subprocess
- [x] `create_socks` — portfwd.go — CreateSocks
- [x] `close_socks` — portfwd.go — CloseSocks
- [x] `backdoor` — execution.go — Backdoor
- [x] `hijack_dll` — execution.go — HijackDLL
- [x] `shellcode_rdi` — generate.go — ShellcodeRDI
- [x] `shellcode_encode` — generate.go — ShellcodeEncoder
- [x] `list_shellcode_encoders` — generate.go — ShellcodeEncoderMap

## Skipped
- Crackstation (~21 methods) — niche, complex
- AI conversations (7) — wrapping AI-in-AI is pointless
- Bidirectional streams (SocksProxy, TunnelData, ClientLog) — incompatible with MCP
- Registration streams (BuilderRegister, CrackstationRegister) — infra, not operator tools
- Events stream — cannot map to tool call
- External builder flow, traffic encoders, C2 profiles, certificates, monitoring — infrastructure config
