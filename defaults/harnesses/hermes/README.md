# Hermes Harness Adapter (Experimental)

The `hermes` adapter is a first-party goYoke integration for the Hermes AI
harness. It is shipped at **experimental** support level: the end-to-end
integration is not yet fully verified and the setup instructions below may
need adjustment for your environment.

## What Gets Installed

Running `goyoke harness link hermes` copies this `README.md` and
`config-template.json` into the goYoke-managed provider directory at:

```
~/.local/share/goyoke/harness/providers/hermes/
```

goYoke does **not** write into any Hermes-owned directory. You must configure
Hermes to read from the goYoke provider directory or copy the config snippet
into your Hermes configuration manually.

## Prerequisites

- `hermes` must be installed and available in your `PATH`.
  If `goyoke harness link hermes` fails with a prerequisite error, install
  Hermes and ensure it is on your `PATH`, then re-run the command.

## Quick Setup

1. Start goYoke in a terminal.

2. Discover the active socket path:
   ```
   goyoke harness status
   ```
   or for a full JSON snippet:
   ```
   goyoke harness print-config hermes
   ```
   The `socket_path` field holds the path to connect to.

3. Configure Hermes with the connection details from `config-template.json`
   in this directory. Hermes must connect to `socket_path` using a persistent
   `SOCK_STREAM` Unix domain socket connection.

4. Exchange JSON envelopes per the `harness-link` protocol:
   - Send a `Request` envelope with `protocol: "harness-link"` and a `kind`.
   - Read the `Response` envelope that arrives in reply.

## Supported Operations

The `hermes` provider declares support for:

| Kind                 | Description                                  |
|----------------------|----------------------------------------------|
| `submit_prompt`      | Send text to the active Claude session       |
| `get_snapshot`       | Retrieve the current session state as JSON   |
| `respond_modal`      | Respond to a modal dialog shown in the TUI   |
| `respond_permission` | Approve or deny a tool-use permission prompt |
| `interrupt`          | Send an interrupt signal to the running session |
| `set_model`          | Change the active Claude model               |
| `set_effort`         | Adjust the reasoning effort level            |

## Request Envelope

```json
{
  "protocol": "harness-link",
  "protocol_version": "1.0.0",
  "kind": "submit_prompt",
  "payload": { "text": "Hello from Hermes" }
}
```

## Response Envelope

```json
{
  "protocol": "harness-link",
  "protocol_version": "1.0.0",
  "kind": "submit_prompt",
  "ok": true
}
```

On failure, `ok` is `false` and an `error` object is present:

```json
{
  "ok": false,
  "error": { "code": "unavailable_state", "message": "no active session" }
}
```

## Important Notes

- The socket path changes every time goYoke starts. Do not cache it across
  restarts; poll `goyoke harness status` or read the active metadata file.
- The `hermes` adapter manages only goYoke-side integration files. It does not
  start, stop, or manage the Hermes process.
- This adapter is `experimental`. When Hermes's config layout is fully
  verified and end-to-end tested, it will be promoted to `supported`.

## Configuration Template

See `config-template.json` in this directory for a JSON snippet you can copy
into your Hermes configuration.
