<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import DynamicBadge from '$lib/components/DynamicBadge.svelte';
	import LoadingBadge from '$lib/components/LoadingBadge.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import LinkButton from '$lib/components/LinkButton.svelte';
	import Markdown from '$lib/components/Markdown.svelte';
	import Dialog from '$lib/components/Dialog.svelte';
	import Card from '$lib/components/Card.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import Separator from '$lib/components/Separator.svelte';
	import {
		CredentialsRequestSchema,
		StartRequestSchema,
		UncoverClueRequestSchema
	} from '$lib/sdk/v1/play/challenge/challenge_pb';
	import { ScoreType, type Challenge, type ScoreEvent } from '$lib/sdk/v1/play/challenge_pb';
	import { toDataURL } from '$lib/utils.svelte.js';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import {
		BadgeCheckIcon,
		DraftingCompassIcon,
		KeyRoundIcon,
		SquareArrowOutUpRightIcon,
		ZapIcon,
		CopyIcon,
		LightbulbIcon,
		PlayIcon
	} from '@lucide/svelte';

	const kitchenFmt = Intl.DateTimeFormat(undefined, {
		hour: '2-digit',
		minute: '2-digit',
		hour12: false
	});

	let {
		challenge,
		active,
		nextInterval,
		refresh
	}: { challenge: Challenge; active: boolean; nextInterval: Date; refresh: () => void } = $props();

	let credentials = $state('');

	let parsedCredentials: { env: [string, string][]; url: string } = $derived.by(() => {
		const parsed = JSON.parse(credentials);
		return {
			env: Object.entries(JSON.parse(credentials)).filter(
				([key, value]) => typeof value === 'string' && value && key !== 'URL'
			) as [string, string][],
			url: parsed.URL.toString()
		};
	});

	let startState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let credsState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let clueState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let adhsTimeout: ReturnType<typeof setTimeout>;
	let adhsSpinner: boolean = $state(false);

	function group(events: ScoreEvent[]): Record<string, ScoreEvent[]> {
		const groups: Record<string, ScoreEvent[]> = {};
		for (const event of events.toReversed()) {
			if (!groups[event.text]) groups[event.text] = [event];
			else groups[event.text].push(event);
		}
		return groups;
	}

	function getOperationColor(event: ScoreEvent): string {
		switch (event.score) {
			case 0:
				return 'text-red-500/70';
			case event.maximum:
				return 'text-emerald-500/70';
			default:
				return 'text-amber-500/70';
		}
	}

	function getDesignColor(event: ScoreEvent): string {
		switch (event.score) {
			case 0:
				return 'text-red-500/70';
			case event.maximum:
				return 'text-emerald-500/70';
			default:
				return 'text-amber-500/70';
		}
	}

	type EventGroup = { key: string; events: ScoreEvent[] };
	const groups = $derived<EventGroup[]>(
		Object.entries(group(challenge.scoreEvents)).map(([key, events]) => ({ key, events }))
	);

	// the history dialog shows every event of the clicked group
	let history: EventGroup | undefined = $state();
	let historyOpen = $state(false);
</script>

<Card class="w-full items-start">
	<div class="flex w-full flex-row flex-wrap items-center gap-2">
		<div class="flex flex-col gap-1">
			<div class="flex flex-row flex-wrap items-center gap-2">
				<h1 class="text-2xl font-semibold text-primary/90">{challenge.title || 'Not started yet'}</h1>
				{#if !challenge.ready && challenge.title}
					<LoadingBadge text="Provisioning Challenge" />
				{/if}
				{#if adhsSpinner}
					<LoadingBadge text="Checking Progress" />
				{/if}
			</div>
			<div class="flex flex-row flex-wrap gap-1">
				{#each challenge.scores as score (score.type)}
					<DynamicBadge variant="primary">
						{#if score.type === ScoreType.Operational}
							<ZapIcon class="size-3" />
						{:else if score.type === ScoreType.Design}
							<DraftingCompassIcon class="size-3" />
						{/if}
						<span>{score.value} / {score.maximum}</span>
					</DynamicBadge>
				{/each}
				{#if !challenge.title}
					<StaticBadge variant="jam" text="not started yet" />
				{/if}
			</div>
		</div>
		<div class="ml-auto flex flex-row flex-wrap gap-2">
			<Button
				variant="primary"
				processing={startState.loading}
				disabled={!active || startState.loading || Boolean(challenge.title)}
				onclick={() =>
					Submit(async () => {
						await Glue.challenge.start(create(StartRequestSchema, { gameId: challenge.gameId, id: challenge.id }));
						challenge.title = '...';
					}, startState)}
			>
				<PlayIcon /> Start
			</Button>

			<Button
				variant="primary"
				disabled={!active || !challenge.ready}
				onclick={() => {
					clearTimeout(adhsTimeout);
					adhsSpinner = true;
					adhsTimeout = setTimeout(() => (adhsSpinner = false), nextInterval.getTime() - Date.now());
				}}
			>
				<BadgeCheckIcon /> Check
			</Button>

			<Button
				variant="primary"
				processing={credsState.loading}
				disabled={!active || !challenge.ready || credsState.loading || !challenge.title}
				onclick={() =>
					Submit(async () => {
						credentials = (
							await Glue.challenge.credentials(
								create(CredentialsRequestSchema, { gameId: challenge.gameId, id: challenge.id })
							)
						).credentials;
					}, credsState)}
			>
				<KeyRoundIcon />
				Credentials
			</Button>
		</div>
	</div>

	{#if startState.error || credsState.error}
		<Alert title="Action failed" message={startState.error || credsState.error} animation={true} />
	{/if}

	{#if credentials}
		<div class="flex w-full flex-row flex-wrap items-center gap-3 rounded-md glass p-3">
			<span class="font-medium text-primary/90">Account credentials</span>
			<!-- TODO: Fix -->
			<Button
				variant="primary"
				onclick={() =>
					navigator.clipboard.writeText(
						parsedCredentials.env.length
							? parsedCredentials.env.map(([name, value]) => `export ${name}=${value}`).join('\n')
							: credentials
					)}
			>
				<CopyIcon /> Copy as environment
			</Button>
			<LinkButton href={parsedCredentials.url}>
				<SquareArrowOutUpRightIcon /> Open AWS Console
			</LinkButton>
		</div>
	{/if}

	{#if challenge.error}
		<Alert
			title="Challenge is malfunctioning. Please contact an administrator!"
			message={challenge.error}
			class="font-mono text-xs break-all"
		/>
	{/if}

	{#if challenge.description.length}
		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Briefing</h2>
			<Markdown source={challenge.description.join('\n\n')} />
		</div>
	{/if}

	{#if Object.keys(challenge.diagrams).length}
		<Separator />

		<div class="flex flex-col gap-3">
			<h2 class="text-xl font-semibold text-primary/80">Infrastructure</h2>
			{#each Object.entries(challenge.diagrams) as [name, diagram] (name)}
				<figure class="flex flex-col gap-2">
					<img class="max-h-96 rounded-md object-contain" src={toDataURL(diagram)} alt={name} />
					<figcaption class="text-center text-xs text-primary/60">{name}</figcaption>
				</figure>
			{/each}
		</div>
	{/if}

	{#if Object.keys(challenge.assets).length}
		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Assets</h2>
			{#each Object.entries(challenge.assets) as [name, asset] (name)}
				<div class="flex flex-row items-center gap-2 text-sm">
					<span class="font-medium">{name}</span>
					<span class="font-mono text-xs break-all text-primary/60">{asset}</span>
				</div>
			{/each}
		</div>
	{/if}

	{#if Object.keys(challenge.clues).length}
		<Separator />

		<div class="flex flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">Clues</h2>
			<p class="text-sm text-primary/60">Uncovering a clue usually costs points.</p>
			<Separator />
			{#each Object.entries(challenge.clues) as [name, text] (name)}
				<div class="flex flex-col justify-center gap-2 text-sm">
					<span class="font-medium">{name}</span>
					{#if text !== '<hidden>'}
						<Markdown source={text} />
					{:else}
						<Button
							variant="primary"
							class="self-start"
							processing={clueState.loading}
							disabled={clueState.loading || challenge.cluePrices[name] === undefined}
							onclick={() =>
								Submit(async () => {
									await Glue.challenge.uncoverClue(
										create(UncoverClueRequestSchema, {
											gameId: challenge.gameId,
											id: challenge.id,
											clue: name
										})
									);
									refresh();
								}, clueState)}
						>
							<LightbulbIcon /> Uncover ({challenge.cluePrices[name] === undefined
								? 'price unavailable'
								: `${-challenge.cluePrices[name]} points`})
						</Button>
					{/if}
				</div>
			{/each}
			{#if clueState.error}
				<Alert class="p-2" message={clueState.error} animation={true} />
			{/if}
		</div>
	{/if}

	{#if challenge.scoreEvents.length}
		<Separator />

		<div class="flex w-full flex-col gap-2">
			<h2 class="text-xl font-semibold text-primary/80">System Status</h2>
			{#key challenge.scoreEvents.length}
				<DataTable
					class="max-h-[60vh]"
					limit={Infinity}
					load={async () => groups}
					loadState={{ error: '', loading: false, forbidden: false }}
					onRowClick={(row) => {
						history = row;
						historyOpen = true;
					}}
					columns={[
						{ title: '', width: '40px', render: renderGroupIcon },
						{ title: 'Time', width: '80px', render: renderGroupTime },
						{ title: 'Result', width: '110px', render: renderGroupResult },
						{ title: 'Event', width: '45%', render: renderGroupText },
						{ title: 'Reason', width: '55%', render: renderGroupReason }
					]}
				/>
			{/key}
			{#snippet renderGroupIcon(row: EventGroup)}
				{@render icon(row.events[0])}
			{/snippet}
			{#snippet renderGroupTime(row: EventGroup)}
				{@render time(row.events[0])}
			{/snippet}
			{#snippet renderGroupResult(row: EventGroup)}
				{@render result(row.events[0])}
			{/snippet}
			{#snippet renderGroupText(row: EventGroup)}
				{row.events[0].text}
			{/snippet}
			{#snippet renderGroupReason(row: EventGroup)}
				<span class="text-primary/60">{row.events[0].reason}</span>
			{/snippet}
		</div>
	{/if}
</Card>

{#snippet icon(event: ScoreEvent)}
	{#if event.type === ScoreType.Operational}
		<ZapIcon class={getOperationColor(event)} />
	{:else if event.type === ScoreType.Design}
		<DraftingCompassIcon class={getDesignColor(event)} />
	{/if}
{/snippet}
{#snippet time(event: ScoreEvent)}
	<span class="font-bold text-primary/60">{kitchenFmt.format(timestampDate(event.timestamp!))}</span>
{/snippet}
{#snippet result(event: ScoreEvent)}
	<StaticBadge variant="primary" text={event.maximum ? `${event.score} / ${event.maximum}` : ''} />
{/snippet}

{#if history}
	<Dialog bind:open={historyOpen} title={history.key} description="Every check of this event, newest first">
		{#key history.key}
			<DataTable
				class="max-h-[60vh]"
				limit={Infinity}
				load={async () => history?.events ?? []}
				loadState={{ error: '', loading: false, forbidden: false }}
				columns={[
					{ title: '', width: '40px', render: icon },
					{ title: 'Time', width: '80px', render: time },
					{ title: 'Result', width: '110px', render: result },
					{ title: 'Reason', width: '100%', render: reason }
				]}
			/>
		{/key}
	</Dialog>
{/if}
{#snippet reason(event: ScoreEvent)}
	<span class="text-primary/60">{event.reason}</span>
{/snippet}
