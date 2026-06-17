import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { StreamEvent } from '$lib/types';

vi.mock('$lib/client/cometmind', () => ({
	getSessionMessages: vi.fn(),
	listChildSessions: vi.fn(),
	streamMessage: vi.fn(),
	abortSession: vi.fn(),
	respondToSubagent: vi.fn()
}));

import {
	getSessionMessages,
	listChildSessions,
	streamMessage
} from '$lib/client/cometmind';
import { chatStore } from './chat.svelte';

async function* eventsOf(...events: StreamEvent[]) {
	for (const event of events) {
		yield event;
	}
}

function mockTranscript(sessionId: string, text: string) {
	return {
		session_id: sessionId,
		items: [{ type: 'user' as const, text }]
	};
}

describe('chatStore session switching', () => {
	beforeEach(() => {
		chatStore.clear();
		vi.clearAllMocks();
		vi.mocked(listChildSessions).mockResolvedValue({ sessions: [] });
	});

	it('loads session B transcript when switching from session A with partial stream', async () => {
		vi.mocked(streamMessage).mockImplementation(async function* (sessionId) {
			if (sessionId === 'sess-a') {
				yield { type: 'text_delta', delta: 'partial A' };
				await new Promise<void>(() => {});
			}
		});

		vi.mocked(getSessionMessages).mockImplementation(async (sessionId) =>
			mockTranscript(sessionId, `history ${sessionId}`)
		);

		chatStore.bindSession('sess-a');
		void chatStore.send('sess-a', 'question A');
		await vi.waitFor(() =>
			expect(chatStore.items.some((item) => item.type === 'assistant')).toBe(true)
		);

		chatStore.bindSession('sess-b');
		await chatStore.loadTranscript('sess-b');

		expect(chatStore.sessionID).toBe('sess-b');
		expect(chatStore.items).toEqual([
			expect.objectContaining({ type: 'user', text: 'history sess-b' })
		]);
		expect(chatStore.isStreamingFor('sess-b')).toBe(false);
	});

	it('restores session A cache when switching back during stream', async () => {
		let releaseA: (() => void) | undefined;
		const aGate = new Promise<void>((resolve) => {
			releaseA = resolve;
		});

		vi.mocked(streamMessage).mockImplementation(async function* (sessionId) {
			if (sessionId === 'sess-a') {
				yield { type: 'text_delta', delta: 'live A' };
				await aGate;
				yield { type: 'done' };
			}
		});

		vi.mocked(getSessionMessages).mockResolvedValue(mockTranscript('sess-b', 'history B'));

		chatStore.bindSession('sess-a');
		void chatStore.send('sess-a', 'question A');
		await vi.waitFor(() =>
			expect(chatStore.items.some((item) => item.type === 'assistant')).toBe(true)
		);

		const assistantBeforeLeave = chatStore.items.find((item) => item.type === 'assistant');

		chatStore.bindSession('sess-b');
		await chatStore.loadTranscript('sess-b');
		expect(chatStore.items.some((item) => item.type === 'user' && item.text === 'history B')).toBe(
			true
		);

		chatStore.bindSession('sess-a');
		const assistantOnReturn = chatStore.items.find((item) => item.type === 'assistant');
		expect(assistantOnReturn?.id).toBe(assistantBeforeLeave?.id);
		expect(assistantOnReturn?.type === 'assistant' ? assistantOnReturn.text : '').toContain(
			'live A'
		);

		releaseA!();
		await vi.waitFor(() => expect(chatStore.isStreamingFor('sess-a')).toBe(false));
	});

	it('allows concurrent sends in different sessions', async () => {
		vi.mocked(streamMessage).mockImplementation(async function* (sessionId) {
			if (sessionId === 'sess-a') {
				yield* eventsOf({ type: 'text_delta', delta: 'answer A' }, { type: 'done' });
				return;
			}
			if (sessionId === 'sess-b') {
				yield* eventsOf({ type: 'text_delta', delta: 'answer B' }, { type: 'done' });
			}
		});

		chatStore.bindSession('sess-a');
		const sendA = chatStore.send('sess-a', 'question A');
		await vi.waitFor(() => expect(chatStore.isStreamingFor('sess-a')).toBe(true));

		chatStore.bindSession('sess-b');
		const sendB = chatStore.send('sess-b', 'question B');
		await vi.waitFor(() => expect(chatStore.isStreamingFor('sess-b')).toBe(true));

		await Promise.all([sendA, sendB]);

		expect(chatStore.isStreamingFor('sess-a')).toBe(false);
		expect(chatStore.isStreamingFor('sess-b')).toBe(false);

		chatStore.bindSession('sess-a');
		expect(
			chatStore.items.some(
				(item) => item.type === 'assistant' && item.text.includes('answer A')
			)
		).toBe(true);

		chatStore.bindSession('sess-b');
		expect(
			chatStore.items.some(
				(item) => item.type === 'assistant' && item.text.includes('answer B')
			)
		).toBe(true);
	});

	it('blocks duplicate send in the same session', async () => {
		let releaseA: (() => void) | undefined;
		const aGate = new Promise<void>((resolve) => {
			releaseA = resolve;
		});

		vi.mocked(streamMessage).mockImplementation(async function* () {
			yield { type: 'text_delta', delta: 'working' };
			await aGate;
			yield { type: 'done' };
		});

		chatStore.bindSession('sess-a');
		void chatStore.send('sess-a', 'first');
		await vi.waitFor(() => expect(chatStore.isStreamingFor('sess-a')).toBe(true));

		await chatStore.send('sess-a', 'second');
		expect(vi.mocked(streamMessage)).toHaveBeenCalledTimes(1);

		releaseA!();
	});
});
