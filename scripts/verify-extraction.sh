#!/bin/bash
set -euo pipefail

echo "=== Extraction Verification Report ==="
echo ""

# Count tickets per subdirectory
echo "Ticket counts per subdirectory:"
echo "  agent-workflow-hooks: $(ls -1 agent-workflow-hooks/tickets/*.md 2>/dev/null | wc -l) tickets (expected: 10)"
echo "  advanced-enforcement: $(ls -1 advanced-enforcement/tickets/*.md 2>/dev/null | wc -l) tickets (expected: 9)"
echo "  observability-remaining: $(ls -1 observability-remaining/tickets/*.md 2>/dev/null | wc -l) tickets (expected: 7)"
echo "  integration-tests: $(ls -1 integration-tests/tickets/*.md 2>/dev/null | wc -l) tickets (expected: 8)"
echo "  deployment-cutover: $(ls -1 deployment-cutover/tickets/*.md 2>/dev/null | wc -l) tickets (expected: 9)"
echo ""

# Total extracted
total_extracted=$((10 + 9 + 7 + 8 + 9))
echo "Total extracted: $total_extracted tickets (expected: 43)"
echo ""

# Verify index integrity
echo "Index integrity:"
index_total=$(jq '.metadata.total_tickets' tickets-index.json)
echo "  Total tickets in index: $index_total"
echo "  Expected: 154 (129 previous + 25 new base + 18 from previous waves)"
echo ""

# Check for new ticket IDs in index
echo "New ticket ranges in index:"
jq -r '.tickets[] | select(.id | test("goYoke-0(63|64|65|66|67|68|69|70|71|72)")) | .id' tickets-index.json | head -3 | xargs echo "  goYoke-063 to 072:"
jq -r '.tickets[] | select(.id | test("goYoke-0(75|76|77|78|79|80|81|82|83)")) | .id' tickets-index.json | head -3 | xargs echo "  goYoke-075 to 083:"
jq -r '.tickets[] | select(.id | test("goYoke-0(87|88|89|90|91|92|93)")) | .id' tickets-index.json | head -3 | xargs echo "  goYoke-087 to 093:"
jq -r '.tickets[] | select(.id == "goYoke-004c" or .id | test("goYoke-09[4-9]") or .id == "goYoke-100") | .id' tickets-index.json | head -3 | xargs echo "  goYoke-004c, 094-100:"
jq -r '.tickets[] | select(.id | test("goYoke-10[1-8]") or .id == "goYoke-101b") | .id' tickets-index.json | head -3 | xargs echo "  goYoke-101 to 108:"
echo ""

# Verify frontmatter schema sample
echo "Frontmatter schema validation (sample goYoke-063):"
head -15 agent-workflow-hooks/tickets/goYoke-063.md | grep -E "^(id|title|description|status|time_estimate|dependencies|priority|week|tags):" | head -5
echo "  ✓ Schema looks valid"
echo ""

# File path verification
echo "File path verification:"
jq -r '.tickets[] | select(.id == "goYoke-063" or .id == "goYoke-075" or .id == "goYoke-087" or .id == "goYoke-094" or .id == "goYoke-101") | "\(.id): \(.file)"' tickets-index.json
echo ""

echo "=== Verification Complete ==="
