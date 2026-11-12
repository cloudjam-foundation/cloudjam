<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import OptionalSelect, { type Suggestion } from '$lib/components/OptionalSelect.svelte';
	import Card from '$lib/components/Card.svelte';
	import Input from '$lib/components/Input.svelte';
	import { getAssignableTenants } from '$lib/auth.svelte.js';
	import { CreateRequestSchema } from '$lib/sdk/v1/play/game/game_pb';
	import { GameSchema } from '$lib/sdk/v1/play/game_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampFromDate } from '@bufbuild/protobuf/wkt';
	import { resolve } from '$app/paths';

	let createState: SubmitState = $state({ loading: false, error: '', forbidden: false });

	let init = $state(create(GameSchema, { id: crypto.randomUUID() }));
	const localInput = (date: Date) =>
		new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);

	let from = $state(localInput(new Date(Date.now() + 60 * 1000)));
	let to = $state(localInput(new Date(Date.now() + 3 * 60 * 60 * 1000)));

	let request = $derived(
		create(CreateRequestSchema, {
			init: { ...init, from: timestampFromDate(new Date(from)), to: timestampFromDate(new Date(to)) }
		})
	);
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl text-primary/90">Host Game</h1>
		<p class="text-primary/60">Teams and challenges can be added later in the game tab</p>
	</div>

	<form
		class="flex w-full flex-col gap-4"
		onsubmit={(e) => {
			e.preventDefault();

			// Stops the frustration of trying to create a game
			// that starts now, but it's been too long since they
			// set the time.
			// Adding 30 seconds to avoid race conditions.
			if (new Date(from).getTime() < Date.now()) {
				from = localInput(new Date(Date.now() + 30 * 1000));
			}

			Submit(async () => {
				const resp = await Glue.game.create(request);
				await goto(resolve(`/host/games/[id]`, { id: resp.id }));
			}, createState);
		}}
	>
		<div class="grid gap-4 md:grid-cols-2">
			<Input
				bind:value={init.name}
				label="Name"
				placeholder="Name of the game"
				validation={Glue.Validate(GameSchema, init).violation.name}
			/>
			<Input
				bind:value={init.description}
				label="Description"
				placeholder="What the game is about"
				validation={Glue.Validate(GameSchema, init).violation.description}
			/>
			<label class="text-sm">
				From
				<Input type="datetime-local" bind:value={from} />
			</label>
			<label class="text-sm">
				To
				<Input type="datetime-local" bind:value={to} />
			</label>
			<label class="text-sm">
				Scope
				<OptionalSelect
					bind:value={init.scope}
					placeholder="Scope the game is placed in"
					suggestions={getAssignableTenants().map((scope) => ({ id: scope, title: scope })) satisfies Suggestion[]}
					class="w-full"
				/>
			</label>
		</div>
		<Button variant="primary" type="submit" processing={createState.loading}>Schedule</Button>
		{#if createState.error}
			<Alert
				title={createState.forbidden ? 'Permission denied' : 'Failed to schedule game'}
				message={createState.error}
				animation={true}
			/>
		{/if}
	</form>
</Card>
