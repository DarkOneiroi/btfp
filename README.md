# btfp 🎵 — BehindTheForestPlayer

`btfp` (BehindTheForestPlayer) is a high-performance, synchronized terminal music player and visualizer written in Go. It features a unique multi-window architecture that allows you to distribute your music experience across multiple terminal panes or windows while maintaining perfect real-time state synchronization.

---

## 🚀 Features

*  **Multi-Window IPC:** Control playback from one window while visualizing in another.
*  **Universal Format Support:** Plays all common audio formats including MP3, WAV, FLAC, OGG/Vorbis, M4A, AAC, and more.
*  **Truecolor Visualizers:** High-resolution spectral analyzers (EQ), karaoke lyrics, and animated backgrounds.
*   **Fast Metadata:** Instant folder scanning with parallel metadata extraction.
*   **Waybar Integration:** Built-in support for Waybar with real-time status updates and remote controls.
*   **Automatic Tagging:** Corrects and synchronizes missing metadata from file structures.
*   **Interactive TUI:** Built with Bubble Tea for a smooth, responsive interface.

---

## 🛠️ Architecture

Unlike traditional music players, `btfp` operates on a **daemon-client** model over a Unix domain socket (`/tmp/btfp.sock`).

1.  **The Server (Daemon):** Manages the shared state, audio engine, metadata synchronization, and background downloads. It is automatically initialized by the first instance of `btfp`.
2.  **The Client (TUI):** Multiple clients can connect simultaneously to the same server. Changes in one client (like skipping a track or adjusting volume) are instantly reflected across all other instances.

---

## 📦 Installation

`btfp` requires Go 1.21+ and ALSA development headers for audio output.

```bash
# Clone the repository
git clone https://github.com/yourusername/btfp.git
cd btfp

# Build and install to ~/go/bin
make install
```

The `make install` command handles:
1. Compiling the `btfp` binary.
2. Installing it to `~/go/bin/btfp`.
3. Initializing the configuration directory at `~/.config/btfp`.
4. **Automatic Waybar integration:** It updates your Waybar `config.jsonc` and `style.css` to include the BTFP status widget.

---

## 🎮 Usage

Launch `btfp` with specialized flags to open specific windows or launch the entire dashboard:

| Command | Description |
| :--- | :--- |
| `btfp` | Launches into your default view (configured in TOML). |
| `btfp --all` | **Dashboard Mode:** Opens the **Library** in the current window and spawns **Playlist**, **Player**, and **Viz** in new terminal windows (using WezTerm). |
| `btfp --view <name>` | Start in a specific view: `library`, `playlist`, `player`, or `viz`. |

---

## 🛰️ Waybar Integration

BTFP provides a built-in Waybar status provider. While `make install` attempts to set this up automatically, you can also do it manually:

1. Add the module to your Waybar `config.jsonc`:
```jsonc
"custom/btfp": {
    "exec": "$HOME/go/bin/btfp --waybar",
    "return-type": "json",
    "on-click": "$HOME/go/bin/btfp --remote pause",
    "interval": 1,
    "tooltip": true
}
```

2. Add styling to your `style.css`:
```css
#custom-btfp {
    margin: 0 10px;
    padding: 0 8px;
    font-weight: bold;
    color: #b4befe;
}
#custom-btfp.playing {
    color: #a6e3a1;
}
```

---

## ⌨️ Controls

### Global
*   **Space:** Play / Pause
*   **L:** Toggle Lyrics mode
*   **M:** Toggle Mute
*   **Q:** Quit
*   **Tab:** Cycle between views (Library ↔ Playlist ↔ Player ↔ Viz)

### Library View
*   **Arrows/JK:** Navigate folders
*   **Enter:** Enter folder / Add song to queue
*   **S:** Search metadata

### Remote Control
You can control `btfp` from scripts or other applications:
```bash
btfp --remote play   # Toggle Play/Pause
btfp --remote next   # Next track
btfp --remote prev   # Previous track
btfp --remote mute   # Toggle Mute
```

---

## 🛠️ Development & Makefile

The project includes a streamlined `Makefile` for common tasks:

| Target | Description |
| :--- | :--- |
| `make build` | Compiles the binary to the local directory. |
| `make install` | Compiles, installs to `~/go/bin`, and runs the Waybar setup script. |
| `make uninstall` | Removes the binary from `~/go/bin`. |
| `make test` | Runs the full test suite. |
| `make clean` | Removes local build artifacts. |
| `make help` | Displays a summary of available targets. |

---

## 🎨 Visuals & Rendering

`btfp` uses a custom rendering engine that employs **Half-Block (`▀`) characters**. By setting different colors for the foreground and background of a single character, it doubles the vertical resolution. Combined with **24-bit Truecolor**, this allows album covers to appear nearly photographic in the terminal.

---

## 📂 Background Tasks

`btfp` acts as a music organizer in the background:
*   **Tag Sync:** If a song has missing tags, `btfp` extracts the Artist/Album from the folder structure and saves them to the file.
*   **Art Fetching:** Automatically searches for and caches album covers from local directories.

---

## ⚙️ Configuration

The config file is located at `~/.config/btfp/config.toml`.

```toml
music_path = "~/Music"
default_view = "player"
accent_color = "63"  # ANSI 256 color code
```

### Themes
Themes are simple TOML files in `~/.config/btfp/themes/`. Colors are defined by **ANSI 256** codes.
