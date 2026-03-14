"""Match GOgent agent IDs to relevant SkillsBench tasks."""

from __future__ import annotations

import logging
import tomllib
from dataclasses import dataclass, field
from pathlib import Path

logger = logging.getLogger(__name__)


@dataclass
class TaskInfo:
    """Parsed metadata from a SkillsBench task."""

    path: Path
    name: str
    category: str
    tags: list[str]
    difficulty: str
    timeout_sec: float


# Maps agent_id -> matching criteria.
# A task matches if ANY tag matches OR ANY category matches.
AGENT_TASK_MAPPING: dict[str, dict] = {
    "python-pro": {
        "tags": ["Python", "python"],
        "categories": ["Compilation & Build", "Parallelization"],
    },
    "react-pro": {
        "tags": ["react", "nextjs", "frontend", "web"],
        "categories": ["web-performance"],
    },
    "typescript-pro": {
        "tags": ["typescript", "javascript", "d3.js", "node"],
        "categories": ["Data Visualization"],
    },
    "go-pro": {
        "tags": ["go", "golang"],
        # NOTE: SkillsBench may have zero Go-tagged tasks currently.
    },
    "r-pro": {
        "tags": ["R", "r-lang", "tidyverse"],
        # NOTE: SkillsBench may have zero R-tagged tasks currently.
    },
    # Fallback: run all tasks (language-agnostic agents)
    "code-reviewer": {"match_all": True},
}

DIFFICULTY_ORDER: dict[str, int] = {"easy": 0, "medium": 1, "hard": 2}


def scan_tasks(tasks_dir: Path) -> list[TaskInfo]:
    """Scan all task.toml files under tasks_dir.

    Walks tasks_dir for subdirectories containing task.toml.
    Parses each with tomllib. Returns list of TaskInfo.
    Handles missing metadata fields gracefully (defaults to empty).
    """
    tasks: list[TaskInfo] = []
    for task_toml in sorted(tasks_dir.glob("*/task.toml")):
        task_dir = task_toml.parent
        try:
            with task_toml.open("rb") as f:
                data = tomllib.load(f)
        except (tomllib.TOMLDecodeError, OSError) as exc:
            logger.warning("Skipping %s: %s", task_toml, exc)
            continue

        metadata = data.get("metadata", {})
        verifier = data.get("verifier", {})

        tasks.append(
            TaskInfo(
                path=task_dir,
                name=task_dir.name,
                category=metadata.get("category", ""),
                tags=metadata.get("tags", []),
                difficulty=metadata.get("difficulty", "medium"),
                timeout_sec=float(verifier.get("timeout_sec", 600.0)),
            )
        )

    logger.debug("Scanned %d tasks from %s", len(tasks), tasks_dir)
    return tasks


def match_tasks(agent_id: str, all_tasks: list[TaskInfo]) -> list[TaskInfo]:
    """Return tasks matching the agent's criteria from AGENT_TASK_MAPPING.

    Match logic:
    1. If agent has 'match_all': True, return all tasks
    2. If agent has 'tags', match if any task tag (case-insensitive) overlaps
    3. If agent has 'categories', match if task category (case-insensitive) overlaps
    4. If agent not in mapping, return empty list with warning

    Returns matched tasks sorted by difficulty (easy first).
    """
    if agent_id not in AGENT_TASK_MAPPING:
        logger.warning("Agent '%s' has no task mapping; returning empty list", agent_id)
        return []

    criteria = AGENT_TASK_MAPPING[agent_id]

    if criteria.get("match_all"):
        matched = list(all_tasks)
    else:
        agent_tags = {t.lower() for t in criteria.get("tags", [])}
        agent_categories = {c.lower() for c in criteria.get("categories", [])}

        matched = [
            task
            for task in all_tasks
            if {tag.lower() for tag in task.tags} & agent_tags
            or task.category.lower() in agent_categories
        ]

    if not matched:
        logger.warning("Zero tasks matched for agent '%s'", agent_id)

    return sorted(matched, key=lambda t: DIFFICULTY_ORDER.get(t.difficulty, 1))


def filter_by_difficulty(
    tasks: list[TaskInfo], max_difficulty: str = "hard"
) -> list[TaskInfo]:
    """Filter tasks to at most the given difficulty level."""
    ceiling = DIFFICULTY_ORDER.get(max_difficulty, 2)
    return [t for t in tasks if DIFFICULTY_ORDER.get(t.difficulty, 1) <= ceiling]


def list_available_agents() -> list[str]:
    """Return all agent IDs that have task mappings."""
    return list(AGENT_TASK_MAPPING.keys())
