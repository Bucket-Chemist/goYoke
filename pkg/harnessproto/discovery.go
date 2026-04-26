package harnessproto

import "time"

// ActiveHarnessEndpoint is written to the path returned by
// config.GetHarnessActiveMetadataPath() after the control server successfully
// binds. Adapters poll or stat this file to discover the live endpoint without
// scanning /proc or using a fixed port.
//
// The file is removed on clean shutdown. Its absence means no harness control
// server is currently listening.
type ActiveHarnessEndpoint struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	// SessionID mirrors the live provider/CLI session identifier once the TUI
	// has received it. It may be empty briefly during startup.
	SessionID       string `json:"session_id,omitempty"`
	SocketPath      string `json:"socket_path"`
	Protocol        string `json:"protocol"`
	ProtocolVersion string `json:"protocol_version"`
}
