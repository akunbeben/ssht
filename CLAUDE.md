# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build            # Build binary (strips symbols, injects version/commit/date via LDFLAGS)
make run              # Build and run immediately
make fmt              # Format source (go fmt)
make tidy             # Tidy go.mod/go.sum
make clean            # Remove binary and temp files
make install          # Install to /usr/local/bin (requires sudo)
make release-snapshot # Dry-run GoReleaser release build
```

No test suite beyond `internal/vpn/shadowsocks_test.go`. Run a single test with `go test ./internal/vpn/ -run TestName`.

## Architecture

**ssht** is a Go SSH & VPN manager with a Bubbletea TUI and Cobra CLI.

### Layers

- **`main.go`** — Entry point. Triggers auto-sync (git pull before / git push after) when sync is enabled, then delegates to `cmd.Execute()`.
- **`cmd/`** — Cobra commands. `root.go` launches the TUI by default. `vpn.go` exposes a hidden `vpn-dial` subcommand used as an SSH `ProxyCommand` for isolated per-session VPN tunneling. `server_import.go` parses zsh/bash/fish history to auto-populate server lists.
- **`internal/config/`** — JSON config stored in `~/.ssht/`. Global state in `config.json`; each profile is a separate file in `profiles/{name}.json` (intentional: reduces Git merge conflicts during sync).
- **`internal/tui/`** — Bubbletea model in `tui.go` dispatches to sub-views (form, list, move, search, profile_switch, vpn, pubkey, delete). Views are rendered via a central `View()` switch on `tui.currentView`.
- **`internal/vpn/`** — Multi-protocol VPN dialers: WireGuard (via gVisor userspace netstack), ShadowSocks, Trojan, OpenVPN. The WireGuard dialer runs entirely in userspace — no root required, no system-level tunnel interface. `dialer.go` selects protocol based on `VPNConf.Type`.
- **`internal/ssh/`** — Builds `ssh` args and `exec`s the system `ssh` binary. Injects a `ProxyCommand` pointing to `ssht vpn-dial` when a VPN is configured. Server-level VPN overrides profile-level VPN. Also handles host key mismatch auto-repair.
- **`internal/key/`** — Scans `~/.ssh/` for public keys, generates ed25519 keys, copies to clipboard.

### VPN Isolation Flow

```
TUI: Connect with VPN
  → ssh.Connect() builds: ssh -o ProxyCommand="ssht vpn-dial ..." host
  → system ssh calls ProxyCommand
  → vpn-dial: vpn.Dial() opens TCP through gVisor netstack (WireGuard/ShadowSocks/Trojan)
  → forwards stdin/stdout over that tunnel
  → ssh traffic travels inside VPN, rest of system is unaffected
```

### Config Schema

`~/.ssht/config.json` — global (last profile, privacy mode, sync settings).
`~/.ssht/profiles/{name}.json` — per profile with `servers[]`, `vpn{}`, and `import_exceptions[]`.

### Key TUI Keybindings

`/` search, `Enter` connect, `a/c/e/d` add/copy/edit/delete, `m` move, `p` profile switch, `v` VPN toggle, `K` pubkey manager, `*` privacy mask, `?` help.

## Release

GoReleaser (`.goreleaser.yaml`) builds for Linux/Darwin/Windows on amd64/arm64. The build injects `version`, `commit`, `date`, and `builder` via `ldflags`.
