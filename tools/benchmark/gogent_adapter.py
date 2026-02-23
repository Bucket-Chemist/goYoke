import logging
from harbor.agents.installed.claude_code import ClaudeCode
from harbor.agents.installed.base import ExecInput

# NOTE: Requires PYTHONPATH to include tools/benchmark/ directory.
# All invocations must set: PYTHONPATH=/path/to/GOgent-Fortress/tools/benchmark
# This is handled by the /benchmark-agent skill.
from identity_injector import (
    build_full_agent_context,
    load_agent_context_requirements,
)

logger = logging.getLogger(__name__)


class GOgentAgent(ClaudeCode):
    """Harbor agent adapter that injects GOgent-Fortress agent identity.

    Usage:
        harbor run -p /path/to/task \
            --agent-import-path "gogent_adapter:GOgentAgent" \
            --ak agent_id=go-pro

    INVARIANT: Identity injection MUST happen on the HOST in create_run_agent_commands(),
    BEFORE calling super(). Inside the Docker container, CLAUDE_CONFIG_DIR is overridden
    to /logs/agent/sessions/ — so ~/.claude/agents/ does NOT exist there. If injection
    were moved into Docker or deferred, it would silently fail to find identity files.
    """

    SUPPORTS_ATIF: bool = True
    _PROMPT_SIZE_WARNING_THRESHOLD = 500_000  # chars

    def __init__(
        self,
        agent_id: str = "python-pro",
        *args,
        **kwargs,
    ):
        super().__init__(*args, **kwargs)
        self._agent_id = agent_id

    @staticmethod
    def name() -> str:
        return "gogent-agent"

    def version(self) -> str:
        return "1.0.0"

    def create_run_agent_commands(self, instruction: str) -> list[ExecInput]:
        """Override to prepend agent identity + conventions to instruction.

        1. Load context requirements for self._agent_id
        2. Build augmented prompt via build_full_agent_context()
        3. Warn if augmented prompt exceeds size threshold
        4. Call super().create_run_agent_commands(augmented_instruction)

        IMPORTANT: Identity injection happens HERE on the host, not in Docker.
        See class docstring for why this invariant must be preserved.
        """
        requirements = load_agent_context_requirements(self._agent_id)
        augmented = build_full_agent_context(
            agent_id=self._agent_id,
            requirements=requirements,
            task_files=[],
            original_prompt=instruction,
        )

        if len(augmented) > self._PROMPT_SIZE_WARNING_THRESHOLD:
            logger.warning(
                "Augmented prompt is %d chars — may approach ARG_MAX. Agent: %s",
                len(augmented),
                self._agent_id,
            )

        return super().create_run_agent_commands(augmented)
