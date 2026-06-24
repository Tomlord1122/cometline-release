// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import {
	buildScrollKey,
	isNearBottom,
	offsetTopRelativeTo,
	userMessageScrollTop
} from './thread-scroll';
import type { ChatItem } from '$lib/stores/chat.svelte';

describe('buildScrollKey', () => {
	it('returns idle key when not streaming', () => {
		const items: ChatItem[] = [
			{ id: 'u1', type: 'user', text: 'hi' },
			{ id: 'a1', type: 'assistant', text: 'hello' }
		];
		expect(buildScrollKey(items, false)).toBe('idle:2:a1');
	});

	it('returns assistant stream key when streaming', () => {
		const items: ChatItem[] = [
			{ id: 'u1', type: 'user', text: 'hi' },
			{ id: 'a1', type: 'assistant', text: 'hel', pending: true }
		];
		expect(buildScrollKey(items, true)).toBe('stream:assistant:a1:3:0:false:true');
	});
});

describe('isNearBottom', () => {
	it('returns true when within threshold of bottom', () => {
		const el = {
			scrollHeight: 1000,
			scrollTop: 904,
			clientHeight: 96
		} as HTMLElement;
		expect(isNearBottom(el, 96)).toBe(true);
	});
});

describe('userMessageScrollTop', () => {
	it('uses first-turn offset for the first user message', () => {
		expect(userMessageScrollTop(200, 1, 800)).toBe(176);
	});

	it('uses follow-up offset for later user messages', () => {
		expect(userMessageScrollTop(400, 2, 800)).toBe(280);
	});
});

describe('offsetTopRelativeTo', () => {
	it('sums offsetTop along the offsetParent chain', () => {
		const ancestor = document.createElement('div');
		const parent = document.createElement('div');
		const child = document.createElement('div');
		ancestor.appendChild(parent);
		parent.appendChild(child);
		Object.defineProperty(parent, 'offsetParent', { value: ancestor });
		Object.defineProperty(child, 'offsetParent', { value: parent });
		Object.defineProperty(parent, 'offsetTop', { value: 40 });
		Object.defineProperty(child, 'offsetTop', { value: 12 });
		expect(offsetTopRelativeTo(child, ancestor)).toBe(52);
	});
});
