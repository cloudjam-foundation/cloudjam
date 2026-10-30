<script lang="ts">
	import { Badge } from '$lib/components/shad/badge';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import ChallengeSection from './ChallengeSection.svelte';
	import GamePanel from './GamePanel.svelte';
	import TeamSection from './TeamSection.svelte';
	import { goto, invalidateAll } from '$app/navigation';
	import { gameState } from './state.svelte.js';
	import DelaySpinner from '$lib/components/custom/DelaySpinner.svelte';
	import Alert from '$lib/components/custom/Alert.svelte';

	let { data } = $props();

	const tabs = ['overview', 'teams', 'challenges'] as const;

	let tab: (typeof tabs)[number] = $state('overview');

	function status(game: Game): string {
		const now = Date.now();
		if (game.from && timestampDate(game.from).getTime() > now) return 'upcoming';
		if (game.to && timestampDate(game.to).getTime() < now) return 'ended';
		return 'running';
	}
</script>

<svelte:head>
	<title>{data.game?.name ?? 'Game'} | CloudJam</title>
	<meta property="og:title" content="Game | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

{#if gameState.loading}
	<DelaySpinner class="mt-10 flex justify-center" delay={200} />
{:else}
	<div class="flex w-full flex-col gap-4">
		<div class="flex flex-row items-center gap-2">
			<Button variant="ghost" class="cursor-pointer" onclick={() => goto('/host/games')}>
				<ChevronLeftIcon />
			</Button>
			<h1 class="text-3xl opacity-80">{data.game?.name ?? 'Game'}</h1>
			{#if data.game}
				<Badge variant="secondary">{status(data.game)}</Badge>
				<Badge variant="outline">scope: {data.game.scope}</Badge>
			{/if}
			<Button
				variant={'primary'}
				class="ml-auto cursor-pointer font-mono text-xs"
				title="Copy game id"
				onclick={() => navigator.clipboard.writeText(data.game?.id ?? '')}
			>
				<CopyIcon />
				{data.game?.id}
			</Button>
		</div>

		<div class="flex flex-row gap-2 border-b">
			{#each tabs as name (name)}
				<button
					type="button"
					class="cursor-pointer border-b-2 px-4 py-2 capitalize transition-all duration-200 {tab === name
						? 'border-primary'
						: 'border-transparent opacity-60 hover:opacity-100'}"
					onclick={() => (tab = name)}
				>
					{name}
				</button>
			{/each}
		</div>

		{#if gameState.forbidden}
			<Alert title="Permission Denied" message="You are not allowed to open this game" animation={true} />
		{:else if tab === 'overview'}
			{#if data.game}
				<GamePanel game={data.game} refresh={() => invalidateAll()} deleted={() => goto('/host/games')} />
			{/if}
		{:else if tab === 'teams'}
			{#if data.game}
				<TeamSection game={data.game} />
			{/if}
		{:else if tab === 'challenges'}
			{#if data.game}
				<ChallengeSection gameId={data.game.id} />
			{/if}
		{/if}

		{#if gameState.error}
			<Alert title="Failed to load the game" message={gameState.error} animation={true} />
		{/if}
	</div>
{/if}
