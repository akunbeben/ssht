# ssht

[![Go Report Card](https://goreportcard.com/badge/github.com/akunbeben/ssht)](https://goreportcard.com/report/github.com/akunbeben/ssht)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

![ssht Banner](assets/banner.jpeg)

ssht is an SSH and per-connection VPN manager designed for speed, privacy, and network isolation. It replaces scattered `~/.ssh/config` entries and manual VPN toggling with a focused Terminal UI (TUI).

## Key Features

### Isolated VPN Sessions
Unlike standard VPN tools that change your global network settings, ssht tunnels only the SSH connections it starts.
- **WireGuard without sudo**: WireGuard runs in userspace using gVisor netstack. No system tunnel interface or global route changes are required.
- **Shared isolated WireGuard hub**: Multiple `ssht` instances using the same WireGuard config share one local userspace tunnel, so concurrent tmux panes work without breaking previous sessions.
- **Other proxy protocols**: ShadowSocks and Trojan are supported as per-connection proxy dialers.
- **OpenVPN note**: OpenVPN configs are recognized, but isolated OpenVPN dialing is not implemented; use system OpenVPN separately if needed.
- **Per-server override**: Set a specific VPN config for an individual server, overriding the profile-level VPN.

### Multi-Device Sync (Git-based)
Never lose your configuration again.
- **Git Backend**: Sync your profiles across multiple devices using any Git remote (GitHub, GitLab, etc.).
- **Modular Config**: Profiles are stored in separate files to minimize merge conflicts.
- **Auto-Sync**: Automatically pulls changes on startup and pushes on exit if enabled.
- **Secure**: Designed to work with Private Repositories to keep your host data safe.

### Privacy Masking
Perfect for streaming, recording, or live demos.
- Press `*` to mask sensitive hostnames, usernames, and VPN configs.
- The masking preference is saved, so ssht opens the same way next time.

### Smart Workflow
- Auto-Import: Scans your shell history (zsh, bash, fish) on startup to find your frequent servers.
- Host-key auto-repair: Detects `REMOTE HOST IDENTIFICATION HAS CHANGED`, removes the stale host key, and retries once.
- Persistent TUI: Returns to the server list after an SSH session exits.

### Pro Management
- Profiles: Group servers into contexts like `production`, `development`, or `personal`.
- Fuzzy search: Find servers by name, host, user, tag, or note.
- Key manager: Generate, list, and copy public keys with `K`.
- Move/migrate: Move servers between profiles from the TUI.

## Installation

### Shell One-liner (macOS/Linux)
```bash
curl -sL https://raw.githubusercontent.com/akunbeben/ssht/main/install.sh | sh
```

### Go Installer
```bash
go install github.com/akunbeben/ssht@latest
```

### Quick Build
```bash
make build
make install # Adds ssht to /usr/local/bin
```

### Download Binaries
Download the raw binary for your platform from the [Latest Releases](https://github.com/akunbeben/ssht/releases) page.
- macOS/Linux: chmod +x ssht && mv ssht /usr/local/bin/
- Windows: Add ssht.exe to your PATH.

## TUI Keybindings

| Key | Action |
|---|---|
| j/k, arrows | Navigate list |
| / | Fuzzy search |
| Enter | Connect to server |
| * | Toggle privacy masking |
| p | Switch Profile |
| a / c / e / d | Add / Copy / Edit / Delete server |
| m | Move server between profiles |
| v | Configure profile VPN |
| K | Pubkey Manager |
| ? | Help Overlay |
| q / Esc | Quit |

## Configuration & Sync

Settings are stored in `~/.ssht/`. Global settings live in `config.json`; each profile is stored separately in `profiles/{name}.json` to reduce Git merge conflicts.

### Multi-Device Synchronization
1. **Setup**: Create a private Git repository and link it:
   ```bash
   ssht sync setup git@github.com:username/my-ssht-config.git
   ```
2. **Push/Pull**: 
   ```bash
   ssht sync push  # Manual push
   ssht sync pull  # Manual pull
   ```
   Note: ssht automatically pulls on startup and pushes on exit when sync is enabled.

### VPN Configurations
ssht supports profile-level VPN config and per-server overrides. Press `v` in the TUI to configure the profile VPN, or set a server-specific VPN while adding/editing a server.

#### Supported VPN config values
- **WireGuard**: path to a `.conf` file, for example `~/vpn/wg0.conf`. Concurrent sessions using the same config share one local userspace hub.
- **ShadowSocks**: `ss://method:password@host:port`
- **Trojan**: `trojan://password@host:port?sni=server_name&allowInsecure=1`
- **OpenVPN**: path to `.ovpn` for system-level use; isolated OpenVPN ProxyCommand dialing is not currently implemented.

#### How isolated WireGuard works
When you connect to a server with a WireGuard VPN configured, ssht launches `ssh` with a hidden `ProxyCommand`. That proxy talks to a local hidden `ssht vpn-hub` process for the WireGuard config. The hub owns one userspace WireGuard device and opens TCP streams through it for each SSH session. System traffic is unaffected.

### Other Commands
```bash
# Check version and build info
ssht version

# Use a specific profile
ssht -p work

# Import manual history file
ssht server import-history --file ~/.zsh_history
```

## Development

```bash
make build   # Build with symbols stripped for small binary size
make run     # Build and launch immediately
make fmt     # Format source code
```

## Platform Support
- macOS: Native (Apple Silicon & Intel)
- Linux: Native
- Windows: Full support (uses Windows OpenSSH)

## License
This project is licensed under the [MIT License](LICENSE).

---
Built using Bubbletea, gVisor, and WireGuard.
