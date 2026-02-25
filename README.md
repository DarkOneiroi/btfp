# btfp 🎵 — BehindTheForestPlayer

`btfp` (BehindTheForestPlayer) is a high-performance, synchronized terminal music player and visualizer written in Go. It features a unique multi-window architecture that allows you to distribute your music experience across multiple terminal panes or windows while maintaining perfect real-time state synchronization.

---

## 🚀 Features

*   **Multi-Window IPC:** Control playback from one window while visualizing in another. Uses a high-performance Unix domain socket for near-zero latency synchronization.
*   **Universal Format Support:** Plays virtually any audio file (MP3, WAV, FLAC, OGG/Vorbis, M4A, AAC, WMA, AIFF, Opus, etc.) using a hybrid strategy of native decoders and an FFmpeg fallback.
*   **Truecolor Visualizers:** High-resolution spectral analyzers (EQ), karaoke lyrics, and animated backgrounds using 24-bit ANSI colors and half-block rendering.
*   **Smart Metadata & Tagging:** Corrects missing metadata from folder structures and avoids common parsing errors (e.g., handles "King Nothing" correctly) by accessing raw ID3v2 frames.
*   **Waybar Integration:** Built-in status provider for Waybar with real-time updates and remote control support.
*   **Interactive TUI:** A responsive interface built with Bubble Tea featuring dynamic icons for selection tracking and playlist status.

---

## 🛠️ Architecture

`btfp` operates on a **daemon-client** model over a Unix domain socket (`/tmp/btfp.sock`).

1.  **The Server (Daemon):**
    *   Manages the shared application state and the audio engine.
    *   Implements a **Strategy Pattern** for audio decoding (Native -> FFmpeg).
    *   Broadcasts state updates to all connected clients.
    *   Handles metadata enrichment and background tasks.
2.  **The Client (TUI):**
    *   Follows a modular **Model-View-Controller (MVC)** pattern.
    *   Multiple clients can connect to one server simultaneously (e.g., one for the library, one for the visualizer).

### Package Structure
*   `tui/`: UI modules (`model`, `view`, `update`, `keys`, `rendering`, `helpers`).
*   `player/`: Core playback engine and FFmpeg integration.
*   `server/`: IPC server and state broadcast logic.
*   `ipc/`: Shared communication protocols and type registrations.
*   `visualizations/`: High-performance terminal rendering logic.

---

## 📦 Installation

### Prerequisites
*   **Go 1.24+**
*   **ALSA Development Headers** (e.g., `libasound2-dev` on Debian/Ubuntu)
*   **FFmpeg** (Required for universal codec support)

### Building from Source
```bash
# Clone the repository
git clone https://github.com/DarkOneiroi/btfp.git
cd btfp

# Run full test suite including linter
make test

# Build and install to ~/go/bin
make install
```

---

## ⌨️ Keybindings

### Global Controls
| Key | Action |
| :--- | :--- |
| `Space` | Play / Pause |
| `M` | Toggle Mute |
| `Q` / `Ctrl+C` | **Global Quit** (Shuts down server and all windows) |
| `Tab` | Cycle Views (Library ↔ Playlist ↔ Player ↔ Viz) |
| `+` / `-` | Volume Up / Down |
| `Left` / `Right` | Seek backward/forward 5 seconds |
| `N` / `B` | Next / Previous track |
| `H` / `?` | Toggle Help Legend |

### Library View
| Key | Action |
| :--- | :--- |
| `Arrows` / `JK` | Navigate files and folders |
| `Space` | **Stage/Select** track for batch addition (`󰄲` icon) |
| `A` | Add all staged tracks to the playlist |
| `Enter` | Enter directory OR Play song immediately (switches to Player) |
| `Backspace` | Go to parent directory |

### Playlist View
| Key | Action |
| :--- | :--- |
| `Enter` | Play selected track (stays in Playlist view) |

### Visualizer View
| Key | Action |
| :--- | :--- |
| `V` | Cycle Background Modes (Viz, EQ, Karaoke, Image, Empty) |
| `C` | Cycle Visualization Patterns |
| `I` | Cycle Color Modes |
| `P` | Cycle Character Palettes |

---

## 🛰️ Waybar Integration

BTFP provides a built-in Waybar status provider. Use the `--waybar` flag to output JSON.

**Waybar Config (`config.jsonc`):**
```jsonc
"custom/btfp": {
    "exec": "$HOME/go/bin/btfp --waybar all",
    "return-type": "json",
    "on-click": "$HOME/go/bin/btfp --remote play",
    "interval": 1,
    "tooltip": true
}
```

---

## ⚙️ Configuration

Configuration is stored in `~/.config/btfp/config.toml`.

```toml
music_path = "~/Music"
default_view = 0           # 0:Library, 1:Playlist, 2:Player, 3:Viz
bg_mode = 0                # Default background mode
auto_download_lyrics = true
auto_download_art = true
update_metadata = true     # Fix missing tags automatically
theme = "default"
```

### Themes
Custom themes are TOML files located in `~/.config/btfp/themes/`. Colors are defined by ANSI 256 codes.

---

## 🛠️ Development

### Quality Standards
We use `golangci-lint` to enforce code standards.
*   `make lint`: Run the linter.
*   `make test`: Run linter + all unit and integration tests.
*   `make lint-install`: Install the required linter version.

### Contribution Rules
1.  **Modular TUI**: Do not add logic to `tui.go`. Use the specific sub-modules (`helpers.go`, `keys.go`, etc.).
2.  **IPC Registration**: New data types sent over IPC must be registered in `ipc/ipc.go`, `main.go`, and `server/server.go`.
3.  **Error Handling**: Check all returned errors or explicitly ignore them with `_ =` if the failure is non-critical.
