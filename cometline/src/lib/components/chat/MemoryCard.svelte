<script lang="ts">
	import { fade, slide } from 'svelte/transition';
	import { Brain, ChevronDown } from '@lucide/svelte';
	import type { InjectedMemory } from '$lib/conversation/thinking-attribution';

	const FOLD_IN = { duration: 180 };
	const CHIP_FADE = { duration: 400 };

	let {
		memories,
		expanded,
		onToggle,
		nested = false,
		contentOnly = false,
		cycling = false
	}: {
		memories: InjectedMemory[];
		expanded: boolean;
		onToggle: () => void;
		nested?: boolean;
		contentOnly?: boolean;
		cycling?: boolean;
	} = $props();

	let cycleTick = $state(0);

	$effect(() => {
		if (!cycling) return;
		const timer = setInterval(() => cycleTick++, 5000);
		return () => clearInterval(timer);
	});
</script>

{#snippet memoryBodyContent()}
	{#if cycling && memories.length > 0}
		{#key cycleTick}
			{@const mem = memories[cycleTick % memories.length]}
			<div class="memory-chip-cycling-wrap">
				<div
					class="memory-chip memory-chip-cycling"
					in:fade={CHIP_FADE}
					title={mem.content}
				>
					{mem.kind}: {mem.content}
				</div>
			</div>
		{/key}
	{:else}
		<div class="memory-chips">
			{#each memories as mem (mem.id)}
				<span class="memory-chip" title={mem.content}>
					{mem.kind}: {mem.content}
				</span>
			{/each}
		</div>
	{/if}
{/snippet}

<div class="fold-panel memory-panel" class:nested class:content-only={contentOnly}>
	{#if !contentOnly}
		<button
			type="button"
			class="fold-toggle memory-toggle"
			aria-expanded={expanded}
			onclick={onToggle}
		>
			<Brain size={13} />
			<span>Memories used · {memories.length}</span>
			<ChevronDown size={13} class={expanded ? 'expanded' : ''} />
		</button>
	{/if}
	{#if contentOnly}
		<div class="fold-body memory-body">
			{@render memoryBodyContent()}
		</div>
	{:else if expanded}
		<div class="fold-body memory-body" transition:slide={FOLD_IN}>
			{@render memoryBodyContent()}
		</div>
	{/if}
</div>

<style>
	/* Base .fold-panel / .fold-toggle / .fold-body styles live in
	   src/lib/styles/fold-panel.css. Only component-specific overrides here. */
	.memory-panel {
		min-width: 0;
		max-width: 100%;
	}

	.fold-panel.nested {
		align-self: stretch;
	}

	.fold-panel.nested .fold-toggle {
		align-self: stretch;
	}

	.fold-panel.content-only .memory-body {
		border: none;
		background: transparent;
		padding: 0;
	}

	.memory-body {
		width: 100%;
		box-sizing: border-box;
		min-width: 0;
		border: 1px solid var(--border-soft);
		border-radius: 10px;
		padding: 8px 10px;
		background: rgba(0, 102, 204, 0.04);
	}

	.memory-chips {
		display: flex;
		min-width: 0;
		flex-direction: column;
		gap: 6px;
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

	.memory-chip-cycling-wrap {
		display: grid;
		min-width: 0;
		width: 100%;
	}

	.memory-chip-cycling {
		grid-column: 1;
		grid-row: 1;
	}
</style>
