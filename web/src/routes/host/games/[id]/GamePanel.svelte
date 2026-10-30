<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/custom/Alert.svelte';
	import StaticBadge from '$lib/components/custom/badges/StaticBadge.svelte';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import Card from '$lib/components/layout/Card.svelte';
	import { Input } from '$lib/components/shad/input';
	import { Separator } from '$lib/components/shad/separator';
	import { DeleteRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/play/game/game_pb';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate, timestampFromDate } from '@bufbuild/protobuf/wkt';

	let { game, refresh, deleted }: { game: Game; refresh: () => void; deleted: () => void } = $props();

	let mod = $state({ ...game });

	// datetime-local inputs work on the local wall clock, protobuf timestamps on utc instants.
	const localInput = (date: Date) =>
		new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);

	let from = $derived(localInput(game.from ? timestampDate(game.from) : new Date()));
	let to = $derived(localInput(game.to ? timestampDate(game.to) : new Date()));

	let confirmDelete = $state(false);

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-2xl font-semibold text-primary/90">{game.name}</h1>
		<p class="text-primary/60">{game.description}</p>
	</div>
	<div class="flex flex-row flex-wrap gap-1">
		<StaticBadge variant="primary" text={`scope: ${game.scope}`} />
		<StaticBadge variant="primary" text={game.id} />
	</div>

	<div class="flex flex-col gap-2">
		<h2 class="text-xl font-semibold text-primary/80">Configuration</h2>
		{#if updateState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to update this game.</p>
		{:else}
			<p class="text-sm text-primary/60">Games that already started cannot be changed anymore.</p>
			<form
				class="flex flex-col gap-4"
				onsubmit={() =>
					Submit(async () => {
						await Glue.game.update(
							create(UpdateRequestSchema, {
								mod: { ...mod, from: timestampFromDate(new Date(from)), to: timestampFromDate(new Date(to)) }
							})
						);
						refresh();
					}, updateState)}
			>
				<div class="grid gap-4 md:grid-cols-2">
					<div class="flex flex-col gap-1">
						<label class="text-sm">
							Name
							<Input bind:value={mod.name} placeholder="Name of the game" />
						</label>
					</div>
					<div class="flex flex-col gap-1">
						<label class="text-sm">
							Description
							<Input bind:value={mod.description} placeholder="What the game is about" />
						</label>
					</div>
					<div class="flex flex-col gap-1">
						<label class="text-sm">
							From
							<Input type="datetime-local" bind:value={from} />
						</label>
					</div>
					<div class="flex flex-col gap-1">
						<label class="text-sm">
							To
							<Input type="datetime-local" bind:value={to} />
						</label>
					</div>
				</div>
				<Button type="submit" variant="primary" processing={updateState.loading}>Save</Button>
			</form>
			{#if updateState.error}
				<Alert class="p-2" message={updateState.error} animation={true} />
			{/if}
		{/if}
	</div>
	<Separator />

	<div class="flex flex-col gap-2">
		<h2 class="text-xl font-semibold text-primary/80">Danger Zone</h2>
		{#if removeState.forbidden}
			<p class="text-sm text-muted-foreground italic">You are not allowed to delete this game.</p>
		{:else}
			<div class="flex flex-row items-center gap-2">
				{#if confirmDelete}
					<Button
						variant="destructive"
						class="cursor-pointer"
						disabled={removeState.loading}
						onclick={() =>
							Submit(async () => {
								await Glue.game.delete(create(DeleteRequestSchema, { id: game.id }));
								deleted();
							}, removeState)}
					>
						Yes, delete {game.name}
					</Button>
					<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
				{:else}
					<Button variant="destructive" onclick={() => (confirmDelete = true)}>Delete Game</Button>
				{/if}
			</div>
			{#if removeState.error}
				<Alert class="p-2" message={removeState.error} animation={true} />
			{/if}
		{/if}
	</div>
</Card>
