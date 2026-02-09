# lazyclaw

A terminal UI for monitoring OpenClaw Gateways, inspired by lazygit and lazydocker.

## Features

- **Single-screen monitoring**: View gateway status, logs, and health at a glance
- **Keyboard-driven**: Full keyboard navigation with vim-style bindings
- **Real-time updates**: Live WebSocket connection to OpenClaw Gateway
- **Configuration persistence**: Remembers your preferences and UI state
- **First-run wizard**: Easy setup for new users

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

## Quick Start

1. **Run lazyclaw**:
   ```bash
   ./lazyclaw
   ```

2. **First run**: The setup wizard will guide you through configuring your first instance.

3. **Navigate** using keyboard shortcuts (press `?` for help).

## Keybindings

| Key | Action |
|-----|--------|
| `q` | Quit |
| `?` | Show help |
| `/` | Search/filter |
| `Tab` | Switch between panes |
| `1-2` | Switch tabs (Overview, Logs) |
| `f` | Toggle log follow mode |
| `r` | Reconnect to gateway |
| `j/k` or arrows | Navigate lists |

## Configuration

Configuration is stored in `~/.config/lazyclaw/config.yml`.

See [config.example.yml](config.example.yml) for a full example.

### Basic Configuration

```yaml
instances:
  - name: "local"
    mode: "local"
    ws_url: "ws://127.0.0.1:18789"

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
| `local` | Connect to localhost (auto-approved by OpenClaw) |
| `ssh_tunnel` | Connect via SSH port forward (recommended for remote) |
| `direct` | Direct remote connection (requires device pairing) |

## Architecture

```
lazyclaw/
├── cmd/lazyclaw/       # Entry point
├── internal/
│   ├── config/         # Configuration loading/saving
│   ├── gateway/        # WebSocket client for OpenClaw
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

# Build
go build -o lazyclaw ./cmd/lazyclaw

# Run tests
go test ./...
```

## License

MIT
