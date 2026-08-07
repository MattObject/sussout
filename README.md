# Sussout

A local-first CLI application for Socratic stress-testing of ideas. Connects to any OpenAI-compatible LLM server (LM Studio, Ollama, omlx, etc.) and engages you in progressive, structured questioning to stress-test your thinking — no SaaS, no accounts, your conversations stay on your machine.

## How It Works

Sussout acts as a critical collaborator. You bring an idea — a project, a design choice, a research question — and it asks probing questions to expose hidden assumptions, structural gaps, and contradictions. The dialogue is stateful: Sussout tracks your position across turns and flags when your thinking drifts or stalls.

Sessions are stored locally in PostgreSQL. You can resume past conversations, export them to Markdown, and switch models or servers mid-session. Everything runs on your hardware against your own LLM.

## Features

- **Local-first** — all data stored in your own PostgreSQL database
- **Any OpenAI-compatible server** — LM Studio, Ollama, omlx, vLLM, and remote APIs all work
- **Interactive TUI** — terminal-native interface with scrolling, color, and panels
- **Session management** — resume past sessions with auto-generated recaps
- **Model switching** — change servers and models mid-session from a picker panel
- **Persistent config** — server and model choices saved to `~/.sussout.yaml` across launches
- **Export** — generate Markdown summaries with `/write`

## Prerequisites

- Go 1.22+
- PostgreSQL
- An OpenAI-compatible LLM server running somewhere

## Install

```bash
git clone https://github.com/MattObject/sussout.git
cd sussout
go build -o sussout .
```

## Quick Start

```bash
# Set up the database
export DATABASE_URL="postgres://user:pass@localhost:5432/sussout"
psql $DATABASE_URL -f db/schema.sql

# Add a preset for your LLM server
./sussout config add

# Start a session
./sussout start
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
./sussout start           # New session or pick from recent ones
./sussout resume <id>     # Resume a specific session by ID
./sussout list            # List all past sessions
```

## Config

Presets are stored in `~/.sussout.yaml`. Each preset defines a server URL, an optional model, and an optional API key.

```bash
./sussout config list     # Show all presets
./sussout config add      # Add a new preset
./sussout config use      # Switch default preset
./sussout config remove   # Remove a preset
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
