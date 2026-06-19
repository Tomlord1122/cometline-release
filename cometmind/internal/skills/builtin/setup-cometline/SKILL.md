---
name: setup-cometline
description: Help configure Cometline providers, model defaults, skills, delegation, memory, storage, and gateways.
---

# Setup Cometline

Use this skill when the user wants help configuring Cometline or CometMind. This includes initial setup, provider API keys, model selection, default model roles, Agent Skills, ACP delegation, Discord gateway settings, memory, storage, import/export, or troubleshooting a broken local setup.

## Workflow

1. Identify what the user wants to configure before changing anything. If the request is ambiguous, ask one focused question.
2. Prefer the Settings UI for user-facing configuration instructions. Mention exact sections such as Settings -> Providers, Settings -> CometMind -> Skills, Settings -> CometMind -> ACP, Settings -> CometMind -> Discord, or Settings -> CometMind -> Storage.
3. Prefer CLI commands when the user wants the agent to inspect or modify settings directly. Use `cometmind settings path`, `cometmind settings show`, `cometmind settings export`, and `cometmind settings import` when available.
4. Treat provider API keys and exported settings as secrets. Warn the user before printing, exporting, or moving files that may contain API keys.
5. After changing runtime settings that affect CometMind, tell the user whether a sidecar restart is expected. The desktop app restarts CometMind automatically for Settings UI saves.

## Provider Setup

Help the user choose a provider method, base URL, API key, enabled models, selected model, and default model roles. Environment variables such as `COMETMIND_PROVIDER`, `COMETMIND_BASE_URL`, `COMETMIND_API_KEY`, `COMETMIND_MODEL`, `ANTHROPIC_API_KEY`, and `OPENAI_API_KEY` can override runtime behavior, so check them when settings appear correct but runtime behavior differs.

## Skills Setup

CometMind discovers skills from configured roots, `~/.cometmind/skills`, workspace `.agents/skills`, workspace `.claude/skills`, and optional OpenCode or Claude skill roots. User-created skills should normally live under `~/.cometmind/skills/{skill-name}/SKILL.md` or the workspace `.agents/skills` directory.

Use `/create-skill` when the user wants the agent to create a reusable skill. The agent should use the `write_skill` tool rather than editing skill files manually when possible.

## Delegation Setup

ACP delegation is configured under Settings -> CometMind -> ACP. Use it when the user wants CometMind to hand coding work to OpenCode, Claude Code, or another ACP-compatible coding agent. Confirm the configured command, arguments, working directory behavior, and whether interactive mode is enabled.

## Discord Gateway

Discord gateway configuration lives under Settings -> CometMind -> Discord. Confirm the bot token environment variable name, workspace path, allowed user IDs, and mention requirements. Do not ask the user to paste bot tokens into chat unless there is no safer option.

## Memory And Storage

Memory and storage settings live under Settings -> CometMind. Explain what will be stored locally, what may be archived, and how provider/model choices affect extraction or summarization if the user is tuning memory behavior.

## Troubleshooting

If settings do not persist, check `~/.cometmind/cometline-settings.json` and file permissions. If the sidecar will not start, check `~/.cometmind/cometline.log`, port `7700`, and the packaged or development CometMind binary path.
