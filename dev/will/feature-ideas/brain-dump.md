# GOgent — Technical Development List

---

## 🔒 Security & Configuration

- [ ] **Review Trail of Bits Claude Code config**
  > [trailofbits/claude-code-config](https://github.com/trailofbits/claude-code-config?tab=readme-ov-file#read-these-first) — read these first
- [ ] Implement sandboxing
- [ ] Remove hard-coded model names — switch to dynamic latest-model resolution at runtime

---

## 🧠 Knowledge Graph & Obsidian Integration

- [ ] **Session handoffs → Obsidian-style**
  - Link into existing KG config (tags, backlinks, metadata)
  - Goal: build a living knowledge base from agent sessions
- [ ] **Learnings output design decision:**
  - Option A: Every agent/orchestrator generates its own Obsidian-linked note
  - Option B: Agents emit a standardised `learnings.json`, synthesised by a dedicated operator into the KG
- [ ] **Spawn a Knowledge Graph Expert subagent** *(high priority)*
- [ ] Research: **RAG with STM vs LTM databases leveraging Obsidian**
  - Investigate [rowboat](https://github.com/rowboatlabs/rowboat) + Obsidian CLI integration
  - Investigate [roam-code](https://github.com/Cranot/roam-code) + Obsidian
- [ ] **zvec** — evaluate for RAG vector backend
  > [alibaba/zvec](https://github.com/alibaba/zvec)

---

## 🛠️ Tool Use & Agent Capabilities

### AskUserQuestion Tool Enhancements
- [ ] Add optional `markdown` preview content per option (code blocks, ASCII diagrams)
- [ ] Add `annotations` field keyed by question text, supporting `markdown` + free-form `notes`
- [ ] Enable preview-driven choices with captured rationale on responses

### Thinking Visibility
- [ ] Stream Claude's reasoning/thinking between each tool call to the UI

### Tool Use Architecture
- [ ] Clarify and document **Tool Use vs Programmatic Tool Use** distinction
  > Reference: [Anthropic Advanced Tool Use](https://www.anthropic.com/engineering/advanced-tool-use)

### Slash Commands → Data Pipeline
- [ ] Implement slash command triggers for:
  - SQL lookups
  - Data structuring
  - Graph building
  - Vectorisation

---

## 👥 Brain Trust — DataSci Review Team

A specialised multi-agent team for data science and ML problem analysis.

- [ ] **Senior Analyst** — scopes the problem, routes to appropriate specialists
- [ ] **Team member pool** (JSON-defined agents), covering:
  - Data Engineer
  - Data Scientist
  - ML Engineer
  - Neural Network Engineer
  - NN Theoretician
  - Data Architect
  - Deep Learning Expert
- [ ] **Problem domains** the team handles:
  - Hyperparameter tuning
  - Assumption validation
  - Model weights & fine-tuning
  - Feature extraction
  - Data cleaning & preprocessing

---

## 📚 Skills System

- [ ] **Codebase-map skill** — implement using [roam-code](https://github.com/Cranot/roam-code)
- [ ] Evaluate [obra/superpowers skills](https://github.com/obra/superpowers/tree/main/skills) — can any be adapted or chained?
- [ ] **Skills vs Best Practices review**
  > Reference: [Complete Guide to Building Skills for Claude (PDF)](https://resources.anthropic.com/hubfs/The-Complete-Guide-to-Building-Skill-for-Claude.pdf)
- [ ] Design **skill chaining** architecture

---

## 🖥️ TUI Cleanup

- [ ] Move all `task()` agents to the **agents/teams subtabs**
- [ ] Standardise stream setup — tools, events, `every 5s display --tail 3`
- [ ] **Subtabs:**
  - Provider switch mechanism available at boot
  - Gemini switcher — determine if a `GoGemini` subfolder acting as 1:1 Claude is needed
  - Add support for KLM and Minimax 2.5 (or equivalent)
- [ ] Implement **Ghostty matrix palette**
  > Reference: [jake-stewart/ghostty-palette gist](https://gist.github.com/jake-stewart/0a8ea46159a7da2c808e5be2177e1783)

---

## 🤖 Agent Teams & Messaging

- [ ] Review and implement [Claude agent teams docs](https://code.claude.com/docs/en/agent-teams)
- [ ] Persistent memory architecture for teams

---

## 📊 Performance, Self-Eval & Telemetry

- [ ] **Recursive subagent self-evaluation** driven by ML telemetry
- [ ] All agents emit structured JSON output including self-eval block
- [ ] **Inefficiency detection pipeline:**
  - Flag sessions with anomalous tool call counts / duration (e.g. 88 calls / 15 min)
  - Capture agent reasoning trace for post-hoc analysis
  - Answer: *Why was it slow? What were its thoughts?*

---

## 🪝 Hooks & Events

| Hook | Action |
|---|---|
| `SubagentStart` / `SubagentStop` | Trigger code review |
| `UserPromptSubmit` | Capture shadow metadata (user behaviour, timing, context) |

---

## 🔬 Research Queue

| Topic | Link | Notes |
|---|---|---|
| Trail of Bits Claude config | [GitHub](https://github.com/trailofbits/claude-code-config) | Security hardening |
| Advanced Tool Use | [Anthropic Eng](https://www.anthropic.com/engineering/advanced-tool-use) | Tool vs programmatic tool use |
| Superpowers skills | [GitHub](https://github.com/obra/superpowers/tree/main/skills) | Skill chaining candidates |
| Skills guide (PDF) | [Anthropic](https://resources.anthropic.com/hubfs/The-Complete-Guide-to-Building-Skill-for-Claude.pdf) | Best practices |
| Agent teams | [Claude docs](https://code.claude.com/docs/en/agent-teams) | Team messaging patterns |
| Rowboat | [GitHub](https://github.com/rowboatlabs/rowboat) | Persistent memory + Obsidian? |
| roam-code | [GitHub](https://github.com/Cranot/roam-code) | Codebase mapping + Obsidian? |
| zvec | [GitHub](https://github.com/alibaba/zvec) | RAG vector backend candidate |
| Ghostty palette | [Gist](https://gist.github.com/jake-stewart/0a8ea46159a7da2c808e5be2177e1783) | TUI theming |
