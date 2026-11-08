<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { getSubject, Glue, Submit, type SubmitState } from '$lib';
	import StateBadge from '$lib/components/StateBadge.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import { ListRequestSchema as ListChallengeRequestSchema } from '$lib/sdk/v1/play/challenge/challenge_pb';
	import type { Challenge } from '$lib/sdk/v1/play/challenge_pb';
	import { GetRequestSchema as GetGameRequestSchema } from '$lib/sdk/v1/play/game/game_pb';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { ListRequestSchema as ListTeamsRequestSchema } from '$lib/sdk/v1/play/team/team_pb';
	import type { Team } from '$lib/sdk/v1/play/team_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import { ChevronLeftIcon, LayersIcon } from '@lucide/svelte';
	import { resolve } from '$app/paths';

	let gameState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let game: Game | undefined = $state();
	$effect(() => {
		Submit(async () => {
			game = (
				await Glue.game.get(
					create(GetGameRequestSchema, {
						id: page.params.game_id
					})
				)
			).game;
		}, gameState);
	});

	let challengesState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let challenges: Challenge[] = $state([]);
	$effect(() => {
		if (!game) return undefined;
		Submit(async () => {
			challenges = (
				await Glue.challenge.list(
					create(ListChallengeRequestSchema, {
						gameId: game?.id,
						limit: 100
					})
				)
			).challenges;
		}, challengesState);
	});

	let teamsState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let teams: Team[] = $state([]);
	$effect(() => {
		if (!game) return undefined;
		Submit(async () => {
			teams = (
				await Glue.team.list(
					create(ListTeamsRequestSchema, {
						gameId: game?.id,
						limit: 100
					})
				)
			).teams.sort((a, b) => a.score - b.score);
		}, teamsState);
	});

	let userTeam: Team | undefined = $state();
	$effect(() => {
		const id = getSubject();
		for (const team of teams) {
			if (id in team.players) {
				userTeam = team;
				return;
			}
		}
		userTeam = undefined;
	});

	let active = $state(false);
	$effect(() => {
		if (!game) {
			active = false;
			return;
		}
		const updateActive = () =>
			(active = timestampDate(game!.from!).getTime() < Date.now() && timestampDate(game!.to!).getTime() > Date.now());
		const interval = setInterval(updateActive, 1000);
		updateActive();
		return () => clearInterval(interval);
	});
</script>

<svelte:head>
	<title>{game?.name ?? 'Game'} | CloudJam</title>
	<meta property="og:title" content="Game | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto(resolve(`/play`))}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">{game?.name ?? 'Game'}</h1>
		{#if active}
			<StateBadge text="running" state="RunningPositive" mapper={undefined} />
		{:else}
			<StateBadge text="ended" state="StableNeutral" mapper={undefined} />
		{/if}
	</div>

	<p class="text-sm text-primary/60">
		{game?.description}
	</p>

	{#if userTeam}
		<div class="flex flex-row flex-wrap gap-4">
			{#each challenges.filter((challenge) => challenge.teamId === userTeam!.id) as challenge (challenge.id)}
				<button
					type="button"
					class="w-full cursor-pointer text-left sm:w-80"
					onclick={() => goto(resolve(`/play/${game?.id}/${challenge.id}`))}
				>
					<Card class="h-full items-start gap-2 duration-300 hover:opacity-70">
						<LayersIcon class="size-8" />
						<h1 class="text-xl">{challenge.title}</h1>
						<p class="text-primary/60">
							{challenge.description.join('\n').slice(0, 20)}
							{challenge.description.length < 1 ? '' : '...'}
						</p>
						<div class="flex flex-row flex-wrap gap-1">
							{#if !challenge.title}
								<StaticBadge variant="primary" text="not discovered" />
							{/if}
							{#if challenge.error}
								<StaticBadge variant="destructive" text="challenge defect" />
							{/if}
						</div>
					</Card>
				</button>
			{/each}
		</div>
	{/if}
</div>

<!-- <h1>TODO: here will be an epic leaderboard</h1> -->
<!-- <div class="flex flex-col"> -->
<!-- 	{#each teams as team (team.id)} -->
<!-- 		<div>{team.name}: {team.scope}</div> -->
<!-- 	{/each} -->
<!-- </div> -->
