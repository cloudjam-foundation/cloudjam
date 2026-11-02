<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Markdown from '$lib/components/Markdown.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import Popup from '$lib/components/Popup.svelte';
	import Select from '$lib/components/Select.svelte';
	import Separator from '$lib/components/Separator.svelte';
	import { DeleteRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/play/challenge/challenge_pb';
	import { ScoreType, ScoreTypeSchema, type Challenge, type ScoreEvent } from '$lib/sdk/v1/play/challenge_pb';
	import type { Team } from '$lib/sdk/v1/play/team_pb';
	import { jpegDataURL } from '$lib/utils';
	import { create, enumToJson } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';

	let {
		open = $bindable(),
		challenge,
		teams,
		refresh
	}: { open: boolean; challenge: Challenge; teams: Team[]; refresh: () => void } = $props();

	let teamId = $derived(challenge.teamId);
	let confirmDelete = $state(false);
	let hasScoreReasons = $derived(challenge.scoreEvents.some((event) => event.reason));
	const scoreTypeName = (type: ScoreType) =>
		type === ScoreType.Unspecified ? '' : enumToJson(ScoreTypeSchema, type)?.toString().toLowerCase();

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });
</script>

<Popup bind:open title={challenge.title || 'Not started yet'} badges={renderBadges} body={renderBody} />

{#snippet renderBadges()}
	<StaticBadge variant="jam" text={`score: ${challenge.scoreEvents.reduce((sum, event) => sum + event.change, 0)}`} />
	{#each challenge.scores as score (score.type)}
		<StaticBadge variant="primary" text={`${scoreTypeName(score.type)}: ${score.value} / ${score.maximum}`} />
	{/each}
	<StaticBadge variant="primary" text={`scope: ${challenge.scope}`} />
	<StaticBadge variant="primary" text={challenge.id} class="font-mono" />
{/snippet}

{#snippet renderBody()}
	{#if challenge.error}
		<Alert title="Challenge reported an error" message={challenge.error} />
	{/if}

	{#if challenge.description.length}
		<div class="flex flex-col gap-2">
			<h1 class="text-xl font-semibold text-primary/80">Briefing</h1>
			<Markdown source={challenge.description.join('\n\n')} />
		</div>
	{/if}

	{#if Object.keys(challenge.diagrams).length}
		<Separator />

		<div class="flex flex-col gap-3">
			<h1 class="text-xl font-semibold text-primary/80">Infrastructure</h1>
			{#each Object.entries(challenge.diagrams) as [name, diagram] (name)}
				<figure class="flex flex-col gap-2">
					<img class="max-h-96 rounded-md object-contain" src={jpegDataURL(diagram)} alt={name} />
					<figcaption class="text-center text-xs text-primary/60">{name}</figcaption>
				</figure>
			{/each}
		</div>
	{/if}

	<Separator />

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Team</h1>
		{#if updateState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to update this challenge.</p>
		{:else}
			<div class="flex flex-row items-center gap-2">
				<Select
					bind:value={teamId}
					class="w-72"
					placeholder="Select a team"
					items={teams.map((team) => ({ value: team.id, label: team.name }))}
				/>
				<Button
					variant="primary"
					processing={updateState.loading}
					disabled={updateState.loading || !teamId || teamId === challenge.teamId}
					onclick={() =>
						Submit(async () => {
							await Glue.challenge.update(create(UpdateRequestSchema, { mod: { ...challenge, teamId: teamId } }));
							refresh();
						}, updateState)}
				>
					Reassign
				</Button>
			</div>
			{#if updateState.error}
				<Alert class="p-2" message={updateState.error} animation={true} />
			{/if}
		{/if}
	</div>

	{#if challenge.scoreEvents.length}
		<Separator />

		<div class="flex flex-col gap-2">
			<h1 class="text-xl font-semibold text-primary/80">Score Events</h1>
			<DataTable
				class="max-h-96"
				limit={Infinity}
				load={async () => challenge.scoreEvents}
				loadState={{ error: '', loading: false, forbidden: false }}
				columns={[
					{ title: 'Time', width: '20%', render: renderTime },
					{ title: 'Event', width: '30%', render: renderEvent },
					{ title: 'Type', width: '10%', render: renderType },
					{ title: 'Result', width: '10%', render: renderResult },
					{ title: 'Change', width: '10%', render: renderChange },
					...(hasScoreReasons ? [{ title: 'Explanation', width: '20%', render: renderReason }] : [])
				]}
			/>
			{#snippet renderTime(event: ScoreEvent)}
				{event.timestamp ? timestampDate(event.timestamp).toLocaleString() : ''}
			{/snippet}
			{#snippet renderEvent(event: ScoreEvent)}
				{event.text}
			{/snippet}
			{#snippet renderType(event: ScoreEvent)}
				<span class="text-primary/60">{scoreTypeName(event.type)}</span>
			{/snippet}
			{#snippet renderResult(event: ScoreEvent)}
				{event.maximum ? `${event.score} / ${event.maximum}` : ''}
			{/snippet}
			{#snippet renderChange(event: ScoreEvent)}
				<StaticBadge variant="jam" text={event.change.toString()} />
			{/snippet}
			{#snippet renderReason(event: ScoreEvent)}
				<span class="text-primary/60">{event.reason}</span>
			{/snippet}
		</div>
	{/if}

	<Separator />

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Danger Zone</h1>
		{#if removeState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to delete this challenge.</p>
		{:else}
			<div class="flex flex-row items-center gap-2">
				{#if confirmDelete}
					<Button
						variant="destructive"
						disabled={removeState.loading}
						onclick={() =>
							Submit(async () => {
								await Glue.challenge.delete(
									create(DeleteRequestSchema, { gameId: challenge.gameId, id: challenge.id })
								);
								refresh();
							}, removeState)}
					>
						Yes, delete this challenge
					</Button>
					<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
				{:else}
					<Button variant="destructive" onclick={() => (confirmDelete = true)}>Delete Challenge</Button>
				{/if}
			</div>
			{#if removeState.error}
				<Alert class="p-2" message={removeState.error} animation={true} />
			{/if}
		{/if}
	</div>
{/snippet}
