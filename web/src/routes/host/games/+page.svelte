<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, Submit, type SubmitState } from '$lib';
	import { ListRequestSchema } from '$lib/sdk/v1/play/game/game_pb';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import { ChevronLeftIcon, PlusIcon } from '@lucide/svelte';
	import StateBadge from '$lib/components/StateBadge.svelte';
	import LinkButton from '$lib/components/LinkButton.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import Card from '$lib/components/Card.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import { resolve } from '$app/paths';

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

<div class="flex h-11/12 w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto(resolve('/host'))}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Games</h1>
		<LinkButton class="ml-auto" href="/host/games/new/">
			<PlusIcon /> Host Game
		</LinkButton>
	</div>

	{#if gamesState.forbidden}
		<div class="flex h-full w-full items-center justify-center">
			<Card>
				<div class="flex flex-col items-start gap-1">
					<h1 class="text-2xl font-bold text-primary/80">Find Game</h1>
					<p class="text-primary/60">You are not allowed to list games, find the one you need by its id</p>
				</div>
				<form class="flex w-full flex-row items-center gap-2" onsubmit={() => goto(resolve(`/host/games/${id}`))}>
					<Input class="w-full" bind:value={id} placeholder="Game id" />
					<Button type="submit" variant="primary" disabled={!id.trim()}>Open</Button>
				</form>
			</Card>
		</div>
	{:else}
		<div class="h-full rounded-sm p-2">
			<DataTable
				class="max-h-full"
				limit={100}
				load={async (startAfter: Game | undefined) => {
					const games: Game[] = [];
					await Submit(async () => {
						const resp = await Glue.game.list(create(ListRequestSchema, { limit: 100, startAfter: startAfter?.id }));
						games.push(...resp.games);
					}, gamesState);
					return games;
				}}
				onRowClick={(game) => {
					goto(resolve(`/host/games/${game.id}`));
				}}
				loadState={gamesState}
				columns={[
					{ title: 'Status', width: '110px', render: renderStatus },
					{ title: 'Name', width: '20%', render: renderName },
					{ title: 'Description', width: '40%', render: renderDescription },
					{ title: 'From', width: '20%', render: renderFrom },
					{ title: 'To', width: '20%', render: renderTo }
				]}
			/>
			{#snippet renderName(game: Game)}
				{game.name}
			{/snippet}
			{#snippet renderDescription(game: Game)}
				<span class="text-primary/60">{game.description}</span>
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
		</div>
	{/if}
</div>
