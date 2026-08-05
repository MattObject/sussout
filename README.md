# Audit

A Socratic tool to audit ideas. Activates critical thinking while developing an idea, acting as an insightful collaborator that challenges your thinking through progressive questioning.

## Prerequisites

- Go 1.22+
- PostgreSQL (for session storage)
- An OpenAI-compatible LLM server (LM Studio, Ollama, omlx, etc.)

## Quick Start

```bash
# Set up the database
export DATABASE_URL="postgres://user:pass@localhost:5432/audit"
psql $DATABASE_URL -f db/schema.sql

# Build
go build -o audit .

# Configure a preset
./audit config add

# Start
./audit start
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

Presets are stored in `~/.audit.yaml`. The `config` subcommand manages them:

```bash
./audit config list     # Show all presets
./audit config add      # Add a new preset
./audit config use      # Switch default preset
./audit config remove   # Remove a preset
```

Environment variables (`LLM_STUDIO_URL`, `LLM_MODEL`, `LLM_API_KEY`) override preset values.
