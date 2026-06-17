import { describe, expect, it } from 'vitest';
import { groupSessionsByWorkspace, workspaceLabel } from './group-by-workspace';
import type { Session } from '$lib/types';

function session(id: string, workspacePath: string, updatedAt: number): Session {
	return {
		id,
		workspace_id: workspacePath,
		workspace_path: workspacePath,
		title: id,
		model_id: 'm',
		provider_id: 'p',
		status: 'active',
		token_usage: { input_tokens: 0, output_tokens: 0, cache_read: 0, cache_write: 0 },
		created_at: updatedAt,
		updated_at: updatedAt
	} as Session;
}

describe('workspaceLabel', () => {
	it('returns the final path segment', () => {
		expect(workspaceLabel('/Users/me/projects/app')).toBe('app');
		expect(workspaceLabel('/Users/me/projects/app/')).toBe('app');
	});
});

describe('groupSessionsByWorkspace', () => {
	it('groups sessions by workspace path', () => {
		const sessions = [
			session('a', '/ws/one', 30),
			session('b', '/ws/two', 20),
			session('c', '/ws/one', 10)
		];
		const groups = groupSessionsByWorkspace(sessions);
		expect(groups).toHaveLength(2);
		const one = groups.find((g) => g.workspacePath === '/ws/one');
		expect(one?.sessions.map((s) => s.id)).toEqual(['a', 'c']);
	});

	it('orders the active workspace first', () => {
		const sessions = [
			session('a', '/ws/one', 30),
			session('b', '/ws/two', 50)
		];
		const groups = groupSessionsByWorkspace(sessions, '/ws/one');
		expect(groups[0].workspacePath).toBe('/ws/one');
	});

	it('orders remaining groups by most recent session', () => {
		const sessions = [
			session('a', '/ws/one', 10),
			session('b', '/ws/two', 90),
			session('c', '/ws/three', 50)
		];
		const groups = groupSessionsByWorkspace(sessions);
		expect(groups.map((g) => g.workspacePath)).toEqual(['/ws/two', '/ws/three', '/ws/one']);
	});
});
