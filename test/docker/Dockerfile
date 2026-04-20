# Dockerfile for testing goYoke single-binary zero-install
#
# Simulates a fresh user who has:
# 1. A clean Linux system
# 2. Claude Code CLI installed
# 3. Downloaded the goYoke binary
#
# Build:  docker build -t goyoke-naked-test test/docker/
# Run:    ./test/docker/run-naked-test.sh

FROM node:22-slim

# Install Claude Code CLI (official npm package)
RUN npm install -g @anthropic-ai/claude-code && \
    claude --version

# Create a non-root user (simulates real user)
RUN useradd -m -s /bin/bash testuser
USER testuser
WORKDIR /home/testuser

# Create minimal directory structure
RUN mkdir -p /home/testuser/.local/bin /home/testuser/.claude

# Copy the pre-built goYoke binary (mounted or copied at build time)
COPY --chown=testuser:testuser goyoke /home/testuser/.local/bin/goyoke
RUN chmod +x /home/testuser/.local/bin/goyoke

# Put goyoke on PATH
ENV PATH="/home/testuser/.local/bin:${PATH}"

# Verify binary runs
RUN goyoke version

# Credentials are mounted at runtime (not baked into image)
# Usage: docker run -v ~/.claude/.credentials.json:/home/testuser/.claude/.credentials.json goyoke-naked-test

# Default: run the TUI
CMD ["goyoke"]
