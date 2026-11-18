# Sliver (delifer)

Custom distribution of [BishopFox/sliver](https://github.com/BishopFox/sliver) with **MCP integration** for AI-driven red team operations.

> **131+ tools** exposed via Model Context Protocol — run Sliver from Claude Code, Codex, or any MCP client.

[![Release](https://img.shields.io/github/v/release/SolaSec/sliver?label=release&style=flat-square)](https://github.com/SolaSec/sliver/releases) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg?style=flat-square)](https://www.gnu.org/licenses/gpl-3.0)

---

## Stack

| Component | Description | Repo |
|-----------|-------------|------|
| **Sliver** | C2 framework with full MCP tool coverage | [SolaSec/sliver](https://github.com/SolaSec/sliver) · [DELISEC/sliver](https://github.com/DELISEC/sliver) |
| **Zergling** | Rust shellcode packer — syscall-indirect, ETW/AMSI patch, sleep mask, multiple injection methods | [SolaSec/zergling](https://github.com/SolaSec/zergling) · [DELISEC/zergling](https://github.com/DELISEC/zergling) |
| **Overlord** | BOF toolkit — 50+ modules across recon, privesc, lateral movement, evasion, persistence, AD | [SolaSec/overlord](https://github.com/SolaSec/overlord) · [DELISEC/overlord](https://github.com/DELISEC/overlord) |

## What's Different

- **MCP server built into the client** — `sliver-client mcp` starts a stdio or HTTP MCP server with 131+ tools covering sessions, beacons, implant generation, file ops, network, registry, pivots, loot, credentials, extensions, BOFs, and more
- **Zergling integration** — generate packed implants via `generate_zergling_implant` with manifest-driven builds
- **Overlord suite runner** — run BOF recon/privesc/lateral suites via `run_overlord_suite`
- **BOF output fix** — WantBOFOutputs/Output field mismatch that silently dropped BeaconPrintf output
- **Beacon staleness checks** — MCP tools reject tasking beacons that missed 3x their check-in window
- **File path support** — MCP tools accept local paths instead of requiring base64
- **Linux ARM** — 32-bit ARM compiler target

## Quick Start

Download binaries from the [latest release](https://github.com/SolaSec/sliver/releases), or build from source:

```sh
git clone https://github.com/SolaSec/sliver.git
cd sliver && git checkout delifer
make          # server + client
make client   # client only (CGO-free)
```

Point your MCP client at the sliver-client binary:

```json
{
  "mcpServers": {
    "sliver": {
      "command": "./sliver-client",
      "args": [
        "mcp",
        "--config", "operator.cfg",
        "--zergling-path", "../zergling/target/release/launcher",
        "--overlord-dir", "../overlord"
      ]
    }
  }
}
```

See [**README.delifer.md**](README.delifer.md) for full setup — server config, zergling/overlord builds, prerequisites, and platform notes.

## Branching

| Branch | Purpose |
|--------|---------|
| `delifer` (default) | Integration branch — all custom features merged |
| `master` | Tracks upstream [BishopFox/sliver](https://github.com/BishopFox/sliver) |
| `feature/*`, `fix/*` | Individual changes, kept alive for potential upstream PRs |

## Upstream

For general Sliver documentation, see [sliver.sh](https://sliver.sh/) and the upstream [wiki](https://github.com/BishopFox/sliver/wiki).

## License

GPLv3 — same as upstream. See [LICENSE](https://www.gnu.org/licenses/gpl-3.0.en.html).
