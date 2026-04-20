package resolve

import (
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"strconv"
	"strings"
)

// MergeAgentIndexJSON merges two agents-index.json documents.
//
// Merge rules:
//   - agents: merged by "id" field; override entries replace base entries with the same ID,
//     new IDs are appended. Base order is preserved.
//   - skill_guards: union merge; override keys extend or replace base keys.
//   - distribution: taken from base only; override value is ignored.
//   - version: major mismatch → log warning, return base unchanged (no error).
//     Minor mismatch → log warning, proceed.
//   - All other top-level keys: taken from base.
//
// Malformed override triggers graceful degradation (returns base unchanged, no error).
// Returns an error only when base is nil/empty or malformed JSON.
func MergeAgentIndexJSON(base, override []byte) ([]byte, error) {
	if len(base) == 0 {
		return nil, fmt.Errorf("resolve: base agents-index.json is required")
	}

	var baseMap map[string]json.RawMessage
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return nil, fmt.Errorf("resolve: malformed base agents-index.json: %w", err)
	}

	if len(override) == 0 {
		return base, nil
	}

	var overrideMap map[string]json.RawMessage
	if err := json.Unmarshal(override, &overrideMap); err != nil {
		log.Printf("resolve: malformed override agents-index.json, using base: %v", err)
		return base, nil
	}

	// Check version compatibility before merging.
	if skipMerge := checkVersions(baseMap, overrideMap); skipMerge {
		return base, nil
	}

	// Start result from base; all unhandled keys inherit base values.
	result := make(map[string]json.RawMessage, len(baseMap))
	maps.Copy(result, baseMap)

	mergeAgents(result, baseMap, overrideMap)
	mergeSkillGuards(result, baseMap, overrideMap)

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("resolve: marshal merged agents-index.json: %w", err)
	}
	return out, nil
}

// checkVersions returns true when a major version mismatch requires aborting the merge.
func checkVersions(baseMap, overrideMap map[string]json.RawMessage) bool {
	baseVerRaw, hasBase := baseMap["version"]
	overrideVerRaw, hasOverride := overrideMap["version"]
	if !hasBase || !hasOverride {
		return false
	}

	var bv, ov string
	if err := json.Unmarshal(baseVerRaw, &bv); err != nil {
		return false
	}
	if err := json.Unmarshal(overrideVerRaw, &ov); err != nil {
		return false
	}

	bParts := parseSemver(bv)
	oParts := parseSemver(ov)
	if len(bParts) == 0 || len(oParts) == 0 {
		return false
	}

	if bParts[0] != oParts[0] {
		log.Printf("resolve: major version mismatch (base %s, override %s), skipping merge", bv, ov)
		return true
	}
	if len(bParts) >= 2 && len(oParts) >= 2 && bParts[1] != oParts[1] {
		log.Printf("resolve: minor version mismatch (base %s, override %s), proceeding with merge", bv, ov)
	}
	return false
}

// mergeAgents merges the agents array into result by ID.
func mergeAgents(result map[string]json.RawMessage, baseMap, overrideMap map[string]json.RawMessage) {
	overrideAgentsRaw, ok := overrideMap["agents"]
	if !ok {
		return
	}

	var overrideAgents []json.RawMessage
	if err := json.Unmarshal(overrideAgentsRaw, &overrideAgents); err != nil || len(overrideAgents) == 0 {
		return
	}

	// Index override agents by ID.
	overrideByID := make(map[string]json.RawMessage, len(overrideAgents))
	overrideOrder := make([]string, 0, len(overrideAgents))
	for _, raw := range overrideAgents {
		id := extractID(raw)
		if id != "" {
			overrideByID[id] = raw
			overrideOrder = append(overrideOrder, id)
		}
	}

	// Walk base agents, replacing any that have an override.
	var baseAgents []json.RawMessage
	if baseRaw, ok2 := baseMap["agents"]; ok2 {
		_ = json.Unmarshal(baseRaw, &baseAgents)
	}

	seenIDs := make(map[string]bool, len(baseAgents))
	merged := make([]json.RawMessage, 0, len(baseAgents)+len(overrideAgents))

	for _, raw := range baseAgents {
		id := extractID(raw)
		if id != "" {
			seenIDs[id] = true
			if replacement, found := overrideByID[id]; found {
				merged = append(merged, replacement)
			} else {
				merged = append(merged, raw)
			}
		} else {
			merged = append(merged, raw)
		}
	}

	// Append override agents whose IDs are not in base.
	for _, id := range overrideOrder {
		if !seenIDs[id] {
			merged = append(merged, overrideByID[id])
		}
	}

	if mergedJSON, err := json.Marshal(merged); err == nil {
		result["agents"] = mergedJSON
	}
}

// mergeSkillGuards performs a union merge of skill_guards maps.
func mergeSkillGuards(result map[string]json.RawMessage, baseMap, overrideMap map[string]json.RawMessage) {
	overrideSGRaw, ok := overrideMap["skill_guards"]
	if !ok {
		return
	}

	var overrideSG map[string]json.RawMessage
	if err := json.Unmarshal(overrideSGRaw, &overrideSG); err != nil {
		return
	}

	baseSG := make(map[string]json.RawMessage)
	if baseSGRaw, ok2 := baseMap["skill_guards"]; ok2 {
		_ = json.Unmarshal(baseSGRaw, &baseSG)
	}

	maps.Copy(baseSG, overrideSG)

	if merged, err := json.Marshal(baseSG); err == nil {
		result["skill_guards"] = merged
	}
}

// extractID returns the "id" field from a raw JSON agent object, or "" on error.
func extractID(raw json.RawMessage) string {
	var meta struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	return meta.ID
}

// parseSemver splits a version string like "2.7.0" into integer parts.
// Returns nil if any part is non-numeric.
func parseSemver(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		result = append(result, n)
	}
	return result
}
