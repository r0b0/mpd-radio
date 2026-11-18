# MPD Radio — Agent Guidelines

## Project Overview

**MPD Radio** is a lightweight Go application that provides a web-based interface for controlling Music Player Daemon (MPD) servers to play internet radio streams. It supports multiple MPD players, radio station management, and volume control via a responsive web UI or a REST-like API.

- **Language**: Go 1.27+
- **Module**: `mpd_radio`
- **Version**: 26.09
- **Default port**: `127.0.0.1:6680`
- **Dependencies**: `golang.org/x/sys` (Windows registry support only)

## Architecture

The application is a single-package (`main`) Go program with a clean separation of concerns across three layers:

### 1. MPD Client Layer (`mpd-client.go`)

Implements a TCP client for the [MPD protocol](https://www.musicpd.org/protocol/). Key responsibilities:

- **Connection management**: TCP dial, auto-reconnect on disconnect, idle timeout (60s disconnect, 30s select timeout).
- **Command/response parsing**: Parses MPD's text-based protocol including key-value pairs, `OK`/`ACK` lines, and binary data (up to 1 MB).
- **Goroutine-based serve loop**: Each `MpdClient` spawns a `Serve()` goroutine that processes commands from a channel, handles ping-based keepalive, and manages connection lifecycle.
- **Thread-safe commands**: Commands are sent via a `chan CommandStruct` and responses are returned via per-command response channels.

Key types:
- `MpdClient` — TCP connection state, command channel, logger.
- `MpdData` — structured command response with `Response map[string]string`, `Binary []byte`, `Error`.

### 2. Application Layer (`application.go`)

Central state manager and business logic:

- **State**: `Application` struct holds the player list, radio list, current status, volume, and selected indices.
- **Player operations**: `Play()`, `Stop()`, `Pause()`, `UpdateVolume()` — all delegate to MPD commands (`clear`, `addid`, `playid`, `stop`, `pause`, `setvol`).
- **Status polling**: `UpdateStatus()` fetches `status` and `currentsong` from MPD, with a 10-second cooldown to avoid excessive polling.
- **Persistence**: `Store()` serializes the entire `Application` struct to JSON via platform-specific `saveConfig()`.
- **Load/Store**: `Load()` deserializes config on startup; `Store()` persists after any mutation (add/remove player or radio).

### 3. HTTP Server Layer (`httpserver.go`)

Single-file HTTP server with embedded assets:

- **Entry point**: `main()` parses flags (`-p` port, `-q` quiet, `-t` no-timestamps, `-m` multi-player), loads config, parses templates, connects players, and starts the server.
- **Routing**: All requests go through `commonHandler()` which dispatches by method + path:

| Method | Path       | Action                              |
|--------|------------|-------------------------------------|
| GET    | `/`        | Render main HTML page               |
| GET    | `/player`  | List players (template fragment)    |
| PUT    | `/player`  | Add a new MPD player                |
| DELETE | `/player`  | Remove a player                     |
| GET    | `/radio`   | List radios (template fragment)     |
| PUT    | `/radio`   | Add a new radio station             |
| DELETE | `/radio`   | Remove a radio station              |
| GET    | `/status`  | Poll player status (HTMX poll)      |
| POST   | `/command` | Play, stop, pause, volume up/down   |

- **Content negotiation**: Responses are rendered as HTML (`text/html`) or plain text (`text/plain`) based on the `Accept` header.
- **Embedded assets**: `template.html`, `template.txt`, and the `static/` directory are embedded via `//go:embed` directives.
- **Static files**: Served from `/static/` via `http.FileServerFS`.

### Platform-Specific Configuration

| Platform | Config location                        | File                  |
|----------|----------------------------------------|-----------------------|
| Linux    | `$XDG_CONFIG_HOME/mpd-radio-config.json` | `platform_linux.go`   |
| macOS    | `~/Library/Application Support/mpd-radio-config.json` | `platform_darwin.go` |
| Windows  | `HKCU\SOFTWARE\MpdRadio\config` (registry) | `platform_windows.go` |

Linux and macOS share identical code; Windows uses the Windows Registry via `golang.org/x/sys/windows/registry`.

## Frontend

- **HTMX** (v4.0.0) for AJAX-driven partial page updates (status polling every 10s, form submissions).
- **Pico CSS** (v2.1.1) for responsive, minimal styling.
- Custom CSS (`static/mpd-radio.css`) for font sizing, grid layout, and music-note animation.
- Minimal JS (`static/mpd-radio.js`) for dialog open/close.

## API

See `api.md` for curl examples. The API supports both HTML and plain-text responses via the `Accept` header.

Key endpoints:
- `GET /status?player=<address>` — returns current song title or "Stopped".
- `POST /command?play&radio=<url>&player=<address>` — play a radio stream.
- `POST /command?stop&player=<address>` — stop playback.
- `POST /command?pause&player=<address>` — pause playback.
- `POST /command?volume_up&player=<address>` — increase volume by 10.
- `POST /command?volume_down&player=<address>` — decrease volume by 10.

## Building

```bash
go build -o mpd_radio .
```

For cross-compilation (e.g., Linux ARM64 for Raspberry Pi):
```bash
GOOS=linux GOARCH=arm64 go build -o mpd_radio .
```

The project has no external dependencies beyond the Go standard library (except `golang.org/x/sys` for Windows).

## Running

```bash
# Default: listen on 127.0.0.1:6680
./mpd_radio

# Custom port
./mpd_radio -p 0.0.0.0:8080

# Quiet mode (no debug logs)
./mpd_radio -q

# Multi-player mode (show player selector in UI)
./mpd_radio -m
```

### Systemd Service

A `mpd_radio.service` file is provided for running as a systemd service on Linux.

## File Structure

```
├── application.go        # Core application state, player/radio CRUD, playback control
├── httpserver.go         # HTTP server, routing, main(), embedded assets
├── mpd-client.go         # MPD protocol TCP client, connection management
├── platform_linux.go     # Linux config persistence (XDG config dir)
├── platform_darwin.go    # macOS config persistence (Application Support)
├── platform_windows.go   # Windows config persistence (Registry)
├── template.html         # HTMX + Pico CSS web UI template
├── template.txt          # Plain-text response templates (for curl/CLI)
├── static/               # Embedded static assets
│   ├── Icon.png          # Favicon
│   ├── mpd-radio.css     # Custom styles
│   └── mpd-radio.js      # Dialog helpers
├── api.md                # API usage examples
├── go.mod / go.sum       # Go module definition
├── mpd_radio.service     # Systemd unit file
├── LICENSE               # License file
└── README.md             # Project readme
```

## Development Notes

- **No external web framework**: The app uses only the Go standard library (`net/http`, `html/template`, `text/template`).
- **No database**: All state is persisted as a single JSON file (or registry key on Windows).
- **Single-package design**: All code is in `package main` — simple and straightforward.
- **HTMX-driven UI**: The frontend relies on HTMX for all dynamic behavior; no SPA framework.
- **MPD protocol**: The client implements a subset of the MPD protocol sufficient for radio playback (clear, addid, playid, stop, pause, setvol, status, currentsong, ping).
- **Connection lifecycle**: MPD connections are ephemeral — they auto-disconnect after 60s of inactivity and reconnect on the next command. This is by design to keep resource usage low.
- **Status cooldown**: `UpdateStatus()` enforces a 10-second minimum interval between status polls to avoid hammering the MPD server.
