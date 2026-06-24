import type { ChatItem } from '$lib/stores/chat.svelte';
import { anyReasoningPending, reasoningTextLength } from './reasoning';

export const SCROLL_PIN_THRESHOLD = 96;
export const USER_SEND_TOP_OFFSET = 24;

export function isNearBottom(element: HTMLElement, threshold = SCROLL_PIN_THRESHOLD) {
	return element.scrollHeight - element.scrollTop - element.clientHeight <= threshold;
}

export function shouldShowJumpToBottom(element: HTMLElement, threshold = SCROLL_PIN_THRESHOLD) {
	const overflowing = element.scrollHeight - element.clientHeight > threshold;
	return overflowing && !isNearBottom(element, threshold);
}

/** Walk the offsetParent chain to get an element's top offset relative to an ancestor. */
export function offsetTopRelativeTo(el: HTMLElement, ancestor: HTMLElement): number {
	let top = 0;
	let cur: HTMLElement | null = el;
	while (cur && cur !== ancestor) {
		top += cur.offsetTop;
		cur = cur.offsetParent as HTMLElement | null;
	}
	return top;
}

export function buildScrollKey(items: readonly ChatItem[], sessionStreaming: boolean) {
	if (!sessionStreaming) {
		const last = items.at(-1);
		return `idle:${items.length}:${last?.id ?? ''}`;
	}
	for (let i = items.length - 1; i >= 0; i--) {
		const item = items[i];
		if (item.type === 'assistant') {
			return `stream:assistant:${item.id}:${item.text.length}:${reasoningTextLength(item)}:${anyReasoningPending(item)}:${item.pending ?? false}`;
		}
		if (item.type === 'tool') {
			return `stream:tool:${item.id}:${item.output?.length ?? 0}:${item.pending ?? false}`;
		}
	}
	return `stream:empty:${items.length}`;
}

export function userMessageScrollTop(
	absoluteTop: number,
	userMessageCount: number,
	viewportHeight: number,
	firstTurnOffset = USER_SEND_TOP_OFFSET
) {
	if (userMessageCount > 1) {
		const followupOffset = viewportHeight > 0 ? Math.round(viewportHeight * 0.15) : 80;
		return Math.max(0, absoluteTop - followupOffset);
	}
	return Math.max(0, absoluteTop - firstTurnOffset);
}
