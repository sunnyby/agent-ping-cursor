# Agent Ping for Cursor

A free command-line utility that plays a macOS sound when Cursor Agent successfully finishes a response. Based on [Agent Ping for Codex](https://github.com/sunnyby/agent-ping).

**macOS 13+ · Apple Silicon and Intel · MIT**

[Инструкция на русском](README.ru.md)

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/sunnyby/agent-ping-cursor/main/install.sh | sh
```

The installer downloads the appropriate binary from GitHub Releases, verifies its SHA-256 checksum, installs it under `~/.local/bin`, and adds the Agent Ping hook to `~/.cursor/hooks.json`. The release binary runs without a Go installation.

Play the default sound immediately to verify your Mac's audio output.

```sh
~/.local/bin/agent-ping-cursor test
```

Send Cursor Agent a short prompt and wait for its reply to verify the hook. Cursor normally reloads hooks automatically. If the hook does not appear, check **Customize → Hooks** and restart Cursor. See [Cursor Hooks](https://cursor.com/docs/hooks#troubleshooting).

To inspect the installer first, download it with `curl -fsSLO` and open `install.sh` before running `sh install.sh`.

## Commands

```sh
~/.local/bin/agent-ping-cursor status
~/.local/bin/agent-ping-cursor test --sound Glass
~/.local/bin/agent-ping-cursor install --sound Glass
~/.local/bin/agent-ping-cursor uninstall
```

`install` connects Agent Ping or updates its sound. The default is `Ping`. `uninstall` removes the managed hook and preserves other hooks and JSON settings. To remove the binary as well, run `rm ~/.local/bin/agent-ping-cursor` after uninstalling.

Pass `--cursor-dir PATH` to `install`, `status`, or `uninstall` to use a different directory containing `hooks.json`. `AGENT_PING_CURSOR_BIN_DIR` selects the installer destination and `AGENT_PING_CURSOR_VERSION` selects a release. To select the sound during installation, pass `--sound Glass` using `sh -s -- --sound Glass` at the end of the installation pipeline.

## Behavior

The utility uses Cursor's [user-level `stop` hook](https://cursor.com/docs/hooks#stop). A `completed` event starts playback on the local Mac. The hook returns an empty response and processes the event in memory. It does not record conversation data or require a running background service. Aborted and failed runs are silent. User-level hooks apply to local Cursor sessions.

Agent Ping merges its entry into a standard JSON version 1 hook file and preserves other entries. The file is formatted as JSON on changes. Invalid JSON and symlinked hook files require manual configuration.

## Build

```sh
go test -race ./...
go vet ./...
go build -o agent-ping-cursor .
sh scripts/release.sh v0.1.1
```

The release workflow tests the source, builds both Mac architectures, and publishes archives and checksums from a version tag. See [LICENSE](LICENSE) for the MIT terms.
