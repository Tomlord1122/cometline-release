<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import { connectionState } from '$lib/stores/runtime.svelte';
	import { settingsStore } from '$lib/stores/settings.svelte';
	import { sessionStore } from '$lib/stores/session.svelte';
	import { shellStore } from '$lib/stores/shell.svelte';
	import { heroComposerCssVars } from '$lib/hero-composer-appearance';
	import { ensureWorkspace, listAllSessions } from '$lib/client/cometmind';

	let { children } = $props();

	onMount(() => {
		connectionState.startPolling();
		void settingsStore.load().then(() => {
			// First launch: play the cinematic intro once.
			if (!settingsStore.settings.app.hasSeenIntro) {
				shellStore.openIntro();
			}
		});
		void initializeWorkspace();
		return () => connectionState.stopPolling();
	});

	$effect(() => {
		const vars = heroComposerCssVars(settingsStore.settings.appearance.heroComposer);
		const root = document.documentElement;
		for (const [key, value] of Object.entries(vars)) {
			root.style.setProperty(key, value);
		}
	});

	$effect(() => {
		const workspacePath = shellStore.workspacePath;
		if (workspacePath && workspacePath !== '/') {
			void loadSessions(workspacePath);
		}
	});

	async function initializeWorkspace() {
		try {
			const workspacePath = (await window.electronAPI?.getWorkspacePath?.()) ?? '/';
			shellStore.setWorkspacePath(workspacePath);
		} catch (err) {
			shellStore.setBootMessage(
				err instanceof Error ? err.message : 'Failed to initialize workspace'
			);
		}
	}

	async function loadSessions(workspacePath: string) {
		try {
			await ensureWorkspace(workspacePath);
			const result = await listAllSessions();
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
