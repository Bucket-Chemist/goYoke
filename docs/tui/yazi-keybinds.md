# Yazi File Manager Keybindings Cheatsheet

Quick reference for essential yazi keybindings. For full documentation, see https://yazi-rs.github.io/docs/configuration/keymap/

## Navigation

| Key(s) | Action |
|--------|--------|
| `j` or `↓` | Next file |
| `k` or `↑` | Previous file |
| `h` or `←` | Back to parent directory |
| `l` or `→` | Enter child directory |
| `H` | Back (history) |
| `L` | Forward (history) |
| `gg` | Jump to top of list |
| `G` | Jump to bottom of list |
| `Ctrl+Home` | Jump to top |
| `Ctrl+End` | Jump to bottom |

## File Operations

| Key(s) | Action |
|--------|--------|
| `o` or `Enter` | Open selected file(s) |
| `O` or `Shift+Enter` | Open with interactive picker |
| `y` | Yank (copy) files |
| `x` | Yank (cut) files |
| `p` | Paste yanked files |
| `P` | Paste with overwrite option |
| `d` | Trash selected files |
| `D` | Permanently delete files |
| `a` | Create file or directory |
| `r` | Rename selected file(s) |

## Selection

| Key(s) | Action |
|--------|--------|
| `Space` | Toggle selection on current file |
| `Ctrl+a` | Select all files in directory |
| `Ctrl+r` | Invert selection (select/deselect all) |
| `v` | Enter visual mode (selection mode) |
| `V` | Visual mode with cursor reset |

## Search and Filter

| Key(s) | Action |
|--------|--------|
| `s` | Search files by name (fd) |
| `S` | Search files by content (ripgrep) |
| `/` | Find next file by name |
| `?` | Find previous file by name |
| `Ctrl+s` | Cancel ongoing search |

## Preview and Display

| Key(s) | Action |
|--------|--------|
| `I` | Toggle file info panel |
| `E` | Toggle hidden files visibility |
| `~` | Jump to home directory |
| `/` | Jump to root directory |

## Tabs

| Key(s) | Action |
|--------|--------|
| `t` | Create new tab |
| `w` | Close current tab |
| `[` | Switch to previous tab |
| `]` | Switch to next tab |
| `{` | Move tab left |
| `}` | Move tab right |

## Exit and Cancel

| Key(s) | Action |
|--------|--------|
| `q` | Quit yazi |
| `Q` | Quit without outputting current directory |
| `Ctrl+c` | Close tab or quit (if last tab) |
| `Esc` or `Ctrl+[` | Exit visual mode, clear selection, or cancel search |

## Tips

- Use relative paths with `a` to create files/directories (e.g., `new-dir/nested-file.txt`)
- Combine selection with file operations: Select multiple files with `Space`, then `y` to copy all
- Visual mode (`v`) lets you select a range of files by moving with arrow keys
- Search is filtered in real-time—press `Esc` to cancel and return to normal browsing
- Yazi integrates with system file associations for `o` (Open) command

---

**Note**: Keybindings are customizable in `~/.config/yazi/keymap.toml`. Defaults shown here are from yazi standard configuration.
