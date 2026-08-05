# HelpMeTalkToMe

A Socratic dialogue TUI companion for stress-testing creative and technical ideas. Acts as an insightful collaborator that challenges your thinking through progressive questioning.

## Prerequisites

- Go 1.22+
- PostgreSQL (for session storage)
- An OpenAI-compatible LLM server (LM Studio, Ollama, omlx, etc.)

## Quick Start

```bash
# Set up the database
export DATABASE_URL="postgres://user:pass@localhost:5432/helpme"
psql $DATABASE_URL -f db/schema.sql

# Build
go build -o helpme .

# Configure a preset
./helpme config add

# Start
./helpme start
```

## Commands

| Command | Action |
|---|---|
| `/?` | Show help |
| `/new` | Save and start a fresh session |
| `/reset` | Clear conversation history |
| `/model` | Select server and model interactively |
| `/write [file] [instructions]` | Export session to markdown |
| `/quit` | Exit |

## Config

Presets are stored in `~/.helpmetalktome.yaml`. The `config` subcommand manages them:

```bash
./helpme config list     # Show all presets
./helpme config add      # Add a new preset
./helpme config use      # Switch default preset
./helpme config remove   # Remove a preset
```

Environment variables (`LLM_STUDIO_URL`, `LLM_MODEL`, `LLM_API_KEY`) override preset values.