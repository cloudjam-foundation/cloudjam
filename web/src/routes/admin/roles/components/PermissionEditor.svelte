<script lang="ts">
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import Select from '$lib/components/Select.svelte';
	import { getActiveScopes } from '$lib/auth.svelte.js';
	import { RBACService } from '$lib/sdk/v1/admin/rbac/rbac_pb';
	import { RoleService } from '$lib/sdk/v1/admin/role/role_pb';
	import { SystemService } from '$lib/sdk/v1/admin/system/system_pb';
	import { UserService } from '$lib/sdk/v1/admin/user/user_pb';
	import { AuthService } from '$lib/sdk/v1/auth/auth_pb';
	import { AccountService } from '$lib/sdk/v1/cloud/account/account_pb';
	import { DefinitionService } from '$lib/sdk/v1/cloud/definition/definition_pb';
	import { ProviderService } from '$lib/sdk/v1/cloud/provider/provider_pb';
	import { ChallengeService } from '$lib/sdk/v1/play/challenge/challenge_pb';
	import { GameService } from '$lib/sdk/v1/play/game/game_pb';
	import { TeamService } from '$lib/sdk/v1/play/team/team_pb';
	import { PlusIcon, Trash2Icon } from '@lucide/svelte';
	import globToRegex from 'glob-to-regexp';

	let { entries = $bindable() }: { entries: { scope: string; patterns: string }[] } = $props();

	const services = [
		AuthService,
		UserService,
		RoleService,
		RBACService,
		SystemService,
		ProviderService,
		AccountService,
		DefinitionService,
		GameService,
		ChallengeService,
		TeamService
	].map((svc) => ({
		name: svc.name,
		functions: svc.methods.map((method) => ({
			label: `${svc.name}/${method.name}`,
			procedure: `/${svc.typeName}/${method.name}`
		}))
	}));
	const functions = services.flatMap((svc) => svc.functions);

	// granted returns a human readable list of functions that are granted according to the specified patterns.
	function granted(patterns: string): string[] {
		const matched = [];
		for (const pattern of patterns.split(',')) {
			const patternRe = globToRegex(pattern, { globstar: true });
			matched.push(...functions.filter((fn) => patternRe.test(fn.procedure)).map((fn) => fn.label));
		}
		return matched;
	}

	function selected(patterns: string): string[] {
		const parts = patterns.split(',');
		return functions.filter((fn) => parts.includes(fn.procedure)).map((fn) => fn.procedure);
	}

	function apply(entry: { scope: string; patterns: string }, picked: string[]) {
		const custom = entry.patterns
			.split(',')
			.filter((pattern) => pattern && !functions.some((fn) => fn.procedure === pattern));
		entry.patterns = [...custom, ...picked].join(',');
	}
</script>

<div class="flex flex-col gap-3">
	{#each entries as entry, i (i)}
		<div class="flex flex-col gap-1">
			<div class="flex flex-row items-center gap-2">
				<OptionalSelect
					bind:value={entry.scope}
					placeholder="Scope"
					suggestions={getActiveScopes().map((scope) => ({ id: scope, title: scope }))}
				/>
				<Select
					multiple={true}
					value={selected(entry.patterns)}
					onchange={(picked) => apply(entry, picked)}
					class="w-48"
					placeholder={`${selected(entry.patterns).length} function(s) picked`}
					groups={services.map((service) => ({
						label: service.name,
						items: service.functions.map((fn) => ({ value: fn.procedure, label: fn.label }))
					}))}
				/>
				<Input
					class="max-w-96 font-mono"
					bind:value={entry.patterns}
					placeholder="Patterns (e.g. /v1.admin.user.UserService/*)"
				/>
				<Button variant="ghost" title="Remove permission" onclick={() => (entries = entries.filter((_, x) => x !== i))}>
					<Trash2Icon />
				</Button>
			</div>
			<div class="flex flex-row flex-wrap items-center gap-1">
				<span class="text-xs text-primary/60">grants:</span>
				{#if granted(entry.patterns).length === functions.length}
					<StaticBadge variant="jam" text="every function" />
				{:else}
					{#each granted(entry.patterns) as label (label)}
						<StaticBadge variant="primary" text={label} />
					{:else}
						<span class="text-xs text-primary/60 italic">no known function</span>
					{/each}
				{/if}
			</div>
		</div>
	{/each}
	<Button variant="primary" class="self-start" onclick={() => (entries = [...entries, { scope: '', patterns: '' }])}>
		<PlusIcon /> Add Permission
	</Button>
</div>
