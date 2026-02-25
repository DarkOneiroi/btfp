# btfp 🎵 — BehindTheForestPlayer

`btfp` (BehindTheForestPlayer) is a high-performance, synchronized terminal music player and visualizer written in Go. It features a unique multi-window architecture that allows you to distribute your music experience across multiple terminal panes or windows while maintaining perfect real-time state synchronization.

---

## 🚀 Features

*  **Multi-Window IPC:** Control playback from one window while visualizing in another. Uses a high-performance Unix domain socket.
*  **Universal Format Support:** Plays all common audio formats including MP3, WAV, FLAC, OGG/Vorbis, M4A, AAC, and more via native decoders and FFmpeg fallback.
*  **Truecolor Visualizers:** High-resolution spectral analyzers (EQ), karaoke lyrics, and animated backgrounds.
*  **Smart Metadata:** Corrects missing metadata and avoids common parsing errors (e.g., "King Nothing" correctly displayed).
*  **Waybar Integration:** Built-in support for Waybar with real-time status updates and remote controls.
*  **Interactive TUI:** Built with Bubble Tea for a smooth, responsive interface with dynamic status icons.

---

## 🛠️ Architecture

`btfp` operates on a **daemon-client** model over a Unix domain socket (`/tmp/btfp.sock`), refactored for modern engineering standards:

1.  **The Server (Daemon):** Manages the shared state, audio engine (Strategy Pattern), metadata synchronization, and background broadcasts.
2.  **The Client (TUI):** Modular Model-View-Controller (MVC) components. Multiple clients can connect simultaneously with instant state reflection.

### Modular Components (tui/)
- `model.go`: Centralized application state.
- `update.go`: Message dispatching and state transitions.
- `view.go`: Main layout orchestration.
- `rendering.go`: Specific UI component rendering logic.
- `keys.go`: Centralized keyboard input handling.
- `helpers.go`: Business logic and IPC communication.

---

## 📦 Installation

`btfp` requires Go 1.24+ and ALSA development headers. `ffmpeg` is recommended for universal codec support.

```bash
# Clone the repository
git clone https://github.com/DarkOneiroi/btfp.git
cd btfp

# Build and install to ~/go/bin
make install
```

---

## 🛠️ Development & Quality

We enforce high code quality using `golangci-lint`:

| Target | Description |
| :--- | :--- |
| `make lint` | Runs the golangci-lint suite. |
| `make test` | Runs linter followed by the full test suite. |
| `make build` | Compiles the binary locally. |
| `make lint-install` | Installs the linter to your GOPATH/bin. |

---

## ⌨️ Controls

### Global
*   **Space:** Play / Pause
*   **M:** Toggle Mute
*   **Q:** Quit (Global shutdown)
*   **Tab:** Cycle between views (Library ↔ Playlist ↔ Player ↔ Viz)
*   **+/-:** Volume control
*   **Left/Right:** Seek 5s

### Library View
*   **Arrows/JK:** Navigate folders
*   **Space:** Stage/Select track for adding
*   **A:** Add all staged tracks to playlist
*   **Enter:** Enter folder / Play song immediately

---

## ⚙️ Configuration

The config file is located at `~/.config/btfp/config.toml`.

```toml
music_path = "~/Music"
default_view = "player"
accent_color = "63"  # ANSI 256 color code
```
