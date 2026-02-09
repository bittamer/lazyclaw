# lazyclaw

A terminal UI for monitoring OpenClaw Gateways, inspired by lazygit and lazydocker.

## Features

- **Single-screen monitoring**: View gateway status, logs, and health at a glance
- **Keyboard-driven**: Full keyboard navigation with vim-style bindings
- **Real-time updates**: Live CLI polling of OpenClaw Gateway status
- **Configuration persistence**: Remembers your preferences and UI state
- **Multi-instance**: Monitor local and remote gateways (via SSH)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/lazyclaw/lazyclaw.git
cd lazyclaw

# Build
go build -o lazyclaw ./cmd/lazyclaw

# Install to PATH (optional)
sudo mv lazyclaw /usr/local/bin/
```

### Requirements

- Go 1.22 or later
- An OpenClaw Gateway running locally or remotely
- `openclaw` CLI installed (locally or on the remote host)

## Quick Start

1. **Run lazyclaw**:
   ```bash
   ./lazyclaw
   ```

2. **Mock mode** (for testing without a running gateway):
   ```bash
   ./lazyclaw --mock
   ```

3. **Navigate** using keyboard shortcuts (press `?` for help).

## Keybindings

| Key | Action |
|-----|--------|
| `q` | Quit |
| `?` | Show help |
| `/` | Search/filter |
| `Tab` | Switch between panes |
| `1-7` | Switch tabs (Overview, Logs, Health, Channels, Agents, Sessions, Events) |
| `8/9/0` | Extra tabs (Memory, Security, System) |
| `f` | Toggle log follow mode |
| `r` | Reconnect to gateway |
| `j/k` or arrows | Navigate lists |

## Tabs

| # | Tab | Content |
|---|-----|---------|
| 1 | Overview | Quick status, gateway, channels, sessions, security summary |
| 2 | Logs | Live log streaming with follow mode and level filters |
| 3 | Health | Gateway health snapshot with probe durations |
| 4 | Channels | Channel readiness, auth age, link status |
| 5 | Agents | Configured agents, workspace, activity |
| 6 | Sessions | Active sessions with token usage indicators |
| 7 | Events | Filtered system events feed (errors, state changes) |
| 8 | Memory | RAG/vector search system details |
| 9 | Security | Security audit findings |
| 0 | System | Services, OS, update status |

## Configuration

Configuration is stored in `~/.config/lazyclaw/config.yml`.

See [config.example.yml](config.example.yml) for a full example.

### Basic Configuration

```yaml
instances:
  - name: "local"
    mode: "local"

ui:
  refresh_ms: 1000
  log_tail_lines: 500

security:
  default_scopes:
    - "operator.read"
```

### Connection Modes

| Mode | Description |
|------|-------------|
| `local` | Run `openclaw` CLI locally (default) |
| `ssh` | Run `openclaw` CLI on a remote host via SSH |

## Architecture

lazyclaw uses a **CLI-first** architecture. It gathers data by executing
`openclaw status --json`, `openclaw health --json`, and `openclaw logs --follow`
either locally or on remote hosts via SSH.

```
lazyclaw/
├── cmd/lazyclaw/       # Entry point
├── internal/
│   ├── config/         # Configuration loading/saving
│   ├── gateway/        # CLI adapter for OpenClaw (local + SSH)
│   ├── models/         # Domain types
│   ├── state/          # UI state persistence
│   └── ui/             # Bubble Tea TUI components
│       ├── keys/       # Keybindings
│       ├── styles/     # Lipgloss styles
│       └── views/      # Tab views
└── docs/               # Design documentation
```

## Development

```bash
# Run in development
go run ./cmd/lazyclaw

# Run with mock data
go run ./cmd/lazyclaw --mock

# Build
go build -o lazyclaw ./cmd/lazyclaw

# Run tests
go test ./...
```

## License

GPL-3.0
