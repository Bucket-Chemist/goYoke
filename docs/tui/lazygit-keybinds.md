# Lazygit Keybindings Cheatsheet

Quick reference for essential lazygit keybindings. For complete documentation, see https://github.com/jesseduffield/lazygit/blob/master/docs/keybindings/Keybindings_en.md

## Panel Navigation

| Key(s) | Action |
|--------|--------|
| `Tab` | Cycle to next panel |
| `Shift+Tab` | Cycle to previous panel |
| `h` or `←` | Move left (or to previous panel) |
| `l` or `→` | Move right (or to next panel) |
| `1` through `5` | Jump to specific panel |

## Files Panel (Staging)

| Key(s) | Action |
|--------|--------|
| `Space` | Stage/unstage file or hunk |
| `a` | Stage all files |
| `u` | Unstage all files |
| `d` | Discard changes (checkout) |
| `e` | Edit file in $EDITOR |
| `o` | Open file in default application |
| `Enter` | Collapse/expand directory |
| `c` | Open commit dialog |

## Commits Panel

| Key(s) | Action |
|--------|--------|
| `c` | Create commit |
| `A` | Amend commit |
| `r` | Reword commit message |
| `s` | Squash commit with previous |
| `f` | Fixup commit (no edit) |
| `e` | Edit commit (via rebase) |
| `d` | Drop commit (delete) |
| `Space` | Checkout as detached HEAD |
| `y` | Copy commit hash |

## Branches Panel

| Key(s) | Action |
|--------|--------|
| `Space` | Checkout branch |
| `n` | Create new branch |
| `d` | Delete branch |
| `M` | Merge branch into current |
| `r` | Rebase current onto branch |
| `f` | Fast-forward branch |
| `R` | Rename branch |
| `u` | Set/unset upstream |

## Stash Panel

| Key(s) | Action |
|--------|--------|
| `Space` | Pop stash (apply and remove) |
| `g` | Pop stash with reapply |
| `d` | Drop stash |

## List Navigation (Universal)

| Key(s) | Action |
|--------|--------|
| `j` or `↓` | Down |
| `k` or `↑` | Up |
| `g` then `g` | Jump to top |
| `G` | Jump to bottom |
| `<` | Scroll to top (long lists) |
| `>` | Scroll to bottom (long lists) |
| `H` | Scroll left (in diffs) |
| `L` | Scroll right (in diffs) |
| `/` | Search by text |
| `v` | Toggle range select |
| `[` | Previous tab |
| `]` | Next tab |

## Push/Pull/Fetch

| Key(s) | Action |
|--------|--------|
| `p` | Push |
| `P` | Pull |
| `f` | Fetch |
| `F` | Fetch and fast-forward |

## Other

| Key(s) | Action |
|--------|--------|
| `?` | Show help (keybindings) |
| `m` | View merge/rebase options |
| `z` | Undo recent action |
| `Ctrl+z` | Redo recent action |
| `Esc` | Exit current dialog/view |
| `q` | Quit lazygit |

## Tips

- Use `?` to view context-sensitive keybindings for current panel
- Search (`/`) filters the current list in real-time
- Range select (`v`) lets you apply operations to multiple commits/files
- Most operations show a confirmation dialog—press `Enter` or `y` to confirm
- Rebase (`r` on branch) can be complex—use `m` to see merge/rebase menu for more options
- Staging individual hunks: Stage file, then `Enter` to expand and `Space` on specific hunks

---

**Note**: Keybindings can be customized in `~/.config/lazygit/config.yml` or via the config panel (`e` on status). Defaults shown here are from lazygit standard configuration.
