import type { ChatItem, ProviderMethod } from '$lib/types';

/**
 * The comet-sdk provider family a Cometline provider method maps to. Mirrors
 * `sdkProviderID` in cometmind/internal/provider/factory.go so the UI can warn
 * about replay behaviour without round-tripping to the backend.
 */
export type SdkFamily = 'anthropic' | 'openai' | 'codex';

export function sdkFamilyForMethod(method: ProviderMethod): SdkFamily {
	switch (method) {
		case 'anthropic':
			return 'anthropic';
		case 'codex':
			return 'codex';
		case 'openai':
		case 'openai-compatible':
		case 'opencode-go':
		default:
			return 'openai';
	}
}

/** A single way the target provider will handle existing history differently. */
export interface ProviderSwitchWarning {
	kind: 'reasoning' | 'tool' | 'image';
	count: number;
	message: string;
}

function transcriptHasReasoning(item: ChatItem): boolean {
	if (item.type !== 'assistant' || !item.reasoning) return false;
	if (item.reasoning.segments?.some((s) => s.text.trim() !== '')) return true;
	return Boolean(item.reasoning.text && item.reasoning.text.trim() !== '');
}

/**
 * Analyzes a transcript against a target provider family and returns the
 * warnings the user should acknowledge before switching. Returns an empty array
 * when the switch is fully lossless (e.g. no relevant history, or a target that
 * replays everything natively).
 *
 * Currently only the Codex family degrades history: it cannot replay
 * chain-of-thought, so prior reasoning is summarized into plain text.
 */
export function analyzeProviderSwitch(
	items: ChatItem[],
	targetMethod: ProviderMethod
): ProviderSwitchWarning[] {
	const family = sdkFamilyForMethod(targetMethod);
	if (family !== 'codex') return [];

	let reasoningCount = 0;
	for (const item of items) {
		if (transcriptHasReasoning(item)) reasoningCount += 1;
	}

	const warnings: ProviderSwitchWarning[] = [];
	if (reasoningCount > 0) {
		warnings.push({
			kind: 'reasoning',
			count: reasoningCount,
			message:
				reasoningCount === 1
					? '1 reasoning step will be condensed into summary text (Codex cannot replay chain-of-thought).'
					: `${reasoningCount} reasoning steps will be condensed into summary text (Codex cannot replay chain-of-thought).`
		});
	}
	return warnings;
}
