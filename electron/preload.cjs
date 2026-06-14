const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
	restartCometMind: () => ipcRenderer.send('cometmind:restart'),
	getWorkspacePath: () => ipcRenderer.invoke('cometline:get-workspace-path'),
	getProviderSettings: () => ipcRenderer.invoke('cometline:get-provider-settings'),
	fetchProviderModels: (config) => ipcRenderer.invoke('cometline:fetch-provider-models', config),
	saveProviderSettings: (settings) =>
		ipcRenderer.invoke('cometline:save-provider-settings', settings),
	setSidebarOpen: (payload) => ipcRenderer.send('cometline:set-sidebar-open', payload),
	getFullScreen: () => ipcRenderer.invoke('cometline:get-fullscreen'),
	onFullScreenChange: (callback) => {
		const handler = (_event, isFullScreen) => callback(Boolean(isFullScreen));
		ipcRenderer.on('cometline:fullscreen-changed', handler);
		return () => ipcRenderer.removeListener('cometline:fullscreen-changed', handler);
	},
	getAppVersion: () => ipcRenderer.invoke('cometline:get-app-version'),
	getUpdateState: () => ipcRenderer.invoke('cometline:get-update-state'),
	checkForUpdates: () => ipcRenderer.invoke('cometline:check-for-updates'),
	installUpdate: () => ipcRenderer.invoke('cometline:install-update'),
	onUpdateState: (callback) => {
		const handler = (_event, state) => callback(state);
		ipcRenderer.on('cometline:update-state', handler);
		return () => ipcRenderer.removeListener('cometline:update-state', handler);
	}
});
