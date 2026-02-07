# Concurrent Config Access Design

## Problem

Multiple goroutines (one per wave member) concurrently:
1. Read `config.Waves[i].Members[j]`
2. Modify `member.Status`, `member.PID`, `member.Cost`
3. Call `writeConfigAtomic(teamDir, config)`

Without synchronization, this causes:
- **Data races**: Detected by `go test -race`
- **Lost updates**: Wave 1 member A's cost update overwrites member B's status update
- **Inconsistent reads**: Wave scheduler reads partial state

## Solution: TeamRunner Struct with Mutex

### Core Struct

```go
// TeamRunner orchestrates wave execution with thread-safe config updates
type TeamRunner struct {
    teamDir  string
    config   *TeamConfig
    configMu sync.Mutex
}

func NewTeamRunner(teamDir string, config *TeamConfig) *TeamRunner {
    return &TeamRunner{
        teamDir: teamDir,
        config:  config,
    }
}
```

### Update Methods

```go
// updateMember safely updates a member's state
func (tr *TeamRunner) updateMember(waveIdx, memberIdx int, fn func(*Member)) error {
    tr.configMu.Lock()
    defer tr.configMu.Unlock()

    if waveIdx >= len(tr.config.Waves) {
        return fmt.Errorf("invalid wave index: %d", waveIdx)
    }
    if memberIdx >= len(tr.config.Waves[waveIdx].Members) {
        return fmt.Errorf("invalid member index: %d", memberIdx)
    }

    member := &tr.config.Waves[waveIdx].Members[memberIdx]
    fn(member) // Apply the update function

    // Write atomically while holding lock
    return writeConfigAtomic(tr.teamDir, tr.config)
}

// updateGlobalCost safely updates budget_remaining_usd
func (tr *TeamRunner) updateGlobalCost(spent float64) error {
    tr.configMu.Lock()
    defer tr.configMu.Unlock()

    tr.config.BudgetRemainingUSD -= spent

    return writeConfigAtomic(tr.teamDir, tr.config)
}

// getConfig safely reads the entire config (returns a copy)
func (tr *TeamRunner) getConfig() TeamConfig {
    tr.configMu.Lock()
    defer tr.configMu.Unlock()

    return *tr.config // Return copy, not pointer
}
```

### Usage in spawnAndWait()

```go
func (tr *TeamRunner) spawnAndWait(waveIdx, memberIdx int, wg *sync.WaitGroup) {
    defer wg.Done()

    member := tr.config.Waves[waveIdx].Members[memberIdx] // Read once outside lock

    for attempt := 0; attempt <= member.MaxRetries; attempt++ {
        // Update status to running
        tr.updateMember(waveIdx, memberIdx, func(m *Member) {
            m.Status = "running"
            m.PID = os.Getpid() // Placeholder, will be child PID
        })

        // Spawn agent, collect output
        cost, err := spawnAgent(member, tr.teamDir)

        if err == nil {
            // Success - update status and cost
            tr.updateMember(waveIdx, memberIdx, func(m *Member) {
                m.Status = "completed"
                m.Cost = cost
            })
            tr.updateGlobalCost(cost)
            return
        }

        // Failure - update retry count
        tr.updateMember(waveIdx, memberIdx, func(m *Member) {
            m.RetryCount = attempt + 1
            m.Status = "pending"
        })
    }

    // All retries exhausted
    tr.updateMember(waveIdx, memberIdx, func(m *Member) {
        m.Status = "failed"
    })
}
```

## Rationale

1. **Mutex scope**: Entire config, not per-member (simpler, less deadlock risk)
2. **Write-on-update**: Every state change immediately persists (crash recovery)
3. **Functional updates**: `fn func(*Member)` pattern makes intent clear
4. **Index-based**: Pass indices instead of pointers (avoids stale pointers)

## Alternative Considered: Per-Member Mutexes

```go
type Member struct {
    // ... existing fields ...
    mu sync.Mutex // One mutex per member
}
```

**Rejected because**:
- Members array is reallocated during unmarshaling (pointers become stale)
- Global budget updates still need a separate mutex
- Complexity: 2-level locking (member + global)
- Deadlock risk: Must acquire locks in consistent order

## Race Detector Verification

All tests must pass with `-race` flag:

```bash
go test -race ./cmd/gogent-team-run/...
```

Specific test case (in TC-011):

```go
func TestConcurrentMemberUpdates(t *testing.T) {
    config := &TeamConfig{
        Waves: []Wave{
            {
                Members: []Member{
                    {Name: "agent-1", Status: "pending"},
                    {Name: "agent-2", Status: "pending"},
                    {Name: "agent-3", Status: "pending"},
                    {Name: "agent-4", Status: "pending"},
                },
            },
        },
    }

    tr := NewTeamRunner("/tmp/test-team", config)

    var wg sync.WaitGroup
    for i := 0; i < 4; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            tr.updateMember(0, idx, func(m *Member) {
                m.Status = "completed"
                m.Cost = 1.23
            })
        }(i)
    }

    wg.Wait()

    // Verify all updates persisted
    final := tr.getConfig()
    for i, member := range final.Waves[0].Members {
        assert.Equal(t, "completed", member.Status, "member %d", i)
        assert.Equal(t, 1.23, member.Cost, "member %d", i)
    }
}
```

## Test Requirements (for TC-011)

### Race Detector Tests

1. **TestConcurrentMemberUpdates**: 4 goroutines update different members
2. **TestConcurrentSameMemberUpdates**: 4 goroutines update SAME member (stress test)
3. **TestConcurrentBudgetUpdates**: 10 goroutines decrement budget_remaining_usd
4. **TestReadDuringWrite**: One goroutine writes, another reads continuously

All must pass: `go test -race -count=10`

### Correctness Tests

1. **TestNoLostUpdates**: Verify all updates persisted (count matches goroutine count)
2. **TestAtomicFileWrites**: Verify config.json valid JSON after concurrent updates
3. **TestFinalState**: After 4 members complete, config has all 4 costs summed correctly

## Files Affected (in TC-008)

- `cmd/gogent-team-run/main.go`: Create `TeamRunner` struct
- `cmd/gogent-team-run/spawn.go`: Use `updateMember()` instead of direct access
- `cmd/gogent-team-run/waves.go`: Use `getConfig()` for wave scheduling
