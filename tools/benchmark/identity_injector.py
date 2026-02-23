"""
identity_injector.py - Python port of pkg/routing identity/context injection.

Ports:
  - StripYAMLFrontmatter  (identity_loader.go)
  - LoadAgentIdentity     (identity_loader.go)
  - BuildFullAgentContext (identity_loader.go)
  - GetClaudeConfigDir    (context_loader.go)
  - LoadConventionContent (context_loader.go)
  - LoadRulesContent      (context_loader.go)
  - ContextRequirements   (context_types.go)
  - ConventionRequirements (context_types.go)
  - ConditionalConvention  (context_types.go)

No caching -- runs once per benchmark invocation.
Dependencies: stdlib only.
"""

from __future__ import annotations

import fnmatch
import json
import os
import sys
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional

# ---------------------------------------------------------------------------
# Marker constants (from prompt_builder.go and identity_loader.go)
# ---------------------------------------------------------------------------

IDENTITY_MARKER = "[AGENT IDENTITY - AUTO-INJECTED]"
IDENTITY_END_MARKER = "[END AGENT IDENTITY]"
CONVENTIONS_MARKER = "[CONVENTIONS - AUTO-INJECTED BY gogent-validate]"
CONVENTIONS_END_MARKER = "[END CONVENTIONS]"


# ---------------------------------------------------------------------------
# Types (from context_types.go)
# ---------------------------------------------------------------------------

@dataclass
class ConditionalConvention:
    pattern: str
    convention: str


@dataclass
class ConventionRequirements:
    base: list[str] = field(default_factory=list)
    conditional: list[ConditionalConvention] = field(default_factory=list)


@dataclass
class ContextRequirements:
    rules: list[str] = field(default_factory=list)
    conventions: ConventionRequirements = field(default_factory=ConventionRequirements)

    def has_context_requirements(self) -> bool:
        return bool(
            self.rules
            or self.conventions.base
            or self.conventions.conditional
        )

    def get_all_conventions(self, task_files: list[str]) -> list[str]:
        """Return base conventions plus any conditional ones whose pattern matches."""
        result = list(self.conventions.base)
        for cond in self.conventions.conditional:
            if _matches_any_file(task_files, cond.pattern):
                result.append(cond.convention)
        return result


def _matches_any_file(files: list[str], pattern: str) -> bool:
    """Check if any file path (or its parent dir) matches the glob pattern."""
    for f in files:
        if fnmatch.fnmatch(f, pattern):
            return True
        parent = str(Path(f).parent)
        if fnmatch.fnmatch(parent, pattern):
            return True
    return False


# ---------------------------------------------------------------------------
# GetClaudeConfigDir (from context_loader.go)
# ---------------------------------------------------------------------------

def get_claude_config_dir() -> Path:
    env_dir = os.environ.get("CLAUDE_CONFIG_DIR", "")
    if env_dir:
        return Path(env_dir)
    return Path.home() / ".claude"


# ---------------------------------------------------------------------------
# StripYAMLFrontmatter (from identity_loader.go)
# ---------------------------------------------------------------------------

def strip_yaml_frontmatter(content: str) -> str:
    trimmed = content.strip()
    if not trimmed.startswith("---"):
        return content  # no frontmatter

    open_idx = content.index("---")
    rest = content[open_idx + 3:]

    close_idx = rest.find("\n---")
    if close_idx == -1:
        return content  # malformed, return as-is

    after_close = rest[close_idx + 4:]

    # Skip rest of the closing --- line
    nl_idx = after_close.find("\n")
    if nl_idx >= 0:
        after_close = after_close[nl_idx + 1:]

    return after_close.lstrip("\n")


# ---------------------------------------------------------------------------
# File loaders (from context_loader.go and identity_loader.go)
# ---------------------------------------------------------------------------

def load_agent_identity(agent_id: str) -> str:
    """Load ~/.claude/agents/{agent_id}/{agent_id}.md body (post-frontmatter)."""
    if not agent_id:
        return ""
    config_dir = get_claude_config_dir()
    path = config_dir / "agents" / agent_id / f"{agent_id}.md"
    try:
        content = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return ""
    except OSError as exc:
        print(f"[identity-injector] Warning: could not read agent identity {agent_id}: {exc}", file=sys.stderr)
        return ""
    return strip_yaml_frontmatter(content)


def load_rules_content(rules_name: str) -> str:
    """Load ~/.claude/rules/{rules_name}. Returns content or empty string."""
    config_dir = get_claude_config_dir()
    path = config_dir / "rules" / rules_name
    try:
        return path.read_text(encoding="utf-8")
    except OSError as exc:
        print(f"[identity-injector] Warning: could not load rules {rules_name}: {exc}", file=sys.stderr)
        return ""


def load_convention_content(convention_name: str) -> str:
    """Load ~/.claude/conventions/{convention_name}. Returns content or empty string."""
    config_dir = get_claude_config_dir()
    path = config_dir / "conventions" / convention_name
    try:
        return path.read_text(encoding="utf-8")
    except OSError as exc:
        print(f"[identity-injector] Warning: could not load convention {convention_name}: {exc}", file=sys.stderr)
        return ""


# ---------------------------------------------------------------------------
# load_agent_context_requirements
# ---------------------------------------------------------------------------

def load_agent_context_requirements(agent_id: str) -> Optional[ContextRequirements]:
    """
    Read agents-index.json and return the ContextRequirements for agent_id.
    Returns None if the agent is not found or the index cannot be read.
    """
    config_dir = get_claude_config_dir()
    index_path = config_dir / "agents" / "agents-index.json"
    try:
        raw = json.loads(index_path.read_text(encoding="utf-8"))
    except OSError as exc:
        print(f"[identity-injector] Warning: could not read agents-index.json: {exc}", file=sys.stderr)
        return None

    for agent in raw.get("agents", []):
        if agent.get("id") != agent_id:
            continue
        cr_raw = agent.get("context_requirements", {})
        conv_raw = cr_raw.get("conventions", {})
        conditionals = [
            ConditionalConvention(pattern=c["pattern"], convention=c["convention"])
            for c in conv_raw.get("conditional", [])
        ]
        return ContextRequirements(
            rules=cr_raw.get("rules", []),
            conventions=ConventionRequirements(
                base=conv_raw.get("base", []),
                conditional=conditionals,
            ),
        )
    return None


# ---------------------------------------------------------------------------
# Model resolution
# ---------------------------------------------------------------------------

# Maps tier names (from agents-index.json) to current model IDs.
# Update these when new model versions are released.
MODEL_TIER_MAP: dict[str, str] = {
    "haiku": "claude-haiku-4-5-20251001",
    "sonnet": "claude-sonnet-4-6",
    "opus": "claude-opus-4-6",
}


def get_agent_model(agent_id: str) -> str:
    """Return the full model ID for an agent based on its tier in agents-index.json.

    Falls back to sonnet tier if agent not found or model field missing.
    """
    config_dir = get_claude_config_dir()
    index_path = config_dir / "agents" / "agents-index.json"
    try:
        raw = json.loads(index_path.read_text(encoding="utf-8"))
    except OSError:
        return MODEL_TIER_MAP["sonnet"]

    for agent in raw.get("agents", []):
        if agent.get("id") == agent_id:
            tier = agent.get("model", "sonnet")
            return MODEL_TIER_MAP.get(tier, MODEL_TIER_MAP["sonnet"])

    return MODEL_TIER_MAP["sonnet"]


# ---------------------------------------------------------------------------
# BuildFullAgentContext (from identity_loader.go)
# ---------------------------------------------------------------------------

def build_full_agent_context(
    agent_id: str,
    requirements: Optional[ContextRequirements],
    task_files: list[str],
    original_prompt: str,
) -> str:
    """
    Build complete agent context: identity + rules + conventions + original prompt.

    Mirrors BuildFullAgentContext in identity_loader.go with session dir injection
    skipped (not needed for benchmark context).
    """
    # Prevent double-injection
    if IDENTITY_MARKER in original_prompt:
        # Identity already present -- only inject conventions if missing
        return _build_augmented_prompt(original_prompt, requirements, task_files)

    sections: list[str] = []
    injected = False

    # 1. Agent identity
    identity = load_agent_identity(agent_id)
    if identity:
        sections += [
            IDENTITY_MARKER,
            f"--- {agent_id} identity ---",
            identity,
            IDENTITY_END_MARKER,
            "",
        ]
        injected = True

    # 2. Rules and conventions
    if requirements is not None and requirements.has_context_requirements():
        if CONVENTIONS_MARKER not in original_prompt:
            conv_sections: list[str] = [CONVENTIONS_MARKER, ""]

            for rules_file in requirements.rules:
                content = load_rules_content(rules_file)
                conv_sections += [f"--- {rules_file} ---", content, ""]

            for conv_file in requirements.get_all_conventions(task_files):
                content = load_convention_content(conv_file)
                conv_sections += [f"--- {conv_file} ---", content, ""]

            conv_sections += [CONVENTIONS_END_MARKER, ""]

            # Only add if something was actually loaded (more than just markers)
            if len(conv_sections) > 4:
                sections.append("\n".join(conv_sections))
                injected = True

    if not injected:
        return original_prompt

    sections += ["---", "", original_prompt]
    return "\n".join(sections)


def _build_augmented_prompt(
    original_prompt: str,
    requirements: Optional[ContextRequirements],
    task_files: list[str],
) -> str:
    """Inject conventions only (identity already present). Mirrors BuildAugmentedPrompt."""
    if requirements is None or not requirements.has_context_requirements():
        return original_prompt
    if CONVENTIONS_MARKER in original_prompt:
        return original_prompt

    sections: list[str] = [CONVENTIONS_MARKER, ""]

    for rules_file in requirements.rules:
        content = load_rules_content(rules_file)
        sections += [f"--- {rules_file} ---", content, ""]

    for conv_file in requirements.get_all_conventions(task_files):
        content = load_convention_content(conv_file)
        sections += [f"--- {conv_file} ---", content, ""]

    sections += [CONVENTIONS_END_MARKER, "", "---", "", original_prompt]
    return "\n".join(sections)
