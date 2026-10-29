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
	import StateBadge from '$lib/components/custom/badges/StateBadge.svelte';
	import LinkButton from '$lib/components/custom/buttons/LinkButton.svelte';
	import DataTable from '$lib/components/layout/DataTable.svelte';
	import ProcessingButton from '$lib/components/custom/buttons/ProcessingButton.svelte';
	import { SquareArrowOutUpRight } from '@lucide/svelte';

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
		<LinkButton ghost href="/host/">
			<ChevronLeftIcon />
		</LinkButton>
		<h1 class="text-3xl opacity-80">Games</h1>
		<LinkButton class="ml-auto" href="/host/games/new/">
			<PlusIcon /> Host Game
		</LinkButton>
	</div>

	<div class="h-full rounded-sm p-2">
		<DataTable
			class="max-h-full"
			load={async (startAfter: Game | undefined) => {
				const games: Game[] = [];
				await Submit(async () => {
					const resp = await Glue.game.list(create(ListRequestSchema, { limit: 100, startAfter: startAfter?.id }));
					games.push(...resp.games);
				}, gamesState);
				return games;
			}}
			loadState={gamesState}
			columns={[
				{ title: 'Status', width: '90px', render: renderStatus },
				{ title: 'Name', width: '20%', render: renderName },
				{ title: 'Description', width: '40%', render: renderDescription },
				{ title: 'From', width: '20%', render: renderFrom },
				{ title: 'To', width: '20%', render: renderTo }
			]}
		/>
		{#snippet renderName(game: Game)}
			<LinkButton
				href={`/host/games/${game.id}`}
				class="justify-between underline active:scale-[0.99]"
				data-sveltekit-preload-data="tap"
			>
				{game.name}
			</LinkButton>
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

	{#if gamesState.forbidden}
		<Card.Root class="w-full">
			<Card.Header>
				<Card.Title>Open a Game</Card.Title>
				<Card.Description>You are not allowed to list games, open the one you host by its id.</Card.Description>
			</Card.Header>
			<Card.Content>
				<form class="flex flex-row items-center gap-2" onsubmit={() => goto(`/host/games/${id}`)}>
					<Input class="max-w-96" bind:value={id} placeholder="Game id" />
					<ProcessingButton type="submit" variant={'primary'} disabled={!id.trim()}>Open</ProcessingButton>
				</form>
			</Card.Content>
		</Card.Root>
	{/if}
</div>
