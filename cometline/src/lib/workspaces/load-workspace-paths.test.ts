import { describe, expect, it } from 'vitest';
import { loadWorkspacePaths, mergeWorkspacePaths } from './load-workspace-paths';

describe('mergeWorkspacePaths', () => {
	it('dedupes recent, current, and registered paths preserving order', () => {
		const { paths } = mergeWorkspacePaths(
			['/a', '/b'],
			[
				{ path: '/b', session_count: 2 },
				{ path: '/c', session_count: 1 }
			],
			'/d'
		);
		expect(paths).toEqual(['/a', '/b', '/d', '/c']);
	});

	it('builds session count map from registered workspaces', () => {
		const { sessionCountByPath } = mergeWorkspacePaths(
			[],
			[
				{ path: '/proj', session_count: 5 },
				{ path: '/other', session_count: 0 }
			]
		);
		expect(sessionCountByPath.get('/proj')).toBe(5);
		expect(sessionCountByPath.get('/other')).toBe(0);
	});
});

describe('loadWorkspacePaths', () => {
	it('filters merged paths through filterExistingWorkspacePaths', async () => {
		const result = await loadWorkspacePaths('/current', {
			listRecentWorkspaces: async () => ['/gone', '/keep'],
			listRegisteredWorkspaces: async () => [{ path: '/keep', session_count: 3 }],
			filterExistingWorkspacePaths: async (paths) => paths.filter((path) => path !== '/gone')
		});
		expect(result.paths).toEqual(['/keep', '/current']);
		expect(result.sessionCountByPath.get('/keep')).toBe(3);
	});
});
