<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import { ListRequestSchema } from '$lib/sdk/v1/play/team/team_pb';
	import type { Team } from '$lib/sdk/v1/play/team_pb';
	import { create } from '@bufbuild/protobuf';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import CreateTeam from './CreateTeam.svelte';
	import TeamPanel from './TeamPanel.svelte';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import StaticBadge from '$lib/components/custom/badges/StaticBadge.svelte';
	import DataTable from '$lib/components/layout/DataTable.svelte';
	import Alert from '$lib/components/custom/Alert.svelte';
	import DynamicBadge from '$lib/components/custom/badges/DynamicBadge.svelte';
	import { toSvg } from 'jdenticon';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import { ClosedCaptionIcon, XIcon } from '@lucide/svelte';

	let { game }: { game: Game } = $props();

	let selected: Team | undefined = $state();
	let openSelected = $state(false);
	let creating = $state(false);

	let teamsState: SubmitState = $state({ error: '', forbidden: false, loading: false });
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<h2 class="text-xl opacity-80">Teams</h2>
		<Button variant="primary" class="ml-auto cursor-pointer" onclick={() => (creating = !creating)}>
			{#if creating}
				<XIcon /> Close Menu
			{:else}
				<PlusIcon /> Create Team
			{/if}
		</Button>
	</div>

	{#if creating}
		<CreateTeam {game} oncreated={() => {}} />
	{/if}

	{#if teamsState.forbidden}
		<Alert title="Permission Denied" message="You are not allowed to list the teams of this game" />
	{:else}
		<DataTable
			class="max-h-[80vh]"
			load={async (startAfter: Team | undefined) => {
				const teams: Team[] = [];
				await Submit(async () => {
					const resp = await Glue.team.list(
						create(ListRequestSchema, { gameId: game.id, limit: 100, startAfter: startAfter?.id })
					);
					teams.push(...resp.teams);
				}, teamsState);
				return teams;
			}}
			onRowClick={(team) => {
				selected = team;
				openSelected = true;
			}}
			loadState={teamsState}
			columns={[
				{ title: 'Name', width: '30%', render: renderName },
				{ title: 'Players', width: '60%', render: renderPlayers },
				{ title: 'Score', width: '30px', render: renderScore }
			]}
		/>
		{#snippet renderName(team: Team)}
			{team.name}
		{/snippet}
		{#snippet renderPlayers(team: Team)}
			<div class="flex flex-row items-center gap-1">
				{#each Object.entries(team.players) as [_, player]}
					<DynamicBadge variant={'primary'}>
						<img
							alt="user profile"
							src={`data:image/svg+xml;base64,${btoa(toSvg(player.pubId, 16))}`}
							height="3rem"
							class="rounded-lg bg-primary/5"
						/>
						{player.username}
					</DynamicBadge>
				{/each}
			</div>
		{/snippet}
		{#snippet renderScore(team: Team)}
			<StaticBadge variant="jam" text={team.score.toString()} />
		{/snippet}
	{/if}

	{#if selected}
		<TeamPanel bind:open={openSelected} team={selected} refresh={() => {}} />
	{/if}
</div>
