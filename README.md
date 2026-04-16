# Skillpatch

A trust-first skill registry and routing layer for Claude Code.

Intercepts prompts via a `UserPromptSubmit` hook, matches them against a local index, and injects relevant skill content directly into the Claude context — **no prompt text ever leaves your machine**.

## How it works

1. User types a prompt in Claude Code
2. The broker hook reads the prompt locally and searches `local_index.json` by keyword
3. Matching skills are ranked by trust score and filtered by your configured risk level
4. The hook fetches the skill content (a static `.md` file) from GitHub by ID
5. Content is injected as `additionalContext` — Claude sees it, the registry never does

The remote registry is **write-only for reads**: it serves static files (index + skill `.md` files). It never receives prompt text. The only outbound write is an optional anonymous event (`skill_id` only, no prompt) sent to the analytics service if you opt in.

## Repo structure

```
broker-plugin/
  hooks/
    prompt_broker.go       # hook binary source (Go)
    prompt_broker          # shell wrapper — picks correct compiled binary
  local_index.json         # local skill metadata cache
  config.json              # risk_level, index update interval
  skills/
    discover-skills/       # /broker skill for searching the registry

storage/
  index.json               # canonical registry index
  skills/                  # skill .md files served as raw GitHub content
    meeting-notes-polisher.md
    landing-page-launcher.md
    pdf-processor.md
    csv-insight-kit.md

analytics-service/
  main.go                  # tiny Go HTTP server — POST /events only

scripts/
  compute_scores.go        # nightly trust score recomputation
  build_hook.sh            # cross-platform binary build

.github/workflows/
  update-scores.yml        # nightly cron: fetch events, recompute, commit index
```

## Trust scores

Each skill carries a `trust_score` (0–5) computed nightly from weighted usage signals:

| Event    | Weight |
|----------|--------|
| inject   | +1     |
| pin      | +5     |
| install  | +10    |
| flag     | −20    |

Verified skills get a score floor of 3.0. Scores are log-scaled so early usage matters but doesn't dominate.

## Risk levels

Set in `broker-plugin/config.json`:

- `strict` — verified skills only, trust score ≥ 4.0
- `balanced` (default) — verified preferred, trust score ≥ 3.0
- `open` — all skills, trust score ≥ 2.0

## Install tiers

**Ephemeral** — broker injects skill content for the current prompt only. Nothing is saved.

**Pinned** — skill is copied to `~/.claude/skills/` so Claude's native system auto-invokes it on future matching prompts.

**Plugin** — full Claude Code plugin install for skills that ship with commands or hooks of their own.

## Building the hook binary

Requires Go 1.22+:

```bash
bash scripts/build_hook.sh
```

Produces binaries for Windows, macOS (arm64/amd64), and Linux in `broker-plugin/hooks/`.

## Testing the hook locally

```bash
cat scripts/demo_prompt_payload.json | broker-plugin/hooks/prompt_broker
```

## Analytics service

The analytics service is a minimal Go HTTP server with a single endpoint:

```
POST /events   { "event": "inject|pin|install|flag", "skill_id": "..." }
GET  /health
```

It never accepts prompt text. Deploy to Railway (or anywhere) and set the `ANALYTICS_URL` secret in this repo's GitHub settings to enable nightly score updates.

## Privacy

- Prompt text never leaves your machine
- The hook only fetches skill content by ID (a static file request)
- Usage events are opt-in and contain only `skill_id` + event type
