<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Markdown from '$lib/components/custom/Markdown.svelte';
	import * as Alert from '$lib/components/shad/alert';
	import { Badge } from '$lib/components/shad/badge';
	import { Button } from '$lib/components/shad/button';
	import * as Card from '$lib/components/shad/card';
	import { Separator } from '$lib/components/shad/separator';
	import Spinner from '$lib/components/shad/spinner/spinner.svelte';
	import * as Table from '$lib/components/shad/table';
	import {
		CredentialsRequestSchema,
		StartRequestSchema,
		UncoverClueRequestSchema
	} from '$lib/sdk/v1/play/challenge/challenge_pb';
	import { ScoreType, ScoreTypeSchema, type Challenge } from '$lib/sdk/v1/play/challenge_pb';
	import { jpegDataURL } from '$lib/utils';
	import { create, enumToJson } from '@bufbuild/protobuf';
	import { timestampDate } from '@bufbuild/protobuf/wkt';
	import { BadgeCheckIcon, KeyRoundIcon, SquareArrowOutUpRightIcon } from '@lucide/svelte';
	import AlertCircleIcon from '@lucide/svelte/icons/alert-circle';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import LightbulbIcon from '@lucide/svelte/icons/lightbulb';
	import PlayIcon from '@lucide/svelte/icons/play';

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
	let hasScoreReasons = $derived(challenge.scoreEvents.some((event) => event.reason));
	const scoreTypeName = (type: ScoreType) =>
		type === ScoreType.Unspecified ? '' : enumToJson(ScoreTypeSchema, type)?.toString().toLowerCase();

	let adhsTimeout: ReturnType<typeof setTimeout>;
	let adhsSpinner: boolean = $state(false);
</script>

<Card.Root class="w-full">
	<Card.Header>
		<Card.Title class="text-2xl">
			{challenge.title || 'Not started yet'}
			{#if !challenge.ready && challenge.title}
				<Badge variant="default">
					<Spinner />
					Provisioning Challenge
				</Badge>
			{/if}
			{#if adhsSpinner}
				<Badge variant="outline">
					<Spinner />
					Checking Progress
				</Badge>
			{/if}
		</Card.Title>
		<Card.Description>
			<Badge variant="secondary">
				score: {challenge.scoreEvents.reduce((sum, event) => sum + event.change, 0)}
			</Badge>
			{#each challenge.scores as score (score.type)}
				<Badge variant="outline">{scoreTypeName(score.type)}: {score.value} / {score.maximum}</Badge>
			{/each}
			{#if !challenge.title}
				<Badge variant="default">not started yet</Badge>
			{/if}
		</Card.Description>
		<Card.Action class="flex flex-row gap-2">
			<Button
				variant="outline"
				class="cursor-pointer"
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
				variant="outline"
				class="cursor-pointer"
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
				variant="outline"
				class="cursor-pointer"
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
		</Card.Action>
	</Card.Header>
	<Card.Content class="flex flex-col gap-6">
		{#if startState.error || credsState.error}
			<Alert.Root variant="destructive">
				<AlertCircleIcon />
				<Alert.Title>Action failed</Alert.Title>
				<Alert.Description>{startState.error || credsState.error}</Alert.Description>
			</Alert.Root>
		{/if}

		{#if credentials}
			<Alert.Root>
				<Alert.Title class="flex flex-row items-center gap-3">
					Account credentials
					<Button
						variant="outline"
						class="cursor-pointer"
						onclick={() =>
							navigator.clipboard.writeText(
								parsedCredentials.env.length
									? parsedCredentials.env.map(([name, value]) => `export ${name}=${value}`).join('\n')
									: credentials
							)}
					>
						<CopyIcon /> Copy as environment
					</Button>
					<Button variant="secondary" class="cursor-pointer" href={parsedCredentials.url}>
						<SquareArrowOutUpRightIcon /> Open AWS Console
					</Button>
				</Alert.Title>
			</Alert.Root>
		{/if}

		{#if challenge.error}
			<Alert.Root variant="destructive">
				<AlertCircleIcon />
				<Alert.Title>Challenge is malfunctioning. Please contact an administrator!</Alert.Title>
				<Alert.Description class="font-mono text-xs break-all">{challenge.error}</Alert.Description>
			</Alert.Root>
		{/if}

		{#if challenge.description.length}
			<div class="flex flex-col gap-2">
				<Card.Title>Briefing</Card.Title>
				<Markdown source={challenge.description.join('\n\n')} />
			</div>
		{/if}

		{#if Object.keys(challenge.diagrams).length}
			<Separator />

			<div class="flex flex-col gap-3">
				<Card.Title>Infrastructure</Card.Title>
				{#each Object.entries(challenge.diagrams) as [name, diagram] (name)}
					<figure class="flex flex-col gap-2">
						<img class="max-h-96 rounded-md object-contain" src={jpegDataURL(diagram)} alt={name} />
						<figcaption class="text-center text-xs text-muted-foreground">{name}</figcaption>
					</figure>
				{/each}
			</div>
		{/if}

		{#if Object.keys(challenge.assets).length}
			<Separator />

			<div class="flex flex-col gap-2">
				<Card.Title>Assets</Card.Title>
				{#each Object.entries(challenge.assets) as [name, asset] (name)}
					<div class="flex flex-row items-center gap-2 text-sm">
						<span class="font-medium">{name}</span>
						<span class="text-muted-foreground font-mono text-xs break-all">{asset}</span>
					</div>
				{/each}
			</div>
		{/if}

		{#if Object.keys(challenge.clues).length}
			<Separator />

			<div class="flex flex-col gap-2">
				<Card.Title>Clues</Card.Title>
				<p class="text-muted-foreground text-sm">Uncovering a clue usually costs points.</p>
				<Separator />
				{#each Object.entries(challenge.clues) as [name, text] (name)}
					<div class="flex flex-col justify-center gap-2 text-sm">
						<span class="font-medium">{name}</span>
						{#if text !== '<hidden>'}
							<span class="text-muted-foreground">{text}</span>
						{:else}
							<Button
								variant="outline"
								class="cursor-pointer"
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
								<LightbulbIcon /> Uncover
								({challenge.cluePrices[name] === undefined ? 'price unavailable' : `${-challenge.cluePrices[name]} points`})
							</Button>
						{/if}
					</div>
				{/each}
				{#if clueState.error}
					<p class="text-destructive text-xs">{clueState.error}</p>
				{/if}
			</div>
		{/if}

		{#if challenge.scoreEvents.length}
			<Separator />

			<div class="flex flex-col gap-2">
				<Card.Title>Activity</Card.Title>
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head>Time</Table.Head>
							<Table.Head>Event</Table.Head>
							<Table.Head>Type</Table.Head>
							<Table.Head>Result</Table.Head>
							<Table.Head>Change</Table.Head>
							{#if hasScoreReasons}<Table.Head>Explanation</Table.Head>{/if}
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each challenge.scoreEvents as event, index (index)}
							<Table.Row>
								<Table.Cell>{event.timestamp ? timestampDate(event.timestamp).toLocaleString() : ''}</Table.Cell>
								<Table.Cell>{event.text}</Table.Cell>
								<Table.Cell>{scoreTypeName(event.type)}</Table.Cell>
								<Table.Cell>{event.maximum ? `${event.score} / ${event.maximum}` : ''}</Table.Cell>
								<Table.Cell>{event.change}</Table.Cell>
								{#if hasScoreReasons}<Table.Cell>{event.reason}</Table.Cell>{/if}
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			</div>
		{/if}
	</Card.Content>
</Card.Root>
