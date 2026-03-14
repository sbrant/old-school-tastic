# old-school-tastic

A terminal UI viewer for [Meshtastic](https://meshtastic.org) devices over serial, built in Go with [bubbletea](https://github.com/charmbracelet/bubbletea).

Cyberpunk-themed, keyboard-driven, single binary, zero runtime dependencies.

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/platform-linux--arm64%20|%20linux--amd64-333)

## Features

- **Packet viewer** — live scrolling list of all mesh traffic with color-coded port types
- **Node list** — discovered nodes with SNR, battery, channel utilization, last heard
- **Chat** — send and receive broadcast messages
- **Direct messages** — DM conversations with individual nodes
- **Device config** — view device, LoRa, bluetooth, security, module configs
- **BBS mode** — auto-responding bulletin board with mailbox system
- **Serial reconnection** — automatic reconnect with backoff on device disconnect
- **SQLite persistence** — packets, nodes, messages, and mail survive restarts
- **PCAP capture** — write raw packets to pcap file for analysis

## Install

### From source

```bash
go install github.com/sbrant/old-school-tastic@latest
```

### Cross-compile for Raspberry Pi

```bash
git clone https://github.com/sbrant/old-school-tastic.git
cd old-school-tastic
GOOS=linux GOARCH=arm64 go build -o meshtastic-cli .
```

### Build on the Pi

```bash
go build -o meshtastic-cli .
```

## Usage

```bash
./meshtastic-cli /dev/ttyUSB0
```

### Options

```
--baud            serial baud rate (default: 115200)
--session         database session name (default: "default")
--packet-limit    max packets to store (default: 1000)
--skip-nodes      skip downloading node database on startup
--skip-config     skip loading device config on startup
--bbs             enable BBS mode
--bot             auto-reply to ping/test messages
--fahrenheit      display temperatures in °F
--pcap <file>     write packets to pcap file
--enable-logging  verbose logging to ~/.config/meshtastic-cli/log
--clear           clear session database and exit
```

## Keyboard

| Key | Action |
|-----|--------|
| `1`-`6` | Switch tabs (Packets, Nodes, Chat, DM, Config, BBS) |
| `j`/`k` | Move down/up |
| `g`/`G` | Go to top/bottom |
| `Ctrl+D`/`Ctrl+U` | Page down/up |
| `Enter` | Select / start typing |
| `Esc` | Back / stop typing |
| `r` | Reload config (config tab) |
| `?` | Help overlay |
| `q` | Quit |

## BBS Mode

Start with `--bbs` to enable. The BBS responds to direct messages on encrypted channels (not channel 0, not broadcast).

| Command | Description |
|---------|-------------|
| `help` | List commands |
| `ping` | Pong |
| `time` | Current date/time |
| `uptime` | BBS uptime |
| `nodes` | List mesh nodes |
| `info` | Your node info |
| `mail read` | Read your messages |
| `mail send <name> <msg>` | Leave a message for a node |
| `mail list` | Mailbox stats |
| `files list` | List available files |
| `files get <name>` | Download a file |

### File Serving

Place files in `~/.config/meshtastic-cli/bbs-files/` to make them available.

- **Text files** (`.txt`, `.md`) are sent as plain text chunks — readable without a client
- **Binary files** are sent as base64-encoded chunks with `FILE:` protocol headers
- The built-in file client automatically reassembles incoming binary file chunks
- Received files are saved to `~/.config/meshtastic-cli/received-files/`
- Max file size: 50KB

## Architecture

```
main.go                     CLI flags, bootstrap
internal/
  serial/                   Serial port, frame parser, reconnection
  proto/                    Protobuf decode/encode
  store/                    SQLite (nodes, messages, packets, mail)
  bbs/                      BBS command engine
  tui/                      Bubbletea UI (tabs, themes, views)
  logger/                   File-based logging
  pcap/                     PCAP file writer
```

Single static binary. No CGO. Pure Go SQLite via [modernc.org/sqlite](https://modernc.org/sqlite).

## License

MIT
