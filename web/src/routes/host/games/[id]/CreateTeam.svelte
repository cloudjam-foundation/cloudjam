<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/custom/Alert.svelte';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import Card from '$lib/components/layout/Card.svelte';
	import Input from '$lib/components/shad/input/input.svelte';
	import LabelInput from '$lib/components/shad/label-input/label-input.svelte';
	import type { Game } from '$lib/sdk/v1/play/game_pb';
	import { CreateRequestSchema } from '$lib/sdk/v1/play/team/team_pb';
	import { TeamSchema } from '$lib/sdk/v1/play/team_pb';
	import { create } from '@bufbuild/protobuf';

	let { game, oncreated }: { game: Game; oncreated: () => void } = $props();

	let init = $state(create(TeamSchema, { id: crypto.randomUUID() }));

	let createState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	function createTeam() {
		Submit(async () => {
			await Glue.team.create(create(CreateRequestSchema, { init: { ...init, gameId: game.id, scope: game.scope } }));
			init.name = '';
			oncreated();
		}, createState);
	}
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Create Team</h1>
		<p class="text-primary/60">Creates a team of the selected game. Players are attached afterwards</p>
	</div>
	<form class="flex flex-col gap-2" onsubmit={createTeam}>
		<div class="flex flex-row items-center justify-center gap-2">
			<Input bind:value={init.name} placeholder="Name of the team" />
			<Button processing={createState.loading} variant="primary" type="submit" class="self-start">Create</Button>
		</div>
		{#if createState.error}
			<Alert
				title={createState.forbidden ? 'Permission denied' : 'Failed to create team'}
				message={createState.error}
				animation={true}
			/>
		{/if}
	</form>
</Card>
