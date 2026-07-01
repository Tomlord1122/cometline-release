import type { ChatItem } from '$lib/stores/chat.svelte';
import { TURN_BOTTOM_CLEARANCE } from './thread-scroll';

export interface ThreadTurnItem {
	item: ChatItem;
	index: number;
}

export interface ThreadTurn {
	id: string;
	user: Extract<ChatItem, { type: 'user' }>;
	userIndex: number;
	items: ThreadTurnItem[];
}

/** Group a flat transcript into user-anchored turns. */
export function groupThreadItemsIntoTurns(items: readonly ChatItem[]): ThreadTurn[] {
	const turns: ThreadTurn[] = [];
	let current: ThreadTurn | null = null;

	for (let index = 0; index < items.length; index++) {
		const item = items[index];
		if (item.type === 'user') {
			current = { id: item.id, user: item, userIndex: index, items: [] };
			turns.push(current);
			continue;
		}
		current?.items.push({ item, index });
	}

	return turns;
}

/** Min-height for the active turn canvas (follow-up turns only). */
export function activeTurnMinHeight(viewportHeight: number, clearance = TURN_BOTTOM_CLEARANCE) {
	if (viewportHeight <= 0) return 0;
	return Math.max(0, viewportHeight - clearance);
}