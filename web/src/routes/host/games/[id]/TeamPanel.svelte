<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/custom/Alert.svelte';
	import DynamicBadge from '$lib/components/custom/badges/DynamicBadge.svelte';
	import StaticBadge from '$lib/components/custom/badges/StaticBadge.svelte';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import OptionalSelect from '$lib/components/custom/OptionalSelect.svelte';
	import Popup from '$lib/components/layout/Popup.svelte';
	import { Input } from '$lib/components/shad/input';
	import { Separator } from '$lib/components/shad/separator';
	import { ListRequestSchema as ListUsersRequestSchema } from '$lib/sdk/v1/admin/user/user_pb';
	import type { User } from '$lib/sdk/v1/admin/user_pb';
	import { DeleteRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/play/team/team_pb';
	import { PlayerSchema, type Player, type Team } from '$lib/sdk/v1/play/team_pb';
	import { create } from '@bufbuild/protobuf';
	import XIcon from '@lucide/svelte/icons/x';
	import { toSvg } from 'jdenticon';
	import { onMount } from 'svelte';

	let { open = $bindable(), team, refresh }: { open: boolean; team: Team; refresh: () => void } = $props();

	let name = $derived(team.name);
	let players: { [key: string]: Player } = $derived({ ...team.players });
	let confirmDelete = $state(false);

	let users: User[] = $state([]);
	let player = $state('');

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let attachState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let usersState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	onMount(() =>
		Submit(async () => {
			users = (await Glue.user.list(create(ListUsersRequestSchema, { limit: 100 }))).users;
		}, usersState)
	);
</script>

<Popup
	bind:open
	title={team.name}
	description={`${Object.keys(team.players).length} players`}
	badges={renderPopupBadges}
	body={renderPopup}
/>

{#snippet renderPopupBadges()}
	<StaticBadge text={`score: ${team.score}`} variant={'jam'} />
	{#each Object.entries(team.players) as [_, player]}
		<DynamicBadge variant={'primary'}>
			<img
				alt="user profile"
				src={`data:image/svg+xml;base64,${btoa(toSvg(player.pubId, 16))}`}
				height="3rem"
				class="rounded-lg bg-primary/5"
			/>
			{player.username}
		</DynamicBadge>
	{/each}
{/snippet}
{#snippet renderPopup()}
	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Name</h1>
		{#if updateState.forbidden}
			<Alert class="p-2" message="You are not allowed to update this team" />
		{:else}
			<form
				class="flex flex-row items-center gap-2"
				onsubmit={() =>
					Submit(async () => {
						await Glue.team.update(create(UpdateRequestSchema, { mod: { ...team, name: name } }));
						refresh();
					}, updateState)}
			>
				<Input class="max-w-96" bind:value={name} placeholder="Name of the team" />
				<Button processing={updateState.loading} type="submit" variant="primary">Save</Button>
			</form>
			{#if updateState.error}
				<Alert class="p-2" message={updateState.error} animation={true} />
			{/if}
		{/if}
	</div>

	<Separator />

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Players</h1>
		<div class="flex flex-row flex-wrap gap-1">
			{#each Object.values(players) as member (member.id)}
				<DynamicBadge variant="primary">
					{member.username}
					<Button
						variant="ghost"
						type="button"
						aria-label="Remove {member.username}"
						onclick={() => {
							delete players[member.id];
							Submit(async () => {
								await Glue.team.update(create(UpdateRequestSchema, { mod: { ...team, players: players } }));
								refresh();
							}, attachState);
						}}
					>
						<XIcon class="size-3" />
					</Button>
				</DynamicBadge>
			{:else}
				<p class="text-muted-foreground text-sm italic">No players attached yet.</p>
			{/each}
		</div>
		<div class="flex flex-row items-center gap-2">
			<OptionalSelect
				bind:value={player}
				placeholder="Select a player"
				suggestions={users.map((user) => ({ id: user.id, title: user.username }))}
			/>

			<Button
				variant="primary"
				processing={attachState.loading}
				disabled={attachState.loading || !player || Boolean(players[player])}
				onclick={() => {
					const user = users.find((user) => user.id === player);
					if (!user) return;
					players[user.id] = create(PlayerSchema, { id: user.id, pubId: user.pubId, username: user.username });
					Submit(async () => {
						await Glue.team.update(create(UpdateRequestSchema, { mod: { ...team, players: players } }));
						refresh();
					}, attachState);
				}}
			>
				Attach
			</Button>
		</div>
		{#if attachState.error}
			<Alert class="p-2" message={attachState.error} animation={true} />
		{/if}
	</div>

	<Separator />

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Danger Zone</h1>
		{#if removeState.forbidden}
			<p class="text-sm text-muted-foreground italic">You are not allowed to delete this team.</p>
		{:else}
			<div class="flex flex-row items-center gap-2">
				{#if confirmDelete}
					<Button
						variant="destructive"
						class="cursor-pointer"
						disabled={removeState.loading}
						onclick={() =>
							Submit(async () => {
								await Glue.team.delete(create(DeleteRequestSchema, { gameId: team.gameId, id: team.id }));
								refresh();
							}, removeState)}
					>
						Yes, delete {team.name}
					</Button>
					<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
				{:else}
					<Button variant="destructive" onclick={() => (confirmDelete = true)}>Delete Team</Button>
				{/if}
			</div>
			{#if removeState.error}
				<Alert class="p-2" message={removeState.error} animation={true} />
			{/if}
		{/if}
	</div>
{/snippet}
