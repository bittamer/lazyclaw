# lazyclaw SRS Amendments

Amendments to the Design Document & Software Requirements Specification (v1.0, February 2026).

## Amendment 1: CLI-First Architecture

**Date:** February 2026
**Supersedes:** SRS sections 4.2, 5.1, 5.2; Design sections NET-1, NET-2, NET-3; FR-CN-1, FR-CN-3

### Change Summary

The transport layer has been changed from **WebSocket-first with CLI fallback** to
**CLI-first**. All gateway communication is performed by executing `openclaw` CLI
commands locally or on remote hosts via SSH.

### Rationale

The CLI adapter provides sufficient data fidelity for v1 monitoring needs via
`openclaw status --json`, `openclaw health --json`, and `openclaw logs --follow`.
This simplifies the codebase, eliminates the need for WebSocket handshake/auth
implementation, and leverages the existing CLI's mature JSON output.

### Affected Requirements

| Original ID | Original Requirement | New Status |
|-------------|---------------------|------------|
| NET-1 | Connect via WebSocket text frames | **Replaced** by CLI adapter (`openclaw` command execution) |
| NET-2 | Gateway WS handshake (connect.challenge) | **Removed** — not applicable to CLI mode |
| NET-3 | Token auth via connect.params.auth.token | **Removed** — SSH key auth handles remote access |
| FR-CN-1 | WS connection with configurable timeout | **Replaced** — CLI command timeout + SSH ConnectTimeout |
| FR-CN-3 | Show handshake/auth failures clearly | **Replaced** — CLI/SSH error messages shown in UI |

### Connection Modes (Updated)

| Mode | Transport | Notes |
|------|-----------|-------|
| `local` | `openclaw <command>` executed locally | Default. Uses openclaw from PATH or configured path. |
| `ssh` | `ssh <host> openclaw <command>` | For remote gateways. Uses SSH key auth, supports proxy jump. |

The `direct` and `ssh_tunnel` modes from the original SRS are removed. The `ssh`
mode executes commands directly on the remote host rather than creating a local
tunnel.

### Data Sources (Updated)

| Data | CLI Command | Usage |
|------|------------|-------|
| Full status | `openclaw status --json` | Overview, Sessions, Agents, Channels, Memory, Security, System tabs |
| Health snapshot | `openclaw health --json` | Health tab |
| Live logs | `openclaw logs --follow` | Logs tab, Events tab |

## Amendment 2: Tab Layout Extension

**Date:** February 2026
**Supersedes:** SRS section 6.1.2

### Change Summary

The tab layout has been extended from 7 tabs to 10 tabs. The original 7 SRS tabs
are retained as tabs 1-7 (accessible via number keys). Three additional tabs are
added for Memory, Security, and System information (accessible via keys 8, 9, 0).

### Updated Tab Layout

| Key | Tab | Content |
|-----|-----|---------|
| 1 | Overview | Reachability, service status, sessions, agents, security summary, channels, memory, recent activity |
| 2 | Logs | Follow/tail with search and level filters |
| 3 | Health | Parsed gateway health snapshot (probe durations, per-channel health) |
| 4 | Channels | Channel readiness, auth age, link status |
| 5 | Agents | Configured agents, workspace path, active turn indicator, heartbeat |
| 6 | Sessions | Active sessions, token usage with progress bars |
| 7 | Events | Filtered system events feed (errors, state changes, connections) |
| 8 | Memory | RAG/vector search system details, source counts, features |
| 9 | Security | Security audit findings with severity badges |
| 0 | System | Gateway info, services, OS, update status |
