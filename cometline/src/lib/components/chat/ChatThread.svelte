<script lang="ts">
	import { fade } from 'svelte/transition';
	import { TriangleAlert } from '@lucide/svelte';
	import { chatStore, type ChatItem } from '$lib/stores/chat.svelte';
	import { settingsStore } from '$lib/stores/settings.svelte';
	import { chatDebug, chatDebugEnabled, summarizeChatItem } from '$lib/debug/chat';
	import {
		assistantThinkingWait,
		toolFoldLabel as formatToolFoldLabel,
		usageText
	} from '$lib/conversation/thread-format';
	import AssistantStack from '$lib/components/chat/AssistantStack.svelte';
	import UserMessageRow from '$lib/components/chat/UserMessageRow.svelte';
	import MemoryEventRow from '$lib/components/chat/MemoryEventRow.svelte';
	import AssistantThinkingWait from '$lib/components/chat/AssistantThinkingWait.svelte';
	import JumpToBottom from '$lib/components/chat/JumpToBottom.svelte';
	import ThreadAvatar from '$lib/components/chat/ThreadAvatar.svelte';
	import ToolFoldPanel from '$lib/components/chat/ToolFoldPanel.svelte';
	import SubagentPanel from '$lib/components/chat/SubagentPanel.svelte';
	import {
		buildAssistantTimeline,
		buildThinkingAttribution,
		pinnedJobProposalToolIds,
		pinnedJobProposalsForAssistant
	} from '$lib/conversation/thinking-attribution';
	import { startsSpeakerRun } from '$lib/conversation/thread-view-helpers';
	import { createFoldController } from '$lib/conversation/thread-fold.svelte';
	import { createThreadScroll } from '$lib/conversation/thread-scroll.svelte';
	import {
		hasReasoning
	} from '$lib/conversation/reasoning';
	import type { ChatTurnPayload } from '$lib/actions/start-chat';
	import type { JobResource } from '$lib/client/cometmind';

	const TRANSCRIPT_IN = { duration: 140 };

	let {
		sessionId,
		awaitingFirstAssistant = false,
		firstTurnFlightDone = false,
		firstTurnHandoffPending = false,
		onNotifyAgent,
		onStartJob
	}: {
		sessionId: string;
		awaitingFirstAssistant?: boolean;
		firstTurnFlightDone?: boolean;
		firstTurnHandoffPending?: boolean;
		onNotifyAgent?: (payload: ChatTurnPayload) => void | Promise<void>;
		onStartJob?: (job: JobResource) => void | Promise<void>;
	} = $props();

	let iconVariant = $derived(settingsStore.settings.app.iconVariant);

	let isSessionSynced = $derived(chatStore.sessionID === sessionId);
	let sessionStreaming = $derived(chatStore.isStreamingFor(sessionId));

	let snapshotItems = $state.raw<ChatItem[]>([]);

	$effect(() => {
		if (chatStore.sessionID !== sessionId) return;
		snapshotItems = chatStore.items;
	});

	let memoryCycleTick = $state(0);
	let copiedId = $state<string | null>(null);
	let copyResetTimer: ReturnType<typeof setTimeout> | null = null;
	let scrollerEl = $state<HTMLDivElement | undefined>(undefined);
	let now = $state(Date.now());
	let threadItems = $derived(isSessionSynced ? chatStore.items : snapshotItems);
	let embeddedPinnedJobIds = $derived(pinnedJobProposalToolIds(threadItems));
	let firstAssistantItem = $derived(
		threadItems.find(
			(item) => item.type === 'assistant' && (item.text.trim() || hasReasoning(item))
		) as Extract<ChatItem, { type: 'assistant' }> | undefined
	);
	let firstAssistantId = $derived(firstAssistantItem?.id ?? null);
	let firstAssistantRowId = $derived(
		threadItems.find((item) => item.type === 'assistant')?.id ?? null
	);
	let streamingAssistantId = $derived.by(() => {
		if (!sessionStreaming) return null;
		const last = threadItems.at(-1);
		if (last?.type === 'user') return null;
		return threadItems.findLast((item) => item.type === 'assistant')?.id ?? null;
	});

	const fold = createFoldController({
		getSessionId: () => sessionId,
		getIsSessionSynced: () => isSessionSynced,
		getItems: () => snapshotItems,
		getStreamingAssistantId: () => streamingAssistantId,
		getSessionStreaming: () => sessionStreaming
	});

	let firstUserId = $derived(threadItems.find((item) => item.type === 'user')?.id ?? null);
	let lastUserId = $derived(threadItems.findLast((item) => item.type === 'user')?.id ?? null);
	let userMessageCount = $derived(
		threadItems.reduce((count, item) => (item.type === 'user' ? count + 1 : count), 0)
	);

	function sessionHasCachedTranscript(targetSessionId: string) {
		if (chatStore.sessionID === targetSessionId && chatStore.items.length > 0) return true;
		if (
			chatStore.isStreamingFor(targetSessionId) ||
			chatStore.hasInFlightTurn(targetSessionId)
		) {
			return true;
		}
		return chatStore.getCachedItemCount(targetSessionId) > 0;
	}

	const scroll = createThreadScroll({
		getSessionId: () => sessionId,
		getIsSessionSynced: () => isSessionSynced,
		getThreadItems: () => threadItems,
		getSessionStreaming: () => sessionStreaming,
		getLastUserId: () => lastUserId,
		getUserMessageCount: () => userMessageCount,
		getIsLoading: () => chatStore.isLoading,
		sessionHasCachedTranscript
	});

	$effect(() => {
		scroll.setScroller(scrollerEl);
	});

	// Reserve a viewport-tall gap below the conversation right after the user
	// sends a message, so the latest user message can scroll up toward the
	// top-right area. Once the assistant starts responding (assistant/tool/
	// subagent items appear), the spacer collapses so the reply grows upward
	// naturally instead of leaving a big empty gap. Only applies once a
	// follow-up turn exists (the first turn keeps its natural layout).
	let bottomSpacerHeight = $derived(
		userMessageCount > 1 &&
			scroll.viewportHeight > 0 &&
			threadItems.at(-1)?.type === 'user'
			? Math.max(0, scroll.viewportHeight - 240)
			: 0
	);
	let showMessages = $derived(
		threadItems.length > 0 || (isSessionSynced && awaitingFirstAssistant && !firstUserId)
	);
	let transcriptFadeIn = $derived(
		awaitingFirstAssistant || scroll.isInitialTranscriptPaint
			? { duration: 0 }
			: TRANSCRIPT_IN
	);

	let thinkingForAssistant = $derived(buildThinkingAttribution(threadItems));

	function isToolInBuffer(item: Extract<ChatItem, { type: 'tool' }>) {
		return thinkingForAssistant.toolIdsInBuffer.has(item.id);
	}

	function isSubagentInBuffer(item: Extract<ChatItem, { type: 'subagent' }>) {
		return thinkingForAssistant.subagentIdsInBuffer.has(item.id);
	}

	function isMemoryInBuffer(item: Extract<ChatItem, { type: 'memory' }>) {
		return thinkingForAssistant.memoryIdsInBuffer.has(item.id);
	}

	async function copyMessage(id: string, text: string) {
		try {
			await navigator.clipboard.writeText(text);
		} catch {
			return;
		}
		copiedId = id;
		if (copyResetTimer) clearTimeout(copyResetTimer);
		copyResetTimer = setTimeout(() => {
			copiedId = null;
			copyResetTimer = null;
		}, 1600);
	}

	let renderDebugSnapshot = $derived.by(() => ({
		awaitingFirstAssistant,
		firstTurnFlightDone,
		firstTurnHandoffPending,
		isStreaming: sessionStreaming,
		firstUserId,
		firstAssistantId,
		firstAssistantRowId,
		firstAssistantItem: firstAssistantItem ? summarizeChatItem(firstAssistantItem) : null,
		firstAssistantVisible: firstAssistantItem ? showAssistantRow(firstAssistantItem) : false,
		items: threadItems.map(summarizeRenderItem)
	}));

	$effect(() => {
		return () => {
			if (copyResetTimer) clearTimeout(copyResetTimer);
		};
	});

	$effect(() => {
		const hasMemory = threadItems.some(
			(item) => item.type === 'memory' && !isMemoryInBuffer(item)
		);
		if (!hasMemory) {
			memoryCycleTick = 0;
			return;
		}
		const timer = setInterval(() => memoryCycleTick++, 5000);
		return () => clearInterval(timer);
	});

	$effect(() => {
		const hasTimedPending = chatStore.items.some(
			(item) =>
				(item.type === 'tool' && item.pending) ||
				(item.type === 'assistant' &&
					item.pendingStartedAt != null &&
					sessionStreaming &&
					item.id === streamingAssistantId &&
					!item.text?.trim())
		);
		if (!hasTimedPending) return;
		const timer = setInterval(() => {
			now = Date.now();
		}, 100);
		return () => clearInterval(timer);
	});

	function toolFoldLabel(item: Extract<ChatItem, { type: 'tool' }>) {
		return formatToolFoldLabel(item, now);
	}

	let heroGlowColor = $derived(settingsStore.settings.appearance.heroComposer.glowColor);

	function showAssistantActivitySpinner(item: Extract<ChatItem, { type: 'assistant' }>) {
		return sessionStreaming && item.id === streamingAssistantId;
	}

	function hasVisibleThinkingBlock(itemId: string) {
		return buildAssistantTimeline(itemId, threadItems, thinkingForAssistant).length > 0;
	}

	function showAssistantPending(item: Extract<ChatItem, { type: 'assistant' }>) {
		if (!sessionStreaming || item.id !== streamingAssistantId) return false;
		if (item.text?.trim()) return false;
		return !hasVisibleThinkingBlock(item.id);
	}

	function showAssistantRow(item: Extract<ChatItem, { type: 'assistant' }>) {
		return Boolean(
			item.text ||
			hasReasoning(item) ||
			hasVisibleThinkingBlock(item.id) ||
			pinnedJobProposalsForAssistant(item.id, threadItems).length > 0 ||
			showAssistantPending(item) ||
			showAssistantActivitySpinner(item)
		);
	}

	function summarizeRenderItem(item: ChatItem, index: number) {
		if (item.type !== 'assistant') return { index, ...summarizeChatItem(item) };
		return {
			index,
			...summarizeChatItem(item),
			showAssistantRow: showAssistantRow(item),
			isFirstAssistant: item.id === firstAssistantId,
			renderedInFirstTurnSlot: awaitingFirstAssistant && item.id === firstAssistantId,
			excludedFromNormalList: awaitingFirstAssistant && item.id === firstAssistantId,
			showAssistantPending: showAssistantPending(item)
		};
	}

	function showFirstTurnAvatarSlot() {
		if (!firstUserId) return false;
		if (firstTurnHandoffPending) return true;
		if (!awaitingFirstAssistant) return false;
		if (!firstTurnFlightDone) return true;
		if (!firstAssistantItem) return true;
		return !showAssistantRow(firstAssistantItem);
	}

	function firstAssistantInNormalList(item: Extract<ChatItem, { type: 'assistant' }>) {
		if (showFirstTurnAvatarSlot()) return false;
		return !(
			awaitingFirstAssistant &&
			item.id === firstAssistantRowId &&
			firstUserId &&
			!(firstTurnFlightDone && showAssistantRow(item))
		);
	}

	function hideAssistantAvatarForFirstTurn(item: Extract<ChatItem, { type: 'assistant' }>) {
		return firstTurnHandoffPending && item.id === firstAssistantRowId;
	}

	function hideFirstTurnDestination() {
		return firstTurnHandoffPending;
	}

	$inspect(renderDebugSnapshot).with((type, snapshot) => {
		if (!chatDebugEnabled()) return;
		chatDebug('thread:$inspect', { type, ...snapshot });
	});
</script>

<div class="thread-wrap">
	<div
		class="thread scrollbar-none"
		bind:this={scrollerEl}
		onscroll={scroll.onScroll}
		aria-live="polite"
	>
		<div class="thread-inner">
			{#if showMessages}
				<div
					class="thread-messages"
					class:hydrating={scroll.isInitialTranscriptPaint}
					in:fade={transcriptFadeIn}
				>
					{#if awaitingFirstAssistant && !firstUserId}
						<div class="row assistant-row flight-placeholder" aria-hidden="true">
							<ThreadAvatar
								variant="avatar"
								{iconVariant}
								flightHidden={firstTurnHandoffPending}
								flightTarget="avatar"
							/>
							{#if firstAssistantItem && showAssistantRow(firstAssistantItem)}
								<div
									class="assistant-column"
									class:first-turn-destination-hidden={hideFirstTurnDestination()}
								>
									<AssistantStack
										item={firstAssistantItem}
										{threadItems}
										{thinkingForAssistant}
										{streamingAssistantId}
										{sessionStreaming}
										{sessionId}
										{now}
										{heroGlowColor}
										{copiedId}
										showActivitySpinner={showAssistantActivitySpinner(firstAssistantItem)}
										{toolFoldLabel}
										onCopyMessage={copyMessage}
										{onNotifyAgent}
										{onStartJob}
										thinkingExpanded={fold.thinkingExpanded}
										toggleThinking={fold.toggleThinking}
										activityGroupExpanded={fold.activityGroupExpanded}
										toggleActivityGroup={fold.toggleActivityGroup}
										memoryInThinkingExpanded={fold.memoryInThinkingExpanded}
										toggleMemoryInThinking={fold.toggleMemoryInThinking}
										toolOutputExpanded={fold.toolOutputExpanded}
										toggleToolOutput={fold.toggleToolOutput}
										subagentExpanded={fold.subagentExpanded}
										toggleSubagent={fold.toggleSubagent}
									/>
								</div>
							{:else if sessionStreaming}
								<div
									class="assistant-stack"
									class:first-turn-destination-hidden={hideFirstTurnDestination()}
								>
									<AssistantThinkingWait
										label={assistantThinkingWait(undefined, now).label}
										detail={assistantThinkingWait(undefined, now).detail}
										color={heroGlowColor}
									/>
								</div>
							{:else}
								<div
									class="assistant-stack"
									class:first-turn-destination-hidden={hideFirstTurnDestination()}
								></div>
							{/if}
						</div>
					{/if}

					{#each threadItems as item, index (item.id)}
						{#if item.type === 'user'}
							<UserMessageRow
								{item}
								{iconVariant}
								continuationRow={!startsSpeakerRun(threadItems, index, 'user')}
								{copiedId}
								onCopyMessage={copyMessage}
							/>
							{#if showFirstTurnAvatarSlot() && item.id === firstUserId}
								<div
									class="row assistant-row"
									class:flight-placeholder={!firstAssistantId}
									aria-hidden={!firstAssistantId}
								>
									<ThreadAvatar
										variant="avatar"
										{iconVariant}
										flightHidden={firstTurnHandoffPending}
										flightTarget="avatar"
									/>
									{#if firstAssistantItem && showAssistantRow(firstAssistantItem)}
										<div
											class="assistant-column"
											class:first-turn-destination-hidden={hideFirstTurnDestination()}
										>
											<AssistantStack
										item={firstAssistantItem}
										{threadItems}
										{thinkingForAssistant}
										{streamingAssistantId}
										{sessionStreaming}
										{sessionId}
										{now}
										{heroGlowColor}
										{copiedId}
										showActivitySpinner={showAssistantActivitySpinner(firstAssistantItem)}
										{toolFoldLabel}
										onCopyMessage={copyMessage}
										{onNotifyAgent}
										{onStartJob}
										thinkingExpanded={fold.thinkingExpanded}
										toggleThinking={fold.toggleThinking}
										activityGroupExpanded={fold.activityGroupExpanded}
										toggleActivityGroup={fold.toggleActivityGroup}
										memoryInThinkingExpanded={fold.memoryInThinkingExpanded}
										toggleMemoryInThinking={fold.toggleMemoryInThinking}
										toolOutputExpanded={fold.toolOutputExpanded}
										toggleToolOutput={fold.toggleToolOutput}
										subagentExpanded={fold.subagentExpanded}
										toggleSubagent={fold.toggleSubagent}
									/>
										</div>
									{:else if sessionStreaming}
										<div
											class="assistant-stack"
											class:first-turn-destination-hidden={hideFirstTurnDestination()}
										>
											<AssistantThinkingWait
												label={assistantThinkingWait(undefined, now).label}
												detail={assistantThinkingWait(undefined, now).detail}
												color={heroGlowColor}
											/>
										</div>
									{:else}
										<div
											class="assistant-stack"
											class:first-turn-destination-hidden={hideFirstTurnDestination()}
										></div>
									{/if}
								</div>
							{/if}
						{:else if item.type === 'assistant' && showAssistantRow(item) && firstAssistantInNormalList(item)}
							<div
								class="row assistant-row"
								class:continuation-row={!startsSpeakerRun(threadItems, index, 'assistant')}
							>
								{#if startsSpeakerRun(threadItems, index, 'assistant')}
									<ThreadAvatar
										variant="avatar"
										{iconVariant}
										flightHidden={hideAssistantAvatarForFirstTurn(item)}
									/>
								{:else}
									<ThreadAvatar variant="gutter" {iconVariant} />
								{/if}
								<div
									class="assistant-column"
									class:first-turn-destination-hidden={hideAssistantAvatarForFirstTurn(item)}
								>
									<AssistantStack
										{item}
										{threadItems}
										{thinkingForAssistant}
										{streamingAssistantId}
										{sessionStreaming}
										{sessionId}
										{now}
										{heroGlowColor}
										{copiedId}
										showActivitySpinner={showAssistantActivitySpinner(item)}
										{toolFoldLabel}
										onCopyMessage={copyMessage}
										{onNotifyAgent}
										{onStartJob}
										thinkingExpanded={fold.thinkingExpanded}
										toggleThinking={fold.toggleThinking}
										activityGroupExpanded={fold.activityGroupExpanded}
										toggleActivityGroup={fold.toggleActivityGroup}
										memoryInThinkingExpanded={fold.memoryInThinkingExpanded}
										toggleMemoryInThinking={fold.toggleMemoryInThinking}
										toolOutputExpanded={fold.toolOutputExpanded}
										toggleToolOutput={fold.toggleToolOutput}
										subagentExpanded={fold.subagentExpanded}
										toggleSubagent={fold.toggleSubagent}
									/>
								</div>
							</div>
						{:else if item.type === 'tool' && !isToolInBuffer(item) && !embeddedPinnedJobIds.has(item.id)}
							<div
								class="row tool-row"
								class:continuation-row={!startsSpeakerRun(threadItems, index, 'assistant')}
							>
								<ThreadAvatar variant="gutter" {iconVariant} />
								<div class="tool-stack">
									<ToolFoldPanel
										{item}
										label={toolFoldLabel(item)}
										expanded={fold.toolOutputExpanded(item)}
										onToggle={() => fold.toggleToolOutput(item.id)}
										{sessionId}
										{onNotifyAgent}
										{onStartJob}
									/>
								</div>
							</div>
						{:else if item.type === 'subagent' && !isSubagentInBuffer(item)}
							<div
								class="row tool-row subagent-row"
								class:continuation-row={!startsSpeakerRun(threadItems, index, 'assistant')}
							>
								<ThreadAvatar variant="gutter" {iconVariant} />
								<div class="subagent-stack">
									<SubagentPanel
										{item}
										expanded={fold.subagentExpanded(item.id)}
										onToggle={() => fold.toggleSubagent(item.id)}
									/>
								</div>
							</div>
						{:else if item.type === 'memory' && !isMemoryInBuffer(item)}
							<MemoryEventRow {item} {memoryCycleTick} />
						{:else if item.type === 'status'}
							<div class="status">{usageText(item)}</div>
						{:else if item.type === 'error'}
							<div class="row event-row">
								<div class="event-card error-card">
									<div class="event-title">
										<TriangleAlert size={14} /><span>Error</span>
									</div>
									<p>{item.text}</p>
								</div>
							</div>
						{/if}
					{/each}
					{#if bottomSpacerHeight > 0}
						<div
							class="thread-bottom-spacer"
							style:height="{bottomSpacerHeight}px"
							aria-hidden="true"
						></div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
	{#if scroll.showJumpToBottom}
		<JumpToBottom onclick={scroll.jumpToBottom} />
	{/if}
</div>

<style>
	.thread-wrap {
		position: absolute;
		inset: 0;
	}

	.thread {
		position: absolute;
		inset: 0;
		overflow-y: auto;
		overflow-x: hidden;
		padding: 32px var(--chat-gutter) var(--thread-padding-bottom);
	}

	.thread-inner {
		--chat-content-column: min(
			var(--chat-content-max),
			calc(100% - var(--chat-avatar-size) - var(--chat-row-gap))
		);
		--chat-assistant-column: var(--chat-content-column);
		width: min(var(--chat-thread-width), 100%);
		margin: 0 auto;
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.thread-messages {
		display: flex;
		flex-direction: column;
		gap: 14px;
		transition: opacity var(--duration-session-switch) var(--ease-smooth);
	}

	.thread-messages.hydrating {
		opacity: 0;
		pointer-events: none;
	}

	.thread-bottom-spacer {
		flex: 0 0 auto;
		pointer-events: none;
	}

	@media (min-width: 768px) {
		.thread {
			padding: 40px var(--chat-gutter) var(--thread-padding-bottom);
		}

		.thread-inner {
			gap: 16px;
		}
	}

	@media (min-width: 1024px) {
		.thread {
			padding: 48px var(--chat-gutter) var(--thread-padding-bottom);
		}

		.thread-inner {
			gap: 18px;
		}
	}

	@media (min-width: 1280px) {
		.thread {
			padding: 56px var(--chat-gutter) var(--thread-padding-bottom);
		}
	}

	.status {
		align-self: center;
		display: inline-flex;
		align-items: center;
		gap: 7px;
		padding: 6px 10px;
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.74);
		border: 1px solid var(--border-soft);
		font-size: 12px;
		color: var(--text-muted);
	}

	.row {
		display: flex;
		width: 100%;
		gap: var(--chat-row-gap);
	}

	.continuation-row {
		margin-top: -6px;
	}

	.tool-row.continuation-row {
		margin-top: -16px;
	}

	.assistant-row,
	.tool-row,
	.event-row {
		justify-content: flex-start;
	}

	.assistant-row,
	.tool-row {
		align-items: flex-start;
	}

	.tool-stack,
	.assistant-column {
		min-width: 0;
		flex: 1;
		max-width: var(--chat-assistant-column);
	}

	.flight-placeholder {
		pointer-events: none;
	}

	.flight-placeholder :global(.assistant-stack) {
		min-height: 1px;
	}

	.first-turn-destination-hidden {
		visibility: hidden;
	}

	.event-row .event-card {
		max-width: var(--chat-content-column);
	}

	.event-card {
		min-width: 0;
		width: 100%;
		max-width: 100%;
		border: 1px solid var(--border-soft);
		background: rgba(255, 255, 255, 0.68);
		border-radius: 14px;
		padding: 10px 12px;
		color: var(--text-muted);
		box-shadow: 0 6px 18px rgba(15, 23, 42, 0.04);
	}

	.event-title {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 12px;
		font-weight: 650;
		color: var(--text-main);
		margin-bottom: 7px;
	}

	.event-title :global(svg:last-child) {
		flex-shrink: 0;
	}

	.error-card p {
		margin: 0;
		font-size: 12px;
		line-height: 1.5;
		white-space: pre-wrap;
	}

	.error-card {
		color: #b42318;
	}

	.error-card {
		background: rgba(255, 245, 245, 0.82);
		border-color: rgba(244, 63, 94, 0.18);
	}

	.subagent-row {
		align-items: flex-start;
	}

	.subagent-stack {
		min-width: 0;
		flex: 1;
		max-width: var(--chat-assistant-column);
	}
</style>
