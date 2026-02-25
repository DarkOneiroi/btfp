# BTFP (BehindTheForestPlayer) Development Rules

## Architectural Principles

1.  **Separation of Concerns**: Keep the TUI (Bubble Tea), Player logic (Beep/FFmpeg), and Server (IPC) decoupled.
2.  **Modular TUI**: For any significant feature addition to the TUI, avoid bloating `tui.go`. Follow the established pattern:
    *   `model.go`: State and initialization.
    *   `messages.go`: Message types.
    *   `update.go`: Main update logic and message dispatching.
    *   `view.go`: Main view orchestration.
    *   `rendering.go`: Specific UI component rendering.
    *   `keys.go`: Input handling.
    *   `helpers.go`: Business logic and utility methods.
3.  **Universal Audio**: Always prioritize native decoders in `player/player.go` but ensure `player/ffmpeg.go` fallback is maintained for non-native formats.
4.  **IPC Integrity**: Any change to `ipc.Command` or `ipc.PlayerState` must be reflected in both the `server` and `tui` packages.

## Coding Standards

*   **Documentation**: Every exported type and function must have a clear doc comment.
*   **Error Handling**: Don't just return errors; provide context where appropriate.
*   **Testing**: Maintain the integration tests in `tests/` and unit tests in component directories. Run `go test ./...` before any commit.
*   **Visual Aesthetics**: Keep the visualization logic in `visualizations/renderer.go` performant. Avoid heavy allocations in the `GeneratePattern` or `Render` loops.

## Tooling

*   Use `go mod tidy` after adding dependencies.
*   Ensure `ffmpeg` is available on the system for universal audio support.
