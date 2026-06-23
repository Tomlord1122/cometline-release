<script lang="ts">
	import { fade, scale } from 'svelte/transition';
	import { untrack } from 'svelte';
	import { Check, ChevronRight, ChevronLeft, LoaderCircle, Sparkles, X } from '@lucide/svelte';
	import { settingsStore } from '$lib/stores/settings.svelte';
	import { shellStore } from '$lib/stores/shell.svelte';
	import { connectionState } from '$lib/stores/runtime.svelte';
	import { cloneProvider } from '$lib/settings/schema';
	import type { ProviderConfig, ProviderMethod, ProviderSettings } from '$lib/types';

	const METHOD_LABELS: Record<ProviderMethod, string> = {
		openai: 'OpenAI',
		anthropic: 'Anthropic',
		'opencode-go': 'OpenCode Go',
		codex: 'ChatGPT Codex',
		'openai-compatible': 'OpenAI-compatible'
	};

	function methodNeedsApiKey(method: ProviderMethod) {
		return method !== 'codex';
	}

	function canFetchModels(provider: ProviderConfig) {
		if (settingsStore.isFetchingModels || !provider.baseURL.trim()) return false;
		return (
			provider.method === 'codex' ||
			provider.method === 'opencode-go' ||
			provider.apiKey.trim().length > 0
		);
	}

	type Step = 'provider' | 'apikey' | 'model' | 'connect';
	const STEP_ORDER: Step[] = ['provider', 'apikey', 'model', 'connect'];
	const STEP_TITLES: Record<Step, string> = {
		provider: 'Choose a provider',
		apikey: 'Connect your account',
		model: 'Pick a model',
		connect: 'Ready to go'
	};

	let step = $state<Step>('provider');
	let saving = $state(false);
	let saveError = $state('');
	let connecting = $state(false);
	let manualModel = $state('');
	let showApiKey = $state(false);

	// Draft built from current settings so the wizard edits don't leak if
	// the user cancels.
	let draft = $state<ProviderSettings>(
		JSON.parse(JSON.stringify(settingsStore.settings)) as ProviderSettings
	);

	let selectedProviderId = $state(
		untrack(() => draft.providers.find((p) => p.id === 'anthropic')?.id ?? draft.providers[0]?.id ?? '')
	);

	let selectedProvider = $derived(
		draft.providers.find((p) => p.id === selectedProviderId) ?? draft.providers[0]
	);

	let stepIndex = $derived(STEP_ORDER.indexOf(step));
	let canAdvance = $derived.by(() => {
		if (step === 'provider') return Boolean(selectedProvider);
		if (step === 'apikey')
			return Boolean(selectedProvider) && (!methodNeedsApiKey(selectedProvider.method) || selectedProvider.apiKey.trim().length > 0);
		if (step === 'model') return Boolean(selectedProvider) && selectedProvider.enabledModels.length > 0;
		return true;
	});

	function patchSelected(patch: Partial<ProviderConfig>) {
		if (!selectedProvider) return;
		draft = {
			...draft,
			providers: draft.providers.map((p) =>
				p.id === selectedProvider.id ? cloneProvider({ ...p, ...patch }) : p
			)
		};
	}

	function selectProvider(id: string) {
		selectedProviderId = id;
		// Reset model state when switching providers.
		manualModel = '';
		patchSelected({ enabledModels: [], selectedModel: '' });
	}

	function next() {
		const idx = STEP_ORDER.indexOf(step);
		if (idx < STEP_ORDER.length - 1) {
			// Skip the apikey step for codex (no key needed).
			const nextStep = STEP_ORDER[idx + 1];
			if (nextStep === 'apikey' && selectedProvider && !methodNeedsApiKey(selectedProvider.method)) {
				step = 'model';
			} else {
				step = nextStep;
			}
		}
	}

	function back() {
		const idx = STEP_ORDER.indexOf(step);
		if (idx > 0) {
			const prevStep = STEP_ORDER[idx - 1];
			if (prevStep === 'apikey' && selectedProvider && !methodNeedsApiKey(selectedProvider.method)) {
				step = 'provider';
			} else {
				step = prevStep;
			}
		}
	}

	async function fetchModels() {
		if (!selectedProvider || !canFetchModels(selectedProvider)) return;
		try {
			const updated = await settingsStore.fetchModelsFor(selectedProvider);
			patchSelected({
				models: updated.models,
				enabledModels: updated.enabledModels,
				selectedModel: updated.selectedModel
			});
		} catch {
			// Error surfaced via settingsStore.error; keep the wizard usable.
		}
	}

	function selectModel(model: string) {
		if (!selectedProvider) return;
		patchSelected({ enabledModels: [model], selectedModel: model });
	}

	function addManualModel() {
		const model = manualModel.trim();
		if (!model || !selectedProvider) return;
		const models = selectedProvider.models.includes(model)
			? selectedProvider.models
			: [...selectedProvider.models, model];
		patchSelected({ models, enabledModels: [model], selectedModel: model });
		manualModel = '';
	}

	async function saveAndConnect() {
		if (!selectedProvider) return;
		// Enable the chosen provider and set it as active + default.
		const finalProviders = draft.providers.map((p) =>
			p.id === selectedProvider.id
				? cloneProvider({ ...p, enabled: true })
				: p
		);
		const finalDraft: ProviderSettings = {
			...draft,
			providers: finalProviders,
			activeProviderId: selectedProvider.id,
			defaultProviderId: selectedProvider.id,
			defaultModelId: selectedProvider.enabledModels[0] ?? selectedProvider.selectedModel
		};

		saving = true;
		saveError = '';
		try {
			await settingsStore.save(finalDraft, { restartCometMind: true });
			draft = JSON.parse(JSON.stringify(settingsStore.settings)) as ProviderSettings;
			// Poll until the sidecar is healthy after restart.
			connecting = true;
			connectionState.reconnect();
			const ready = await waitForReady(20);
			connecting = false;
			if (ready) {
				await settingsStore.markSetupComplete();
				shellStore.closeSetup();
			} else {
				saveError = 'CometMind is still starting up. Your settings were saved — try sending a message in a moment.';
			}
		} catch (err) {
			saveError = err instanceof Error ? err.message : 'Failed to save settings.';
		} finally {
			saving = false;
			connecting = false;
		}
	}

	async function waitForReady(maxAttempts: number): Promise<boolean> {
		for (let i = 0; i < maxAttempts; i++) {
			if (connectionState.status === 'ready') return true;
			await new Promise((resolve) => setTimeout(resolve, 500));
		}
		return false;
	}

	function skip() {
		shellStore.closeSetup();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			skip();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="wizard-layer" transition:fade={{ duration: 120 }}>
	<button class="scrim" aria-label="Close setup wizard" onclick={skip}></button>
	<div
		class="wizard-modal"
		role="dialog"
		aria-modal="true"
		aria-labelledby="wizard-title"
		transition:scale={{ start: 0.97, duration: 140 }}
	>
		<header>
			<div class="title-mark"><Sparkles size={16} /></div>
			<div>
				<h2 id="wizard-title">Welcome to Cometline</h2>
				<p>{STEP_TITLES[step]} — step {stepIndex + 1} of {STEP_ORDER.length}</p>
			</div>
			<button class="close-btn" aria-label="Close" onclick={skip}><X size={16} /></button>
		</header>

		<div class="step-body">
			{#if step === 'provider'}
				<p class="step-intro">
					Pick the LLM provider you'd like to use. You can add more later in Settings.
				</p>
				<div class="provider-list">
					{#each draft.providers as provider (provider.id)}
						<button
							class="provider-option"
							class:selected={provider.id === selectedProviderId}
							onclick={() => selectProvider(provider.id)}
						>
							<div class="provider-option-copy">
								<strong>{provider.name || METHOD_LABELS[provider.method]}</strong>
								<span>{provider.baseURL || 'Custom endpoint'}</span>
							</div>
							{#if provider.id === selectedProviderId}
								<Check size={16} />
							{/if}
						</button>
					{/each}
				</div>
			{:else if step === 'apikey'}
				{#if selectedProvider}
					<p class="step-intro">
						Enter your API key for {selectedProvider.name || METHOD_LABELS[selectedProvider.method]}.
						It's stored locally and never sent anywhere except the provider.
					</p>
					<label class="field">
						<span class="field-label">API key</span>
						<div class="api-key-row">
							<input
								type={showApiKey ? 'text' : 'password'}
								class="field-input"
								placeholder={methodNeedsApiKey(selectedProvider.method) ? 'Paste your API key' : 'Not required'}
								value={selectedProvider.apiKey}
								disabled={!methodNeedsApiKey(selectedProvider.method)}
								oninput={(e) => patchSelected({ apiKey: e.currentTarget.value })}
							/>
							<button
								class="toggle-visibility"
								onclick={() => (showApiKey = !showApiKey)}
								type="button"
							>
								{showApiKey ? 'Hide' : 'Show'}
							</button>
						</div>
					</label>
					<label class="field">
						<span class="field-label">Base URL</span>
						<input
							type="text"
							class="field-input"
							value={selectedProvider.baseURL}
							oninput={(e) => patchSelected({ baseURL: e.currentTarget.value })}
						/>
					</label>
				{/if}
			{:else if step === 'model'}
				{#if selectedProvider}
					<p class="step-intro">
						Choose which model to use. Fetch the available list from the provider, or type a model ID manually.
					</p>
					<div class="fetch-row">
						<button
							class="fetch-btn"
							onclick={fetchModels}
							disabled={!canFetchModels(selectedProvider)}
						>
							{#if settingsStore.isFetchingModels}<LoaderCircle size={14} class="spin" />{/if}
							Fetch models
						</button>
						{#if selectedProvider.models.length > 0}
							<span class="fetch-hint">{selectedProvider.models.length} models available</span>
						{/if}
					</div>
					{#if selectedProvider.models.length > 0}
						<div class="model-list">
							{#each selectedProvider.models as model (model)}
								<button
									class="model-option"
									class:selected={selectedProvider.enabledModels.includes(model)}
									onclick={() => selectModel(model)}
								>
									<span>{model}</span>
									{#if selectedProvider.enabledModels.includes(model)}
										<Check size={14} />
									{/if}
								</button>
							{/each}
						</div>
					{/if}
					<div class="manual-model">
						<input
							type="text"
							class="field-input"
							placeholder="Or type a model ID (e.g. claude-sonnet-4-5)"
							value={manualModel}
							oninput={(e) => (manualModel = e.currentTarget.value)}
							onkeydown={(e) => {
								if (e.key === 'Enter') {
									e.preventDefault();
									addManualModel();
								}
							}}
						/>
						<button class="manual-add" onclick={addManualModel} disabled={!manualModel.trim()}>
							Add
						</button>
					</div>
				{/if}
			{:else if step === 'connect'}
				<p class="step-intro">
					{selectedProvider?.name || 'Your provider'} is ready to go. Save your settings and
					Cometline will connect.
				</p>
				<div class="review">
					<div class="review-row">
						<span>Provider</span>
						<strong>{selectedProvider?.name || METHOD_LABELS[selectedProvider?.method ?? 'anthropic']}</strong>
					</div>
					<div class="review-row">
						<span>Model</span>
						<strong>{(selectedProvider?.enabledModels[0] ?? selectedProvider?.selectedModel) || '—'}</strong>
					</div>
					<div class="review-row">
						<span>API key</span>
						<strong>{methodNeedsApiKey(selectedProvider?.method ?? 'anthropic') ? (selectedProvider?.apiKey ? 'Set' : 'Missing') : 'Not required'}</strong>
					</div>
				</div>
				{#if saveError}
					<p class="wizard-error">{saveError}</p>
				{/if}
			{/if}
		</div>

		<footer>
			<button class="wizard-btn secondary" onclick={skip}>Skip setup</button>
			<div class="footer-right">
				{#if stepIndex > 0}
					<button class="wizard-btn secondary" onclick={back}>
						<ChevronLeft size={14} />
						Back
					</button>
				{/if}
				{#if step !== 'connect'}
					<button class="wizard-btn primary" onclick={next} disabled={!canAdvance}>
						Next
						<ChevronRight size={14} />
					</button>
				{:else}
					<button class="wizard-btn primary" onclick={saveAndConnect} disabled={saving || connecting}>
						{#if saving || connecting}<LoaderCircle size={14} class="spin" />{/if}
						{connecting ? 'Connecting…' : saving ? 'Saving…' : 'Save & connect'}
					</button>
				{/if}
			</div>
		</footer>
	</div>
</div>

<style>
	.wizard-layer {
		position: fixed;
		inset: 0;
		z-index: 85;
		display: grid;
		place-items: center;
		padding: 30px;
	}

	.scrim {
		position: absolute;
		inset: 0;
		border: none;
		background: rgba(17, 24, 39, 0.18);
		backdrop-filter: blur(12px);
	}

	.wizard-modal {
		position: relative;
		width: min(560px, 100%);
		max-height: min(680px, 90vh);
		display: flex;
		flex-direction: column;
		background: var(--panel-bg, #fff);
		border: 1px solid var(--border-soft);
		border-radius: var(--radius-card, 16px);
		box-shadow: var(--shadow-card);
		overflow: hidden;
	}

	header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 18px 20px;
		border-bottom: 1px solid var(--border-soft);
	}

	.title-mark {
		width: 32px;
		height: 32px;
		border-radius: 11px;
		background: rgba(0, 102, 204, 0.09);
		color: var(--accent);
		display: grid;
		place-items: center;
		flex-shrink: 0;
	}

	header h2 {
		margin: 0;
		font-size: 17px;
		font-weight: 600;
		color: var(--text-main);
	}

	header p {
		margin: 2px 0 0;
		font-size: 12px;
		color: var(--text-muted);
	}

	.close-btn {
		margin-left: auto;
		border: none;
		background: transparent;
		color: var(--text-muted);
		cursor: pointer;
		padding: 4px;
		border-radius: 6px;
		display: grid;
		place-items: center;
	}

	.close-btn:hover {
		background: rgba(15, 23, 42, 0.05);
		color: var(--text-main);
	}

	.step-body {
		padding: 20px;
		overflow-y: auto;
		flex: 1;
	}

	.step-intro {
		margin: 0 0 16px;
		font-size: 13px;
		line-height: 1.5;
		color: var(--text-muted);
	}

	.provider-list {
		display: grid;
		gap: 8px;
	}

	.provider-option {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 12px 14px;
		border: 1px solid var(--border-soft);
		border-radius: 10px;
		background: var(--panel-bg, #fff);
		cursor: pointer;
		text-align: left;
		transition: border-color 0.15s ease;
	}

	.provider-option:hover {
		border-color: rgba(0, 102, 204, 0.3);
	}

	.provider-option.selected {
		border-color: var(--accent);
		box-shadow: 0 0 0 2px rgba(0, 102, 204, 0.12);
	}

	.provider-option-copy {
		display: grid;
		gap: 2px;
	}

	.provider-option-copy strong {
		font-size: 13px;
		font-weight: 600;
		color: var(--text-main);
	}

	.provider-option-copy span {
		font-size: 11px;
		color: var(--text-muted);
	}

	.field {
		display: grid;
		gap: 6px;
		margin-bottom: 16px;
	}

	.field-label {
		font-size: 12px;
		font-weight: 600;
		color: var(--text-main);
	}

	.field-input {
		width: 100%;
		padding: 9px 11px;
		border: 1px solid var(--border-soft);
		border-radius: 8px;
		background: var(--panel-bg, #fff);
		color: var(--text-main);
		font: inherit;
		font-size: 13px;
	}

	.field-input:focus {
		outline: none;
		border-color: var(--accent);
		box-shadow: 0 0 0 2px rgba(0, 102, 204, 0.12);
	}

	.api-key-row {
		display: flex;
		gap: 8px;
	}

	.api-key-row .field-input {
		flex: 1;
	}

	.toggle-visibility {
		border: 1px solid var(--border-soft);
		border-radius: 8px;
		background: var(--panel-bg, #fff);
		color: var(--text-muted);
		font: inherit;
		font-size: 12px;
		padding: 0 12px;
		cursor: pointer;
	}

	.toggle-visibility:hover {
		color: var(--text-main);
	}

	.fetch-row {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-bottom: 14px;
	}

	.fetch-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 14px;
		border: 1px solid var(--border-soft);
		border-radius: 8px;
		background: var(--panel-bg, #fff);
		color: var(--text-main);
		font: inherit;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
	}

	.fetch-btn:hover:not(:disabled) {
		border-color: var(--accent);
	}

	.fetch-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.fetch-hint {
		font-size: 12px;
		color: var(--text-muted);
	}

	.model-list {
		display: grid;
		gap: 6px;
		margin-bottom: 16px;
		max-height: 220px;
		overflow-y: auto;
	}

	.model-option {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		padding: 9px 12px;
		border: 1px solid var(--border-soft);
		border-radius: 8px;
		background: var(--panel-bg, #fff);
		cursor: pointer;
		font: inherit;
		font-size: 12px;
		color: var(--text-main);
		text-align: left;
	}

	.model-option:hover {
		border-color: rgba(0, 102, 204, 0.3);
	}

	.model-option.selected {
		border-color: var(--accent);
		box-shadow: 0 0 0 2px rgba(0, 102, 204, 0.12);
	}

	.manual-model {
		display: flex;
		gap: 8px;
	}

	.manual-model .field-input {
		flex: 1;
	}

	.manual-add {
		border: 1px solid var(--border-soft);
		border-radius: 8px;
		background: var(--panel-bg, #fff);
		color: var(--text-main);
		font: inherit;
		font-size: 12px;
		font-weight: 600;
		padding: 0 14px;
		cursor: pointer;
	}

	.manual-add:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.review {
		display: grid;
		gap: 10px;
		padding: 16px;
		border: 1px solid var(--border-soft);
		border-radius: 10px;
	}

	.review-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 13px;
	}

	.review-row span {
		color: var(--text-muted);
	}

	.review-row strong {
		color: var(--text-main);
		font-weight: 600;
	}

	.wizard-error {
		margin: 12px 0 0;
		font-size: 12px;
		color: #b42318;
	}

	footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 16px 20px;
		border-top: 1px solid var(--border-soft);
	}

	.footer-right {
		display: flex;
		gap: 8px;
	}

	.wizard-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 8px 14px;
		border: 1px solid var(--border-soft);
		border-radius: 8px;
		font: inherit;
		font-size: 13px;
		font-weight: 600;
		cursor: pointer;
	}

	.wizard-btn.secondary {
		background: var(--panel-bg, #fff);
		color: var(--text-main);
	}

	.wizard-btn.secondary:hover {
		border-color: rgba(15, 23, 42, 0.18);
	}

	.wizard-btn.primary {
		background: var(--accent);
		color: white;
		border-color: transparent;
	}

	.wizard-btn.primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
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
