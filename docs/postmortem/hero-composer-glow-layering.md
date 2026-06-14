# Hero composer glow layering and animation

**Date:** 2026-06-14  
**Components:** `HeroComposerFrame.svelte`, `ChatView.svelte`, `+page.svelte`, `Composer.svelte`, `FirstTurnFlight.svelte`, `app.css`

## Symptom

Several iterations of the hero composer “pink aura” failed to match the design intent:

1. **Glow on the wrong layer** — aura sat on an outer shell or on `Composer.svelte` pseudos, misaligned with the white hero card (wrong radius, too large).
2. **Wrong sequence** — border ring appeared before or at the same time as the glow; design called for glow to rise first, then the red border after the glow wraps the card.
3. **Lost “rise from bottom” feel** — after splitting ring and glow, the glow only faded in with `opacity`, with no upward motion.
4. **First-turn exit out of sync** — on first send, glow ran its own sink animation while `composer-wrapper` moved on a separate 560ms `bottom`/`transform` transition; aura and composer felt like two beats.
5. **Giant static wash at rest** — glow filled a tall box from viewport bottom to above the composer; at animation end the full box stayed visible, so the hero looked like a large background gradient instead of a tight halo around the card.

## Root cause

### 1. Layer ownership

Hero aura is **not** composer chrome (padding, border-radius, shadow). It is a **frame effect** that must:

- sit outside `Composer.svelte` (so dock variant does not carry glow CSS),
- align to the hero card’s 24px radius and width,
- mount in both home (`+page.svelte`) and session hero (`ChatView.svelte`).

Putting `::before`/`::after` on `.composer.hero` worked for alignment but mixed flight/dock styling with decorative animation. The durable split is `HeroComposerFrame.svelte` wrapping `<Composer variant="hero" />`.

### 2. One element used for both “travel distance” and “rest size”

To make glow “burst from the UI bottom”, an intermediate fix **extended the glow element** downward:

```css
/* broken resting geometry */
bottom: calc(-1 * var(--hero-glow-travel) - 10px);
transform: scaleY(1); /* after enter */
```

`--hero-glow-travel` is the pixel distance from the frame bottom to `.chat-home` bottom (often 200–400px on centered session hero). With `scaleY(1)`, radial gradients painted across that entire height — strong wash above and below the composer, detached from the card edge.

**Animation geometry ≠ resting geometry.** Rising from the viewport bottom is an enter path; the hold state must stay tight (`inset: -16px -12px -10px`).

### 3. Competing motion on first turn

`HeroComposerFrame` `{#if active}` removed ring/glow instantly when `composerVariant` became `dock`. Later, separate exit keyframes (`translateY` sink, 500ms) ran while `.composer-wrapper` already transitioned over `--duration-flight` (560ms). Two motion systems on different clocks.

### 4. Enter started before layout was measured

Glow animation began with `--hero-glow-travel: 0` before `getBoundingClientRect()`, so the first frame could flash from the wrong origin until `ResizeObserver` updated.

## Fix (applied)

### Architecture: `HeroComposerFrame.svelte`

Two layers, both outside `Composer.svelte`:

| Layer | Role | Enter timing |
| ----- | ---- | ------------ |
| `.hero-composer-glow` | Pink radial blur + outer ring shadow | 0 → 0.65s: `translateY(travel)` + `scaleY` from bottom |
| `.hero-composer-ring` | 1px red border, `clip-path` rise | 0.55 → 1.0s (delay + 0.45s) |

Tokens in `app.css`:

```css
--duration-hero-sequence: 1s;
--duration-hero-glow-rise: 0.65s;
--duration-hero-ring-delay: 0.55s;
--duration-hero-ring-rise: 0.45s;
--duration-hero-exit-ring: 0.24s;
```

### Enter: tight box, travel via transform

- Glow box stays **composer-sized** (`inset: -16px -12px -10px`).
- Measure `glowTravelPx` = `.chat-home` bottom − frame bottom; set `--hero-glow-travel`.
- Wait for measure (`glowReady`) before starting animations.
- Enter: `translateY(var(--hero-glow-travel)) scaleY(0.35)` → `translateY(0) scaleY(1)` with `transform-origin: center bottom`.

### First-turn exit: move with wrapper, don’t sink independently

In `ChatView.svelte`:

```typescript
onPrepareFlight={() => {
  if (composerVariant === 'hero') heroFrameExiting = true;
  shellStore.dockComposer();
}}
```

`HeroComposerFrame` props:

- `active={composerVariant === 'hero' && !heroFrameExiting}`
- `exiting={heroFrameExiting}` — keeps layers mounted until outro finishes
- `onExitComplete` clears `heroFrameExiting`

Exit behavior:

- **Ring:** quick `clip-path` collapse (`--duration-hero-exit-ring`, 240ms).
- **Glow:** **opacity only** over `--duration-flight` (560ms) — no `translateY` exit. Physical motion comes from `.composer-wrapper`’s `bottom`/`transform` transition (`overflow: visible` on wrapper).

Do **not** set `--hero-glow-travel: 0` on exit to “collapse” the box; that was tied to the broken extended-height model.

## How to avoid regressions

- **Do not put hero aura back on `Composer.svelte`.** Dock variant must not inherit glow pseudos or animation.

- **Do not extend glow `bottom` by `--hero-glow-travel` for the resting state.** Use `translateY(travel)` only in enter keyframes on a tight inset box.

- **Enter order:** glow rise completes wrapping → then ring (`--duration-hero-ring-delay` ≈ glow rise − ring rise overlap).

- **First-turn exit:** one motion source for position — `composer-wrapper` + `--duration-flight`. Glow exits with opacity (and optional ring clip), not a second vertical animation.

- **Measure before animate:** gate ring/glow with `glowReady` after `measureGlowTravel()`; observe frame + `.chat-home` with `ResizeObserver`.

- **Related postmortem:** composer dock timing and `onPrepareFlight` are documented in [hero-composer-dock-transition-jank.md](./hero-composer-dock-transition-jank.md). Changing dock hooks or `--duration-flight` affects glow exit sync.

- **Home vs session:** travel distance differs (grid vs absolute hero). Always measure from DOM; don’t hard-code pixel travel.

## Verification

1. **Home (`/`)** — refresh: glow rises from bottom, tight halo at rest; ring appears ~0.55s in; total enter ≤ 1s.
2. **Empty session hero** — same enter; halo hugs card, not a full-width background wash.
3. **First send** — ring collapses quickly; glow fades while composer docks; no independent “sink” lagging behind the card.
4. **Docked / existing session** — no glow (`active={false}`); no flash on load.
5. **Resize** — re-measure; enter not required again until remount.
6. **`prefers-reduced-motion`** — glow and ring show final state; exit clears without waiting on keyframes.
