<script lang="ts">
	import { page } from '$app/state';
	import ChatView from '$lib/components/ChatView.svelte';
	import { shellStore } from '$lib/stores/shell.svelte';

	let sessionId = $derived(page.params.id);
</script>

{#if sessionId}
	<!--
		SvelteKit reuses this page component across /session/[id] navigations, so
		key ChatView on the id to remount it per session. ChatView then runs its
		own per-session init (select session, consume pending message) on mount.
	-->
	{#key sessionId}
		<ChatView {sessionId} bootMessage={shellStore.bootMessage} />
	{/key}
{/if}
