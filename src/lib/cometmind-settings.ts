export interface CometMindACPSettings {
	command: string;
	args: string[];
	timeout: string;
	interactive: boolean;
}

export interface CometMindDiscordGatewaySettings {
	enabled: boolean;
	botToken: string;
	botTokenEnv: string;
	providerId: string;
	modelId: string;
	allowedUsers: string[];
	allowedChannels: string[];
	requireMention: boolean;
	workspacePath: string;
}

export interface CometMindSkillsSettings {
	enabled: boolean;
	roots: string[];
	includeOpenCode: boolean;
	includeClaude: boolean;
	mirrorToCometMind: boolean;
}

export interface CometMindMemorySettings {
	embedding: {
		providerId: string;
		provider: string;
		model: string;
		baseURL: string;
		apiKey: string;
	};
}

export interface CometMindSettings {
	acp: CometMindACPSettings;
	skills: CometMindSkillsSettings;
	memory: CometMindMemorySettings;
	gateway: {
		discord: CometMindDiscordGatewaySettings;
	};
}

function looksLikeDiscordBotToken(value: string): boolean {
	const parts = value.split('.');
	if (parts.length !== 3) return false;
	return parts[0].length >= 18 && parts[1].length >= 4 && parts[2].length >= 20;
}

function migrateDiscordTokenFields(discord: Partial<CometMindDiscordGatewaySettings>) {
	const defaults = defaultCometMindSettings().gateway.discord;
	let botToken = String(discord.botToken ?? '').trim();
	let botTokenEnv =
		String(discord.botTokenEnv ?? defaults.botTokenEnv).trim() || defaults.botTokenEnv;
	if (!botToken && looksLikeDiscordBotToken(botTokenEnv)) {
		botToken = botTokenEnv;
		botTokenEnv = defaults.botTokenEnv;
	}
	return { botToken, botTokenEnv };
}

export function defaultCometMindSettings(workspacePath = ''): CometMindSettings {
	return {
		acp: {
			command: 'opencode',
			args: ['acp'],
			timeout: '30m',
			interactive: true
		},
		skills: {
			enabled: true,
			roots: [],
			includeOpenCode: true,
			includeClaude: true,
			mirrorToCometMind: false
		},
		memory: {
			embedding: {
				providerId: '',
				provider: '',
				model: '',
				baseURL: '',
				apiKey: ''
			}
		},
		gateway: {
			discord: {
				enabled: false,
				botToken: '',
				botTokenEnv: 'DISCORD_BOT_TOKEN',
				providerId: '',
				modelId: '',
				allowedUsers: [],
				allowedChannels: [],
				requireMention: true,
				workspacePath
			}
		}
	};
}

function cleanStringList(values: unknown): string[] {
	if (!Array.isArray(values)) return [];
	return values.map((v) => String(v).trim()).filter(Boolean);
}

export function normalizeCometMindSettings(
	input: Partial<CometMindSettings> | undefined,
	fallbackWorkspacePath = ''
): CometMindSettings {
	const defaults = defaultCometMindSettings(fallbackWorkspacePath);
	const acp: Partial<CometMindACPSettings> = input?.acp ?? {};
	const skills: Partial<CometMindSkillsSettings> = input?.skills ?? {};
	const memory: Partial<CometMindMemorySettings> = input?.memory ?? {};
	const embedding: Partial<CometMindMemorySettings['embedding']> = memory.embedding ?? {};
	const discord: Partial<CometMindDiscordGatewaySettings> = input?.gateway?.discord ?? {};
	const args = Array.isArray(acp.args)
		? acp.args.map((a) => String(a).trim()).filter(Boolean)
		: defaults.acp.args;
	const { botToken, botTokenEnv } = migrateDiscordTokenFields(discord);

	return {
		acp: {
			command: String(acp.command ?? defaults.acp.command).trim() || defaults.acp.command,
			args: args.length > 0 ? args : defaults.acp.args,
			timeout: String(acp.timeout ?? defaults.acp.timeout).trim() || defaults.acp.timeout,
			interactive:
				typeof acp.interactive === 'boolean' ? acp.interactive : defaults.acp.interactive
		},
		skills: {
			enabled: typeof skills.enabled === 'boolean' ? skills.enabled : defaults.skills.enabled,
			roots: cleanStringList(skills.roots),
			includeOpenCode:
				typeof skills.includeOpenCode === 'boolean'
					? skills.includeOpenCode
					: defaults.skills.includeOpenCode,
			includeClaude:
				typeof skills.includeClaude === 'boolean'
					? skills.includeClaude
					: defaults.skills.includeClaude,
			mirrorToCometMind:
				typeof skills.mirrorToCometMind === 'boolean'
					? skills.mirrorToCometMind
					: defaults.skills.mirrorToCometMind
		},
		memory: {
			embedding: {
				providerId: String(embedding.providerId ?? defaults.memory.embedding.providerId).trim(),
				provider: String(embedding.provider ?? defaults.memory.embedding.provider).trim(),
				model: String(embedding.model ?? defaults.memory.embedding.model).trim(),
				baseURL: String(embedding.baseURL ?? defaults.memory.embedding.baseURL).trim(),
				apiKey: String(embedding.apiKey ?? defaults.memory.embedding.apiKey).trim()
			}
		},
		gateway: {
			discord: {
				enabled:
					typeof discord.enabled === 'boolean' ? discord.enabled : defaults.gateway.discord.enabled,
				botToken,
				botTokenEnv,
				providerId: String(discord.providerId ?? defaults.gateway.discord.providerId).trim(),
				modelId: String(discord.modelId ?? defaults.gateway.discord.modelId).trim(),
				allowedUsers: cleanStringList(discord.allowedUsers),
				allowedChannels: cleanStringList(discord.allowedChannels),
				requireMention:
					typeof discord.requireMention === 'boolean'
						? discord.requireMention
						: defaults.gateway.discord.requireMention,
				workspacePath:
					String(discord.workspacePath ?? defaults.gateway.discord.workspacePath).trim() ||
					defaults.gateway.discord.workspacePath
			}
		}
	};
}

export function cloneCometMindSettings(settings: CometMindSettings): CometMindSettings {
	return {
		acp: {
			command: settings.acp.command,
			args: [...settings.acp.args],
			timeout: settings.acp.timeout,
			interactive: settings.acp.interactive
		},
		skills: {
			...settings.skills,
			roots: [...settings.skills.roots]
		},
		memory: {
			embedding: { ...settings.memory.embedding }
		},
		gateway: {
			discord: {
				...settings.gateway.discord,
				allowedUsers: [...settings.gateway.discord.allowedUsers],
				allowedChannels: [...settings.gateway.discord.allowedChannels]
			}
		}
	};
}

/** Parse comma- or newline-separated IDs for text inputs. */
export function parseIdList(raw: string): string[] {
	return raw
		.split(/[\n,]+/)
		.map((part) => part.trim())
		.filter(Boolean);
}

export function formatIdList(ids: string[]): string {
	return ids.join('\n');
}
