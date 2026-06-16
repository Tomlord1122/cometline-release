export interface BuiltinSlashCommand {
	name: string;
	description: string;
}

export const BUILTIN_SLASH_COMMANDS: BuiltinSlashCommand[] = [
	{
		name: 'create-skill',
		description: 'Build a new Agent Skill in ~/.cometmind/skills'
	}
];

export function expandCreateSkillCommand(userText: string): string {
	const rest = userText.trim();
	let prompt =
		'Create a new Agent Skill for CometMind.\n\n' +
		'Target directory: ~/.cometmind/skills/{skill-name}/\n\n' +
		'Requirements:\n' +
		'1. Use the `write_skill` tool to create SKILL.md (YAML frontmatter with name and description, then markdown body).\n' +
		'2. Follow Agent Skills conventions: clear trigger scenarios, step-by-step workflow, examples, and constraints.\n' +
		'3. Skill names use lowercase letters, numbers, and hyphens only.\n' +
		'4. If the request is vague, ask up to two clarifying questions before writing.\n' +
		'5. After writing, summarize the skill name, what it does, and how to invoke it with /{skill-name} in Cometline.';
	if (rest) {
		prompt += '\n\nUser request:\n' + rest;
	}
	return prompt;
}

export function expandBuiltinSlashCommand(text: string): string | null {
	const match = /^\s*\/([\w-]+)(?:\s+([\s\S]*))?$/.exec(text);
	if (!match) return null;
	const name = match[1];
	const builtin = BUILTIN_SLASH_COMMANDS.find((cmd) => cmd.name === name);
	if (!builtin) return null;
	const rest = match[2]?.trimStart() ?? '';
	if (name === 'create-skill') {
		return expandCreateSkillCommand(rest);
	}
	return null;
}

export type SlashMenuOption =
	| { kind: 'builtin'; name: string; description: string }
	| { kind: 'skill'; name: string; description: string };

export function filterSlashMenuOptions(
	query: string,
	skills: { name: string; description: string }[]
): SlashMenuOption[] {
	const q = query.toLowerCase();
	const builtins = BUILTIN_SLASH_COMMANDS.filter(
		(cmd) =>
			!q ||
			cmd.name.toLowerCase().includes(q) ||
			cmd.description.toLowerCase().includes(q)
	).map((cmd) => ({ kind: 'builtin' as const, name: cmd.name, description: cmd.description }));
	const skillOptions = skills
		.filter(
			(skill) =>
				!q ||
				skill.name.toLowerCase().includes(q) ||
				skill.description.toLowerCase().includes(q)
		)
		.map((skill) => ({
			kind: 'skill' as const,
			name: skill.name,
			description: skill.description
		}));
	return [...builtins, ...skillOptions];
}
