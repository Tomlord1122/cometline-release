<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import { connectionState } from '$lib/stores/runtime.svelte';
	import { settingsStore } from '$lib/stores/settings.svelte';
	import { sessionStore } from '$lib/stores/session.svelte';
	import { shellStore } from '$lib/stores/shell.svelte';
	import { heroComposerCssVars } from '$lib/hero-composer-appearance';
	import { listSessions } from '$lib/client/cometmind';

	let { children } = $props();

	onMount(() => {
		connectionState.startPolling();
		void settingsStore.load();
		void initialize();
		return () => connectionState.stopPolling();
	});

	$effect(() => {
		const vars = heroComposerCssVars(settingsStore.settings.appearance.heroComposer);
		const root = document.documentElement;
		for (const [key, value] of Object.entries(vars)) {
			root.style.setProperty(key, value);
		}
	});

	async function initialize() {
		try {
			const workspacePath = (await window.electronAPI?.getWorkspacePath?.()) ?? '/';
			shellStore.setWorkspacePath(workspacePath);
			const result = await listSessions(workspacePath);
			sessionStore.setSessions(result.sessions);
			shellStore.setBootMessage('');
		} catch (err) {
			shellStore.setBootMessage(
				err instanceof Error ? err.message : 'Failed to load sessions'
			);
		}
	}
</script>

<AppShell workspacePath={shellStore.workspacePath}>
	{@render children()}
</AppShell>
