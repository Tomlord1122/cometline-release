<script lang="ts">
	import { onDestroy, tick } from 'svelte';
	import type { CaretTrailSettings } from '$lib/types';
	import { faviconUrl, domainFromUrl, isHttpUrl } from '$lib/markdown/embed';
	import { openLink } from '$lib/open-link';

	let {
		value = $bindable(''),
		placeholder = '',
		ariaLabel = 'Message input',
		skillNames = [],
		caretTrail = { enabled: true, intensity: 0.72, speed: 0.68 },
		caretColor = '#72c0ff',
		onkeydown,
		onfiles
	}: {
		value?: string;
		placeholder?: string;
		ariaLabel?: string;
		skillNames?: string[];
		caretTrail?: CaretTrailSettings;
		caretColor?: string;
		onkeydown?: (e: KeyboardEvent) => void;
		onfiles?: (files: File[]) => void;
	} = $props();

	type TrailPoint = { x: number; y: number; t: number };

	let wrap = $state<HTMLDivElement | null>(null);
	let editor = $state<HTMLDivElement | null>(null);
	let customCaret = $state<HTMLSpanElement | null>(null);
	let trailPath = $state<SVGPathElement | null>(null);
	let focused = $state(false);
	// Guard so our own DOM writes don't recursively re-trigger input handling.
	let syncing = false;
	// IME composition guard — Enter during candidate selection must not trigger send.
	let composing = false;
	let raf = 0;
	let caretReady = $state(false);
	let currentX = 0;
	let currentY = 0;
	let targetX = 0;
	let targetY = 0;
	let lastMeasuredX = 0;
	let lastMeasuredY = 0;
	let lastMeasuredAt = 0;
	let trailPoints: TrailPoint[] = [];

	let caretTrailEnabled = $derived(caretTrail.enabled);
	let caretStyle = $derived(
		`--rce-caret-color: ${caretColor}; --rce-trail-opacity: ${0.28 + caretTrail.intensity * 0.42}`
	);

	/**
	 * Serializes the contenteditable DOM back to plain text. URL chips serialize
	 * to their full URL (stored in data-url); <br>/block boundaries become \n.
	 */
	function serialize(root: HTMLElement): string {
		let out = '';
		const walk = (node: Node) => {
			for (const child of Array.from(node.childNodes)) {
				if (child.nodeType === Node.TEXT_NODE) {
					out += child.textContent ?? '';
				} else if (child instanceof HTMLElement) {
					if (child.dataset.url) {
						out += child.dataset.url;
					} else if (child.dataset.skillCommand) {
						out += child.dataset.skillCommand;
					} else if (child.tagName === 'BR') {
						out += '\n';
					} else {
						const isBlock = /^(DIV|P)$/.test(child.tagName);
						if (isBlock && out && !out.endsWith('\n')) out += '\n';
						walk(child);
					}
				}
			}
		};
		walk(root);
		return out;
	}

	function readValue() {
		if (!editor) return;
		value = serialize(editor);
	}

	function clampUnit(value: number): number {
		if (!Number.isFinite(value)) return 0;
		return Math.min(1, Math.max(0, value));
	}

	function setCaretVisual(x: number, y: number) {
		if (!customCaret) return;
		customCaret.style.transform = `translate3d(${x}px, ${y}px, 0)`;
	}

	function setTrailVisual(now: number) {
		if (!trailPath) return;
		const lifetime = 120 + clampUnit(caretTrail.intensity) * 420;
		trailPoints = trailPoints.filter((point) => now - point.t <= lifetime);
		if (trailPoints.length < 2) {
			trailPath.setAttribute('d', '');
			return;
		}
		trailPath.setAttribute(
			'd',
			trailPoints
				.map((point, index) => `${index === 0 ? 'M' : 'L'} ${point.x.toFixed(1)} ${point.y.toFixed(1)}`)
				.join(' ')
		);
	}

	function measureCaret() {
		if (!wrap || !editor || !caretTrailEnabled || !focused) return;
		const selection = window.getSelection();
		if (!selection || selection.rangeCount === 0 || !selection.isCollapsed) {
			resetCaretTrail();
			return;
		}
		const range = selection.getRangeAt(0);
		const container = range.commonAncestorContainer;
		if (container !== editor && !editor.contains(container)) {
			resetCaretTrail();
			return;
		}

		const rect = range.getClientRects()[0] ?? range.getBoundingClientRect();
		const wrapRect = wrap.getBoundingClientRect();
		const editorRect = editor.getBoundingClientRect();
		const lineHeight = Number.parseFloat(getComputedStyle(editor).lineHeight) || 22.5;
		const measuredX = rect && rect.left ? rect.left - wrapRect.left : editorRect.left - wrapRect.left;
		const measuredY = rect && rect.top ? rect.top - wrapRect.top : editorRect.top - wrapRect.top;
		const now = performance.now();
		const dt = lastMeasuredAt > 0 ? Math.max(8, now - lastMeasuredAt) : 16;
		const vx = (measuredX - lastMeasuredX) / dt;
		const vy = (measuredY - lastMeasuredY) / dt;
		const predictionMs = 14 + clampUnit(caretTrail.speed) * 30;
		targetX = measuredX + vx * predictionMs;
		targetY = measuredY + vy * predictionMs;
		lastMeasuredX = measuredX;
		lastMeasuredY = measuredY;
		lastMeasuredAt = now;
		if (customCaret) customCaret.style.height = `${lineHeight}px`;
		startCaretAnimation();
	}

	function animateCaret(now: number) {
		const follow = 0.2 + clampUnit(caretTrail.speed) * 0.52;
		currentX += (targetX - currentX) * follow;
		currentY += (targetY - currentY) * follow;
		setCaretVisual(currentX, currentY);
		trailPoints.push({ x: currentX + 1, y: currentY + 11, t: now });
		if (trailPoints.length > 42) trailPoints = trailPoints.slice(-42);
		setTrailVisual(now);

		if (Math.abs(targetX - currentX) > 0.35 || Math.abs(targetY - currentY) > 0.35) {
			raf = requestAnimationFrame(animateCaret);
		} else {
			raf = 0;
		}
	}

	function startCaretAnimation() {
		if (!caretTrailEnabled || !focused) return;
		if (!caretReady) {
			currentX = targetX;
			currentY = targetY;
			caretReady = true;
			setCaretVisual(currentX, currentY);
		}
		if (!raf) raf = requestAnimationFrame(animateCaret);
	}

	function resetCaretTrail() {
		if (raf) cancelAnimationFrame(raf);
		raf = 0;
		caretReady = false;
		trailPoints = [];
		trailPath?.setAttribute('d', '');
	}

	function scheduleCaretMeasure() {
		if (!caretTrailEnabled) return;
		requestAnimationFrame(measureCaret);
	}

	/** Build a non-editable inline chip element for a URL. */
	function makeChip(url: string): HTMLElement {
		const chip = document.createElement('span');
		chip.className = 'rce-chip';
		chip.contentEditable = 'false';
		chip.dataset.url = url;
		chip.title = url;

		const img = document.createElement('img');
		img.className = 'rce-chip-icon';
		img.src = faviconUrl(url);
		img.alt = '';
		img.width = 14;
		img.height = 14;
		img.addEventListener('error', () => (img.style.visibility = 'hidden'));

		const label = document.createElement('span');
		label.className = 'rce-chip-label';
		label.textContent = domainFromUrl(url);

		chip.appendChild(img);
		chip.appendChild(label);
		return chip;
	}

	function makeSkillChip(name: string): HTMLElement {
		const chip = document.createElement('span');
		chip.className = 'rce-chip rce-skill-chip';
		chip.contentEditable = 'false';
		chip.dataset.skillCommand = `/${name}`;
		chip.title = `Use the ${name} skill`;

		const label = document.createElement('span');
		label.className = 'rce-chip-label';
		label.textContent = `/${name}`;
		chip.appendChild(label);
		return chip;
	}

	function escapeRegex(value: string): string {
		return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	}

	function skillNameRegex() {
		const names = skillNames.map((name) => name.trim()).filter(Boolean);
		if (names.length === 0) return null;
		const pattern = names.sort((a, b) => b.length - a.length).map(escapeRegex).join('|');
		return new RegExp(`(^|\\s)\\/(${pattern})(?=\\s|$)`, 'g');
	}

	/**
	 * Scans text nodes in the editor and replaces any complete bare URL with a
	 * chip. Only runs on text the user isn't actively typing the tail of (we
	 * require a trailing boundary char or that the URL isn't at the caret end).
	 */
	function linkifyEditor(opts?: { allowCaretEnd?: boolean }) {
		const allowCaretEnd = opts?.allowCaretEnd ?? false;
		if (!editor) return;
		const urlRe = /https?:\/\/[^\s<]+/g;
		const walker = document.createTreeWalker(editor, NodeFilter.SHOW_TEXT);
		const textNodes: Text[] = [];
		let n = walker.nextNode();
		while (n) {
			// Skip text already inside a chip.
			if (!(n.parentElement && n.parentElement.closest('.rce-chip'))) {
				textNodes.push(n as Text);
			}
			n = walker.nextNode();
		}

		const sel = window.getSelection();
		const caretNode = sel && sel.rangeCount > 0 ? sel.focusNode : null;
		const caretOffset = sel && sel.rangeCount > 0 ? sel.focusOffset : 0;

		let didChange = false;
		for (const textNode of textNodes) {
			const text = textNode.textContent ?? '';
			urlRe.lastIndex = 0;
			let match: RegExpExecArray | null = urlRe.exec(text);
			if (!match) continue;

			// Build replacement fragment.
			const frag = document.createDocumentFragment();
			let cursor = 0;
			let replaced = false;
			urlRe.lastIndex = 0;
			while ((match = urlRe.exec(text)) !== null) {
				const start = match.index;
				const end = start + match[0].length;
				// Don't chipify a URL the caret is still typing at the end of —
				// unless this is a paste, where we chipify immediately.
				const caretInThisNode = caretNode === textNode;
				const caretAtUrlEnd = caretInThisNode && caretOffset === end;
				if (caretAtUrlEnd && !allowCaretEnd) continue;
				let url = match[0];
				const trailing = /[.,;:!?)\]}'"]+$/.exec(url);
				if (trailing) url = url.slice(0, url.length - trailing[0].length);
				if (!isHttpUrl(url)) continue;

				if (start > cursor)
					frag.appendChild(document.createTextNode(text.slice(cursor, start)));
				frag.appendChild(makeChip(url));
				const suffix = match[0].slice(url.length);
				if (suffix) frag.appendChild(document.createTextNode(suffix));
				cursor = end;
				replaced = true;
			}
			if (!replaced) continue;
			if (cursor < text.length) frag.appendChild(document.createTextNode(text.slice(cursor)));

			// Append a trailing space + place caret after the inserted content so
			// typing continues normally after a chip.
			const trailingSpace = document.createTextNode('\u00a0');
			frag.appendChild(trailingSpace);
			textNode.replaceWith(frag);
			didChange = true;

			// Restore caret to just after the trailing space.
			const range = document.createRange();
			range.setStartAfter(trailingSpace);
			range.collapse(true);
			sel?.removeAllRanges();
			sel?.addRange(range);
		}
		return didChange;
	}

	function skillifyEditor(opts?: { allowCaretEnd?: boolean }) {
		const re = skillNameRegex();
		if (!editor || !re) return;
		const allowCaretEnd = opts?.allowCaretEnd ?? false;
		const walker = document.createTreeWalker(editor, NodeFilter.SHOW_TEXT);
		const textNodes: Text[] = [];
		let n = walker.nextNode();
		while (n) {
			if (!(n.parentElement && n.parentElement.closest('.rce-chip'))) {
				textNodes.push(n as Text);
			}
			n = walker.nextNode();
		}

		const sel = window.getSelection();
		const caretNode = sel && sel.rangeCount > 0 ? sel.focusNode : null;
		const caretOffset = sel && sel.rangeCount > 0 ? sel.focusOffset : 0;

		let didChange = false;
		for (const textNode of textNodes) {
			const text = textNode.textContent ?? '';
			re.lastIndex = 0;
			if (!re.test(text)) continue;

			const frag = document.createDocumentFragment();
			let cursor = 0;
			let replaced = false;
			re.lastIndex = 0;
			let match: RegExpExecArray | null;
			while ((match = re.exec(text)) !== null) {
				const prefix = match[1] ?? '';
				const name = match[2];
				const commandStart = match.index + prefix.length;
				const commandEnd = commandStart + name.length + 1;
				const caretInThisNode = caretNode === textNode;
				const caretAtCommandEnd = caretInThisNode && caretOffset === commandEnd;
				const hasTrailingBoundary = commandEnd < text.length && /\s/.test(text[commandEnd]);
				if (caretAtCommandEnd && !allowCaretEnd && !hasTrailingBoundary) continue;

				if (commandStart > cursor) {
					frag.appendChild(document.createTextNode(text.slice(cursor, commandStart)));
				}
				frag.appendChild(makeSkillChip(name));
				cursor = commandEnd;
				replaced = true;
			}
			if (!replaced) continue;
			if (cursor < text.length) frag.appendChild(document.createTextNode(text.slice(cursor)));

			textNode.replaceWith(frag);
			didChange = true;
		}
		if (didChange) {
			const range = document.createRange();
			range.selectNodeContents(editor);
			range.collapse(false);
			sel?.removeAllRanges();
			sel?.addRange(range);
		}
		return didChange;
	}

	function decorateEditor(opts?: { allowCaretEnd?: boolean }) {
		const didSkillify = skillifyEditor(opts);
		const didLinkify = linkifyEditor(opts);
		return didSkillify || didLinkify;
	}

	function onInput() {
		if (syncing || !editor) return;
		syncing = true;
		decorateEditor();
		syncing = false;
		readValue();
		scheduleCaretMeasure();
	}

	function onPaste(e: ClipboardEvent) {
		const files = Array.from(e.clipboardData?.files ?? []).filter((file) =>
			file.type.startsWith('image/')
		);
		if (files.length > 0) {
			e.preventDefault();
			onfiles?.(files);
			return;
		}

		// Force plain-text paste so we don't inherit foreign HTML, then linkify
		// immediately so a pasted URL becomes a chip without needing an extra
		// keystroke.
		const text = e.clipboardData?.getData('text/plain');
		if (text == null) return;
		e.preventDefault();
		document.execCommand('insertText', false, text);
		if (!editor) return;
		syncing = true;
		decorateEditor({ allowCaretEnd: true });
		syncing = false;
		readValue();
		scheduleCaretMeasure();
	}

	function onCompositionStart() {
		composing = true;
	}

	function onCompositionEnd() {
		// Defer reset: some browsers fire the confirming keydown after compositionend.
		setTimeout(() => {
			composing = false;
			scheduleCaretMeasure();
		}, 0);
	}

	function onKeydownInternal(e: KeyboardEvent) {
		if (composing || e.isComposing) return;
		onkeydown?.(e);
	}

	function onEditorClick(e: MouseEvent) {
		const target = e.target;
		if (!(target instanceof Element)) return;
		const chip = target.closest('.rce-chip');
		if (!(chip instanceof HTMLElement) || !chip.dataset.url) return;
		// A plain click on a chip opens its link.
		e.preventDefault();
		openLink(chip.dataset.url);
	}

	function onFocus() {
		focused = true;
		scheduleCaretMeasure();
	}

	function onBlur() {
		focused = false;
		resetCaretTrail();
	}

	function insertPlainText(text: string) {
		if (!editor) return;
		focus();
		document.execCommand('insertText', false, text);
		readValue();
		scheduleCaretMeasure();
	}

	export function focus() {
		editor?.focus({ preventScroll: true });
		// Move caret to end.
		if (editor) {
			const range = document.createRange();
			range.selectNodeContents(editor);
			range.collapse(false);
			const sel = window.getSelection();
			sel?.removeAllRanges();
			sel?.addRange(range);
		}
		scheduleCaretMeasure();
	}

	export async function focusAsync() {
		await tick();
		focus();
	}

	export function insertText(text: string) {
		insertPlainText(text);
	}

	export function setText(text: string) {
		if (!editor) {
			value = text;
			return;
		}
		editor.textContent = text;
		value = text;
		focus();
		syncing = true;
		decorateEditor({ allowCaretEnd: true });
		syncing = false;
		readValue();
		scheduleCaretMeasure();
	}

	/** Clears the editor (used after send). */
	export function clear() {
		if (editor) editor.innerHTML = '';
		value = '';
		resetCaretTrail();
	}

	// Keep the DOM in sync when `value` is set externally to empty (e.g. cleared
	// after send). We only handle the clear case to avoid clobbering chips.
	$effect(() => {
		if (value === '' && editor && editor.textContent !== '') {
			editor.innerHTML = '';
		}
	});

	$effect(() => {
		const key = skillNames.join('\n');
		if (!editor || key === '') return;
		syncing = true;
		decorateEditor();
		syncing = false;
		readValue();
		scheduleCaretMeasure();
	});

	$effect(() => {
		if (!caretTrailEnabled) {
			resetCaretTrail();
			return;
		}
		const onSelectionChange = () => scheduleCaretMeasure();
		const onResize = () => scheduleCaretMeasure();
		document.addEventListener('selectionchange', onSelectionChange);
		window.addEventListener('resize', onResize);
		scheduleCaretMeasure();
		return () => {
			document.removeEventListener('selectionchange', onSelectionChange);
			window.removeEventListener('resize', onResize);
			resetCaretTrail();
		};
	});

	onDestroy(resetCaretTrail);

	let isEmpty = $derived(value.trim() === '');
</script>

<div bind:this={wrap} class="rce-wrap" style={caretStyle}>
	{#if isEmpty}
		<div class="rce-placeholder" aria-hidden="true">{placeholder}</div>
	{/if}
	{#if caretTrailEnabled}
		<div class="rce-caret-layer" class:visible={focused && caretReady} aria-hidden="true">
			<svg class="rce-trail" focusable="false">
				<path bind:this={trailPath}></path>
			</svg>
			<span bind:this={customCaret} class="rce-caret"></span>
		</div>
	{/if}
	<div
		bind:this={editor}
		class="rce-editor"
		class:trail-enabled={caretTrailEnabled}
		contenteditable="true"
		role="textbox"
		tabindex="0"
		aria-multiline="true"
		aria-label={ariaLabel}
		oninput={onInput}
		onpaste={onPaste}
		onkeydown={onKeydownInternal}
		oncompositionstart={onCompositionStart}
		oncompositionend={onCompositionEnd}
		onclick={onEditorClick}
		onfocus={onFocus}
		onblur={onBlur}
	></div>
</div>

<style>
	.rce-wrap {
		position: relative;
		width: 100%;
	}

	.rce-placeholder {
		position: absolute;
		inset: 0;
		pointer-events: none;
		color: var(--text-soft);
		font-size: 15px;
		line-height: 1.5;
		white-space: pre-wrap;
	}

	.rce-editor {
		width: 100%;
		min-height: calc(1.5em * 3);
		max-height: calc(1.5em * 8);
		overflow-y: auto;
		font-size: 15px;
		line-height: 1.5;
		color: var(--text-main);
		outline: none;
		white-space: pre-wrap;
		word-break: break-word;
		font-family: inherit;
	}

	.rce-editor.trail-enabled {
		caret-color: transparent;
	}

	.rce-caret-layer {
		position: absolute;
		inset: 0;
		pointer-events: none;
		z-index: 2;
		overflow: hidden;
		opacity: 0;
		transition: opacity 0.08s ease;
	}

	.rce-caret-layer.visible {
		opacity: 1;
	}

	.rce-trail {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		overflow: visible;
	}

	.rce-trail path {
		fill: none;
		stroke: var(--rce-caret-color);
		stroke-width: 2.5;
		stroke-linecap: round;
		stroke-linejoin: round;
		opacity: var(--rce-trail-opacity);
		filter: drop-shadow(0 0 7px var(--rce-caret-color));
	}

	.rce-caret {
		position: absolute;
		top: 0;
		left: 0;
		width: 2px;
		height: 1.5em;
		border-radius: 999px;
		background: var(--rce-caret-color);
		box-shadow: 0 0 9px var(--rce-caret-color);
		will-change: transform;
	}

	.rce-caret::after {
		content: '';
		position: absolute;
		inset: -5px -4px;
		border-radius: 999px;
		background: var(--rce-caret-color);
		opacity: 0.14;
		filter: blur(5px);
	}

	.rce-editor :global(.rce-chip) {
		display: inline-flex;
		align-items: center;
		gap: 0.3em;
		max-width: 16rem;
		vertical-align: middle;
		padding: 0.1em 0.45em;
		margin: 0 2px;
		border: 1px solid var(--border-soft);
		border-radius: 6px;
		background: rgba(15, 23, 42, 0.04);
		font-size: 0.92em;
		line-height: 1.3;
		color: var(--text-muted);
		white-space: nowrap;
		user-select: none;
		cursor: pointer;
	}

	.rce-editor :global(.rce-skill-chip) {
		border-color: rgba(37, 99, 235, 0.18);
		background: rgba(37, 99, 235, 0.06);
		color: #31517a;
		font-weight: 650;
	}

	.rce-editor :global(.rce-chip-icon) {
		flex-shrink: 0;
		width: 1em;
		height: 1em;
		object-fit: contain;
		border-radius: 3px;
	}

	.rce-editor :global(.rce-chip-label) {
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
