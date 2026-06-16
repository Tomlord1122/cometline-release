<script lang="ts">
	import SettingsToggle from '$lib/components/SettingsToggle.svelte';

	let {
		openAtLogin = $bindable(false),
		onOpenAtLoginChange
	}: {
		openAtLogin: boolean;
		onOpenAtLoginChange?: (enabled: boolean) => void | Promise<void>;
	} = $props();
</script>

<section class="general-panel">
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
</style>
