# Terminal Compatibility

This document tracks terminal emulator compatibility for the GOgent TUI application.

## Test Matrix

The following terminal emulators have been tested using `scripts/terminal-test.sh`:

| Terminal | Platform | Support Level | Version Tested | Date |
|----------|----------|---------------|----------------|------|
| Alacritty | Linux/macOS/Windows | ⏳ Pending | - | - |
| iTerm2 | macOS | ⏳ Pending | - | - |
| Kitty | Linux/macOS | ⏳ Pending | - | - |
| GNOME Terminal | Linux | ⏳ Pending | - | - |
| macOS Terminal | macOS | ⏳ Pending | - | - |
| Windows Terminal | Windows | ⏳ Pending | - | - |

**Support Levels:**
- ✅ **Full Support**: All features work as expected
- ⚠️ **Partial Support**: Core features work, some degradation
- ❌ **Not Supported**: Significant issues prevent normal use
- ⏳ **Pending**: Not yet tested

## Fully Supported Terminals

_Testing in progress. Results will be documented as terminals are verified._

### Example Format:
```
### Alacritty (Linux)
- Version: 0.13.0
- TERM: alacritty
- Test Date: YYYY-MM-DD
- Features:
  - ✅ 24-bit color (truecolor)
  - ✅ Unicode box drawing characters
  - ✅ Resize handling
  - ✅ Input handling (Tab, arrows, text)
  - ✅ Error boundaries display correctly
  - ✅ No visual artifacts
```

## Partial Support Terminals

_Testing in progress. Results will be documented as terminals are verified._

### Example Format:
```
### macOS Terminal
- Version: 2.13
- TERM: xterm-256color
- Test Date: YYYY-MM-DD
- Features:
  - ⚠️ 256 colors only (no truecolor)
  - ✅ Unicode box drawing characters
  - ✅ Resize handling
  - ✅ Input handling
- Degraded Features:
  - Color gradients may appear banded
  - Recommend using iTerm2 or Alacritty for best experience
```

## Known Issues

_No issues documented yet. This section will be populated after testing._

### Example Format:
```
### Windows Terminal (Windows 11)
- Issue: Border rendering occasionally flickers during rapid resize
- Workaround: Resize slowly or use fixed window size
- Severity: Low (cosmetic only)
```

## Graceful Degradation Strategy

The TUI implements the following fallback mechanisms for terminals with limited support:

### Color Support Detection

The app detects color capabilities via the `COLORTERM` environment variable:

- **COLORTERM=truecolor**: Use full 24-bit color palette
- **TERM=*-256color**: Fallback to 256-color palette
- **Other**: Fallback to 16-color ANSI palette

### Border Style Fallback

If Unicode box-drawing characters don't render correctly:

- **Primary**: Use Ink's `borderStyle="single"` (Unicode)
- **Fallback**: Use ASCII borders (`+-|`) if rendering issues detected

### Resize Handling

- Terminal resize events are handled by Ink's built-in system
- Layout components use percentage-based widths for fluid adaptation
- Minimum terminal size: 80x24 (standard VT100 size)

### Error Recovery

- `ErrorBoundary` component catches React errors and prevents full crashes
- Errors are logged to `~/.cache/gofortress-tui/debug.log` (when DEBUG=true)
- User sees red-bordered error message instead of blank screen
- App remains interactive after component errors

## Testing Procedure

To test a new terminal emulator:

1. Run the test script:
   ```bash
   cd packages/tui
   ./scripts/terminal-test.sh
   ```

2. Verify all items in the checklist displayed by the script

3. Document results in this file using the format shown above

4. Capture screenshots of any rendering issues

5. Test with both light and dark terminal themes (if applicable)

## Environment Variables

The following environment variables affect terminal compatibility:

| Variable | Values | Effect |
|----------|--------|--------|
| `TERM` | `xterm-256color`, `alacritty`, etc. | Terminal capability detection |
| `COLORTERM` | `truecolor`, `24bit` | Enables 24-bit color support |
| `DEBUG` | `true`, `false` | Enables debug logging to `~/.cache/gofortress-tui/debug.log` |

## Minimum Requirements

For acceptable TUI experience:

- **Terminal Size**: Minimum 80x24 characters
- **Color Support**: At least 256 colors (16-color fallback possible but not recommended)
- **Unicode Support**: UTF-8 encoding for box-drawing characters
- **Input Support**: Standard keyboard input (arrows, Tab, Enter, Ctrl+C)

## Resources

- [Ink Terminal Compatibility](https://github.com/vadimdemedes/ink#faq)
- [Terminal Color Support Detection](https://gist.github.com/XVilka/8346728)
- [ANSI Escape Codes Reference](https://en.wikipedia.org/wiki/ANSI_escape_code)

---

**Last Updated**: 2026-02-04
**Maintainer**: typescript-pro (via TUI-019)
