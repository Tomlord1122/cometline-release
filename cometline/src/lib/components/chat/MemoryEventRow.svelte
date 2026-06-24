<script lang="ts">
	import { fade } from 'svelte/transition';
	import { Brain } from '@lucide/svelte';
	import type { ChatItem } from '$lib/stores/chat.svelte';

	let {
		item,
		memoryCycleTick
	}: {
		item: Extract<ChatItem, { type: 'memory' }>;
		memoryCycleTick: number;
	} = $props();
</script>

<div class="row event-row">
	<div class="event-card memory-card">
		<div class="event-title">
			<Brain size={14} /><span>Memories used · {item.memories.length}</span>
		</div>
		{#if item.memories.length > 0}
			{#key memoryCycleTick}
				{@const mem = item.memories[memoryCycleTick % item.memories.length]}
				<div class="memory-chip-rotator">
					<span
						class="memory-chip memory-chip-cycling"
						in:fade={{ duration: 500 }}
						title={mem.content}
					>{mem.kind}: {mem.content}</span
					>
				</div>
			{/key}
		{/if}
	</div>
</div>

<style>
	.row {
		display: flex;
		width: 100%;
		gap: var(--chat-row-gap);
	}

	.event-row {
		justify-content: flex-start;
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

	.memory-chip {
		display: block;
		width: 100%;
		min-width: 0;
		overflow: hidden;
		white-space: nowrap;
		text-overflow: ellipsis;
		padding: 5px 10px;
		border-radius: 10px;
		background: rgba(0, 102, 204, 0.08);
		color: var(--text-main);
		font-size: 11px;
		line-height: 1.45;
	}

	.memory-chip-rotator {
		display: grid;
		min-width: 0;
		width: 100%;
	}

	.memory-chip-cycling {
		grid-column: 1;
		grid-row: 1;
	}
</style>
