<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { Glue, Submit, type SubmitState } from '$lib';
	import DynamicBadge from '$lib/components/DynamicBadge.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Card from '$lib/components/Card.svelte';
	import { GetRequestSchema as GetDefinitionRequestSchema } from '$lib/sdk/v1/cloud/definition/definition_pb';
	import type { Definition } from '$lib/sdk/v1/cloud/definition_pb';
	import { GetRequestSchema as GetChallengeRequestSchema } from '$lib/sdk/v1/play/challenge/challenge_pb';
	import type { Challenge } from '$lib/sdk/v1/play/challenge_pb';
	import { GetRequestSchema as GetGameRequestSchema } from '$lib/sdk/v1/play/game/game_pb';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { GetRequestSchema as GetTeamRequestSchema } from '$lib/sdk/v1/play/team/team_pb';
	import type { Team } from '$lib/sdk/v1/play/team_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import { ChevronLeftIcon, LockIcon } from '@lucide/svelte';
	import { toSvg } from 'jdenticon';
	import PlayChallenge from './components/PlayChallenge.svelte';
	import { resolve } from '$app/paths';

	const reloadInterval = 5000;

	let gameId: string | undefined = $derived(page.params.game_id);
	let gameState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let game: Game | undefined = $state();
	$effect(() => {
		if (!gameId) return;
		Submit(async () => {
			game = (
				await Glue.game.get(
					create(GetGameRequestSchema, {
						id: gameId
					})
				)
			).game;
		}, gameState);
	});

	let teamId: string | undefined = $state('');
	let teamState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let team: Team | undefined = $state();
	async function reloadTeam() {
		if (!teamId) return undefined;
		if (!gameId) return undefined;
		await Submit(async () => {
			team = (
				await Glue.team.get(
					create(GetTeamRequestSchema, {
						gameId: gameId,
						id: teamId
					})
				)
			).team;
		}, teamState);
	}

	let definitionId: string | undefined = $state();
	let definitionProviderId: string | undefined = $state();
	let definitionState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let definition: Definition | undefined = $state();
	$effect(() => {
		if (!definitionId) return undefined;
		if (!definitionProviderId) return undefined;
		Submit(async () => {
			definition = (
				await Glue.definition.get(
					create(GetDefinitionRequestSchema, {
						providerId: definitionProviderId,
						id: definitionId
					})
				)
			).definition;
		}, definitionState);
	});

	let challengeTimeout: ReturnType<typeof setTimeout>;
	let nextInterval: Date = $state(new Date());

	async function reloadChallenge() {
		clearTimeout(challengeTimeout);
		async function poll() {
			await Submit(async () => {
				challenge = (
					await Glue.challenge.get(
						create(GetChallengeRequestSchema, {
							gameId: gameId,
							id: challengeId
						})
					)
				).challenge;
			}, challengeState);
			teamId = challenge?.teamId;
			await reloadTeam();
			definitionId = challenge?.definitionId;
			definitionProviderId = challenge?.definitionProviderId;
			nextInterval = new Date(Date.now() + reloadInterval);
			challengeTimeout = setTimeout(poll, reloadInterval);
		}
		poll();
	}

	let challengeId: string | undefined = $derived(page.params.challenge_id);
	let challengeState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let challenge: Challenge | undefined = $state();
	$effect(() => {
		if (!gameId) return undefined;
		if (!challengeId) return undefined;
		reloadChallenge();
	});

	let score = $derived(challenge?.scoreEvents.reduce((sum, event) => sum + event.change, 0) ?? 0);

	let active = $state(false);
	let remaining = $state('');
	function formatRemaining(milliseconds: number): string {
		const seconds = Math.max(0, Math.ceil(milliseconds / 1000));
		const hours = Math.floor(seconds / 3600);
		const minutes = Math.floor((seconds % 3600) / 60);
		return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${(seconds % 60).toString().padStart(2, '0')}`;
	}
	$effect(() => {
		if (!game) {
			active = false;
			remaining = '';
			return;
		}
		const updateActive = () => {
			const now = Date.now();
			const end = timestampDate(game!.to!).getTime();
			active = timestampDate(game!.from!).getTime() < now && end > now;
			remaining = formatRemaining(end - now);
		};
		updateActive();
		const interval = setInterval(updateActive, 1000);
		return () => clearInterval(interval);
	});
</script>

<svelte:head>
	<title>{challenge?.title || definition?.name || 'Challenge'} | CloudJam</title>
	<meta property="og:title" content="Challenge | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto(resolve(`/play/${page.params.game_id}`))}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">{challenge?.title || definition?.name || 'Challenge'}</h1>
		{#if definition}
			<StaticBadge variant="primary" text={definition?.name} />
		{/if}
		{#if !active}
			<DynamicBadge variant="primary">
				<LockIcon class="size-3" />
				Locked
			</DynamicBadge>
		{:else}
			<StaticBadge variant="primary" text={`ends in ${remaining}`} />
		{/if}
		<StaticBadge variant="jam" text={`score: ${score}`} />
	</div>

	<Card class="w-full items-start gap-2 p-4">
		<div class="flex flex-row items-center gap-2">
			<h1 class="text-2xl font-semibold text-primary/90">{team?.name}</h1>
			<StaticBadge variant="jam" text={`score: ${team?.score}`} />
		</div>
		<div class="flex flex-row flex-wrap gap-1">
			{#each Object.values(team?.players ?? {}) as player (player.id)}
				<DynamicBadge variant="primary">
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
	</Card>
	{#if challenge}
		<PlayChallenge {challenge} {active} {nextInterval} refresh={() => reloadChallenge()} />
	{/if}
</div>
