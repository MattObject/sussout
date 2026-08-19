# Sussout

A local-first CLI application for Socratic stress-testing of ideas. Connects to any OpenAI-compatible LLM server (LM Studio, Ollama, omlx, etc.) and engages you in progressive, structured questioning to stress-test your thinking — no SaaS, no accounts, your conversations stay on your machine.

## How It Works

Sussout acts as a critical collaborator. You bring an idea — a project, a design choice, a research question — and it asks probing questions to expose hidden assumptions, structural gaps, and contradictions. The dialogue is stateful: Sussout tracks your position across turns and flags when your thinking drifts or stalls.

Sessions are stored locally in SQLite (`~/.sussout/sussout.db`). You can resume past conversations, export them to Markdown, and switch models or servers mid-session. Everything runs on your hardware against your own LLM.

## Features

- **Standalone** — embedded SQLite (`~/.sussout/sussout.db`), no Docker or external database
- **Any OpenAI-compatible server** — LM Studio, Ollama, omlx, vLLM, and remote APIs all work
- **Interactive TUI** — terminal-native interface with scrolling, color, and panels
- **Session management** — resume past sessions with auto-generated recaps
- **Model switching** — change servers and models mid-session from a picker panel
- **Persistent config** — server and model saved to `~/.sussout.yaml` across launches
- **Export** — generate Markdown summaries with `/write`

## Prerequisites

- Go 1.22+
- An OpenAI-compatible LLM server running somewhere (LM Studio, Ollama, omlx, etc.)

## Install

```bash
git clone https://github.com/MattObject/sussout.git
cd sussout
make setup    # installs binary to $GOPATH/bin
```

No database setup required — SQLite is embedded and creates `~/.sussout/sussout.db` automatically on first run.

## Quick Start

```bash
# Add a preset for your LLM server
sussout config add my-server

# Start a session
sussout start
```

## Commands

| Command | Action |
|---|---|
| `/?` | Show help |
| `/new` | Save and start a fresh session |
| `/reset` | Clear conversation history in current session |
| `/model` | Select server and model interactively |
| `/write [file] [instructions]` | Export session to a Markdown file |
| `/quit` | Exit |

## Session Management

```bash
sussout start              # New session or pick from recent ones
sussout resume <id>        # Resume a specific session by ID
sussout list               # List all past sessions
```

## Config

All settings are stored in `~/.sussout.yaml`.

```bash
sussout config              # Show active settings and presets
sussout config list         # List all presets with details
sussout config add <name> [url]  # Add a preset, optionally with URL
sussout config use <name>   # Switch default preset
sussout config remove <name># Remove a preset
```

Environment variables (`LLM_STUDIO_URL`, `LLM_MODEL`, `LLM_API_KEY`) override preset values at runtime.

## How the Socratic Dialogue Works

The system prompt enforces 16 rules across several domains:

- **Session Dynamics** — starts gently, becomes more rigorous, tracks contradictions and momentum
- **Question Discipline** — one question per response, concise framing, stays on thread
- **Conversation Rules** — builds on user input, focuses on structural choices, no empty praise
- **Termination** — suggests pivoting when thinking stalls, not just "this idea is bad"
- **Evidence** — asks what backs research claims without pretending to verify sources
- **Under-interpretation** — doesn't hallucinate context from fragments; asks for clarification

## Development

```bash
make build       # compile binary
make dev         # go run . (no build step)
make run ARGS="start"  # build and run with args
make test        # run tests
make vet         # static analysis
make clean       # remove binary
```
