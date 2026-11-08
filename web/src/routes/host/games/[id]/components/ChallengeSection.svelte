<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { ListRequestSchema } from '$lib/sdk/v1/play/challenge/challenge_pb.ts';
	import type { Challenge } from '$lib/sdk/v1/play/challenge_pb.ts';
	import { ListRequestSchema as ListTeamRequestSchema } from '$lib/sdk/v1/play/team/team_pb.ts';
	import type { Team } from '$lib/sdk/v1/play/team_pb.ts';
	import { create } from '@bufbuild/protobuf';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import XIcon from '@lucide/svelte/icons/x';
	import { onMount } from 'svelte';
	import ChallengePanel from './ChallengePanel.svelte';
	import CreateChallenge from './CreateChallenge.svelte';

	let { gameId }: { gameId: string } = $props();

	let teams: Team[] = $state([]);

	let selected: Challenge | undefined = $state();
	let openSelected = $state(false);
	let creating = $state(false);
	// bumping the version remounts the table and therefore reloads the challenges.
	let version = $state(0);

	const score = (challenge: Challenge) => challenge.scoreEvents.reduce((sum, event) => sum + event.change, 0);

	function teamName(id: string): string {
		return teams.find((team) => team.id === id)?.name ?? id;
	}

	let challengesState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let teamsState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	onMount(() =>
		Submit(async () => {
			teams = (await Glue.team.list(create(ListTeamRequestSchema, { gameId: gameId, limit: 100 }))).teams;
		}, teamsState)
	);
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<h2 class="text-xl opacity-80">Challenges</h2>
		<Button variant="primary" class="ml-auto" onclick={() => (creating = !creating)}>
			{#if creating}
				<XIcon /> Close Menu
			{:else}
				<PlusIcon /> Provision Challenges
			{/if}
		</Button>
	</div>

	{#if creating}
		<CreateChallenge {gameId} {teams} oncreated={() => version++} />
	{/if}

	{#if challengesState.forbidden}
		<Alert title="Permission Denied" message="You are not allowed to list the challenges of this game" />
	{:else}
		{#key version}
			<DataTable
				class="max-h-[80vh]"
				limit={100}
				load={async (startAfter: Challenge | undefined) => {
					const challenges: Challenge[] = [];
					await Submit(async () => {
						const resp = await Glue.challenge.list(
							create(ListRequestSchema, { gameId: gameId, limit: 100, startAfter: startAfter?.id })
						);
						challenges.push(...resp.challenges);
					}, challengesState);
					return challenges;
				}}
				onRowClick={(challenge) => {
					selected = challenge;
					openSelected = true;
				}}
				loadState={challengesState}
				columns={[
					{ title: 'Title', width: '40%', render: renderTitle },
					{ title: 'Team', width: '30%', render: renderTeam },
					{ title: 'Score', width: '90px', render: renderScore },
					{ title: 'Errors', width: '90px', render: renderErrors }
				]}
			/>
		{/key}
		{#snippet renderTitle(challenge: Challenge)}
			{challenge.title || 'not started'}
		{/snippet}
		{#snippet renderTeam(challenge: Challenge)}
			<span class="text-primary/60">{teamName(challenge.teamId)}</span>
		{/snippet}
		{#snippet renderScore(challenge: Challenge)}
			<StaticBadge variant="jam" text={score(challenge).toString()} />
		{/snippet}
		{#snippet renderErrors(challenge: Challenge)}
			{#if challenge.error}
				<StaticBadge variant="destructive" text="defect" />
			{/if}
		{/snippet}
	{/if}

	{#if selected}
		<ChallengePanel
			bind:open={openSelected}
			challenge={selected}
			{teams}
			refresh={() => {
				openSelected = false;
				version++;
			}}
		/>
	{/if}
</div>
