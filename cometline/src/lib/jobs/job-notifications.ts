import { listJobs } from '$lib/client/cometmind';
import type { CometMindJobsNotificationSettings } from '$lib/cometmind-settings';

type JobSnapshot = {
	id: string;
	status: string;
	assigned_session_id?: string | null;
	description: string;
};

export function startJobNotificationPoller(opts: {
	getSettings: () => CometMindJobsNotificationSettings;
	intervalMs?: number;
	onNotify: (title: string, body: string) => void;
}): () => void {
	const intervalMs = opts.intervalMs ?? 30_000;
	const last = new Map<string, JobSnapshot>();

	async function poll() {
		const settings = opts.getSettings();
		if (!settings.enabled) return;
		try {
			const res = await listJobs();
			for (const job of res.jobs ?? []) {
				if (job.deleted_at) continue;
				const snap: JobSnapshot = {
					id: job.id,
					status: job.status,
					assigned_session_id: job.assigned_session_id,
					description: job.description
				};
				const prev = last.get(job.id);
				if (prev) {
					if (
						settings.onClaimed &&
						!prev.assigned_session_id &&
						job.assigned_session_id &&
						job.status === 'ongoing'
					) {
						opts.onNotify('Job claimed', job.description);
					}
					if (settings.onCompleted && prev.status !== 'done' && job.status === 'done') {
						opts.onNotify('Job completed', job.description);
					}
					if (settings.onReleased && prev.status === 'ongoing' && job.status === 'todo') {
						opts.onNotify('Job released', job.description);
					}
				}
				last.set(job.id, snap);
			}
		} catch {
			// Sidecar may be offline.
		}
	}

	void poll();
	const timer = setInterval(() => void poll(), intervalMs);
	return () => clearInterval(timer);
}
