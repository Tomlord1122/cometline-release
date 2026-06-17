# Session switch: in-flight response lost, stuck, or replayed from scratch

**Date:** 2026-06-17  
**Components:** `conversation-controller.ts`, `ChatView.svelte`, `ChatThread.svelte`, `AssistantMarkdown.svelte`, `chat.svelte.ts`, `chat.svelte.test.ts`, `conversation-controller.test.ts`

## Symptom

While Session A was waiting on the LLM (reasoning or partial response streaming over SSE), switching to Session B and sending there, then switching back to Session A caused several failures:

1. **Response vanished** — only the user bubble remained (e.g. `"write a 1000 words joke"`), with no assistant/reasoning content. The session felt **stuck** until the user sent a **second** message on A.
2. **Early switch before SSE** — sending on A and switching away before reasoning started had the same stuck/orphan behavior.
3. **Response replayed on return** — when content did appear after switching back during an in-flight stream, the assistant text **re-rendered from empty** (typewriter catch-up) instead of showing the accumulated text immediately; only **new** tokens should animate.

Desired behavior: sessions stream **concurrently in the background**; switching away and back should show the **current** partial or completed transcript without re-fetch clobbering cache or replaying already-streamed text.

## Root cause

Four independent bugs compounded; two affected **data**, two affected **presentation**.

### 1. Turn bound to live `sessionId` (orphaned send)

`ChatView` passed the **live** `sessionId` prop into `chatStore.send` / `refreshConversationSession`:

```ts
send: (payload, opts) => chatStore.send(sessionId, payload, opts),
```

`runTurn` is async (user-bubble flight → `await send`). If the user switched to Session B before the flight finished, `send()` ran against **B's id** while the queued turn belonged to **A**. A's SSE never started correctly.

### 2. `loadTranscript` clobbered in-flight cache on return

`onMount()` always called `loadTranscript`, while `$effect.pre` used `shouldSkipTranscriptLoad()` — **two inconsistent paths**.

When returning to A:

- Background stream might be finished or `isStreamingFor(A)` false while server had not persisted the assistant turn yet.
- `loadTranscript` fetched a **user-only** transcript and `writeSessionItems` **overwrote** the cache that still held partial reasoning/response.

That matches the screenshot symptom and explains why a **second send** “fixed” A (new stream + `preAssistant`, not reliance on clobbered cache).

### 3. `bindSession` saved stale visible `items`

On leave, `bindSession` did `sessionCache.set(sessionID, items)` using the **visible** `items` ref, not `getCachedItems(sessionID)`. Background stream updates wrote to `sessionCache`; visible `items` could lag, producing snapshots missing assistant/reasoning rows.

### 4. `AssistantMarkdown` remount replayed typewriter from empty

`ChatThread` uses `{#each threadItems as item (item.id)}`. Switching sessions destroys A's DOM nodes and recreates them on return. `AssistantMarkdown` remounted with `displaySource = ''` while `streaming={true}`, so the reveal animation **replayed from scratch** even though `source` already held kilobytes of text.

Separately, `isInitialTranscriptPaint` could run hydration (opacity 0 → settle) when returning to a session that already had cached items, adding a “whole transcript fading in again” feel.

## Fix

### Data layer (`chat.svelte.ts`, `conversation-controller.ts`, `ChatView.svelte`)

1. **Capture turn session at enqueue** — `ensureQueue()` stores `queueForSessionId`; `runTurn` calls `deps.send(turnSessionId, …)` and `deps.refreshSession(turnSessionId)`. ChatView deps take explicit `sid`.
2. **`hasInFlightTurn(sessionId)`** — true when streaming, a stream handle exists, or cache has a pending assistant/reasoning item. Used by `shouldSkipTranscriptLoad`, `loadTranscript` early/post-fetch guards, and `onMount`.
3. **`bindSession` leave snapshot** — flush batch, then `sessionCache.set(sessionID, getCachedItems(sessionID))`.
4. **`send()` cleanup** — throw when `isStreamingFor` blocks duplicate send; abort handle on run mismatch; `finally` only finishes when `streamHandles.get(sessionId) === handle`.
5. **UI state on switch** — `awaitingFirstAssistant` restored via `chatStore.isAwaitingFirstAssistant(sessionId)` instead of unconditional reset.

### Presentation layer (`AssistantMarkdown.svelte`, `ChatThread.svelte`)

1. **Mount snap** — on first mount with non-empty `source`, set `displaySource = source` once (`snappedExistingSource`). Remount mid-stream shows accumulated text immediately; only **new** deltas use the typewriter reveal.
2. **Skip transcript hydration when cache exists** — `sessionHasCachedTranscript()` uses `getCachedItemCount`, `isStreamingFor`, and `hasInFlightTurn` so returning to A does not run `isInitialTranscriptPaint` settle/fade on already-hydrated content.

## How to avoid regressions

- **Never close over live route/prop session id inside async turn pipelines.** Capture session id when the turn is enqueued or when the per-session queue is created.
- **One gate for transcript load.** `onMount`, `$effect.pre`, and `loadTranscript` must share the same skip rules; prefer `hasInFlightTurn` over `isStreamingFor && items.length` alone.
- **Cache is authoritative for background streams.** On `bindSession` leave, persist `getCachedItems`, not visible `items`.
- **Server transcript lags in-flight SSE.** Do not overwrite local cache with fetch results while a turn is in flight or pending assistant rows exist locally.
- **Keyed `{#each}` remounts children on session switch.** Any streaming UI with local animation state (`displaySource`, typewriter) must **snap to current `source` on mount** when remounting mid-stream.
- **Fresh mount ≠ empty session.** If `getCachedItemCount(sessionId) > 0` or the session is streaming, skip `isInitialTranscriptPaint` hydration.

## Verification

1. `cd cometline && pnpm run test` — includes session-switch, in-flight clobber, turn-session capture, and duplicate-send throw cases.
2. Manual:
   - Session A: send a long prompt; wait for reasoning or partial text.
   - Switch to B; send another question; wait for B to respond.
   - Switch back to A → A's reasoning/response visible **immediately** at current length, continuing to update if still streaming; **no** user-only stuck state; **no** full typewriter replay from empty.

## Related postmortems

- [session-switch-slow-and-stuck-load.md](./session-switch-slow-and-stuck-load.md) — superseded load / empty snapshot on rapid switch (load promise cleanup, `snapshotLoading`).
- [composer-phase-and-positioning.md](./composer-phase-and-positioning.md) — composer phase vs transcript load timing.
- [streaming-ui-not-live-updating.md](./streaming-ui-not-live-updating.md) — `$state.raw` + immutable item updates for live SSE rendering.
