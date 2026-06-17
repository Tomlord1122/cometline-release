<script lang="ts">
	import { tick } from 'svelte';
	import { fly, fade } from 'svelte/transition';
	import { Check, ChevronDown, Search, Sparkles } from '@lucide/svelte';
	import SettingsToggle from '$lib/components/SettingsToggle.svelte';
	import type { CometMindStorageSettings } from '$lib/settings/schema';
	import type { ProviderConfig } from '$lib/types';
	import { isEmbeddingModelName } from '$lib/embedding-models';

	interface ModelEntry {
		id: string;
		label: string;
		providerId: string;
		providerName: string;
		modelId: string;
	}

	let {
		openAtLogin = $bindable(false),
		storage = $bindable<CometMindStorageSettings>(),
		defaultModelId = $bindable(''),
		defaultProviderId = $bindable(''),
		providers = [],
		onOpenAtLoginChange
	}: {
		openAtLogin: boolean;
		storage: CometMindStorageSettings;
		defaultModelId: string;
		defaultProviderId: string;
		providers: ProviderConfig[];
		onOpenAtLoginChange?: (enabled: boolean) => void | Promise<void>;
	} = $props();

	let modelMenuOpen = $state(false);
	let modelSearch = $state('');
	let modelSearchInput = $state<HTMLInputElement | null>(null);

	function labelForModel(modelID: string) {
		return modelID
			.split(/[_/]+/)
			.filter(Boolean)
			.map((part) => part.charAt(0).toUpperCase() + part.slice(1).toUpperCase())
			.join(' ');
	}

	let modelOptions = $derived.by(() => {
		const options: ModelEntry[] = [];
		for (const provider of providers) {
			if (!provider.enabled) continue;
			for (const modelId of provider.enabledModels) {
				if (isEmbeddingModelName(modelId)) continue;
				options.push({
					id: `${provider.id}:${modelId}`,
					label: labelForModel(modelId),
					providerId: provider.id,
					providerName: provider.name || provider.id,
					modelId
				});
			}
		}
		return options;
	});

	let filteredModelOptions = $derived.by(() => {
		const query = modelSearch.trim().toLowerCase();
		if (!query) return modelOptions;
		return modelOptions.filter(
			(option) =>
				option.label.toLowerCase().includes(query) ||
				option.modelId.toLowerCase().includes(query) ||
				option.providerName.toLowerCase().includes(query)
		);
	});

	let groupedModelOptions = $derived.by(() => {
		const groups: {
			providerId: string;
			providerName: string;
			options: ModelEntry[];
		}[] = [];
		for (const option of filteredModelOptions) {
			let group = groups.find((item) => item.providerId === option.providerId);
			if (!group) {
				group = {
					providerId: option.providerId,
					providerName: option.providerName,
					options: []
				};
				groups.push(group);
			}
			group.options.push(option);
		}
		return groups;
	});

	let selectedLabel = $derived.by(() => {
		if (!defaultModelId || !defaultProviderId) return 'First enabled model';
		const match = modelOptions.find(
			(o) => o.providerId === defaultProviderId && o.modelId === defaultModelId
		);
		return match?.label ?? 'First enabled model';
	});

	function selectDefaultModel(option: ModelEntry) {
		defaultModelId = option.modelId;
		defaultProviderId = option.providerId;
		modelMenuOpen = false;
		modelSearch = '';
	}

	function clearDefaultModel() {
		defaultModelId = '';
		defaultProviderId = '';
		modelMenuOpen = false;
		modelSearch = '';
	}

	async function openModelMenu() {
		if (modelOptions.length === 0) return;
		modelMenuOpen = true;
		modelSearch = '';
		await tick();
		modelSearchInput?.focus();
		modelSearchInput?.select();
	}

	function toggleModelMenu() {
		if (modelMenuOpen) {
			modelMenuOpen = false;
			modelSearch = '';
			return;
		}
		void openModelMenu();
	}

	function closeModelMenu(e: FocusEvent) {
		const next = e.relatedTarget as Node | null;
		const current = e.currentTarget as Node;
		if (next && current.contains(next)) return;
		modelMenuOpen = false;
		modelSearch = '';
	}

	function patchStorage(patch: Partial<CometMindStorageSettings>) {
		storage = { ...storage, ...patch };
	}

	function onRetentionDaysInput(event: Event) {
		const value = Number((event.currentTarget as HTMLInputElement).value);
		patchStorage({ retentionDays: Number.isFinite(value) ? Math.max(0, Math.floor(value)) : 0 });
	}

	function onMaxSessionsInput(event: Event) {
		const value = Number((event.currentTarget as HTMLInputElement).value);
		patchStorage({
			maxSessionsPerWorkspace: Number.isFinite(value) ? Math.max(0, Math.floor(value)) : 0
		});
	}

	function onPurgeDaysInput(event: Event) {
		const value = Number((event.currentTarget as HTMLInputElement).value);
		patchStorage({
			archivedMemoryPurgeDays: Number.isFinite(value) ? Math.max(0, Math.floor(value)) : 0
		});
	}
</script>

<section class="general-panel">
	<div class="section-block">
		<div class="section-heading">
			<h3>Default model</h3>
			<p>Choose which model new chats use by default. You can still switch models per session.</p>
		</div>
		<div class="default-model-picker" onfocusout={closeModelMenu}>
			<button
				class="model-button"
				aria-label="Select default model"
				aria-expanded={modelMenuOpen}
				disabled={modelOptions.length === 0}
				onclick={toggleModelMenu}
			>
				<Sparkles size={14} stroke-width={1.8} />
				<span>{selectedLabel}</span>
				<ChevronDown size={12} stroke-width={2} />
			</button>
			{#if defaultModelId}
				<button class="clear-default-button" onclick={clearDefaultModel} title="Clear default (use first enabled model)">
					&times;
				</button>
			{/if}

			{#if modelMenuOpen}
				<div class="model-menu" transition:fly={{ y: 6, duration: 120 }}>
					<input
						class="model-search"
						bind:this={modelSearchInput}
						bind:value={modelSearch}
						placeholder="Search models..."
						spellcheck="false"
					/>
					{#each groupedModelOptions as group (group.providerId)}
						<div class="model-group" transition:fade={{ duration: 90 }}>
							<div class="model-group-heading">
								<strong>{group.providerName}</strong>
							</div>
							{#each group.options as option (option.id)}
								<button
									class="model-option"
									onclick={() => selectDefaultModel(option)}
								>
									<span class="model-check">
										{#if option.providerId === defaultProviderId && option.modelId === defaultModelId}<Check
												size={14}
												stroke-width={2}
											/>{/if}
									</span>
									<span class="model-option-copy">
										<strong>{option.label}</strong>
										<small>{option.modelId}</small>
									</span>
								</button>
							{/each}
						</div>
					{:else}
						<p class="model-empty">No enabled models match your search.</p>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<div class="section-block">
		<div class="section-heading">
			<h3>Startup</h3>
			<p>Control how Cometline launches on your Mac.</p>
		</div>
		<SettingsToggle
			label="Open at login"
			description="Launch Cometline when you sign in. On macOS 13+, you may need to approve it in System Settings → Login Items."
			bind:checked={openAtLogin}
			disabled={!window.electronAPI?.setOpenAtLogin}
			onchange={onOpenAtLoginChange}
		/>
	</div>

	<div class="section-block">
		<div class="section-heading">
			<h3>Storage & retention</h3>
			<p>
				Automatic cleanup runs when CometMind starts. Set a field to 0 to disable that rule.
			</p>
		</div>

		<label class="field">
			<span>Session retention (days)</span>
			<input
				type="number"
				min="0"
				step="1"
				value={storage.retentionDays}
				oninput={onRetentionDaysInput}
			/>
			<small>
				{#if storage.retentionDays === 0}
					Disabled — sessions are not deleted by age.
				{:else}
					Delete sessions with no activity for {storage.retentionDays} days.
				{/if}
			</small>
		</label>

		<label class="field">
			<span>Max sessions per workspace</span>
			<input
				type="number"
				min="0"
				step="1"
				value={storage.maxSessionsPerWorkspace}
				oninput={onMaxSessionsInput}
			/>
			<small>
				{#if storage.maxSessionsPerWorkspace === 0}
					Disabled — no limit on session count.
				{:else}
					Keep the {storage.maxSessionsPerWorkspace} most recently updated sessions; delete older ones.
				{/if}
			</small>
		</label>

		<label class="field">
			<span>Purge archived memories (days)</span>
			<input
				type="number"
				min="0"
				step="1"
				value={storage.archivedMemoryPurgeDays}
				oninput={onPurgeDaysInput}
			/>
			<small>
				{#if storage.archivedMemoryPurgeDays === 0}
					Disabled — archived memories stay on disk.
				{:else}
					Hard-delete archived memories older than {storage.archivedMemoryPurgeDays} days.
				{/if}
			</small>
		</label>

		<SettingsToggle
			label="Vacuum database after purge"
			description="Reclaim disk space in cometmind.db after sessions or memories are deleted."
			checked={storage.vacuumAfterPurge}
			onchange={(enabled) => patchStorage({ vacuumAfterPurge: enabled })}
		/>

		<p class="discord-note">
			Deleting a session also removes its Discord channel mapping. The next message in that channel
			starts a fresh session without prior Cometline history.
		</p>
	</div>
</section>

<style>
	.general-panel {
		display: flex;
		flex-direction: column;
		gap: 28px;
	}

	.section-block {
		display: flex;
		flex-direction: column;
		gap: 14px;
	}

	.section-heading h3 {
		margin: 0 0 4px;
		font-size: 15px;
		font-weight: 650;
		color: var(--text-main);
	}

	.section-heading p {
		margin: 0;
		font-size: 12px;
		line-height: 1.5;
		color: var(--text-muted);
	}

	.default-model-picker {
		position: relative;
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.model-button {
		display: inline-flex;
		align-items: center;
		gap: 7px;
		padding: 8px 12px;
		border-radius: 11px;
		border: 1px solid var(--border-soft);
		background: rgba(255, 255, 255, 0.76);
		color: var(--text-main);
		font: inherit;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: border-color 0.15s, box-shadow 0.15s;
	}

	.model-button:hover:not(:disabled) {
		border-color: rgba(0, 102, 204, 0.3);
	}

	.model-button:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.clear-default-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		border: none;
		background: rgba(0, 0, 0, 0.06);
		color: var(--text-muted);
		font-size: 14px;
		line-height: 1;
		cursor: pointer;
		transition: background 0.15s;
	}

	.clear-default-button:hover {
		background: rgba(0, 0, 0, 0.12);
		color: var(--text-main);
	}

	.model-menu {
		position: absolute;
		top: calc(100% + 6px);
		left: 0;
		z-index: 100;
		min-width: 280px;
		max-height: 320px;
		overflow-y: auto;
		padding: 6px;
		border-radius: 12px;
		border: 1px solid var(--border-soft);
		background: rgba(255, 255, 255, 0.96);
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
		backdrop-filter: blur(20px);
	}

	.model-search {
		width: 100%;
		padding: 8px 10px;
		border-radius: 8px;
		border: 1px solid var(--border-soft);
		background: rgba(255, 255, 255, 0.8);
		font: inherit;
		font-size: 12px;
		outline: none;
		margin-bottom: 4px;
	}

	.model-search:focus {
		border-color: rgba(0, 102, 204, 0.35);
	}

	.model-search::placeholder {
		color: var(--text-muted);
	}

	.model-group {
		margin-top: 4px;
	}

	.model-group-heading {
		padding: 4px 8px 2px;
		font-size: 11px;
		font-weight: 600;
		color: var(--text-muted);
		text-transform: uppercase;
		letter-spacing: 0.03em;
	}

	.model-option {
		display: flex;
		align-items: center;
		gap: 8px;
		width: 100%;
		padding: 7px 8px;
		border: none;
		border-radius: 8px;
		background: transparent;
		color: var(--text-main);
		font: inherit;
		font-size: 13px;
		cursor: pointer;
		text-align: left;
	}

	.model-option:hover {
		background: rgba(0, 102, 204, 0.08);
	}

	.model-check {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		flex-shrink: 0;
		color: rgba(0, 102, 204, 0.8);
	}

	.model-option-copy {
		display: flex;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
	}

	.model-option-copy strong {
		font-weight: 550;
		font-size: 13px;
	}

	.model-option-copy small {
		font-size: 11px;
		color: var(--text-muted);
	}

	.model-empty {
		padding: 12px 8px;
		margin: 0;
		text-align: center;
		font-size: 12px;
		color: var(--text-muted);
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
		font-size: 13px;
		color: var(--text-main);
	}

	.field input {
		max-width: 160px;
		padding: 10px 11px;
		border-radius: 11px;
		border: 1px solid var(--border-soft);
		background: rgba(255, 255, 255, 0.76);
		color: var(--text-main);
		font: inherit;
		font-size: 13px;
		outline: none;
	}

	.field input:focus {
		border-color: rgba(0, 102, 204, 0.35);
		box-shadow: 0 0 0 3px rgba(0, 102, 204, 0.1);
	}

	.field small {
		font-size: 12px;
		line-height: 1.45;
		color: var(--text-muted);
	}

	.discord-note {
		margin: 4px 0 0;
		padding: 10px 12px;
		border-radius: 8px;
		background: color-mix(in srgb, var(--text-muted) 8%, transparent);
		font-size: 12px;
		line-height: 1.5;
		color: var(--text-muted);
	}
</style>
