<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, Submit, type SubmitState } from '$lib';
	import * as Card from '$lib/components/shad/card';
	import { Input } from '$lib/components/shad/input';
	import { ListRequestSchema } from '$lib/sdk/v1/play/game/game_pb';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import DataTable from './DataTable.svelte';
	import StateBadge from '$lib/components/custom/badges/StateBadge.svelte';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import LinkButton from '$lib/components/custom/buttons/LinkButton.svelte';

	let id = $state('');

	function status(game: Game): string {
		const now = Date.now();
		if (game.from && timestampDate(game.from).getTime() > now) return 'upcoming';
		if (game.to && timestampDate(game.to).getTime() < now) return 'ended';
		return 'running';
	}
	let gamesState: SubmitState = $state({ error: '', loading: true, forbidden: false });
</script>

<svelte:head>
	<title>Games | CloudJam</title>
	<meta property="og:title" content="Games | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" size="icon" class="cursor-pointer" href="/host/">
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Games</h1>
		<Button variant="outline" class="ml-auto cursor-pointer" href="/host/games/new/">
			<PlusIcon /> Host Game
		</Button>
	</div>

	<DataTable
		load={async (startAfter: Game | undefined) => {
			const games: Game[] = [];
			await Submit(async () => {
				const resp = await Glue.game.list(create(ListRequestSchema, { limit: 3, startAfter: startAfter?.id }));
				games.push(...resp.games);
			}, gamesState);
			return games;
		}}
		loadState={gamesState}
		columns={[
			{ title: 'Status', width: '90px', render: renderStatus },
			{ title: 'Name', width: '10%', render: renderName },
			{ title: 'Description', width: '20%', render: renderDescription },
			{ title: 'From', width: '20%', render: renderFrom },
			{ title: 'To', width: '20%', render: renderTo },
			{ title: 'Action', width: '10%', render: renderAction }
		]}
	/>
	{#snippet renderName(game: Game)}
		<b>{game.name}</b>
	{/snippet}
	{#snippet renderDescription(game: Game)}
		{game.description}
	{/snippet}
	{#snippet renderFrom(game: Game)}
		{game.from ? timestampDate(game.from).toLocaleString() : ''}
	{/snippet}
	{#snippet renderTo(game: Game)}
		{game.to ? timestampDate(game.to).toLocaleString() : ''}
	{/snippet}
	{#snippet renderStatus(game: Game)}
		<StateBadge
			text={status(game)}
			state={status(game)}
			mapper={{ running: 'StablePositive', upcoming: 'StableWarning', ended: 'StableNeutral' }}
		/>
	{/snippet}
	{#snippet renderAction(game: Game)}
		<LinkButton text="Open" href={`/host/games/${game.id}`} data-sveltekit-preload-data="tap" />
	{/snippet}

	{#if gamesState.forbidden}
		<Card.Root class="w-full">
			<Card.Header>
				<Card.Title>Open a Game</Card.Title>
				<Card.Description>You are not allowed to list games, open the one you host by its id.</Card.Description>
			</Card.Header>
			<Card.Content>
				<form class="flex flex-row items-center gap-2" onsubmit={() => goto(`/host/games/${id}`)}>
					<Input class="max-w-96" bind:value={id} placeholder="Game id" />
					<Button type="submit" class="cursor-pointer" disabled={!id.trim()}>Open</Button>
				</form>
			</Card.Content>
		</Card.Root>
	{/if}
</div>
