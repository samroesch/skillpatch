# Skill Broker — Product Plan

## Competitive context

OpenClaw (formerly MoltBot/Clawdbot) already built a skills registry. ClawHub, their public registry, hosts 13,729 community skills as of February 2026. The market is validated — the frontend-design skill alone has 277k installs.

They also handed us the opening: in February 2026, hundreds of malicious skills were published to ClawHub performing data exfiltration and prompt injection. ClawHub has no meaningful security checks. Researchers found payloads visible in plain text. An awesome-list of 5,400 curated skills exists because ClawHub's own discovery is too noisy to use directly.

**We are not early to the registry idea. We are early to the trusted registry.**

The SKILL.md format is also going cross-platform — Claude Code, Cursor, Gemini CLI, Codex CLI. A trust registry that works across all of them is a larger moat than Claude-only.

---

## What this is

A trust-first skills registry and routing layer. Fewer skills than ClawHub, all verified. The npm of AI skills — not the first registry, but the one people trust when it matters.

The delivery mechanism: a broker plugin with a local-first hook that routes the right skill to Claude at the right moment, invisibly. The user never manages a skill list.

---

## Positioning

| | ClawHub (OpenClaw) | Us |
|--|-------------------|-----|
| **Skills** | 13,729, anyone publishes | Curated, verified, security-reviewed |
| **Security** | Malware incident Feb 2026 | Gated submission, static analysis, commit SHA pinning |
| **Discovery** | Browse + awesome-lists | Dynamic, prompt-time routing |
| **Delivery** | Manual install | Auto-inject inline or recommend |
| **Scope** | Claude Code | Cross-platform (SKILL.md universal format) |
| **Trust signal** | None | Verified badge, usage ranking, publisher reputation |

---

## Core value proposition

> The registry where quality is guaranteed. Every skill is reviewed, pinned to a commit SHA, and ranked by real usage. The broker delivers the right one automatically — no browsing, no awesome-lists, no malware.

---

## Three delivery modes

| Mode | What happens | When |
|------|-------------|------|
| **Inject inline** | Skill content fetched and injected as context — ephemeral, no install | High confidence match, lightweight skill |
| **Recommend** | Skill surfaced with install command at end of response | High confidence match, skill needs install for full capability |
| **Silent** | Nothing | No confident match |

---

## Three install tiers

What "install" means depends on how much the user wants to commit:

| Tier | What it is | How |
|------|-----------|-----|
| **Ephemeral** | Injected for this prompt only. Nothing stored. Already working. | Automatic via broker hook |
| **Pinned** | Added to local `pinned_skills.json`. Broker always injects when relevant. No slash command overhead. | `/broker pin <skill-id>` |
| **Full install** | Permanent Claude Code plugin with slash command. | `claude plugin install` |

For most users, pinning is the right middle ground. Full install is for skills used constantly that warrant a dedicated slash command.

Trust signal mapping:
- No flag after injection → weak implicit acceptance
- User pins skill → explicit acceptance
- Full install → strong acceptance

---

## Architecture

### Hook (local-first, Go binary)

Prompts never leave the machine. The hook searches a local index, only hitting the network on a match — and then only to fetch skill content by ID.

```
Plugin ships with:
  hooks/prompt_broker          shell wrapper (selects binary per OS)
  hooks/prompt_broker_*        compiled Go binaries (Win/Mac/Linux)
  local_index.json             metadata index (updated daily from CDN)
  config.json                  user settings (risk level)
  cache/                       fetched skill content, cached locally
  pinned_skills.json           user's pinned skills (always inject when relevant)

Hook flow (per prompt):
  1. Read prompt from stdin
  2. Search local_index.json — no network, prompt never leaves machine
  3. If no match above threshold → silent ({})
  4. Apply risk gate (strict / balanced / open)
  5. If match:
       a. Check cache for skill content
       b. If not cached → GET skill .md file from CDN (skill ID only, never prompt)
       c. Inject content inline OR surface recommendation
  6. Opt-in: fire signal to analytics service (skill ID only)
```

### Static layer (GitHub / CDN) — no server required

```
storage/
  index.json              full skill index — hook pulls daily
  skills/
    meeting-notes-polisher.md
    landing-page-launcher.md
    pdf-processor.md
    csv-insight-kit.md
    ...
```

Served as raw files from the GitHub repo. Free. No maintenance. `content_url` in the index points directly to raw GitHub URLs.

### Analytics service (tiny Go server — Railway)

The only dynamic component. Receives write events only — never reads prompts.

```
POST /events    { "event": "inject"|"pin"|"install"|"flag", "skill_id": "..." }
```

A nightly batch job (GitHub Action) reads the events log, recomputes trust scores, and writes a new `storage/index.json`. The static layer self-updates.

**No Python anywhere in the stack.**

---

## Trust layer

The design principle: **trust-by-transparency, not trust-by-gatekeeping.**

### User risk tolerance (set at install, changeable anytime)

```
How would you like the skill broker to handle unverified skills?

  [1] Strict   — verified skills only, high trust score required
  [2] Balanced — verified or strong community trust (recommended)
  [3] Open     — show me everything, I'll decide

Change anytime with: /broker settings
```

| | Strict | Balanced | Open |
|--|--------|----------|------|
| Verified + high score | Auto-inject | Auto-inject | Auto-inject |
| Verified + low score | Skip | Show with warning | Show with score |
| Unverified + high community | Skip | Show with warning | Auto-inject |
| Unverified + low score | Skip | Skip | Show with warning |

### Trust score

```
Trust score = weighted combination of:

  Security scan        automated: no shell calls, no exfiltration patterns,
                       SHA pinned post-approval, known injection signatures
  Verification         human reviewed, meets submission criteria
  Community signal     implicit acceptance (injection → no flag)
                       + pin (explicit) + install (strong)
  Publisher reputation track record across their other skills
  Age + stability      longevity, no suspicious changes
  Flag rate            negative signal, weighted heavily
```

### Surfaced in recommendations

High trust:
```
skill-broker: `meeting-notes-polisher` looks relevant
  ✓ Verified  ✓ Security scanned  ★ 4.8 (2,847 users)
Install: claude plugin install meeting-notes-polisher@skill-broker
```

Lower trust:
```
skill-broker: `fastapi-expert` looks relevant
  ⚠ Unverified  ✓ Security scanned  ★ 3.1 (41 users · 3 weeks old)
[Inject once]  [Skip]  [Never suggest this]
```

---

## Gaps and risks

### Critical
- **Cold start** — ClawHub has 13,729 skills; we launch with far fewer. Quality must compensate immediately. Seed with verified imports before public launch.
- **Quality gating** — one confidently wrong skill poisons trust. Need automated triage + human review for verified badge. Define the two-tier clearly before launch.
- **Capture mechanism** — flywheel needs `/broker publish` that extracts a skill from the current session with minimal friction.

### Serious
- **OpenClaw recovers** — they could add security checks post-incident. Move fast on trust infrastructure.
- **Anthropic competitive risk** — they own the hook system and official marketplace. Hedge: cross-platform scope (Cursor, Gemini CLI) reduces dependency.
- **Model drift** — skills degrade silently as models update. Need versioning and compatibility flags.
- **False positive noise** — already visible in dev. Keyword scorer needs improvement before launch.

### Manageable
- **Privacy** — prompts never leave the machine. Analytics is opt-in, skill-ID-only.
- **Monetization** — private team registries, verified publisher program, promoted listings. Not blocking for MVP.

---

## Phases

### Phase 1 — Fix the core loop ✓ DONE
- Replaced placeholder `_build_context()` with real recommendations
- Added score threshold — silent when no confident match
- Removed debug artifact

### Phase 2 — Local-first architecture + Go binary ✓ DONE
- Go binary replaces Python hook — zero runtime dependency
- Built for Windows / Mac (Intel + Apple Silicon) / Linux
- Local index search — prompt never leaves machine
- Risk gate — reads `config.json` risk level
- Trust badge surfaces in recommendations
- Content cached after first fetch
- Cross-platform wrapper script

### Phase 3 — Retire Python server, move to static + Go analytics
**Goal:** No Python anywhere. Static files for read path. Tiny Go service for write path.

1. Delete `registry-service/` (Python/FastAPI — retired)
2. Create `storage/skills/*.md` — one SKILL.md per skill
3. Update `local_index.json` content_url fields to point to raw GitHub URLs
4. Write tiny Go analytics service (`analytics-service/main.go`)
   - Single endpoint: `POST /events`
   - Appends to events log
   - No prompt data accepted
5. Write GitHub Action for nightly trust score recomputation → updates `storage/index.json`
6. Deploy analytics service to Railway

### Phase 4 — Improve matching quality
**Goal:** False positive rate low enough that recommendations mean something.

1. Audit keyword scorer against 20 real prompts
2. Add minimum word-match requirement (already at ≥2, tune further)
3. Evaluate embedding-based scoring (cached at index build time)
4. Publisher-defined negative keywords in index
5. Definition of done: 20 prompts, fewer than 2 false positives

### Phase 5 — Pinning + trust signals
**Goal:** Users can commit to skills they like; those signals feed ranking.

1. Add `pinned_skills.json` — broker always injects pinned skills when relevant
2. `/broker pin <skill-id>` skill command
3. Pinning fires an acceptance event to analytics service
4. Install detection fires a strong acceptance event
5. Nightly job uses signals to update trust scores in index

### Phase 6 — Seed and publish
**Goal:** Enough quality skills at launch that the trust story is immediately credible.

1. Import and verify top skills from awesome-openclaw-skills and obra/superpowers
2. Write `/broker publish` — capture + submit skill from current session
3. Finalize plugin for public install
4. Cross-platform testing (Cursor, Gemini CLI)
5. Submit to official Anthropic marketplace

### Future roadmap (not scheduled)
- **Skill conflict detection** — flag when multiple installed/pinned skills overlap in domain, preventing skill bloat and dilution. Registry can surface conflicts at install time ("you already have X which covers this").
- **Skill deprecation** — formal process for retiring outdated skills as models improve
- **Skill composition** — combine multiple skills into a named pack

---

### Phase 7 — Trust layer at scale
**Goal:** Quality signals at scale, private registries, publisher program.

1. Full security scan pipeline (automated + human review queue)
2. Commit SHA pinning on all approved skills
3. Community flagging → quarantine flow
4. Model version compatibility flags
5. Private registry tier for teams
6. Publisher reputation system

---

## Current state (2026-04-16)

- Hook: Go binary, all platforms ✓
- Local-first search: prompt never leaves machine ✓
- Risk gate + trust badge in recommendations ✓
- Python server (`registry-service/`): still present, needs deletion
- Skill content: not yet in `storage/skills/*.md`
- Analytics service: not built
- No deployed registry

**Next action:** Phase 3 — delete Python server, create skill files, write Go analytics service.
