# Skill Broker MVP

A starter kit for a Claude Code "skill broker" that:

1. installs one always-on broker plugin locally
2. intercepts user prompts with a `UserPromptSubmit` hook
3. queries a lightweight remote registry for relevant skills
4. injects a compact recommendation block back into Claude Code
5. optionally tracks anonymous usage events for ranking

This is designed as a **cheap-first architecture**:
- plugin source can live in GitHub
- marketplace can be a static JSON file in Git
- registry can be a tiny HTTP service
- real skills can remain distributed across many repos

## What is included

- `broker-plugin/` — Claude Code plugin skeleton
- `registry-service/` — tiny FastAPI registry/search service
- `storage/skills.json` — example skill index
- `marketplace.json` — example catalog entry for the broker plugin
- `scripts/demo_prompt_payload.json` — sample payload for local testing

## Suggested architecture

### Local pieces
- **Broker plugin**: the only thing users must install first
- **Hook**: runs on `UserPromptSubmit` and sends prompt text to the registry
- **Local skill install path**: later, selected skills can be installed as regular plugins

### Remote pieces
- **Registry/index**: metadata only
- **Analytics**: optional event logging and ranking
- **Package hosting**: GitHub repos owned by maintainers

## Flow

1. User types a prompt in Claude Code
2. Broker hook receives the prompt JSON on stdin
3. Hook sends prompt text to the registry `/search`
4. Registry returns top matching skills plus install hints
5. Hook emits `additionalContext` telling Claude about the most relevant skills
6. Claude can follow the recommendation immediately
7. If the user installs a skill later, usage can be logged via the broker plugin or registry

## Why this MVP shape

It avoids betting on unsupported remote skill loading.
Instead, it uses:
- dynamic search at prompt time
- local plugin install for durable capabilities
- cheap static hosting for marketplace/plugin metadata

## Local dev quickstart

### 1) Start the registry

```bash
cd registry-service
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn app:app --reload --port 8787
```

### 2) Test the hook script directly

```bash
cat scripts/demo_prompt_payload.json | \
  REGISTRY_URL=http://127.0.0.1:8787 \
  python broker-plugin/hooks/prompt_broker.py
```

Expected result: JSON with `continue: true` and an `additionalContext` block.

### 3) Package the plugin

Validate against your local Claude Code installation:

```bash
claude plugin validate broker-plugin
```

### 4) Install locally for testing

From a repo containing this plugin and marketplace:

```bash
claude plugin marketplace add .
claude plugin install skill-broker@skill-broker-market
```

## Notes

This repo intentionally keeps the plugin simple and conservative:
- no hidden telemetry by default
- usage logging is opt-in via environment variable
- remote registry only sees the current prompt text unless you expand the payload

## Next upgrades

- add `/broker-search` command backed by MCP
- add install helper that shells out to `claude plugin install`
- add trust badges and commit SHA pinning
- add creator reputation and success-rate ranking
- add local cache of recent registry responses
