<script lang="ts">
	import { goto } from '$app/navigation';
	import StateBadge from '$lib/components/StateBadge.svelte';
	import TabPage from '$lib/components/TabPage.svelte';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import ChallengeSection from './components/ChallengeSection.svelte';
	import GamePanel from './components/GamePanel.svelte';
	import TeamSection from './components/TeamSection.svelte';
	import { resolve } from '$app/paths';
	import CopyableValue from '$lib/components/CopyableValue.svelte';

	let { data } = $props();

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

{#snippet state()}
	{#if data.game}
		<StateBadge
			text={status(data.game)}
			state={status(data.game)}
			mapper={{
				upcoming: 'StableWarning',
				running: 'RunningPositive',
				ended: 'StableNeutral'
			}}
		/>
	{/if}
{/snippet}

{#snippet uuid()}
	<CopyableValue descriptor="Game ID" value={data.game?.id} />
{/snippet}

{#snippet overview()}
	{#if data.game}
		<GamePanel game={data.game} refresh={() => data.refresh()} deleted={() => goto(resolve('/host/games'))} />
	{/if}
{/snippet}

{#snippet teams()}
	{#if data.game}
		<TeamSection game={data.game} />
	{/if}
{/snippet}

{#snippet challenges()}
	<ChallengeSection gameId={data.game?.id ?? ''} />
{/snippet}

<TabPage
	title={data.game?.name ?? 'Game'}
	headers={[state, uuid]}
	back={() => goto(resolve('/host/games'))}
	dataState={data.gameState}
	tabs={['overview', 'teams', 'challenges']}
	tabPages={[overview, teams, challenges]}
/>
