package main

import (
	"fmt"
	"slices"

	routing "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// ValidationResult holds the outcome of a relationship validation check.
// Valid is true when no errors were produced. Warnings are advisory and do
// not block the spawn; they arise when a claimed (unverified) identity
// violates a can_spawn constraint.
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// validateRelationship enforces spawned_by and can_spawn constraints from the
// agent index before allowing a spawn_agent call to proceed.
//
// Parameters:
//   - index: the loaded agent index (caller must have already resolved it)
//   - parentType: the verified parent identity, sourced from GOGENT_PARENT_AGENT env var
//   - childAgentID: the agent being spawned
//   - callerType: self-reported identity passed via caller_type input field (unverified)
//
// Resolution order for effective parent:
//  1. parentType (verified, from env)
//  2. callerType (claimed, unverified) — triggers warning-only for can_spawn violations
//  3. "router" (implicit root session default)
func validateRelationship(index *routing.AgentIndex, parentType string, childAgentID string, callerType string) ValidationResult {
	// Step 1: Resolve effective parent.
	effectiveParent := parentType
	claimedIdentity := false
	if effectiveParent == "" && callerType != "" {
		effectiveParent = callerType
		claimedIdentity = true // caller_type is self-reported, not verified
	}
	if effectiveParent == "" {
		effectiveParent = "router" // root-level spawn from the router session
	}

	// Step 2: Load child agent config.
	child, err := index.GetAgentByID(childAgentID)
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Errors: []string{fmt.Sprintf("unknown child agent: %s", childAgentID)},
		}
	}

	var result ValidationResult

	// Step 3: Check spawned_by constraint on child.
	// If the child has a non-empty SpawnedBy list, the effective parent must be
	// present in that list. The effectiveParent for a root session is already
	// resolved to "router" in step 1, so a spawned_by entry of ["router"] will
	// pass when the router session spawns — no special bypass is needed here.
	if len(child.SpawnedBy) > 0 && !slices.Contains(child.SpawnedBy, effectiveParent) {
		result.Errors = append(result.Errors,
			fmt.Sprintf("agent %q may only be spawned by %v, not by %q",
				childAgentID, child.SpawnedBy, effectiveParent))
	}

	// Step 4: Check can_spawn constraint on parent (bidirectional check).
	// The router has implicit permission to spawn anything; skip for router.
	if effectiveParent != "router" {
		parent, err := index.GetAgentByID(effectiveParent)
		if err == nil && len(parent.CanSpawn) > 0 && !slices.Contains(parent.CanSpawn, childAgentID) {
			if claimedIdentity {
				// Claimed identity — cannot verify, emit warning only.
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("claimed parent %q does not have %q in can_spawn list -- claimed identity not verified",
						effectiveParent, childAgentID))
			} else {
				// Verified parent (from env) — treat as hard error.
				result.Errors = append(result.Errors,
					fmt.Sprintf("agent %q is not permitted to spawn %q (can_spawn constraint)",
						effectiveParent, childAgentID))
			}
		}
	}

	// Step 5: Finalize.
	result.Valid = len(result.Errors) == 0
	return result
}
