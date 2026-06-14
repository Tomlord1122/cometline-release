<script lang="ts">
	import type { HeroComposerAppearance } from '$lib/types';
	import {
		DEFAULT_HERO_COMPOSER_APPEARANCE,
		heroComposerCssVarStyle
	} from '$lib/hero-composer-appearance';

	let {
		appearance = $bindable({ ...DEFAULT_HERO_COMPOSER_APPEARANCE })
	}: { appearance: HeroComposerAppearance } = $props();

	let previewStyle = $derived(heroComposerCssVarStyle(appearance));

	function resetDefaults() {
		appearance = { ...DEFAULT_HERO_COMPOSER_APPEARANCE };
	}
</script>

<section class="appearance-panel">
	<div class="appearance-heading">
		<div>
			<h3>Hero composer glow</h3>
			<p>Customize the rising glow and border on the new-chat composer.</p>
		</div>
		<button class="secondary" type="button" onclick={resetDefaults}>Reset defaults</button>
	</div>

	<div class="appearance-grid">
		<div class="appearance-fields">
			<label>
				<span>Glow color</span>
				<div class="color-field">
					<input type="color" bind:value={appearance.glowColor} aria-label="Glow color" />
					<input
						type="text"
						bind:value={appearance.glowColor}
						spellcheck="false"
						pattern="^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$"
					/>
				</div>
			</label>

			<label>
				<span>Border color</span>
				<div class="color-field">
					<input type="color" bind:value={appearance.ringColor} aria-label="Border color" />
					<input
						type="text"
						bind:value={appearance.ringColor}
						spellcheck="false"
						pattern="^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$"
					/>
				</div>
			</label>
		</div>

		<div class="appearance-preview" style={previewStyle}>
			<div class="preview-glow" aria-hidden="true"></div>
			<div class="preview-ring" aria-hidden="true"></div>
			<div class="preview-card">
				<span class="preview-placeholder">Ask anything...</span>
			</div>
		</div>
	</div>
</section>

<style>
	.appearance-panel {
		border: 1px solid var(--border-soft);
		border-radius: 18px;
		background: rgba(251, 251, 250, 0.72);
		padding: 16px;
	}

	.appearance-heading,
	.color-field {
		display: flex;
		align-items: center;
	}

	.appearance-heading {
		justify-content: space-between;
		gap: 12px;
		margin-bottom: 16px;
	}

	.appearance-heading h3,
	.appearance-heading p {
		margin: 0;
	}

	.appearance-heading h3 {
		font-size: 15px;
		font-weight: 700;
	}

	.appearance-heading p {
		margin-top: 4px;
		font-size: 12px;
		line-height: 1.45;
		color: var(--text-muted);
	}

	.appearance-grid {
		display: grid;
		grid-template-columns: minmax(0, 280px) minmax(0, 1fr);
		gap: 16px;
		align-items: center;
	}

	.appearance-fields {
		display: grid;
		gap: 12px;
	}

	label {
		display: grid;
		gap: 6px;
		font-size: 12px;
		font-weight: 600;
		color: var(--text-muted);
	}

	.color-field {
		gap: 8px;
	}

	input[type='color'] {
		width: 42px;
		height: 38px;
		padding: 2px;
		border: 1px solid var(--border-soft);
		border-radius: 10px;
		background: rgba(255, 255, 255, 0.76);
		cursor: pointer;
	}

	input[type='text'] {
		flex: 1;
		border: 1px solid var(--border-soft);
		border-radius: 11px;
		background: rgba(255, 255, 255, 0.76);
		padding: 10px 11px;
		font: inherit;
		font-size: 13px;
		color: var(--text-main);
		outline: none;
	}

	input[type='text']:focus,
	input[type='color']:focus {
		border-color: rgba(0, 102, 204, 0.35);
		box-shadow: 0 0 0 3px rgba(0, 102, 204, 0.1);
	}

	.appearance-preview {
		position: relative;
		min-height: 168px;
		display: grid;
		place-items: center;
		padding: 28px 20px;
		border-radius: 16px;
		background: linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(248, 250, 252, 0.88));
		border: 1px solid var(--border-soft);
		overflow: hidden;
	}

	.preview-glow,
	.preview-ring {
		position: absolute;
		pointer-events: none;
		border-radius: 24px;
	}

	.preview-glow {
		inset: 36px 18% 28px;
		background:
			radial-gradient(
				ellipse 118% 92% at 50% 100%,
				var(--hero-composer-glow-strong),
				transparent 70%
			),
			radial-gradient(
				ellipse 88% 68% at 50% 0%,
				var(--hero-composer-glow-soft),
				transparent 74%
			);
		filter: blur(16px);
		box-shadow: 0 0 36px var(--hero-composer-glow-ring);
	}

	.preview-ring {
		inset: 44px 22% 36px;
		border: 1px solid var(--hero-composer-ring);
		box-shadow: 0 0 0 1px rgba(255, 255, 255, 0.42) inset;
	}

	.preview-card {
		position: relative;
		z-index: 1;
		width: min(100%, 320px);
		padding: 18px 18px 14px;
		border-radius: 24px;
		background: rgba(255, 255, 255, 0.74);
		border: 1px solid var(--border-soft);
		box-shadow: 0 18px 60px rgba(15, 23, 42, 0.12);
	}

	.preview-placeholder {
		font-size: 14px;
		color: var(--text-soft);
	}

	.secondary {
		border: none;
		border-radius: 10px;
		padding: 8px 11px;
		font: inherit;
		font-size: 12px;
		font-weight: 600;
		background: rgba(15, 23, 42, 0.04);
		color: var(--text-main);
	}

	.secondary:hover {
		background: rgba(15, 23, 42, 0.05);
	}

	@media (max-width: 780px) {
		.appearance-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
