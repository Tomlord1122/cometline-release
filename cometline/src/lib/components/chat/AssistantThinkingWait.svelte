<script lang="ts">
	import { Brain, LoaderCircle } from '@lucide/svelte';

	let {
		label,
		detail
	}: {
		label: string;
		detail: string;
	} = $props();
</script>

<div class="fold-panel thinking-panel thinking-wait" aria-live="polite">
	<div class="fold-toggle thinking-toggle thinking-wait-toggle" aria-busy="true">
		<Brain size={13} />
		<span>{label}</span>
		<LoaderCircle size={12} class="spin" />
	</div>
	<div class="fold-body thinking-body">
		<div class="thinking-reasoning">
			<p>{detail}</p>
		</div>
	</div>
</div>

<style>
	.fold-panel {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.thinking-wait-toggle {
		cursor: default;
		pointer-events: none;
	}

	.fold-toggle {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		align-self: flex-start;
		border: 1px solid var(--border-soft);
		background: rgba(255, 255, 255, 0.72);
		color: var(--text-muted);
		border-radius: 999px;
		padding: 5px 10px;
		font-size: 12px;
		font-weight: 600;
	}

	.fold-body {
		border: 1px solid var(--border-soft);
		background: rgba(255, 255, 255, 0.68);
		border-radius: 12px;
		padding: 10px 12px;
		color: var(--text-muted);
		box-shadow: 0 6px 18px rgba(15, 23, 42, 0.04);
	}

	.thinking-body {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.thinking-reasoning p {
		margin: 0;
		font-size: 12px;
		line-height: 1.5;
		white-space: pre-wrap;
		word-break: break-word;
		color: var(--text-muted);
	}

	:global(.spin) {
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
