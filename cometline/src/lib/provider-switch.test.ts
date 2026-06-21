import { describe, expect, it } from 'vitest';
import { analyzeProviderSwitch, sdkFamilyForMethod } from './provider-switch';
import type { ChatItem } from '$lib/types';

const userItem: ChatItem = { id: 'u1', type: 'user', text: 'hi' };
const reasoningAssistant: ChatItem = {
	id: 'a1',
	type: 'assistant',
	text: 'answer',
	reasoning: { segments: [{ text: 'thinking' }] }
};
const emptyReasoningAssistant: ChatItem = {
	id: 'a2',
	type: 'assistant',
	text: 'answer',
	reasoning: { segments: [{ text: '   ' }] }
};
const toolItem: ChatItem = { id: 't1', type: 'tool', toolName: 'read', input: {} };

describe('sdkFamilyForMethod', () => {
	it('maps methods to comet-sdk families', () => {
		expect(sdkFamilyForMethod('anthropic')).toBe('anthropic');
		expect(sdkFamilyForMethod('codex')).toBe('codex');
		expect(sdkFamilyForMethod('openai')).toBe('openai');
		expect(sdkFamilyForMethod('openai-compatible')).toBe('openai');
		expect(sdkFamilyForMethod('opencode-go')).toBe('openai');
	});
});

describe('analyzeProviderSwitch', () => {
	it('returns no warnings for native families even with reasoning', () => {
		expect(analyzeProviderSwitch([reasoningAssistant], 'anthropic')).toEqual([]);
		expect(analyzeProviderSwitch([reasoningAssistant], 'openai')).toEqual([]);
	});

	it('warns about reasoning summarization when switching to codex', () => {
		const warnings = analyzeProviderSwitch([reasoningAssistant, toolItem], 'codex');
		expect(warnings).toHaveLength(1);
		expect(warnings[0].kind).toBe('reasoning');
		expect(warnings[0].count).toBe(1);
	});

	it('counts multiple reasoning steps', () => {
		const warnings = analyzeProviderSwitch(
			[reasoningAssistant, { ...reasoningAssistant, id: 'a3' }],
			'codex'
		);
		expect(warnings[0].count).toBe(2);
		expect(warnings[0].message).toContain('2 reasoning steps');
	});

	it('ignores empty reasoning and non-assistant items', () => {
		const warnings = analyzeProviderSwitch(
			[userItem, emptyReasoningAssistant, toolItem],
			'codex'
		);
		expect(warnings).toEqual([]);
	});
});
