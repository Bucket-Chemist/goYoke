package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTeamRunUDSClient_Noop verifies that an empty socket path produces a
// noop client that never allocates or blocks.
func TestNewTeamRunUDSClient_Noop(t *testing.T) {
	t.Parallel()

	c := NewTeamRunUDSClient("")
	assert.True(t, c.isNoop())
	assert.Nil(t, c.conn)
	assert.Nil(t, c.sendCh)

	// notify must not panic or block on a noop client.
	c.notify(typeAgentRegister, agentRegisterPayload{AgentID: "test"})
	c.Close() // must be safe to call on noop
}

// TestNewTeamRunUDSClient_ConnectAndSend verifies that the client connects to a
// UDS socket and delivers NDJSON messages with the correct format.
func TestNewTeamRunUDSClient_ConnectAndSend(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "test.sock")

	// Start a mock UDS server.
	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	defer ln.Close()

	var (
		received []ipcRequest
		mu       sync.Mutex
		wg       sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		dec := json.NewDecoder(conn)
		for {
			var req ipcRequest
			if err := dec.Decode(&req); err != nil {
				return
			}
			mu.Lock()
			received = append(received, req)
			mu.Unlock()
		}
	}()

	c := NewTeamRunUDSClient(sockPath)
	require.False(t, c.isNoop())
	require.NotNil(t, c.sendCh)

	// Send an agent register notification.
	payload := agentRegisterPayload{
		AgentID:     "agent-1",
		AgentType:   "go-pro",
		Model:       "sonnet",
		Description: "test agent",
	}
	c.notify(typeAgentRegister, payload)

	// Send an agent update notification.
	c.notify(typeAgentUpdate, agentUpdatePayload{
		AgentID: "agent-1",
		Status:  "running",
		PID:     12345,
	})

	c.Close()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, received, 2)

	// Verify first message — agent_register.
	assert.Equal(t, typeAgentRegister, received[0].Type)
	assert.NotEmpty(t, received[0].ID)

	var reg agentRegisterPayload
	require.NoError(t, json.Unmarshal(received[0].Payload, &reg))
	assert.Equal(t, "agent-1", reg.AgentID)
	assert.Equal(t, "go-pro", reg.AgentType)
	assert.Equal(t, "sonnet", reg.Model)

	// Verify second message — agent_update.
	assert.Equal(t, typeAgentUpdate, received[1].Type)

	var upd agentUpdatePayload
	require.NoError(t, json.Unmarshal(received[1].Payload, &upd))
	assert.Equal(t, "agent-1", upd.AgentID)
	assert.Equal(t, "running", upd.Status)
	assert.Equal(t, 12345, upd.PID)
}

// TestNotify_ChannelFull verifies that notify is non-blocking when the send
// buffer is full: excess messages are dropped without panicking or blocking.
func TestNotify_ChannelFull(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "full.sock")

	// Listener that accepts but reads slowly.
	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	defer ln.Close()

	// Accept the connection but don't read, so the sender stalls after the
	// kernel buffer fills.  We only need the dial to succeed.
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Hold the connection open but don't read.
		time.Sleep(2 * time.Second)
		conn.Close()
	}()

	c := NewTeamRunUDSClient(sockPath)
	require.False(t, c.isNoop())

	// Fill the channel buffer beyond its capacity (256).  The loop must
	// complete quickly (non-blocking).
	start := time.Now()
	for range 300 {
		c.notify(typeAgentActivity, agentActivityPayload{
			AgentID: "a",
			Tool:    "Read",
			Target:  "/tmp/file",
		})
	}
	elapsed := time.Since(start)

	// All 300 calls must return well within 1 second.
	assert.Less(t, elapsed, time.Second, "notify should be non-blocking even when buffer is full")

	// Drain without reading to avoid senderLoop hanging on write.
	// Drain the channel manually so Close() doesn't block forever.
	for len(c.sendCh) > 0 {
		<-c.sendCh
	}
	close(c.sendCh)
	<-c.done
	c.conn.Close()
}

// TestClose_DrainsMessages verifies that Close() waits for the senderLoop to
// drain all enqueued messages before returning.
func TestClose_DrainsMessages(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "drain.sock")

	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	defer ln.Close()

	var (
		count int
		mu    sync.Mutex
		wg    sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		dec := json.NewDecoder(conn)
		for {
			var req ipcRequest
			if err := dec.Decode(&req); err != nil {
				return
			}
			mu.Lock()
			count++
			mu.Unlock()
		}
	}()

	c := NewTeamRunUDSClient(sockPath)
	require.False(t, c.isNoop())

	const numMessages = 10
	for range numMessages {
		c.notify(typeAgentTodoUpdate, agentTodoUpdatePayload{
			AgentID: "agent-drain",
			Todos: []todoItem{
				{Content: "todo", Status: "pending"},
				{Content: "item", Status: "in_progress"},
			},
		})
	}

	c.Close()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, numMessages, count, "all messages should be delivered before Close returns")
}

// TestNewTeamRunUDSClient_DialFailure verifies that a dial error to a
// non-existent socket produces a noop client with a logged warning.
func TestNewTeamRunUDSClient_DialFailure(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "nonexistent.sock")
	// Ensure the path does not exist.
	os.Remove(sockPath)

	c := NewTeamRunUDSClient(sockPath)
	assert.True(t, c.isNoop(), "dial failure should produce noop client")
	c.Close() // safe on noop
}
