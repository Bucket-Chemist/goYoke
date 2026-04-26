# Manual Harness Adapter

The `manual` adapter is the universal fallback for external harness integrations
that do not have a first-party goYoke adapter. It provides the connection
parameters and protocol information needed to wire any external process to a
running goYoke TUI session.

## How It Works

goYoke exposes a Unix domain socket that external processes connect to. The
socket speaks the `harness-link` JSON protocol: request/response envelopes
encoded as newline-delimited JSON over a persistent connection.

## Quick Setup

1. Start goYoke in a terminal.

2. Discover the active socket path:
   ```
   goyoke harness status
   ```
   or for a full JSON snippet:
   ```
   goyoke harness print-config manual
   ```
   The `socket_path` field holds the path to connect to.

3. Connect to the socket from your external process. The connection is a
   persistent TCP-like stream over a Unix domain socket (SOCK_STREAM).

4. Exchange JSON envelopes per the `harness-link` protocol:
   - Send a `Request` envelope with `protocol: "harness-link"` and a `kind`.
   - Read the `Response` envelope that arrives in reply.

## Request Envelope

```json
{
  "protocol": "harness-link",
  "protocol_version": "1.0.0",
  "kind": "submit_prompt",
  "payload": { "text": "Hello from my harness" }
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

## Supported Operations

The `manual` provider declares support for:

| Kind            | Description                            |
|-----------------|----------------------------------------|
| `submit_prompt` | Send text to the active Claude session |
| `get_snapshot`  | Retrieve the current session state     |

## Important Notes

- The socket path changes every time goYoke starts. Do not cache it across
  restarts; poll `goyoke harness status` or read the active metadata file.
- The `manual` adapter does not start or manage the goYoke process. Start
  goYoke first, then connect.
- For a fully supported first-party integration, see the `hermes` provider.

## Configuration Template

See `config-template.json` in this directory for a JSON snippet you can copy
into your harness configuration.
