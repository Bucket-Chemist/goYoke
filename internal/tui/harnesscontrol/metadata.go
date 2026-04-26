package harnesscontrol

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	pkgconfig "github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// WriteActiveMetadata writes the active harness endpoint discovery file.
// The file is written atomically via os.WriteFile with mode 0600.
// Call this immediately after Server.Start() succeeds.
func WriteActiveMetadata(pid int, socketPath, sessionID string) error {
	ep := harnessproto.ActiveHarnessEndpoint{
		PID:             pid,
		StartedAt:       time.Now().UTC(),
		SessionID:       sessionID,
		SocketPath:      socketPath,
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
	}
	return writeMetadataFile(ep)
}

// UpdateSessionID rewrites the active metadata file with an updated session ID.
// Returns an error if the metadata file does not yet exist.
func UpdateSessionID(sessionID string) error {
	path := pkgconfig.GetHarnessActiveMetadataPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read active harness metadata: %w", err)
	}
	var ep harnessproto.ActiveHarnessEndpoint
	if err := json.Unmarshal(data, &ep); err != nil {
		return fmt.Errorf("decode active harness metadata: %w", err)
	}
	ep.SessionID = sessionID
	return writeMetadataFile(ep)
}

// RemoveActiveMetadata removes the active harness endpoint discovery file.
// It is idempotent: if the file does not exist, nil is returned.
func RemoveActiveMetadata() error {
	path := pkgconfig.GetHarnessActiveMetadataPath()
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writeMetadataFile(ep harnessproto.ActiveHarnessEndpoint) error {
	data, err := json.MarshalIndent(ep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal harness metadata: %w", err)
	}
	path := pkgconfig.GetHarnessActiveMetadataPath()
	return os.WriteFile(path, data, 0600)
}
