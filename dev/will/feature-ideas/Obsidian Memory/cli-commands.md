Obsidian CLI

Usage: obsidian <command> [options]

Options:
  vault=<name>              Target a specific vault by name

Commands:
  __completions
  __files
  aliases [all] [file=<name>] [path=<path>] [total] [verbose]  List aliases in the vault or file
  append [file=<name>] [path=<path>] content=<text> [inline]  Append content to a file
  backlinks [file=<name>] [path=<path>] [counts] [total]  List backlinks to a file
  base:create [name=<name>] [content=<text>] [silent] [newtab]  Create a new item in the current base view
  base:query [file=<name>] [path=<path>] [view=<name>] [format=json|csv|tsv|md|paths]  Query a base and return results
  base:views                                              List views in the current base file
  bases                                                   List all base files in vault
  bookmark [file=<path>] [subpath=<subpath>] [folder=<path>] [search=<query>] [url=<url>] [title=<title>]  Add a bookmark
  bookmarks [total] [verbose]                             List bookmarks
  command id=<command-id>                                 Execute an Obsidian command
  commands [filter=<prefix>]                              List available command IDs
  create [name=<name>] [path=<path>] [content=<text>] [template=<name>] [overwrite] [silent] [newtab]  Create a new file
  daily [paneType=tab|split|window] [silent]              Open daily note
  daily:append content=<text> [inline] [silent] [paneType=tab|split|window]  Append content to daily note
  daily:prepend content=<text> [inline] [silent] [paneType=tab|split|window]  Prepend content to daily note
  daily:read                                              Read daily note contents
  deadends [total] [all]                                  List files with no outgoing links
  delete [file=<name>] [path=<path>] [permanent]          Delete a file
  diff [file=<name>] [path=<path>] [from=<n>] [to=<n>] [filter=local|sync]  List or diff local/sync versions
  file [file=<name>] [path=<path>]                        Show file info
  files [folder=<path>] [ext=<extension>] [total]         List files in the vault
  folder path=<path> [info=files|folders|size]            Show folder info
  folders [folder=<path>] [total]                         List folders in the vault
  help                                                    Show list of all available commands
  history [file=<name>] [path=<path>]                     List file history versions
  history:list                                            List files with history
  history:open [file=<name>] [path=<path>]                Open file recovery
  history:read [file=<name>] [path=<path>] [version=<n>]  Read a file history version
  history:restore [file=<name>] [path=<path>] version=<n> Restore a file history version
  hotkey id=<command-id> [verbose]                        Get hotkey for a command
  hotkeys [total] [all] [verbose]                         List hotkeys
  links [file=<name>] [path=<path>] [total]               List outgoing links from a file
  move [file=<name>] [path=<path>] to=<path>              Move or rename a file
  open [file=<name>] [path=<path>] [newtab]               Open a file
  orphans [total] [all]                                   List files with no incoming links
  outline [file=<name>] [path=<path>] [format=tree|md] [total]  Show headings for the current file
  plugin id=<plugin-id>                                   Get plugin info
  plugin:disable id=<id> [filter=core|community]          Disable a plugin
  plugin:enable id=<id> [filter=core|community]           Enable a plugin
  plugin:install id=<id> [enable]                         Install a community plugin
  plugin:reload id=<id>                                   Reload a plugin (for developers)
  plugin:uninstall id=<id>                                Uninstall a community plugin
  plugins [filter=core|community] [versions]              List installed plugins
  plugins:enabled [filter=core|community] [versions]      List enabled plugins
  plugins:restrict [on] [off]                             Toggle or check restricted mode
  prepend [file=<name>] [path=<path>] content=<text> [inline]  Prepend content to a file
  properties [all] [file=<name>] [path=<path>] [name=<name>] [total] [sort=count] [counts] [format=yaml|tsv]  List properties in the vault or for a file
  property:read name=<name> [file=<name>] [path=<path>]   Read a property value from a file
  property:remove name=<name> [file=<name>] [path=<path>] Remove a property from a file
  property:set name=<name> value=<value> [type=text|list|number|checkbox|date|datetime] [file=<name>] [path=<path>]  Set a property on a file
  random [folder=<path>] [newtab] [silent]                Open a random note
  random:read [folder=<path>]                             Read a random note
  read [file=<name>] [path=<path>]                        Read file contents
  recents [total]                                         List recently opened files
  reload                                                  Reload the vault
  restart                                                 Restart the app
  search query=<text> [path=<folder>] [limit=<n>] [total] [matches] [case] [format=text|json]  Search vault for text
  search:open [query=<text>]                              Open search view
  snippet:disable name=<name>                             Disable a CSS snippet
  snippet:enable name=<name>                              Enable a CSS snippet
  snippets                                                List installed CSS snippets
  snippets:enabled                                        List enabled CSS snippets
  sync [on] [off]                                         Pause or resume sync
  sync:deleted [total]                                    List deleted files in sync
  sync:history [file=<name>] [path=<path>] [total]        List sync version history for a file
  sync:open [file=<name>] [path=<path>]                   Open sync history
  sync:read [file=<name>] [path=<path>] version=<n>       Read a sync version
  sync:restore [file=<name>] [path=<path>] version=<n>    Restore a sync version
  sync:status                                             Show sync status
  tab:open [group=<id>] [file=<path>] [view=<type>]       Open a new tab
  tabs [ids]                                              List open tabs
  tag name=<tag> [total] [verbose]                        Get tag info
  tags [all] [file=<name>] [path=<path>] [total] [counts] [sort=count]  List tags in the vault or file
  task [ref=<path:line>] [file=<name>] [path=<path>] [line=<n>] [toggle] [done] [todo] [daily] [status="<char>"]  Show or update a task
  tasks [all] [daily] [file=<name>] [path=<path>] [total] [done] [todo] [status="<char>"] [verbose]  List tasks in the vault or file
  template:insert name=<template>                         Insert template into active file
  template:read name=<template> [resolve] [title=<title>] Read template content
  templates [total]                                       List templates
  theme [name=<name>]                                     Show active theme or get info
  theme:install name=<name> [enable]                      Install a community theme
  theme:set name=<name>                                   Set active theme
  theme:uninstall name=<name>                             Uninstall a theme
  themes [versions]                                       List installed themes
  unresolved [total] [counts] [verbose]                   List unresolved links in vault
  vault [info=name|path|files|folders|size]               Show vault info
  vaults [total] [verbose]                                List known vaults
  version                                                 Show Obsidian version
  wordcount [file=<name>] [path=<path>] [words] [characters]  Count words and characters
  workspace [ids]                                         Show workspace tree

Developer:
  dev:cdp method=<CDP.method> [params=<json>]             Run a Chrome DevTools Protocol command
  dev:console [clear] [limit=<n>] [level=log|warn|error|info|debug]  Show captured console messages
  dev:css selector=<css> [prop=<name>]                    Inspect CSS with source locations
  dev:debug [on] [off]                                    Attach/detach Chrome DevTools Protocol debugger
  dev:dom selector=<css> [total] [text] [inner] [all] [attr=<name>] [css=<prop>]  Query DOM elements
  dev:errors [clear]                                      Show captured errors
  dev:mobile [on] [off]                                   Toggle mobile emulation
  dev:screenshot [path=<filename>]                        Take a screenshot
  devtools                                                Toggle Electron dev tools
  eval code=<javascript>                                  Execute JavaScript and return result

Examples:
  obsidian help
  obsidian vault=Notes commands
  obsidian command id=app:open-settings
  obsidian plugin:enable id=canvas
  