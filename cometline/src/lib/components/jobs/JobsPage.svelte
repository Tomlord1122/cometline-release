<script lang="ts">
	import { LoaderCircle, Trash2 } from '@lucide/svelte';
	import { onMount } from 'svelte';
	import {
		createJob,
		deleteJob,
		listJobEvents,
		listJobs,
		updateJob,
		type JobEventResource,
		type JobResource
	} from '$lib/client/cometmind';

	type StatusFilter = 'todo' | 'ongoing' | 'done' | 'deleted';

	let jobs = $state<JobResource[]>([]);
	let loading = $state(true);
	let error = $state('');
	let statusFilter = $state<StatusFilter>('todo');
	let selectedJob = $state<JobResource | null>(null);
	let events = $state<JobEventResource[]>([]);
	let saving = $state(false);

	let newDescription = $state('');
	let newDod = $state('');
	let newPriority = $state(0);
	let newWorkspacePath = $state('');

	let editDescription = $state('');
	let editDod = $state('');
	let editPriority = $state(0);
	let editWorkspacePath = $state('');

	async function loadJobs() {
		loading = true;
		error = '';
		try {
			const includeDeleted = statusFilter === 'deleted';
			const res = await listJobs({
				status:
					includeDeleted || statusFilter === 'deleted'
						? undefined
						: statusFilter,
				include_deleted: includeDeleted
			});
			jobs = res.jobs ?? [];
			if (statusFilter === 'deleted') {
				jobs = jobs.filter((j) => j.deleted_at);
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load jobs';
		} finally {
			loading = false;
		}
	}

	async function selectJob(job: JobResource) {
		selectedJob = job;
		editDescription = job.description;
		editDod = job.definition_of_done ?? '';
		editPriority = job.priority ?? 0;
		editWorkspacePath = job.workspace_path ?? '';
		try {
			const res = await listJobEvents(job.id);
			events = res.events ?? [];
		} catch {
			events = [];
		}
	}

	async function handleCreate() {
		if (!newDescription.trim()) return;
		saving = true;
		try {
			await createJob({
				description: newDescription.trim(),
				definition_of_done: newDod.trim(),
				priority: newPriority,
				workspace_path: newWorkspacePath.trim() || undefined,
				created_by: 'user',
				source_platform: 'desktop'
			});
			newDescription = '';
			newDod = '';
			newPriority = 0;
			newWorkspacePath = '';
			await loadJobs();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create job';
		} finally {
			saving = false;
		}
	}

	async function handleSaveTodo() {
		if (!selectedJob || selectedJob.status !== 'todo') return;
		saving = true;
		try {
			const updated = await updateJob(selectedJob.id, {
				description: editDescription.trim(),
				definition_of_done: editDod.trim(),
				priority: editPriority,
				workspace_path: editWorkspacePath.trim() || undefined
			});
			selectedJob = updated;
			await loadJobs();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to update job';
		} finally {
			saving = false;
		}
	}

	async function handleDelete(job: JobResource) {
		if (!confirm(`Delete job ${job.id}?`)) return;
		try {
			await deleteJob(job.id);
			if (selectedJob?.id === job.id) selectedJob = null;
			await loadJobs();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete job';
		}
	}

	onMount(() => {
		void loadJobs();
	});

	$effect(() => {
		statusFilter;
		void loadJobs();
	});
</script>

<div class="jobs-page">
	<header class="jobs-header">
		<h1>Jobs</h1>
		<p>Global work queue shared across sessions.</p>
	</header>

	<div class="jobs-filters" role="tablist" aria-label="Job status">
		{#each ['todo', 'ongoing', 'done', 'deleted'] as status (status)}
			<button
				type="button"
				class:active={statusFilter === status}
				onclick={() => (statusFilter = status as StatusFilter)}
			>
				{status}
			</button>
		{/each}
	</div>

	{#if error}
		<p class="jobs-error">{error}</p>
	{/if}

	<div class="jobs-layout">
		<section class="jobs-list-panel">
			{#if loading}
				<p class="jobs-muted"><LoaderCircle size={16} class="spin" /> Loading…</p>
			{:else if jobs.length === 0}
				<p class="jobs-muted">No jobs in this view.</p>
			{:else}
				<ul class="jobs-list">
					{#each jobs as job (job.id)}
						<li>
							<button
								type="button"
								class:selected={selectedJob?.id === job.id}
								onclick={() => selectJob(job)}
							>
								<span class="job-id">{job.id.slice(0, 8)}…</span>
								<span class="job-desc">{job.description}</span>
								<span class="job-meta">p={job.priority}</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}

			{#if statusFilter === 'todo'}
				<form class="jobs-create" onsubmit={(e) => { e.preventDefault(); void handleCreate(); }}>
					<h2>New job</h2>
					<label>
						<span>Description</span>
						<textarea bind:value={newDescription} rows="2" required></textarea>
					</label>
					<label>
						<span>Definition of done</span>
						<textarea bind:value={newDod} rows="2"></textarea>
					</label>
					<label>
						<span>Priority</span>
						<input type="number" bind:value={newPriority} />
					</label>
					<label>
						<span>Workspace path (optional)</span>
						<input type="text" bind:value={newWorkspacePath} placeholder="/path/to/project" />
					</label>
					<button type="submit" disabled={saving || !newDescription.trim()}>Create</button>
				</form>
			{/if}
		</section>

		<section class="jobs-detail-panel">
			{#if selectedJob}
				<header class="detail-header">
					<h2>{selectedJob.description}</h2>
					<button type="button" class="icon-btn" title="Delete" onclick={() => handleDelete(selectedJob!)}>
						<Trash2 size={16} />
					</button>
				</header>
				<p class="jobs-muted">ID: {selectedJob.id} · {selectedJob.status}</p>
				{#if selectedJob.assigned_session_id}
					<p class="jobs-muted">Assigned: {selectedJob.assigned_session_id}</p>
				{/if}
				{#if selectedJob.progress}
					<div class="progress-block">
						<h3>Progress</h3>
						<pre>{selectedJob.progress}</pre>
					</div>
				{/if}
				{#if selectedJob.status === 'todo'}
					<form class="jobs-edit" onsubmit={(e) => { e.preventDefault(); void handleSaveTodo(); }}>
						<label>
							<span>Description</span>
							<textarea bind:value={editDescription} rows={2}></textarea>
						</label>
						<label>
							<span>Definition of done</span>
							<textarea bind:value={editDod} rows={2}></textarea>
						</label>
						<label>
							<span>Priority</span>
							<input type="number" bind:value={editPriority} />
						</label>
						<label>
							<span>Workspace path</span>
							<input type="text" bind:value={editWorkspacePath} />
						</label>
						<button type="submit" disabled={saving}>Save</button>
					</form>
				{/if}
				{#if events.length > 0}
					<div class="events-block">
						<h3>Events</h3>
						<ul>
							{#each events as ev (ev.id)}
								<li><code>{ev.action}</code> {ev.detail}</li>
							{/each}
						</ul>
					</div>
				{/if}
			{:else}
				<p class="jobs-muted">Select a job to view details.</p>
			{/if}
		</section>
	</div>
</div>

<style>
	.jobs-page {
		padding: 1.5rem 2rem;
		max-width: 1100px;
		margin: 0 auto;
		height: 100%;
		overflow: auto;
	}
	.jobs-header h1 {
		margin: 0 0 0.25rem;
		font-size: 1.5rem;
	}
	.jobs-header p {
		margin: 0 0 1rem;
		color: var(--text-muted, #888);
	}
	.jobs-filters {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}
	.jobs-filters button {
		padding: 0.35rem 0.75rem;
		border-radius: 6px;
		border: 1px solid var(--border-subtle, #333);
		background: transparent;
		cursor: pointer;
	}
	.jobs-filters button.active {
		background: var(--accent-muted, #334);
	}
	.jobs-layout {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1.5rem;
		align-items: start;
	}
	.jobs-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.jobs-list button {
		width: 100%;
		text-align: left;
		padding: 0.6rem 0.75rem;
		border: none;
		border-radius: 8px;
		background: transparent;
		cursor: pointer;
		display: grid;
		grid-template-columns: auto 1fr auto;
		gap: 0.5rem;
		align-items: center;
	}
	.jobs-list button:hover,
	.jobs-list button.selected {
		background: var(--surface-hover, rgba(255, 255, 255, 0.06));
	}
	.job-id {
		font-family: monospace;
		font-size: 0.75rem;
		opacity: 0.7;
	}
	.job-desc {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.job-meta {
		font-size: 0.75rem;
		opacity: 0.6;
	}
	.jobs-create,
	.jobs-edit {
		margin-top: 1.5rem;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}
	label {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
		font-size: 0.85rem;
	}
	textarea,
	input {
		padding: 0.5rem;
		border-radius: 6px;
		border: 1px solid var(--border-subtle, #333);
		background: var(--surface-elevated, #1a1a1a);
		color: inherit;
	}
	.jobs-muted {
		color: var(--text-muted, #888);
	}
	.jobs-error {
		color: #f66;
	}
	.detail-header {
		display: flex;
		justify-content: space-between;
		align-items: start;
		gap: 1rem;
	}
	.detail-header h2 {
		margin: 0;
		font-size: 1.1rem;
	}
	.icon-btn {
		border: none;
		background: transparent;
		cursor: pointer;
		opacity: 0.7;
	}
	.progress-block pre,
	.events-block ul {
		font-size: 0.85rem;
		white-space: pre-wrap;
	}
	:global(.spin) {
		animation: spin 1s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
