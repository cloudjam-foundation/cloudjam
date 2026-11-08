<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import Select from '$lib/components/Select.svelte';
	import { ListRequestSchema as ListDefinitionRequestSchema } from '$lib/sdk/v1/cloud/definition/definition_pb';
	import type { Definition } from '$lib/sdk/v1/cloud/definition_pb';
	import { ListRequestSchema as ListProviderRequestSchema } from '$lib/sdk/v1/cloud/provider/provider_pb';
	import type { Provider } from '$lib/sdk/v1/cloud/provider_pb';
	import { CreateRequestSchema } from '$lib/sdk/v1/play/challenge/challenge_pb';
	import { ChallengeSchema, type Challenge } from '$lib/sdk/v1/play/challenge_pb';
	import type { Team } from '$lib/sdk/v1/play/team_pb';
	import { create } from '@bufbuild/protobuf';
	import { onMount } from 'svelte';

	let {
		gameId,
		teams,
		onCreated
	}: {
		gameId: string;
		teams: Team[];
		onCreated: () => void;
	} = $props();

	let assignedTeams: string[] = $state([]);
	let blueprint: Challenge = $state(create(ChallengeSchema, { id: crypto.randomUUID() }));

	let createState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let providers: Provider[] = $state([]);
	async function loadProviders() {
		try {
			providers = (
				await Glue.provider.list(
					create(ListProviderRequestSchema, {
						limit: 100
					})
				)
			).providers;
		} catch {
			providers = [];
		}
	}

	let definitions: Definition[] = $state([]);
	$effect(() => {
		loadDefinitions(blueprint.definitionProviderId);
	});

	async function loadDefinitions(providerId: string) {
		try {
			definitions = (
				await Glue.definition.list(
					create(ListDefinitionRequestSchema, {
						providerId: providerId,
						limit: 100
					})
				)
			).definitions;
		} catch {
			definitions = [];
		}
	}

	function createChallenges() {
		Submit(async () => {
			for (const team of assignedTeams) {
				await Glue.challenge.create(
					create(CreateRequestSchema, {
						init: { ...blueprint, teamId: team, gameId: gameId }
					})
				);
			}
			onCreated();
		}, createState);
	}

	onMount(() => loadProviders());
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Provision Challenges</h1>
		<p class="text-primary/60">
			Rollout a challenge to the specified teams. The details are revealed once the team starts the challenge.
		</p>
	</div>
	<form class="flex w-full flex-col gap-4" onsubmit={createChallenges}>
		<div class="grid gap-4 md:grid-cols-2">
			<div class="flex flex-col gap-1">
				<span class="text-sm">Provider</span>
				{#if providers.length > 0}
					<Select
						bind:value={blueprint.definitionProviderId}
						class="w-full"
						placeholder="Select provider"
						items={providers.map((provider) => ({ value: provider.id, label: provider.name }))}
					/>
				{:else}
					<Input id="select-provider" bind:value={blueprint.definitionProviderId} placeholder="Definition ID" />
				{/if}
			</div>

			<div class="flex flex-col gap-1">
				<span class="text-sm">Definition</span>
				{#if definitions.length > 0}
					<Select
						bind:value={blueprint.definitionId}
						class="w-full"
						placeholder="Select definition"
						items={definitions.map((definition) => ({ value: definition.id, label: definition.name }))}
					/>
				{:else}
					<Input id="select-definition" bind:value={blueprint.definitionId} placeholder="Definition ID" />
				{/if}
			</div>

			<div class="flex flex-col gap-1">
				<span class="text-sm">Teams</span>
				{#if teams.length > 0}
					<Select
						multiple={true}
						bind:value={assignedTeams}
						class="w-full"
						placeholder="Select Teams"
						items={teams.map((team) => ({ value: team.id, label: team.name }))}
					/>
				{:else}
					<Input id="select-teams" bind:value={assignedTeams} placeholder="Comma separated team ids" />
				{/if}
			</div>
		</div>
		<Button type="submit" variant="primary" processing={createState.loading} class="self-start">Rollout</Button>
		{#if createState.error}
			<Alert
				title={createState.forbidden ? 'Permission denied' : 'Failed to hand out challenge'}
				message={createState.error}
				animation={true}
			/>
		{/if}
	</form>
</Card>
