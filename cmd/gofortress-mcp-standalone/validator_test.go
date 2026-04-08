package main

import (
	"strings"
	"testing"

	routing "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// testValidationIndex builds a hand-crafted *routing.AgentIndex suitable for
// validateRelationship tests. It does NOT call AgentIndex.Validate() or
// LoadAgentIndex, so it never hits disk and the version string is irrelevant
// to these tests. Only the Agents slice is used by GetAgentByID.
func testValidationIndex() *routing.AgentIndex {
	return &routing.AgentIndex{
		Version:     "2.7.0",
		GeneratedAt: "2026-03-24T00:00:00Z",
		Description: "test validation index",
		Agents: []routing.Agent{
			{
				// router-agent: only the router is permitted to spawn this.
				ID:        "router-agent",
				Name:      "Router Agent",
				Model:     "sonnet",
				Tier:      float64(2),
				Category:  "implementation",
				Path:      "agents/test/router-agent.md",
				Tools:     []string{"Read"},
				SpawnedBy: []string{"router"},
				CanSpawn:  []string{},
			},
			{
				// mozart: spawned only by router; can spawn einstein and beethoven.
				ID:        "mozart",
				Name:      "Mozart",
				Model:     "opus",
				Tier:      float64(3),
				Category:  "orchestrator",
				Path:      "agents/mozart/mozart.md",
				Tools:     []string{"Read", "Glob"},
				SpawnedBy: []string{"router"},
				CanSpawn:  []string{"einstein", "beethoven"},
			},
			{
				// einstein: spawned only by mozart; no can_spawn restriction.
				ID:        "einstein",
				Name:      "Einstein",
				Model:     "opus",
				Tier:      float64(3),
				Category:  "analysis",
				Path:      "agents/einstein/einstein.md",
				Tools:     []string{"Read"},
				SpawnedBy: []string{"mozart"},
				CanSpawn:  []string{},
			},
			{
				// beethoven: spawned only by mozart; no can_spawn restriction.
				ID:        "beethoven",
				Name:      "Beethoven",
				Model:     "opus",
				Tier:      float64(3),
				Category:  "synthesis",
				Path:      "agents/beethoven/beethoven.md",
				Tools:     []string{"Read"},
				SpawnedBy: []string{"mozart"},
				CanSpawn:  []string{},
			},
			{
				// open-agent: no spawned_by or can_spawn restrictions — any parent may spawn.
				ID:        "open-agent",
				Name:      "Open Agent",
				Model:     "sonnet",
				Tier:      float64(2),
				Category:  "implementation",
				Path:      "agents/open/open.md",
				Tools:     []string{"Read"},
				SpawnedBy: []string{},
				CanSpawn:  []string{},
			},
		},
	}
}

// TestValidateRelationship runs all 8 required test cases against
// validateRelationship using the hand-crafted fixture index.
func TestValidateRelationship(t *testing.T) {
	index := testValidationIndex()

	tests := []struct {
		name             string
		parentType       string
		callerType       string
		childAgentID     string
		wantValid        bool
		wantErrorSubstr  []string // each substring must appear in at least one error
		wantWarnSubstr   []string // each substring must appear in at least one warning
		wantNoErrors     bool
		wantNoWarnings   bool
	}{
		{
			// 1. Router spawning router-agent.
			// router-agent.SpawnedBy=["router"]. effectiveParent="router".
			// spawned_by: "router" is not in list but "router" IS effectiveParent and is in list. PASS.
			// can_spawn: effectiveParent=="router", skip. PASS.
			name:           "RouterSpawnsRouterAgent",
			parentType:     "",
			callerType:     "",
			childAgentID:   "router-agent",
			wantValid:      true,
			wantNoErrors:   true,
			wantNoWarnings: true,
		},
		{
			// 2. Router spawning open-agent.
			// open-agent.SpawnedBy=[]. No constraint. PASS.
			// can_spawn: effectiveParent=="router", skip. PASS.
			name:           "RouterSpawnsOpenAgent",
			parentType:     "",
			callerType:     "",
			childAgentID:   "open-agent",
			wantValid:      true,
			wantNoErrors:   true,
			wantNoWarnings: true,
		},
		{
			// 3. Router spawning einstein.
			// einstein.SpawnedBy=["mozart"]. effectiveParent="router".
			// "router" is NOT in einstein.SpawnedBy (only "mozart" is), AND "router" is not in spawned_by. FAIL.
			name:            "RouterSpawnsEinstein",
			parentType:      "",
			callerType:      "",
			childAgentID:    "einstein",
			wantValid:       false,
			wantErrorSubstr: []string{"may only be spawned by"},
			wantNoWarnings:  true,
		},
		{
			// 4. Mozart (claimed via callerType) spawning einstein.
			// effectiveParent="mozart", claimedIdentity=true.
			// einstein.SpawnedBy=["mozart"]. "mozart" IS in list. spawned_by PASS.
			// can_spawn: mozart.CanSpawn=["einstein","beethoven"]. "einstein" IS in list. PASS.
			name:           "MozartClaimedSpawnsEinstein",
			parentType:     "",
			callerType:     "mozart",
			childAgentID:   "einstein",
			wantValid:      true,
			wantNoErrors:   true,
			wantNoWarnings: true,
		},
		{
			// 5. Mozart (claimed) spawning router-agent.
			// effectiveParent="mozart", claimedIdentity=true.
			// router-agent.SpawnedBy=["router"]. "mozart" NOT in list, "router" NOT in spawned_by. ERROR.
			// can_spawn: mozart.CanSpawn=["einstein","beethoven"]. "router-agent" NOT in list.
			//   claimedIdentity=true → WARNING (not error).
			name:            "MozartClaimedSpawnsRouterAgent",
			parentType:      "",
			callerType:      "mozart",
			childAgentID:    "router-agent",
			wantValid:       false,
			wantErrorSubstr: []string{"may only be spawned by"},
			wantWarnSubstr:  []string{"claimed parent", "can_spawn"},
		},
		{
			// 6. Mozart (verified via parentType env) spawning router-agent.
			// effectiveParent="mozart", claimedIdentity=false.
			// router-agent.SpawnedBy=["router"]. "mozart" NOT in list, "router" NOT in spawned_by. ERROR.
			// can_spawn: mozart.CanSpawn=["einstein","beethoven"]. "router-agent" NOT in list.
			//   claimedIdentity=false → ERROR.
			name:            "MozartVerifiedSpawnsRouterAgent",
			parentType:      "mozart",
			callerType:      "",
			childAgentID:    "router-agent",
			wantValid:       false,
			wantErrorSubstr: []string{"may only be spawned by", "not permitted to spawn"},
			wantNoWarnings:  true,
		},
		{
			// 7. Unknown child agent ID.
			// GetAgentByID returns error → ValidationResult{Valid:false, Errors:["unknown child agent: ..."]}
			name:            "UnknownChildAgent",
			parentType:      "",
			callerType:      "",
			childAgentID:    "nonexistent-xyz",
			wantValid:       false,
			wantErrorSubstr: []string{"unknown child agent"},
			wantNoWarnings:  true,
		},
		{
			// 8. Verified parent (mozart) can_spawn violation with unconstrained child.
			// effectiveParent="mozart", claimedIdentity=false.
			// open-agent.SpawnedBy=[]. No constraint. spawned_by PASS.
			// can_spawn: mozart.CanSpawn=["einstein","beethoven"]. "open-agent" NOT in list.
			//   claimedIdentity=false → ERROR.
			name:            "VerifiedParentCanSpawnViolation",
			parentType:      "mozart",
			callerType:      "",
			childAgentID:    "open-agent",
			wantValid:       false,
			wantErrorSubstr: []string{"not permitted to spawn"},
			wantNoWarnings:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			vr := validateRelationship(index, tc.parentType, tc.childAgentID, tc.callerType)

			if vr.Valid != tc.wantValid {
				t.Errorf("Valid = %v, want %v (Errors=%v, Warnings=%v)",
					vr.Valid, tc.wantValid, vr.Errors, vr.Warnings)
			}

			if tc.wantNoErrors && len(vr.Errors) > 0 {
				t.Errorf("expected no errors but got: %v", vr.Errors)
			}

			if tc.wantNoWarnings && len(vr.Warnings) > 0 {
				t.Errorf("expected no warnings but got: %v", vr.Warnings)
			}

			for _, substr := range tc.wantErrorSubstr {
				found := false
				for _, e := range vr.Errors {
					if strings.Contains(e, substr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected an error containing %q but got: %v", substr, vr.Errors)
				}
			}

			for _, substr := range tc.wantWarnSubstr {
				found := false
				for _, w := range vr.Warnings {
					if strings.Contains(w, substr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected a warning containing %q but got: %v", substr, vr.Warnings)
				}
			}
		})
	}
}
