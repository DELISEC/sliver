# Sliver (delifer branch)

Custom distribution of [BishopFox/sliver](https://github.com/BishopFox/sliver) (based on v1.7.9) with MCP integration, zergling payload packing, and overlord BOF suites.

Maintained on the `delifer` branch at [DELISEC/sliver](https://github.com/DELISEC/sliver) and [SolaSec/sliver](https://github.com/SolaSec/sliver).

For general Sliver documentation, see the [upstream wiki](https://github.com/BishopFox/sliver/wiki).

---

## What's Different

| Feature | Description |
|---------|-------------|
| **131+ MCP tools** | Full Sliver API surface exposed via Model Context Protocol for AI-driven operations from Claude Code or Codex |
| **Zergling integration** | Rust shellcode packer with manifest-driven builds, indirect syscalls, ETW/AMSI patching, sleep masking, multiple injection methods |
| **Overlord suite runner** | Run BOF suites (recon, network, privesc, lateral, AD, evasion, persistence) with one MCP call |
| **BOF output fix** | Fixed WantBOFOutputs/Output field mismatch that silently dropped all BeaconPrintf output |
| **Beacon staleness checks** | MCP tools reject tasking beacons that missed 3x their check-in window |
| **File path support** | MCP tools accept local file paths (bof_path, file_path) instead of requiring base64 |
| **Loot crash fix** | Prevent nil dereference in LootAdd |
| **Linux ARM support** | 32-bit ARM added as a compiler target |

---

## Prerequisites

All platforms need:

| Dependency | Version | Purpose |
|-----------|---------|---------|
| Go | 1.27+ | Build Sliver server and client |
| make | any | Build orchestration |
| mingw-w64 | any | Cross-compile Windows implants and BOFs |
| Rust (nightly) | nightly-2026-11-14+ | Build zergling packer |
| Git | any | Clone repos |

### macOS (arm64)

```sh
brew install go mingw-w64 git
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
rustup toolchain install nightly
rustup target add x86_64-pc-windows-gnu --toolchain nightly
```

### Linux (amd64)

```sh
# Debian/Ubuntu
apt install golang gcc-mingw-w64 git make

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
rustup toolchain install nightly
rustup target add x86_64-pc-windows-gnu --toolchain nightly
```

---

## Setup

### 1. Clone the repos

```sh
# Sliver (this repo)
git clone https://github.com/DELISEC/sliver.git
cd sliver && git checkout delifer

# Zergling (shellcode packer)
git clone https://github.com/DELISEC/zergling.git

# Overlord (BOF suites)
git clone https://github.com/DELISEC/overlord.git
```

### 2. Build Sliver

```sh
cd sliver

# Server + client (CGO required for server)
make

# Client only (CGO-free, any GOOS/GOARCH)
make client
```

This produces `sliver-server` and `sliver-client` in the repo root.

### 3. Build zergling

Zergling has two components: the **launcher** (operator-side CLI that builds payloads) and **zerg_exe** (the Windows payload template).

```sh
cd zergling

# Build the launcher for your operator platform
cargo build --release --features launcher

# Build the Windows payload template (cross-compiled)
make
```

The launcher binary lands at `target/release/launcher` (Linux) or `target/aarch64-apple-darwin/release/launcher` (macOS arm64 with appropriate target). This is what `--zergling-path` points to.

### 4. Build overlord BOFs

```sh
cd overlord
make
```

This cross-compiles all BOFs into `dist/`. The repo root is what `--overlord-dir` points to. The `suites.json` manifest is included.

---

## Running the Server

```sh
./sliver-server
```

On first run, this initializes the database and creates the config directory at `~/.sliver-server/`.

From the Sliver console, generate an operator config for the client:

```
sliver > new-operator --name <operator-name> --lhost <server-ip>

[*] Generating new client certificate, please wait ...
[*] Saved new client config to /path/to/operator-name_server-ip.cfg
```

Start a listener:

```
sliver > mtls --lhost 0.0.0.0 --lport 8888
```

---

## Running the Client with MCP

The client runs as an MCP server, exposing all 131+ tools to Claude Code, Codex, or any MCP-compatible client.

```sh
./sliver-client mcp \
    --config /path/to/operator.cfg \
    --zergling-path /path/to/zergling/target/release/launcher \
    --overlord-dir /path/to/overlord
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--config` | Yes | Path to Sliver operator config (.cfg) |
| `--zergling-path` | No | Path to zergling launcher binary (enables `generate_zergling_implant`) |
| `--overlord-dir` | No | Path to overlord repo root (enables `run_overlord_suite`, `list_overlord_suites`) |
| `--bof-modules-dir` | No | Path to BOF modules directory (enables `run_bof_module`, `list_bof_modules`) |
| `--transport` | No | `stdio` (default), `http`, or `sse` |
| `--listen` | No | Listen address for HTTP/SSE (default `127.0.0.1:8080`) |

---

## Claude Code Integration

Create `.mcp.json` in your working directory:

```json
{
  "mcpServers": {
    "sliver": {
      "command": "/path/to/sliver-client",
      "args": [
        "mcp",
        "--config", "/path/to/operator.cfg",
        "--zergling-path", "/path/to/zergling/target/release/launcher",
        "--overlord-dir", "/path/to/overlord"
      ]
    }
  }
}
```

Then start Claude Code in that directory. The Sliver MCP tools will be available automatically.

### Codex Integration

Place the same MCP configuration in Codex's settings. The stdio transport works with any MCP-compatible client.

---

## MCP Tool Categories

| Category | Tools | Examples |
|----------|-------|---------|
| Sessions & Beacons | 6 | `list_sessions_and_beacons`, `session_info`, `kill_session` |
| Filesystem | 12 | `fs_ls`, `fs_cat`, `upload_file`, `download_file` |
| Execution | 7 | `execute`, `execute_assembly`, `execute_bof`, `sideload` |
| Network | 8 | `network_interfaces`, `network_connections`, `start_rport_fwd` |
| Processes | 4 | `list_processes`, `terminate_process`, `process_dump` |
| Privilege | 7 | `get_privs`, `impersonate`, `run_as`, `get_system` |
| Services | 5 | `list_services`, `start_service`, `remove_service` |
| Registry | 7 | `registry_read`, `registry_write`, `registry_list_subkeys` |
| Data (Loot/Creds/Hosts) | 14 | `loot_add`, `creds_list`, `hosts_list` |
| Pivots | 5 | `pivot_graph`, `pivot_start_listener` |
| Generation | 5 | `generate_implant`, `generate_zergling_implant`, `generate_stage` |
| Extensions & BOFs | 6 | `execute_bof`, `run_bof_module`, `call_extension` |
| Overlord Suites | 2 | `run_overlord_suite`, `list_overlord_suites` |
| Listeners | 7 | `start_mtls_listener`, `start_https_listener`, `list_jobs` |
| Advanced | 6 | `shellcode_rdi`, `shellcode_encode`, `backdoor`, `hijack_dll` |
| Environment | 3 | `get_environment`, `set_environment` |
| Screenshots | 1 | `screenshot` |
| Websites | 5 | `list_websites`, `website_add_content` |
| WireGuard | 4 | `wg_start_socks`, `wg_start_port_forward` |
| SSH | 1 | `run_ssh_command` |

### Overlord Suites

Available via `list_overlord_suites` / `run_overlord_suite`:

| Suite | BOFs | Description |
|-------|------|-------------|
| recon | 7 | User, system, env, defender, processes |
| network | 4 | Adapters, netstat, ARP, DNS cache |
| privesc | 3 | Potato, UAC, unquoted paths |
| services | 4 | Services, tasks, autoruns, firewall |
| software | 4 | Installed software, .NET, PowerShell |
| credentials | 5 | Cred stores, SAM, WiFi, interesting files |
| tokens | 5 | Token enum, certs, clipboard, cloud, event logs |
| ad | 6 | Domain info, users, groups, trusts, SPNs, computers |
| lateral | 3 | WinRM, WMI, PsExec |
| evasion | 3 | AMSI patch, ETW patch, ntdll unhook |
| persistence | 4 | Run key, startup, service, schtask |
| full | 16 | recon + network + privesc + shares |
| fullrecon | 36 | All recon/network/privesc/cred/infra BOFs |

---

## Branching Strategy

```
upstream/master (BishopFox/sliver)
    |
    v
  master ---- tracks upstream, periodic merge
    |
    |--- feature/mcp-full-coverage  (PR-ready for upstream)
    |--- feature/linux-arm-support  (PR-ready for upstream)
    |--- fix/getsystem-db-save      (PR-ready for upstream)
    |
    v
  delifer ---- integration branch, all features merged
    |
    v
  tags: delifer-v1.7.9-1, delifer-v1.7.9-2, ...
```

- **`master`** syncs with `upstream/master` via fetch + merge
- **Feature branches** stay alive and clean for potential upstream PRs
- **`delifer`** merges all feature branches; this is what you run
- **Tags** on `delifer` mark releases

To update from upstream:

```sh
git fetch upstream
git checkout master
git merge upstream/master
git checkout delifer
git merge master
```

---

## License

Same as upstream Sliver: [GPLv3](https://github.com/BishopFox/sliver/blob/master/LICENSE).
