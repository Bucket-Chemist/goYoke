// Package teams implements the team list and detail views for the
// goYoke TUI. It polls goyoke-team-run config.json files to track
// team execution state.
//
// The TeamConfig, Wave, and Member types are imported from the shared
// internal/teamconfig package, which both the TUI and cmd/goyoke-team-run
// consume. Changes to the config.json schema produce compile-time errors
// in both consumers.
package teams

import (
	"sort"
	"sync"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/teamconfig"
)

// Re-export types from teamconfig for backward compatibility within this
// package. New code outside this package should use teamconfig directly.
type TeamConfig = teamconfig.TeamConfig
type Wave = teamconfig.Wave
type Member = teamconfig.Member

// ---------------------------------------------------------------------------
// TeamState
// ---------------------------------------------------------------------------

// TeamState is the TUI's cached view of a single team derived from its
// config.json file.
type TeamState struct {
	// Dir is the absolute filesystem path to the team directory.
	Dir string
	// Config is the most recently parsed config.json.
	Config TeamConfig
	// LastPolled records when the config.json was last successfully read.
	LastPolled time.Time
	// StreamSizes maps agent name to the size of its stream_{agent}.ndjson file in bytes.
	StreamSizes map[string]int64
}

// copyOf returns a shallow copy of the TeamState. Config.Waves and their
// Members slices are duplicated so callers cannot mutate internal state.
func (ts *TeamState) copyOf() *TeamState {
	cp := *ts
	if ts.Config.Waves != nil {
		cp.Config.Waves = make([]Wave, len(ts.Config.Waves))
		for i, w := range ts.Config.Waves {
			wCopy := w
			if w.Members != nil {
				wCopy.Members = make([]Member, len(w.Members))
				copy(wCopy.Members, w.Members)
				for j := range wCopy.Members {
					m := &wCopy.Members[j]
					if m.ProcessPID != nil {
						v := *m.ProcessPID
						m.ProcessPID = &v
					}
					if m.LastActivityTime != nil {
						v := *m.LastActivityTime
						m.LastActivityTime = &v
					}
				}
			}
			cp.Config.Waves[i] = wCopy
		}
	}
	if ts.StreamSizes != nil {
		cp.StreamSizes = make(map[string]int64, len(ts.StreamSizes))
		for k, v := range ts.StreamSizes {
			cp.StreamSizes[k] = v
		}
	}
	return &cp
}

// TotalCostUSD returns the sum of all member CostUSD values across all waves.
func (ts *TeamState) TotalCostUSD() float64 {
	var total float64
	for _, w := range ts.Config.Waves {
		for _, m := range w.Members {
			total += m.CostUSD
		}
	}
	return total
}

// CurrentWaveNumber returns the 1-based number of the last wave that contains
// any non-pending member, or 1 when no waves exist. This approximates which
// wave is currently active.
func (ts *TeamState) CurrentWaveNumber() int {
	for i := len(ts.Config.Waves) - 1; i >= 0; i-- {
		for _, m := range ts.Config.Waves[i].Members {
			if m.Status != "pending" {
				return ts.Config.Waves[i].WaveNumber
			}
		}
	}
	if len(ts.Config.Waves) > 0 {
		return ts.Config.Waves[0].WaveNumber
	}
	return 1
}

// ---------------------------------------------------------------------------
// TeamRegistry
// ---------------------------------------------------------------------------

// TeamRegistry is a thread-safe store for all known team states.
//
// The zero value is not usable; use NewTeamRegistry instead.
type TeamRegistry struct {
	teams map[string]*TeamState // keyed by team directory path
	mu    sync.RWMutex
}

// NewTeamRegistry allocates and returns an empty TeamRegistry.
func NewTeamRegistry() *TeamRegistry {
	return &TeamRegistry{
		teams: make(map[string]*TeamState),
	}
}

// Update inserts or replaces the TeamState for the given directory path.
// It is safe to call from any goroutine.
func (r *TeamRegistry) Update(dir string, config TeamConfig, streamSizes map[string]int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.teams[dir] = &TeamState{
		Dir:         dir,
		Config:      config,
		LastPolled:  time.Now(),
		StreamSizes: streamSizes,
	}
}

// Get returns a deep copy of the TeamState for the given directory, or nil
// when no entry exists. The returned value is safe to mutate.
func (r *TeamRegistry) Get(dir string) *TeamState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ts, ok := r.teams[dir]
	if !ok {
		return nil
	}
	return ts.copyOf()
}

// All returns deep copies of all registered team states, sorted by CreatedAt
// descending (newest first). The returned slice is safe to mutate.
func (r *TeamRegistry) All() []*TeamState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*TeamState, 0, len(r.teams))
	for _, ts := range r.teams {
		result = append(result, ts.copyOf())
	}

	sort.Slice(result, func(i, j int) bool {
		// Descending: later CreatedAt values sort first.
		return result[i].Config.CreatedAt > result[j].Config.CreatedAt
	})

	return result
}

// Count returns the number of registered teams.
func (r *TeamRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.teams)
}

// MostRecentRunning returns a deep copy of the running team, or the most
// recently created team as a fallback when no team is running. It returns
// nil when the registry is empty.
func (r *TeamRegistry) MostRecentRunning() *TeamState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First pass: find a running team.
	for _, ts := range r.teams {
		if ts.Config.Status == "running" {
			return ts.copyOf()
		}
	}

	// Fallback: return most recent by CreatedAt.
	var best *TeamState
	for _, ts := range r.teams {
		if best == nil || ts.Config.CreatedAt > best.Config.CreatedAt {
			best = ts
		}
	}
	if best != nil {
		return best.copyOf()
	}
	return nil
}
