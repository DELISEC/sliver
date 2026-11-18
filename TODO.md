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
- [ ] `server_version` — meta.go — GetVersion
- [ ] `list_operators` — meta.go — GetOperators
- [ ] `session_info` — sessions.go — GetSessions (filter by ID)
- [ ] `kill_session` — sessions.go — Kill
- [ ] `beacon_info` — sessions.go — GetBeacon
- [ ] `kill_beacon` — sessions.go — RmBeacon
- [ ] `list_beacon_tasks` — sessions.go — GetBeaconTasks
- [ ] `cancel_beacon_task` — sessions.go — CancelBeaconTask
- [ ] `list_jobs` — listeners.go — GetJobs
- [ ] `start_mtls_listener` — listeners.go — StartMTLSListener
- [ ] `start_https_listener` — listeners.go — StartHTTPSListener
- [ ] `start_http_listener` — listeners.go — StartHTTPListener
- [ ] `start_dns_listener` — listeners.go — StartDNSListener
- [ ] `start_wg_listener` — listeners.go — StartWGListener
- [ ] `kill_job` — listeners.go — KillJob
- [ ] `execute` — execution.go — Execute (smart shell detection)
- [ ] `download_file` — filesystem.go — Download
- [ ] `upload_file` — filesystem.go — Upload
- [ ] `list_processes` — processes.go — Ps
- [ ] `network_interfaces` — network.go — Ifconfig
- [ ] `network_connections` — network.go — Netstat
- [ ] `get_environment` — environment.go — GetEnv
- [ ] `registry_read` — registry.go — RegistryRead
- [ ] `screenshot` — screenshot.go — Screenshot

## Tier 2 — Core Operations (22 new → 55 total)
- [ ] `fs_grep` — filesystem.go — Grep
- [ ] `fs_chtimes` — filesystem.go — Chtimes
- [ ] `fs_mount` — filesystem.go — Mount
- [ ] `fs_memfiles_list` — filesystem.go — MemfilesList
- [ ] `fs_memfiles_add` — filesystem.go — MemfilesAdd
- [ ] `fs_memfiles_rm` — filesystem.go — MemfilesRm
- [ ] `terminate_process` — processes.go — Terminate
- [ ] `process_dump` — processes.go — ProcessDump
- [ ] `set_environment` — environment.go — SetEnv
- [ ] `unset_environment` — environment.go — UnsetEnv
- [ ] `execute_assembly` — execution.go — ExecuteAssembly
- [ ] `sideload` — execution.go — Sideload
- [ ] `spawn_dll` — execution.go — SpawnDll
- [ ] `run_ssh_command` — execution.go — RunSSHCommand
- [ ] `registry_write` — registry.go — RegistryWrite
- [ ] `registry_create_key` — registry.go — RegistryCreateKey
- [ ] `registry_delete_key` — registry.go — RegistryDeleteKey
- [ ] `registry_list_subkeys` — registry.go — RegistryListSubKeys
- [ ] `registry_list_values` — registry.go — RegistryListValues
- [ ] `registry_read_hive` — registry.go — RegistryReadHive
- [ ] `reconfigure_implant` — sessions.go — Reconfigure
- [ ] `rename_implant` — sessions.go — Rename

## Tier 3 — Privilege and Services (13 new → 68 total)
- [ ] `run_as` — privilege.go — RunAs
- [ ] `impersonate` — privilege.go — Impersonate
- [ ] `rev_to_self` — privilege.go — RevToSelf
- [ ] `get_system` — privilege.go — GetSystem
- [ ] `make_token` — privilege.go — MakeToken
- [ ] `current_token_owner` — privilege.go — CurrentTokenOwner
- [ ] `get_privs` — privilege.go — GetPrivs
- [ ] `list_services` — services.go — Services
- [ ] `service_detail` — services.go — ServiceDetail
- [ ] `start_service_by_name` — services.go — StartServiceByName
- [ ] `start_service` — services.go — StartService
- [ ] `stop_service` — services.go — StopService
- [ ] `remove_service` — services.go — RemoveService

## Tier 4 — Data Management (17 new → 85 total)
- [ ] `loot_add` — loot.go — LootAdd
- [ ] `loot_rm` — loot.go — LootRm
- [ ] `loot_update` — loot.go — LootUpdate
- [ ] `loot_content` — loot.go — LootContent
- [ ] `loot_list` — loot.go — LootAll
- [ ] `creds_list` — credentials.go — Creds
- [ ] `creds_add` — credentials.go — CredsAdd
- [ ] `creds_rm` — credentials.go — CredsRm
- [ ] `creds_update` — credentials.go — CredsUpdate
- [ ] `creds_get_by_id` — credentials.go — GetCredByID
- [ ] `creds_get_by_hash_type` — credentials.go — GetCredsByHashType
- [ ] `hosts_list` — hosts.go — Hosts
- [ ] `host_info` — hosts.go — Host
- [ ] `host_rm` — hosts.go — HostRm
- [ ] `host_ioc_rm` — hosts.go — HostIOCRm
- [ ] `restart_jobs` — listeners.go — RestartJobs
- [ ] `start_tcp_stager` — listeners.go — StartTCPStagerListener

## Tier 5 — Networking and Pivots (13 new → 98 total)
- [ ] `pivot_start_listener` — pivots.go — PivotStartListener
- [ ] `pivot_stop_listener` — pivots.go — PivotStopListener
- [ ] `pivot_session_listeners` — pivots.go — PivotSessionListeners
- [ ] `pivot_graph` — pivots.go — PivotGraph
- [ ] `start_rport_fwd` — portfwd.go — StartRportFwdListener
- [ ] `list_rport_fwd` — portfwd.go — GetRportFwdListeners
- [ ] `stop_rport_fwd` — portfwd.go — StopRportFwdListener
- [ ] `wg_start_port_forward` — portfwd.go — WGStartPortForward
- [ ] `wg_stop_port_forward` — portfwd.go — WGStopPortForward
- [ ] `wg_start_socks` — portfwd.go — WGStartSocks
- [ ] `wg_stop_socks` — portfwd.go — WGStopSocks
- [ ] `wg_list_forwarders` — portfwd.go — WGListForwarders
- [ ] `wg_list_socks_servers` — portfwd.go — WGListSocksServers

## Tier 6 — Generation and Websites (16 new → 114 total)
- [ ] `generate_implant` — generate.go — Generate
- [ ] `generate_stage` — generate.go — GenerateStage
- [ ] `regenerate_implant` — generate.go — Regenerate
- [ ] `list_implant_builds` — generate.go — ImplantBuilds
- [ ] `delete_implant_build` — generate.go — DeleteImplantBuild
- [ ] `list_implant_profiles` — generate.go — ImplantProfiles
- [ ] `save_implant_profile` — generate.go — SaveImplantProfile
- [ ] `delete_implant_profile` — generate.go — DeleteImplantProfile
- [ ] `get_compiler` — generate.go — GetCompiler
- [ ] `list_canaries` — generate.go — Canaries
- [ ] `list_websites` — websites.go — Websites
- [ ] `website_info` — websites.go — Website
- [ ] `website_remove` — websites.go — WebsiteRemove
- [ ] `website_add_content` — websites.go — WebsiteAddContent
- [ ] `website_update_content` — websites.go — WebsiteUpdateContent
- [ ] `website_remove_content` — websites.go — WebsiteRemoveContent

## Tier 7 — Extensions and BOF Modules (6 new → 120 total)
- [ ] `call_extension` — extensions.go — CallExtension
- [ ] `list_extensions` — extensions.go — ListExtensions
- [ ] `register_extension` — extensions.go — RegisterExtension
- [ ] `execute_bof` — extensions.go — CallExtension (IsBOF=true, BOFArgsBuffer)
- [ ] `list_bof_modules` — extensions.go — Scan BOF_MODULES_DIR, read manifests
- [ ] `run_bof_module` — extensions.go — Resolve module, validate args, delegate to execute_bof

## Tier 8 — Advanced (8 new → 128 total)
- [ ] `generate_packed_implant` — integrations.go — Generate + zergling subprocess
- [ ] `create_socks` — portfwd.go — CreateSocks
- [ ] `close_socks` — portfwd.go — CloseSocks
- [ ] `backdoor` — execution.go — Backdoor
- [ ] `hijack_dll` — execution.go — HijackDLL
- [ ] `shellcode_rdi` — generate.go — ShellcodeRDI
- [ ] `shellcode_encode` — generate.go — ShellcodeEncoder
- [ ] `list_shellcode_encoders` — generate.go — ShellcodeEncoderMap

## Skipped
- Crackstation (~21 methods) — niche, complex
- AI conversations (7) — wrapping AI-in-AI is pointless
- Bidirectional streams (SocksProxy, TunnelData, ClientLog) — incompatible with MCP
- Registration streams (BuilderRegister, CrackstationRegister) — infra, not operator tools
- Events stream — cannot map to tool call
- External builder flow, traffic encoders, C2 profiles, certificates, monitoring — infrastructure config
