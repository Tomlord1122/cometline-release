<script lang="ts">
	import { Check, Copy } from '@lucide/svelte';
	import AssistantMarkdown from '$lib/components/AssistantMarkdown.svelte';
	import AssistantThinkingWait from '$lib/components/chat/AssistantThinkingWait.svelte';
	import ThinkingBlock from '$lib/components/chat/ThinkingBlock.svelte';
	import MemoryCard from '$lib/components/chat/MemoryCard.svelte';
	import ToolFoldPanel from '$lib/components/chat/ToolFoldPanel.svelte';
	import SubagentPanel from '$lib/components/chat/SubagentPanel.svelte';
	import AssistantActivityGroup from '$lib/components/chat/AssistantActivityGroup.svelte';
	import { assistantThinkingWait } from '$lib/conversation/thread-format';
	import {
		buildAssistantTimeline,
		isTimelineEntryToggleDisabled,
		pinnedJobProposalsForAssistant,
		shouldGroupAssistantTimeline,
		type ThinkingAttribution
	} from '$lib/conversation/thinking-attribution';
	import { timelineEntryKey } from '$lib/conversation/thread-view-helpers';
	import { memoryUpdateHint, memoryUpdateTooltip } from '$lib/memory-updates';
	import type { ChatItem } from '$lib/stores/chat.svelte';
	import type { ChatTurnPayload } from '$lib/actions/start-chat';
	import type { JobResource } from '$lib/client/cometmind';

	type AssistantItem = Extract<ChatItem, { type: 'assistant' }>;

	let {
		item,
		threadItems,
		thinkingForAssistant,
		streamingAssistantId,
		sessionStreaming,
		sessionId,
		now,
		heroGlowColor,
		copiedId,
		showActivitySpinner,
		toolFoldLabel,
		onCopyMessage,
		onNotifyAgent,
		onStartJob,
		thinkingExpanded,
		toggleThinking,
		activityGroupExpanded,
		toggleActivityGroup,
		memoryInThinkingExpanded,
		toggleMemoryInThinking,
		toolOutputExpanded,
		toggleToolOutput,
		subagentExpanded,
		toggleSubagent
	}: {
		item: AssistantItem;
		threadItems: readonly ChatItem[];
		thinkingForAssistant: ThinkingAttribution;
		streamingAssistantId: string | null;
		sessionStreaming: boolean;
		sessionId: string;
		now: number;
		heroGlowColor: string;
		copiedId: string | null;
		showActivitySpinner: boolean;
		toolFoldLabel: (tool: Extract<ChatItem, { type: 'tool' }>) => string;
		onCopyMessage: (id: string, text: string) => void | Promise<void>;
		onNotifyAgent?: (payload: ChatTurnPayload) => void | Promise<void>;
		onStartJob?: (job: JobResource) => void | Promise<void>;
		thinkingExpanded: (
			assistant: AssistantItem,
			segmentKey: string,
			segmentIndex: number,
			pending?: boolean
		) => boolean;
		toggleThinking: (
			assistant: AssistantItem,
			segmentKey: string,
			segmentIndex: number,
			pending?: boolean
		) => void;
		activityGroupExpanded: (assistantId: string, assistant: AssistantItem) => boolean;
		toggleActivityGroup: (assistantId: string, assistant: AssistantItem) => void;
		memoryInThinkingExpanded: (segmentKey: string) => boolean;
		toggleMemoryInThinking: (segmentKey: string) => void;
		toolOutputExpanded: (tool: Extract<ChatItem, { type: 'tool' }>) => boolean;
		toggleToolOutput: (id: string) => void;
		subagentExpanded: (id: string) => boolean;
		toggleSubagent: (id: string) => void;
	} = $props();

	const timeline = $derived(
		buildAssistantTimeline(item.id, threadItems, thinkingForAssistant)
	);
	const grouped = $derived(shouldGroupAssistantTimeline(item, timeline));
	const pinnedJobTools = $derived(pinnedJobProposalsForAssistant(item.id, threadItems));
	const maxVisible = $derived(
		item.id === streamingAssistantId && sessionStreaming ? 3 : 0
	);
	const cycling = $derived(item.id === streamingAssistantId && sessionStreaming);
	const thinkingWait = $derived(assistantThinkingWait(item, now));

	function thinkingActive(pending?: boolean) {
		return pending === true;
	}
</script>

<div class="assistant-stack">
	{#if grouped}
		<AssistantActivityGroup
			assistant={item}
			assistantId={item.id}
			{timeline}
			parentExpanded={activityGroupExpanded(item.id, item)}
			onToggleParent={() => toggleActivityGroup(item.id, item)}
			{timelineEntryKey}
			{toolFoldLabel}
			{thinkingExpanded}
			{toggleThinking}
			{memoryInThinkingExpanded}
			{toggleMemoryInThinking}
			{thinkingActive}
			showThinkingSpinner={!item.text.trim() &&
				!(item.id === streamingAssistantId && sessionStreaming)}
			{toolOutputExpanded}
			{toggleToolOutput}
			{subagentExpanded}
			{toggleSubagent}
			{sessionId}
			{onNotifyAgent}
			{onStartJob}
			maxVisibleReasoning={maxVisible}
			{cycling}
		/>
	{:else}
		{#each timeline as entry (timelineEntryKey(entry))}
			{@const toggleDisabled = isTimelineEntryToggleDisabled(entry)}
			{#if entry.kind === 'reasoning'}
				{@const segmentKey = `${item.id}-seg-${entry.segmentIndex}`}
				<ThinkingBlock
					text={entry.text}
					pending={entry.pending}
					expanded={thinkingExpanded(item, segmentKey, entry.segmentIndex, entry.pending)}
					showSpinner={thinkingActive(entry.pending) &&
						!item.text.trim() &&
						!(item.id === streamingAssistantId && sessionStreaming)}
					{toggleDisabled}
					onToggle={() =>
						toggleThinking(item, segmentKey, entry.segmentIndex, entry.pending)}
				/>
			{:else if entry.kind === 'memory'}
				{@const memoryKey = `${item.id}-memory`}
				<MemoryCard
					memories={entry.memories}
					expanded={memoryInThinkingExpanded(memoryKey)}
					onToggle={() => toggleMemoryInThinking(memoryKey)}
				/>
			{:else if entry.kind === 'tool'}
				<ToolFoldPanel
					item={entry.tool}
					label={toolFoldLabel(entry.tool)}
					expanded={toolOutputExpanded(entry.tool)}
					{toggleDisabled}
					onToggle={() => toggleToolOutput(entry.tool.id)}
					{sessionId}
					{onNotifyAgent}
					{onStartJob}
				/>
			{:else}
				<SubagentPanel
					item={entry.subagent}
					expanded={subagentExpanded(entry.subagent.id)}
					{toggleDisabled}
					onToggle={() => toggleSubagent(entry.subagent.id)}
				/>
			{/if}
		{/each}
	{/if}
	{#if item.text}
		<div class="bubble assistant-bubble">
			<AssistantMarkdown source={item.text} streaming={item.id === streamingAssistantId} />
		</div>
	{/if}
	{#each pinnedJobTools as jobTool (jobTool.id)}
		<ToolFoldPanel
			item={jobTool}
			label={toolFoldLabel(jobTool)}
			expanded={toolOutputExpanded(jobTool)}
			onToggle={() => toggleToolOutput(jobTool.id)}
			{sessionId}
			{onNotifyAgent}
			{onStartJob}
		/>
	{/each}
	{#if item.text && item.id !== streamingAssistantId}
		<div class="message-actions m-1">
			{#if item.memoryUpdates?.length}
				<span
					class="message-action memory-hint"
					title={memoryUpdateTooltip(item.memoryUpdates)}
					aria-label={memoryUpdateTooltip(item.memoryUpdates)}
				>
					{memoryUpdateHint(item.memoryUpdates)}
				</span>
			{/if}
			<button
				type="button"
				class="message-action m-1"
				class:copied={copiedId === item.id}
				title="Copy message"
				aria-label="Copy message"
				onclick={() => onCopyMessage(item.id, item.text)}
			>
				{#if copiedId === item.id}
					<Check size={13} />
					<span>Copied</span>
				{:else}
					<Copy size={13} />
					<span>Copy</span>
				{/if}
			</button>
		</div>
	{/if}
	{#if showActivitySpinner}
		<AssistantThinkingWait
			label={thinkingWait.label}
			detail={thinkingWait.detail}
			color={heroGlowColor}
		/>
	{/if}
</div>

<style>
	.assistant-stack {
		display: flex;
		flex-direction: column;
		gap: 8px;
		width: 100%;
		max-width: var(--chat-assistant-column);
		min-width: 0;
		flex: 0 1 auto;
		align-items: flex-start;
		--assistant-activity-width: 80%;
	}

	.assistant-stack > :global(.memory-panel),
	.assistant-stack > :global(.tool-fold-panel),
	.assistant-stack > :global(.thinking-panel),
	.assistant-stack > :global(.subagent-panel),
	.assistant-stack :global(.activity-group > .fold-body) {
		align-self: flex-start;
		width: var(--assistant-activity-width);
		max-width: 100%;
		min-width: 0;
		box-sizing: border-box;
	}

	.assistant-stack > :global(.memory-panel .memory-body) {
		width: 100%;
		box-sizing: border-box;
	}

	.assistant-stack :global(.activity-group) {
		align-self: stretch;
		width: 100%;
		min-width: 0;
	}

	.message-actions {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-top: -2px;
		opacity: 0;
		transition: opacity var(--duration-fast) var(--ease-smooth);
	}

	.assistant-stack:hover .message-actions,
	.message-actions:focus-within {
		opacity: 1;
	}

	.message-action {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		padding: 4px 8px;
		border: 1px solid transparent;
		border-radius: 7px;
		background: transparent;
		color: var(--text-soft);
		font-size: 11px;
		font-weight: 600;
		line-height: 1;
		cursor: pointer;
		transition:
			color var(--duration-fast) var(--ease-smooth),
			background var(--duration-fast) var(--ease-smooth),
			border-color var(--duration-fast) var(--ease-smooth);
	}

	.message-action:hover {
		color: var(--text-main);
		background: rgba(255, 255, 255, 0.92);
		border-color: var(--border-soft);
	}

	.message-action.copied {
		color: #15803d;
	}

	.memory-hint {
		cursor: default;
	}

	.memory-hint:hover {
		background: transparent;
		border-color: transparent;
		color: var(--text-soft);
	}

	.message-action :global(svg) {
		flex-shrink: 0;
	}

	@media (prefers-reduced-motion: reduce) {
		.message-actions {
			transition: none;
		}
	}
</style>
