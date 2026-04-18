#!/usr/bin/env bash
# integrate-rust-pro.sh — Integrates rust-pro agent into goYoke routing system
# Generated from staff-architect audit findings
# Run from repo root: bash scripts/integrate-rust-pro.sh

set -euo pipefail

CLAUDE_DIR="$HOME/Documents/goYoke/.claude"

echo "=== Phase 1: Rename convention file ==="
if [ -f "$CLAUDE_DIR/conventions/rust-conventions.md" ]; then
    mv "$CLAUDE_DIR/conventions/rust-conventions.md" "$CLAUDE_DIR/conventions/rust.md"
    echo "  ✓ rust-conventions.md → rust.md"
elif [ -f "$CLAUDE_DIR/conventions/rust.md" ]; then
    echo "  ✓ rust.md already exists (skip)"
else
    echo "  ✗ ERROR: Neither rust-conventions.md nor rust.md found!"
    exit 1
fi

echo ""
echo "=== Phase 2a: Add rust-pro to agents-index.json ==="
# Insert rust-pro entry before go-cli (after go-pro block ends)
python3 -c "
import json, sys

with open('$CLAUDE_DIR/agents/agents-index.json', 'r') as f:
    data = json.load(f)

# Check if rust-pro already exists
agents = data['agents']
if any(a['id'] == 'rust-pro' for a in agents):
    print('  ✓ rust-pro already in agents list (skip)')
else:
    # Find go-cli index to insert before it
    go_cli_idx = next(i for i, a in enumerate(agents) if a['id'] == 'go-cli')

    rust_pro = {
        'id': 'rust-pro',
        'parallelization_template': 'D',
        'name': 'Rust Pro',
        'model': 'sonnet',
        'thinking': True,
        'thinking_budget': 14000,
        'thinking_budget_refactor': 18000,
        'thinking_budget_debug': 24000,
        'tier': 2,
        'category': 'language',
        'path': 'rust-pro',
        'triggers': [
            'implement', 'refactor', 'optimize', 'create struct',
            'add function', 'write test', 'rust code', 'cargo',
            'crate', 'trait', 'lifetime', 'borrow'
        ],
        'tools': ['Read', 'Write', 'Edit', 'Bash', 'Grep', 'Glob'],
        'cli_flags': {
            'allowed_tools': ['Read', 'Glob', 'Grep', 'Bash'],
            'additional_flags': ['--permission-mode', 'delegate']
        },
        'auto_activate': {
            'languages': ['Rust'],
            'file_patterns': ['Cargo.toml', '*.rs']
        },
        'spawned_by': ['router', 'orchestrator', 'impl-manager', 'architect'],
        'conventions_required': ['rust.md'],
        'sharp_edges_count': 16,
        'description': 'Expert Rust development with modern patterns. Auto-activated for Rust projects. Single-binary desktop distribution focus with edition 2024.',
        'context_requirements': {
            'rules': ['agent-guidelines.md'],
            'conventions': {
                'base': ['rust.md'],
                'conditional': []
            }
        }
    }

    agents.insert(go_cli_idx, rust_pro)
    print('  ✓ rust-pro inserted into agents list')

# Phase 2c: Update can_spawn lists
# orchestrator
for agent in agents:
    if agent['id'] == 'orchestrator' and 'can_spawn' in agent:
        if 'rust-pro' not in agent['can_spawn']:
            agent['can_spawn'].append('rust-pro')
            print('  ✓ Added rust-pro to orchestrator.can_spawn')
        else:
            print('  ✓ rust-pro already in orchestrator.can_spawn (skip)')

    if agent['id'] == 'impl-manager' and 'can_spawn' in agent:
        if 'rust-pro' not in agent['can_spawn']:
            agent['can_spawn'].append('rust-pro')
            print('  ✓ Added rust-pro to impl-manager.can_spawn')
        else:
            print('  ✓ rust-pro already in impl-manager.can_spawn (skip)')

# Update routing_rules section (NOT 'routing' — that key doesn't exist)
routing_rules = data.get('routing_rules', {})

# auto_fire
auto_fire = routing_rules.get('auto_fire', {})
if 'implementation_rust' not in auto_fire:
    auto_fire['implementation_rust'] = 'rust-pro'
    routing_rules['auto_fire'] = auto_fire
    print('  ✓ Added implementation_rust to auto_fire')

# model_tiers
model_tiers = routing_rules.get('model_tiers', {})
sonnet_tier = model_tiers.get('sonnet', [])
if 'rust-pro' not in sonnet_tier:
    sonnet_tier.append('rust-pro')
    model_tiers['sonnet'] = sonnet_tier
    routing_rules['model_tiers'] = model_tiers
    print('  ✓ Added rust-pro to model_tiers.sonnet')

data['routing_rules'] = routing_rules

with open('$CLAUDE_DIR/agents/agents-index.json', 'w') as f:
    json.dump(data, f, indent=2)
    f.write('\n')

print('  ✓ agents-index.json saved')
"

echo ""
echo "=== Phase 2b: Update routing-schema.json ==="
python3 -c "
import json

with open('$CLAUDE_DIR/routing-schema.json', 'r') as f:
    data = json.load(f)

changes = 0

# 1. Add rust-pro to sonnet tier agents
sonnet_agents = data['tiers']['sonnet']['agents']
if 'rust-pro' not in sonnet_agents:
    sonnet_agents.append('rust-pro')
    print('  ✓ Added rust-pro to tiers.sonnet.agents')
    changes += 1

# 2. Add to agent_subagent_mapping
mapping = data['agent_subagent_mapping']
if 'rust-pro' not in mapping:
    mapping['rust-pro'] = 'Rust Pro'
    print('  ✓ Added rust-pro → \"Rust Pro\" to agent_subagent_mapping')
    changes += 1

# 3. Add to subagent_types.implementation.agents
impl_agents = data['subagent_types']['implementation']['agents']
if 'rust-pro' not in impl_agents:
    impl_agents.append('rust-pro')
    print('  ✓ Added rust-pro to subagent_types.implementation.agents')
    changes += 1

# 4. Add .rs to direct_impl_check.implementation_extensions
extensions = data['direct_impl_check']['implementation_extensions']
if '.rs' not in extensions:
    extensions.append('.rs')
    print('  ✓ Added .rs to direct_impl_check.implementation_extensions')
    changes += 1

# 5. Add Rust patterns to sonnet tier patterns
sonnet_patterns = data['tiers']['sonnet']['patterns']
rust_patterns = ['cargo', 'crate', 'trait', 'lifetime', 'borrow', 'rust']
for p in rust_patterns:
    if p not in sonnet_patterns:
        sonnet_patterns.append(p)
        changes += 1
if changes > 0:
    print('  ✓ Added Rust trigger patterns to sonnet tier')

# 6. Add auto_fire for rust
if 'auto_fire' in data:
    if 'implementation_rust' not in data['auto_fire']:
        data['auto_fire']['implementation_rust'] = 'rust-pro'
        print('  ✓ Added implementation_rust to auto_fire')
        changes += 1

# 7. Add escalation path for rust-pro → python-architect (reuse for rust-architect later)
# Skip for now — no rust-architect exists yet

with open('$CLAUDE_DIR/routing-schema.json', 'w') as f:
    json.dump(data, f, indent=2)
    f.write('\n')

print(f'  ✓ routing-schema.json saved ({changes} changes)')
"

echo ""
echo "=== Phase 3: Update CLAUDE.md dispatch table ==="
# Add Rust row to Tier 2 Sonnet section
# Add .rs to trigger resolution priority
python3 -c "
import re

with open('$CLAUDE_DIR/CLAUDE.md', 'r') as f:
    content = f.read()

changes = 0

# 1. Add Rust row to Tier 2 dispatch table (after react-pro row)
react_row = '| react, component, hook, useState, ink        | \`react-pro\`                       | React Pro                        |'
rust_row = '| Rust: implement, cargo, crate, trait, lifetime | \`rust-pro\`                        | Rust Pro                         |'
if 'rust-pro' not in content:
    content = content.replace(react_row, react_row + '\n' + rust_row)
    print('  ✓ Added Rust Pro row to Tier 2 dispatch table')
    changes += 1
else:
    print('  ✓ rust-pro already in CLAUDE.md (skip)')

# 2. Add .rs to file-type auto-activation
r_line = '   - \`.R\` files → r-pro'
rs_line = '   - \`.rs\` files → rust-pro'
if '.rs' not in content:
    content = content.replace(r_line, r_line + '\n' + rs_line)
    print('  ✓ Added .rs to trigger resolution priority')
    changes += 1

# 3. Add Rust to Convention Auto-Loading comment (simple note)
# The convention auto-loading section is Python-specific, so we add a Rust section
rust_convention_note = '''
### Rust Convention Auto-Loading

| File Pattern               | Conventions Loaded |
| -------------------------- | ------------------ |
| \`**/src/**/*.rs\`           | rust.md            |
| \`**/Cargo.toml\`           | rust.md            |'''

if 'Rust Convention' not in content:
    # Insert after the Python convention table
    python_ml_row = '| \`python-ml.md\`      | ML/NN implementation          | PyTorch patterns, attention mechanisms, loss functions, training, ONNX   |'
    content = content.replace(python_ml_row, python_ml_row + '\n' + rust_convention_note)
    print('  ✓ Added Rust Convention Auto-Loading section')
    changes += 1

with open('$CLAUDE_DIR/CLAUDE.md', 'w') as f:
    f.write(content)

print(f'  ✓ CLAUDE.md saved ({changes} changes)')
"

echo ""
echo "=== Phase 4: Update rust-pro.md (thinking budgets + remove phantom conventions) ==="
python3 -c "
with open('$CLAUDE_DIR/agents/rust-pro/rust-pro.md', 'r') as f:
    content = f.read()

changes = 0

# 1. Bump thinking budgets
content = content.replace('  budget: 10000', '  budget: 14000')
content = content.replace('  budget_refactor: 14000', '  budget_refactor: 18000')
content = content.replace('  budget_debug: 18000', '  budget_debug: 24000')
print('  ✓ Bumped thinking budgets to 14K/18K/24K')
changes += 1

# 2. Fix conventions_required to match actual filename
# Already correct as rust.md (matches the renamed file)

# 3. Remove phantom convention references at the bottom
old_refs = '''- \`~/.claude/conventions/rust.md\` (core)
- \`~/.claude/conventions/rust-tokio.md\` (if async)
- \`~/.claude/conventions/rust-cli.md\` (if CLI with clap)'''

new_refs = '''- \`~/.claude/conventions/rust.md\` (core Rust conventions)'''

content = content.replace(old_refs, new_refs)
print('  ✓ Removed phantom convention references (rust-tokio.md, rust-cli.md)')
changes += 1

with open('$CLAUDE_DIR/agents/rust-pro/rust-pro.md', 'w') as f:
    f.write(content)

print(f'  ✓ rust-pro.md saved ({changes} changes)')
"

echo ""
echo "=== Verification ==="
echo "Checking integration points..."

# Verify convention file exists
if [ -f "$CLAUDE_DIR/conventions/rust.md" ]; then
    echo "  ✓ conventions/rust.md exists"
else
    echo "  ✗ conventions/rust.md MISSING"
fi

# Verify agents-index.json has rust-pro
if python3 -c "
import json
with open('$CLAUDE_DIR/agents/agents-index.json') as f:
    data = json.load(f)
agents = [a['id'] for a in data['agents']]
assert 'rust-pro' in agents, 'rust-pro not in agents'
print('  ✓ rust-pro in agents-index.json')

# Check can_spawn
for a in data['agents']:
    if a['id'] == 'orchestrator' and 'can_spawn' in a:
        assert 'rust-pro' in a['can_spawn'], 'rust-pro not in orchestrator.can_spawn'
        print('  ✓ rust-pro in orchestrator.can_spawn')
    if a['id'] == 'impl-manager' and 'can_spawn' in a:
        assert 'rust-pro' in a['can_spawn'], 'rust-pro not in impl-manager.can_spawn'
        print('  ✓ rust-pro in impl-manager.can_spawn')

# Check model_tiers
tiers = data.get('routing', {}).get('model_tiers', {}).get('sonnet', [])
assert 'rust-pro' in tiers, 'rust-pro not in model_tiers.sonnet'
print('  ✓ rust-pro in model_tiers.sonnet')
"; then true; else echo "  ✗ agents-index.json verification FAILED"; fi

# Verify routing-schema.json
if python3 -c "
import json
with open('$CLAUDE_DIR/routing-schema.json') as f:
    data = json.load(f)

assert 'rust-pro' in data['tiers']['sonnet']['agents'], 'not in sonnet agents'
print('  ✓ rust-pro in routing-schema sonnet tier')

assert data['agent_subagent_mapping'].get('rust-pro') == 'Rust Pro', 'missing subagent mapping'
print('  ✓ rust-pro → Rust Pro in subagent mapping')

assert 'rust-pro' in data['subagent_types']['implementation']['agents'], 'not in implementation agents'
print('  ✓ rust-pro in subagent_types.implementation')

assert '.rs' in data['direct_impl_check']['implementation_extensions'], '.rs not in extensions'
print('  ✓ .rs in implementation_extensions')
"; then true; else echo "  ✗ routing-schema.json verification FAILED"; fi

# Verify CLAUDE.md
if grep -q 'rust-pro' "$CLAUDE_DIR/CLAUDE.md"; then
    echo "  ✓ rust-pro in CLAUDE.md dispatch table"
else
    echo "  ✗ rust-pro NOT in CLAUDE.md"
fi

# Verify thinking budgets
if grep -q 'budget: 14000' "$CLAUDE_DIR/agents/rust-pro/rust-pro.md"; then
    echo "  ✓ Thinking budgets updated (14K/18K/24K)"
else
    echo "  ✗ Thinking budgets NOT updated"
fi

# Check for phantom conventions
if grep -q 'rust-tokio.md' "$CLAUDE_DIR/agents/rust-pro/rust-pro.md"; then
    echo "  ✗ Phantom convention rust-tokio.md still referenced"
else
    echo "  ✓ No phantom convention references"
fi

echo ""
echo "=== Integration complete ==="
echo "All phases executed. Review verification output above for any failures."
