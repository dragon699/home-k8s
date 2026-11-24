# OpenClaw agent playbook

How to build a lean, scheduled OpenClaw agent the way Agent 007 (`tidy-jellyfin-library`) was rebuilt on 2026-10-09. Follow it when creating a new agent or automation. The reference implementation lives in the OpenClaw PVC under `/home/node/.openclaw/workspace-agent-007/` and the cron job `tidy-jellyfin-library`.

Measured effect on Agent 007: per-call context dropped from about 22,800 input tokens (75 tool schemas, 20 skills, 6.8 KB persona files, 56 KB organizer output) to about 12,000, and a run went from roughly 12 model calls to 4 or 5.

## 1. Division of labour

- The **model orchestrates, scripts decide.** Every deterministic rule (naming, matching, cleanup, verification, dedup of notifications) is implemented in a script. The model only runs commands, reads their JSON, applies one or two numeric conditions and writes a short message.
- **A rule lives in exactly one place.** Naming/cleanup rules: the convention file. Workflow: the automation prompt. Facts: `MEMORY.md`. Preferences: `USER.md`. Never restate a rule in another file, a skill or memory. When a rule changes, change the script and the convention file together.
- **Nothing the model reads may contradict anything else it reads.** Remove superseded lines instead of adding "update:" lines.

## 2. Workspace files (what to write, size targets)

| File | Loaded when | Target size | Content |
|---|---|---|---|
| `AGENTS.md` | every run (Codex harness reads it natively) | ≤ 2 KB | sections `Purpose`, `Local notes`, `Rules of engagement`, `Memory` |
| `SOUL.md` | every call unless the job uses light context | ≤ 300 chars | 2–3 sentences of tone |
| `IDENTITY.md` | same | ≤ 150 chars | 4 bullets: Name, Creature, Vibe, Emoji |
| `USER.md` | same, truncated at 4,000 chars | ≤ 2.5 KB | owner preferences as directives, one per line |
| `MEMORY.md` | main chat session (injected at start, plus memory search); not injected into isolated cron runs, which rely on the prompt, `AGENTS.md` and script state files instead | ≤ 1.5 KB | durable facts only |
| `memory/YYYY-MM-DD.md` | main session startup, also indexed for memory search | short | what changed and why |
| `<DOMAIN>_CONVENTION.md` | only when the model calls `read` | ≤ 3.5 KB | the specification the scripts implement |
| `media/*.py` (or equivalent) | never injected | n/a | source copies of the deployed scripts |

### AGENTS.md

- `Purpose`: one paragraph. Name the automation, its schedule and which file holds which rules.
- `Local notes`: absolute paths, script names and flags, exact tool call syntax with real arguments, for example `message(action="send", channel="slack", target="user:U0AFBG79RRV", message=...)` and `jellyfin_library_manage(action=scan)`. Note channel formatting constraints (bullets, no markdown tables in Slack/Discord).
- `Rules of engagement`: scheduled runs act without asking, never retry a failed command, report exactly what changed; destructive operations are permanent deletes, never overwrite; dry-run first for ambiguous owner requests.
- `Memory`: what goes into the daily log, `MEMORY.md` and `USER.md`; "rewrite in place, never keep superseded lines".
- No stock template text, no "read file X then file Y" chains, no tool catalogues, no example transcripts.

### SOUL.md and IDENTITY.md

Persona in two or three sentences; one playful line is enough for a notification. No lists of values, no back-story, no instructions (those belong in `AGENTS.md`).

### USER.md

- One directive per preference, prefixed with an HTML comment `<!-- observed: YYYY-MM-DD | status: active -->`.
- State the current rule only. Merge related preferences into one line (for example the whole subtitle policy is one line).
- Stay under the 4,000-char injection budget; anything beyond it is silently cut.

### MEMORY.md

Facts that a fresh session needs and cannot derive: paths and mounts, service names, library stats with a verification date, deployed script locations, job settings, known pitfalls with the date they were fixed. No procedures, no preferences, no rules.

### Convention file

Sections: layout, filenames, subtitles (or the domain equivalent), other material, gate conditions (what is protected), open decisions. Present tense, concrete examples with real paths. Mark undecided behaviour under "Open decisions" so the model leaves it alone.

## 3. Scripts

- Deployed under `/home/martin/.local/bin/<agent>-<task>.py` on the worker host and invoked via `node_exec` with absolute paths. Keep a source copy in the workspace (`media/` for Agent 007) and update both.
- Support `--dry-run` and keep runs idempotent.
- **Stdout is the model's input, so keep it small.** Print one compact JSON summary (counts plus the derived fields the prompt branches on) and write the full event log to `~/.local/state/<agent>/last-run.json`. Agent 007 went from 56 KB to about 1 KB of stdout.
- Derived fields do the thinking for the model: `video_moves` (count of video files that moved, including those inside a moved folder), `titles`, `moved_torrents` (inode- and size-verified), `incomplete_torrents` (count), `collisions`, `uncertain`, `events_file`, `recent_dms`.
- Verification belongs in the script (inode/size checks, "target exists → refuse"), not in the prompt.
- Never let the model carry long paths or arguments between steps: it shortened paths and split calls even when told to run them verbatim. Have the script write the follow-up commands into a shell file and print one short command (Agent 007's `cleanup_command`, `sh ~/.local/state/agent-007/cleanup.sh`).
- Delete permanently (`unlink`), never to Trash. Never overwrite an existing video.
- Notification dedup: the script that sends or logs a DM keeps `~/.local/state/<agent>/dm-history.txt` (last 40 texts); the organizer prints them as `recent_dms` so the prompt can demand different wording.
- Cross-check with upstream tools that touch the same files (for Agent 007: the fetch-api downloader already renames; the downloader was patched to skip samples and never rename onto an existing file).
- No explanatory comments in code; rationale goes in the commit message or reply.

## 4. Automation prompt

- Numbered steps, each naming the exact tool and the exact command. Target ≤ 1,500 chars.
- First line sets the mode: "Scheduled <job> run on <host>. Use the tools normally and report actual results; never retry a failed command."
- One tool call per step. When several shell commands belong to one step, say "in ONE node_exec call, joined with `;`" (each tool call is a model turn that re-reads the whole context).
- Fire-and-forget where possible: "trigger one scan; do not wait for it, poll it or check its result."
- Conditions reference numeric fields from the script output ("If `video_moves` is at least 1 … If 0, send nothing"), never judgement calls like "if something interesting happened".
- Notifications: exact call syntax, length cap, "send regardless of the other outcomes", must differ from every entry in `recent_dms` (name themes to vary), then log the exact text with the dm-log script.
- Slack formatting goes through Markdown conversion: `**bold**` renders bold, a single `*asterisk*` renders italic; line breaks are kept.
- Last step: "Reply with a short summary: counts, trigger result, cleanup results, DM sent (titles) or not, blockers."
- No sha or checksum verification steps, no re-reading of convention files unless needed, no "think about" instructions.

## 5. Job settings (`openclaw cron edit <id> …`)

| Setting | Value | Why |
|---|---|---|
| `--session isolated` | yes | runs do not pile into the main chat; `MEMORY.md` is not injected |
| `--tools` / `payload.toolsAllow` | minimal, e.g. `node_exec,message,read,jellyfin__*` | each tool schema is sent on every call; 75 entries cost about 6,000 tokens per call |
| `--light-context` | on | skips SOUL/IDENTITY/USER injection; the prompt must be self-sufficient |
| `--thinking low` | low | orchestration only; `off` is clamped to low on Sol |
| `--timeout-seconds` | 600 | normal run is 40–170 s |
| model | `openai/gpt-6-luna` | enough once the prompt is unambiguous |
| delivery | none | the run sends its own conditional DM |
| name/description | what it does (`tidy-jellyfin-library`) | the job id never changes |

MCP tool globs are `server__*` (for example `teslamate__*`), never `mcp__server__*`; unknown entries are ignored silently. Tool policy changes apply only to new sessions.

Chat and OpenAI-endpoint sessions get only today's date in the prompt (the clock time is reserved for cron runs). An agent that computes time windows must have `session_status` in its allowlist and be told to call it for the exact time; models convert raw epochs wrongly, so never make them derive "now" from `time()`.

## 6. Per-agent config (`agents.entries.<id>` in `openclaw.json`)

- Create the agent with `openclaw agents add <id> --workspace /home/node/.openclaw/workspace-<id> --agent-dir /home/node/.openclaw/agents/<id>/agent --model openai/gpt-6-luna --non-interactive`, then replace the seeded stock files and delete `BOOTSTRAP.md` (a first-run onboarding script none of the agents keep).
- `skills` is an allowlist for that agent only; an unset key means every eligible skill (20+ descriptions, about 5.5 KB per call). Never leave it unset or empty on a new agent: every agent gets at least the default set below. Do not change skills globally unless the owner says so.
- Default skills for every agent (owner decision 2026-10-14): `["summarize", "FFmpeg", "gog", "visualize", "diagram-maker", "meme-maker", "block-kit"]`. Append domain skills to this list; leave out `canvas` (macOS panel only). FFmpeg and gog live in the shared managed folder `/home/node/.openclaw/skills` (visible to all agents) and their CLIs plus `summarize` are installed by the `init-skills` init container into `/home/node/.openclaw/tools/bin` (on `PATH`); new shared skills go into its `SHARED_SKILLS` list, not into one agent's workspace.
- `tools.allow` for the agent's main chat session, `thinkingDefault` for the main session. Whenever an agent has a `tools.allow`, it must contain `show_widget`, `read`, `image` and `exec` in addition to its domain globs (owner decision 2026-10-14). On the Codex harness a finite allowlist disables Codex's native file and shell access, and OpenClaw never projects its own `read`, `write` or `edit` into Codex turns, so `exec` (the shell fallback in the OpenClaw pod) is the only way such an agent can read skill files, `FINANCE_PROCESS.md`-style workspace files, or run the summarize, FFmpeg and gog CLIs. Without it the agent guesses tool formats (for example a rejected `show_widget` call) and answers with text instead of charts.
- An agent with a filtered `tools.allow` receives the developer notice "Configured MCP is incomplete for this run: ... omitted by policy". Add one line to `AGENTS.md` saying the notice is expected and harmless, or the model reports its access as blocked.
- Use `openclaw config` or edit the live file; GitOps `openclaw.json` is deep-merged (`configMode: merge`), so keys set only live stay live.

## 7. Verify and measure

- Before changing anything: copy the live files into a proposal directory with `current/`, `proposed/`, `diff/`, `backup-before-apply/` and a README with a size table; apply only after the owner says so; back up the cron job definition with `openclaw cron list --json`.
- Dry-run the script, then trigger a manual run and read `openclaw cron runs --id <job> --json`: `usage.total_tokens` includes cached re-reads of the context for every turn, so it scales with the number of tool calls; `input_tokens` is the fresh context. The summary and the DM history file show whether the conditional notification fired.
- Expected numbers for a healthy run: 4–5 tool calls, 11–16k fresh input tokens, 35–80k total, 15–30 s.

## 8. Checklist for a new agent

1. Write the convention file and the scripts first; make the script print the decision fields the prompt needs.
2. Write `AGENTS.md` (≤ 2 KB) with real paths and exact tool syntax; `SOUL.md`, `IDENTITY.md` tiny; `USER.md` as dated directives; `MEMORY.md` facts only.
3. Create the cron job: isolated, light context, thinking low, minimal `toolsAllow`, delivery none, descriptive name, prompt ≤ 1,500 chars in numbered steps.
4. Set `agents.entries.<id>.skills` to the default set from section 6 (plus domain skills) and, if the agent has a chat session, `tools.allow` with its `server__*` globs plus `show_widget`, `read`, `image` and `exec`; add the "MCP incomplete notice is harmless" line to `AGENTS.md`. Verify with `openclaw gateway call tools.effective --params '{"sessionKey":"agent:<id>:main"}'` and reset the main session (`sessions.reset`) so the new policy applies.
5. Dry-run, manual run, check `cron runs --json`, confirm the notification path with the dedup log.
6. Commit the GitOps part (`<app>: <Change>`); keep workspace files on the PVC with dated backups.
