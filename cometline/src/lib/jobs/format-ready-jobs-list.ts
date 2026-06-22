import type { JobResource } from '$lib/client/cometmind';

export function formatReadyJobsList(jobs: JobResource[]): string {
	const ready = jobs.filter((job) => !job.deleted_at);
	if (ready.length === 0) return 'No ready jobs.';
	return ready.map((job) => `• ${job.description}`).join('\n');
}

export function listJobsUserDisplayText(jobs: JobResource[]): string {
	return `/list-jobs\n\n${formatReadyJobsList(jobs)}`;
}
