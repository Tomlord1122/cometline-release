# macOS tray icon oversized and gray

**Date:** 2026-06-16  
**Components:** `electron/main.cjs`, `buildResources/trayIcon.png`, `buildResources/trayTemplate.png`, `package.json`

## Symptom

The Cometline menu bar icon on macOS was visibly larger than the surrounding system tray icons and appeared as a gray/white blob instead of the project avatar.

## Root cause

`buildResources/trayTemplate.png` was a colored anime-style avatar, but the Electron main process treated any file ending in `trayTemplate.png` as a macOS template image. `setTemplateImage(true)` turned the colored artwork into a monochrome mask, producing the gray blob.

The asset was also `44×44` pixels and the code skipped resizing for template assets, so it was rendered at its native pixel size — larger than the standard macOS menu bar icon size of 16pt (32px on Retina displays).

```javascript
// electron/main.cjs (before)
if (process.platform === 'darwin') {
    const isTemplateAsset = candidate.endsWith('trayTemplate.png');
    if (!isTemplateAsset) {
        image = image.resize({ width: 22, height: 22, quality: 'best' });
    }
    image.setTemplateImage(true); // always applied on macOS
    return image;
}
```

The `22×22` resize for non-template fallbacks was also inconsistent with the code comment that said 16pt / 32px Retina backing.

## Fix

### 1. Dedicated colored tray icon

Generated `buildResources/trayIcon.png` (32×32) from `static/project_avatar_96.png` so the menu bar shows a small, recognizable colored avatar instead of a template mask.

```bash
sips -z 32 32 static/project_avatar_96.png --out buildResources/trayIcon.png
```

### 2. Prefer the colored icon in candidate resolution

`resolveTrayIconCandidates()` now lists `trayIcon.png` first, with `trayTemplate.png` kept as a backward-compatible fallback.

### 3. Resize every macOS candidate to the correct size

Template assets are no longer exempt from resizing. Every macOS candidate is now resized to `32×32` so the rendered icon matches the standard 16pt menu bar size on Retina displays.

### 4. Only template actual template assets

`setTemplateImage(true)` is now applied only when the candidate filename ends with `Template.png`. `trayIcon.png` remains colored.

```javascript
// electron/main.cjs (after)
if (process.platform === 'darwin') {
    const isTemplateAsset = candidate.endsWith('trayTemplate.png');
    // macOS menu bar icons read best at 16pt (32px backing on Retina).
    image = image.resize({ width: 32, height: 32, quality: 'best' });
    if (image.isEmpty()) continue;
    if (isTemplateAsset) {
        image.setTemplateImage(true);
    }
    return image;
}
```

### 5. Bundle the new asset

`package.json` `extraResources` now copies `buildResources/trayIcon.png` into the app bundle so the colored icon is available in packaged builds.

## If changing the tray icon later

| What to change | File | Notes |
| -------------- | ---- | ----- |
| Colored avatar source | `static/project_avatar_96.png` | Regenerate `buildResources/trayIcon.png` with `sips -z 32 32`. |
| Monochrome menu bar icon | `buildResources/trayTemplate.png` | Must be a true template (single color with alpha). Keep the `Template.png` suffix so `setTemplateImage(true)` is applied. |
| Candidate order | `electron/main.cjs` | First-match wins; put the desired icon first. |
| Menu bar icon size | `electron/main.cjs` | Keep `32×32` for Retina 16pt unless providing both 1x and 2x assets. |
| Packaged asset inclusion | `package.json` | Add any new tray icon to `extraResources`. |

## How to avoid regressions

- Do not name a colored image `*Template.png` and then call `setTemplateImage(true)` on it. macOS will treat it as a mask and render it as a monochrome blob.

- Do not skip resizing for template assets. All menu bar icons should render at the same logical size regardless of the source asset's pixel dimensions.

- Keep the resize dimension aligned with the comment. If the comment says 16pt (32px Retina), resize to `32×32`, not `22×22`.

- When adding a new tray icon, add it to both the dev candidate list and `package.json` `extraResources` so packaged builds can find it.

## Verification

1. Run the desktop app with `pnpm dev` on a Retina Mac; the menu bar icon should be the same size as neighboring system icons.
2. The icon should display in color, not as a gray/white silhouette.
3. Package the app (`pnpm build:mac`) and launch the `.app`; the menu bar icon should still be the colored avatar at the correct size.
4. Temporarily delete `buildResources/trayIcon.png`; the app should fall back to `trayTemplate.png` and render it as a smaller monochrome blob (not oversized).
5. `node --check electron/main.cjs` should pass.
