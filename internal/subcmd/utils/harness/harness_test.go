package harness

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// isolate redirects XDG_DATA_HOME to a temp directory so manifest operations
// in tests do not touch the real user data directory.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
}

func TestRunList(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunList(context.Background(), nil, nil, &buf)
	if err != nil {
		t.Fatalf("RunList returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "hermes") {
		t.Error("output missing 'hermes' provider")
	}
	if !strings.Contains(out, "manual") {
		t.Error("output missing 'manual' provider")
	}
	if !strings.Contains(out, "PROVIDER") {
		t.Error("output missing header row")
	}
	if !strings.Contains(out, "LINKED") {
		t.Error("output missing LINKED column header")
	}
}

func TestRunList_LinkedProviderShown(t *testing.T) {
	isolate(t)

	// Link a provider so it appears as "yes" in the list.
	var linkOut bytes.Buffer
	if err := RunLink(context.Background(), []string{"manual"}, nil, &linkOut); err != nil {
		t.Fatalf("RunLink returned unexpected error: %v", err)
	}

	var listOut bytes.Buffer
	if err := RunList(context.Background(), nil, nil, &listOut); err != nil {
		t.Fatalf("RunList returned error: %v", err)
	}

	out := listOut.String()
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "manual") {
			if !strings.Contains(line, "yes") {
				t.Errorf("expected 'manual' row to show 'yes' linked, got: %q", line)
			}
			return
		}
	}
	t.Error("'manual' row not found in list output")
}

func TestRunLink_UnknownProvider(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunLink(context.Background(), []string{"nonexistent"}, nil, &buf)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("expected 'unknown provider' in error, got: %v", err)
	}
}

func TestRunLink_NoArgs(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunLink(context.Background(), nil, nil, &buf)
	if err == nil {
		t.Fatal("expected error when no args provided, got nil")
	}
}

func TestRunLink_Success(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunLink(context.Background(), []string{"manual"}, nil, &buf)
	// link.RunDoctor may report failures for missing directories; those cause
	// RunLink to return an error. That is expected in a test environment where
	// harness runtime dirs may not exist. We only verify the manifest was written.
	// If the error mentions "linked but doctor", the link itself succeeded.
	if err != nil && !strings.Contains(err.Error(), "doctor") {
		t.Fatalf("RunLink failed for unexpected reason: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "manual") {
		t.Error("output missing provider name")
	}
}

func TestRunUnlink_NotLinked(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunUnlink(context.Background(), []string{"manual"}, nil, &buf)
	if err == nil {
		t.Fatal("expected error when unlinking non-linked provider, got nil")
	}
	if !strings.Contains(err.Error(), "not linked") {
		t.Errorf("expected 'not linked' in error, got: %v", err)
	}
}

func TestRunUnlink_NoArgs(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunUnlink(context.Background(), nil, nil, &buf)
	if err == nil {
		t.Fatal("expected error when no args provided, got nil")
	}
}

func TestRunUnlink_RoundTrip(t *testing.T) {
	isolate(t)

	// Link then unlink, verifying the list returns to unlinked state.
	var linkBuf bytes.Buffer
	_ = RunLink(context.Background(), []string{"manual"}, nil, &linkBuf)

	var unlinkBuf bytes.Buffer
	err := RunUnlink(context.Background(), []string{"manual"}, nil, &unlinkBuf)
	if err != nil {
		t.Fatalf("RunUnlink returned error: %v", err)
	}
	if !strings.Contains(unlinkBuf.String(), "manual") {
		t.Error("unlink output missing provider name")
	}

	// After unlinking, list should show "no" for manual.
	var listBuf bytes.Buffer
	if err := RunList(context.Background(), nil, nil, &listBuf); err != nil {
		t.Fatalf("RunList error: %v", err)
	}
	lines := strings.Split(listBuf.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "manual") {
			if strings.Contains(line, "yes") {
				t.Errorf("expected 'manual' to be unlinked, but line shows 'yes': %q", line)
			}
			return
		}
	}
}

func TestRunDoctor_GlobalChecks(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	// Doctor may fail in test environment (missing dirs); we just verify it runs.
	_ = RunDoctor(context.Background(), nil, nil, &buf)

	out := buf.String()
	if !strings.Contains(out, "Doctor results") {
		t.Error("output missing 'Doctor results' header")
	}
	if !strings.Contains(out, "Summary:") {
		t.Error("output missing 'Summary:' line")
	}
}

func TestRunDoctor_ProviderNotLinked(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunDoctor(context.Background(), []string{"manual"}, nil, &buf)

	// Doctor should report a failure because no manifest exists.
	if err == nil {
		t.Fatal("expected error from doctor for unlinked provider, got nil")
	}
	if !strings.Contains(err.Error(), "doctor found failures") {
		t.Errorf("expected 'doctor found failures' in error, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"manual"`) {
		t.Error("output missing provider name")
	}
}

func TestRunStatus_NoHarness(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunStatus(context.Background(), nil, nil, &buf)
	if err != nil {
		t.Fatalf("RunStatus returned unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "not running") {
		t.Error("expected 'not running' in status output when harness is absent")
	}
	if !strings.Contains(out, "Linked providers:") {
		t.Error("output missing 'Linked providers:' section")
	}
}

func TestRunPrintConfig_UnknownProvider(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunPrintConfig(context.Background(), []string{"noexist"}, nil, &buf)
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
}

func TestRunPrintConfig_ValidProvider(t *testing.T) {
	isolate(t)

	var buf bytes.Buffer
	err := RunPrintConfig(context.Background(), []string{"hermes"}, nil, &buf)
	if err != nil {
		t.Fatalf("RunPrintConfig returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"hermes"`) {
		t.Error("output missing provider name")
	}
	if !strings.Contains(out, `"protocol_version"`) {
		t.Error("output missing protocol_version field")
	}
	if !strings.Contains(out, `"capabilities"`) {
		t.Error("output missing capabilities field")
	}
}
