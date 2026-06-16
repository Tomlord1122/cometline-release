import type { ChatItem } from '$lib/types';

export type InjectedMemory = Extract<ChatItem, { type: 'memory' }>['memories'][number];

export type ThinkingBlock = {
	reasoning?: { text: string; pending?: boolean };
	tools: Extract<ChatItem, { type: 'tool' }>[];
	memories: InjectedMemory[];
};

export type ThinkingAttribution = {
	map: Map<string, ThinkingBlock>;
	toolIdsInBuffer: Set<string>;
	memoryIdsInBuffer: Set<string>;
};

/** Attribute memory/tool rows to the assistant in the same user turn (full transcript scan). */
export function buildThinkingAttribution(items: readonly ChatItem[]): ThinkingAttribution {
	const map = new Map<string, ThinkingBlock>();
	const toolIdsInBuffer = new Set<string>();
	const memoryIdsInBuffer = new Set<string>();
	let currentAssistantId: string | null = null;
	let pendingMemories: InjectedMemory[] = [];

	for (let index = 0; index < items.length; index++) {
		const item = items[index];
		if (item.type === 'user' || item.type === 'status' || item.type === 'error') {
			currentAssistantId = null;
			pendingMemories = [];
			continue;
		}
		if (item.type === 'memory') {
			pendingMemories = item.memories;
			memoryIdsInBuffer.add(item.id);
			continue;
		}
		if (item.type === 'assistant') {
			currentAssistantId = item.id;
			const existing = map.get(item.id);
			if (!existing) {
				map.set(item.id, {
					reasoning: item.reasoning,
					tools: [],
					memories: pendingMemories
				});
			} else {
				if (item.reasoning && !existing.reasoning) {
					existing.reasoning = item.reasoning;
				}
				if (pendingMemories.length > 0) {
					existing.memories = pendingMemories;
				}
			}
			pendingMemories = [];
		} else if (item.type === 'tool' && currentAssistantId) {
			const block = map.get(currentAssistantId);
			if (block) {
				block.tools.push(item);
				toolIdsInBuffer.add(item.id);
			}
		}
	}

	return { map, toolIdsInBuffer, memoryIdsInBuffer };
}
