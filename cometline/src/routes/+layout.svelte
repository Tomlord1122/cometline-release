<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import AppShell from '$lib/components/AppShell.svelte';
	import { connectionState } from '$lib/stores/runtime.svelte';
	import { settingsStore } from '$lib/stores/settings.svelte';
	import { runtimeProviders } from '$lib/stores/settings.svelte';
	import { sessionStore } from '$lib/stores/session.svelte';
	import { shellStore } from '$lib/stores/shell.svelte';
	import { heroComposerCssVars } from '$lib/hero-composer-appearance';
	import { ensureWorkspace, listAllSessions } from '$lib/client/cometmind';
	import { startJobNotificationPoller } from '$lib/jobs/job-notifications';

	let { children } = $props();

	let settingsLoaded = $state(false);
	// Prevents the setup wizard from re-opening after the user skips it.
	let setupAutoTriggered = false;

	onMount(() => {
		connectionState.startPolling();
		const stopJobNotifications = startJobNotificationPoller({
			getSettings: () => settingsStore.settings.cometmind.jobs.notifications,
			onNotify: (title, body) => {
				window.electronAPI?.notifyJob?.({ title, body });
			}
		});
		void settingsStore.load().then(() => {
			settingsLoaded = true;
			// The sync localStorage read in shell.svelte.ts already sets introOpen
			// correctly for the first frame. This IPC result is the authoritative
			// source and handles edge cases:
			// - localStorage cleared but JSON file still has hasSeenIntro=true
			//   → close any intro that the sync read left open.
			// - Fresh install with no localStorage → hasSeenIntro=false
			//   → intro already open; openIntro() is a no-op.
			if (settingsStore.settings.app.hasSeenIntro) {
				shellStore.closeIntro();
			} else {
				shellStore.openIntro();
			}
		});
		void initializeWorkspace();
		return () => {
			connectionState.stopPolling();
			stopJobNotifications();
		};
	});

	// Auto-open the setup wizard once when the intro has finished and the user
	// hasn't completed setup, or when no provider is configured (state fallback).
	$effect(() => {
		if (!settingsLoaded || shellStore.introOpen || shellStore.setupOpen) return;
		if (setupAutoTriggered) return;
		const hasProvider = runtimeProviders(settingsStore.settings).length > 0;
		if (!settingsStore.settings.app.hasCompletedSetup || !hasProvider) {
			setupAutoTriggered = true;
			shellStore.openSetup();
		}
	});

	$effect(() => {
		const vars = heroComposerCssVars(settingsStore.settings.appearance.heroComposer);
		const root = document.documentElement;
		for (const [key, value] of Object.entries(vars)) {
			root.style.setProperty(key, value);
		}
	});

	$effect(() => {
		if (connectionState.status !== 'ready') return;
		if (shellStore.bootMessage) {
			shellStore.setBootMessage('');
		}
		if (!sessionsLoaded) {
			sessionsLoaded = true;
			void loadSessions();
		}
	});

	let sessionsLoaded = false;
	let lastEnsuredWorkspace = '';

	$effect(() => {
		const workspacePath = shellStore.workspacePath;
		if (!workspacePath || workspacePath === '/') return;
		if (connectionState.status !== 'ready') return;
		// Register the workspace in the background (needed for forks / new
		// chats) without blocking the session list or transcript loads. Only
		// re-register when the path actually changes.
		if (workspacePath !== lastEnsuredWorkspace) {
			lastEnsuredWorkspace = workspacePath;
			void ensureWorkspace(workspacePath).catch(() => {});
		}
	});

	async function initializeWorkspace() {
		try {
			const workspacePath = (await window.electronAPI?.getWorkspacePath?.()) ?? '/';
			shellStore.initializeDefaultWorkspace(workspacePath);
		} catch (err) {
			shellStore.setBootMessage(
				err instanceof Error ? err.message : 'Failed to initialize workspace'
			);
		}
	}

	async function loadSessions() {
		try {
			const result = await listAllSessions();
			sessionStore.setSessions(result.sessions);
			shellStore.setBootMessage('');
		} catch (err) {
			if (connectionState.status === 'connecting') {
				// Let the runtime overlay own startup copy while the sidecar is still
				// warming up; we'll retry automatically once it reports healthy.
				sessionsLoaded = false;
				return;
			}
			// Allow a later workspace/effect tick to retry (e.g. backend not
			// healthy yet at startup).
			sessionsLoaded = false;
			shellStore.setBootMessage(
				err instanceof Error ? err.message : 'Failed to load sessions'
			);
		}
	}
</script>

<AppShell>
	{@render children()}
</AppShell>
