import { describe, expect, it } from 'vitest';
import { activeTurnMinHeight, groupThreadItemsIntoTurns } from './thread-turns';
import type { ChatItem } from '$lib/stores/chat.svelte';

describe('groupThreadItemsIntoTurns', () => {
	it('groups follow-up items under the preceding user message', () => {
		const items: ChatItem[] = [
			{ id: 'u1', type: 'user', text: 'first' },
			{ id: 'a1', type: 'assistant', text: 'reply' },
			{ id: 't1', type: 'tool', toolName: 'read', input: {}, output: 'ok' },
			{ id: 'u2', type: 'user', text: 'second' },
			{ id: 'a2', type: 'assistant', text: 'again' }
		];

		const turns = groupThreadItemsIntoTurns(items);

		expect(turns).toHaveLength(2);
		expect(turns[0].id).toBe('u1');
		expect(turns[0].items.map((entry) => entry.item.id)).toEqual(['a1', 't1']);
		expect(turns[1].id).toBe('u2');
		expect(turns[1].items.map((entry) => entry.item.id)).toEqual(['a2']);
	});

	it('returns an empty list for an empty transcript', () => {
		expect(groupThreadItemsIntoTurns([])).toEqual([]);
	});
});

describe('activeTurnMinHeight', () => {
	it('subtracts clearance from the viewport height', () => {
		expect(activeTurnMinHeight(600)).toBe(504);
	});

	it('returns zero when the viewport is unknown', () => {
		expect(activeTurnMinHeight(0)).toBe(0);
	});
});