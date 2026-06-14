export function syncTrafficLightsForSidebar(open: boolean) {
	window.electronAPI?.setSidebarOpen?.(open);
}
