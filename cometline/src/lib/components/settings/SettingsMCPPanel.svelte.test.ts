// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, waitFor } from '@testing-library/svelte';
import { flushSync } from 'svelte';
import Harness from './SettingsMCPPanel.harness.svelte';

vi.mock('$lib/client/cometmind', () => ({
	listMcpServers: () => Promise.resolve([]),
	listMcpTools: () => Promise.resolve([]),
	reconnectMcpServer: vi.fn(),
	startMcpOAuth: vi.fn()
}));

afterEach(() => {
	vi.clearAllMocks();
});

function clickAddServer(container: HTMLElement) {
	const addButton = [...container.querySelectorAll('button')].find((button) =>
		button.textContent?.includes('Add server')
	);
	expect(addButton).toBeTruthy();
	flushSync(() => addButton!.click());
}

describe('SettingsMCPPanel add server', () => {
	it('adds a server and shows the expanded editor', async () => {
		const { container } = render(Harness);

		await waitFor(() => {
			expect(container.textContent).toContain('No servers configured yet');
		});

		clickAddServer(container);

		await waitFor(() => {
			expect(container.querySelector('[data-testid="server-count"]')?.textContent).toBe('1');
			expect(container.textContent).toContain('MCP Server 1');
		});

		expect(container.querySelector('.mcp-server-editor')).toBeTruthy();
	});

	it('keeps environment variable text while typing incomplete lines', async () => {
		const { container } = render(Harness);

		clickAddServer(container);

		await waitFor(() => {
			expect(container.querySelector('.mcp-server-editor')).toBeTruthy();
		});

		const envField = container.querySelector(
			'textarea'
		) as HTMLTextAreaElement | null;
		expect(envField).toBeTruthy();

		await fireEvent.input(envField!, { target: { value: 'MY_KEY' } });
		expect(envField!.value).toBe('MY_KEY');

		await fireEvent.input(envField!, { target: { value: 'MY_KEY=secret' } });
		expect(envField!.value).toBe('MY_KEY=secret');
	});

	it('assigns unique server ids when adding multiple servers', async () => {
		const { container } = render(Harness);

		clickAddServer(container);
		clickAddServer(container);
		clickAddServer(container);

		await waitFor(() => {
			expect(container.querySelector('[data-testid="server-count"]')?.textContent).toBe('3');
		});

		const ids = container.querySelector('[data-testid="server-ids"]')?.textContent?.split(',') ?? [];
		expect(ids).toHaveLength(3);
		expect(new Set(ids).size).toBe(3);
	});
});