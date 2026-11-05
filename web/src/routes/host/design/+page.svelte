<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { toDigest } from '$lib/utils.svelte';
	import { ListRequestSchema } from '$lib/sdk/v1/cloud/definition/definition_pb';
	import type { Definition } from '$lib/sdk/v1/cloud/definition_pb';
	import {
		GetRequestSchema as GetProviderRequestSchema,
		ListRequestSchema as ListProviderRequestSchema
	} from '$lib/sdk/v1/cloud/provider/provider_pb';
	import type { Provider } from '$lib/sdk/v1/cloud/provider_pb';
	import { create } from '@bufbuild/protobuf';
	import { WandSparklesIcon } from '@lucide/svelte';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import { onMount } from 'svelte';
	import CreateDefinition from './CreateDefinition.svelte';
	import DefinitionPanel from './DefinitionPanel.svelte';
	import Help from './Help.svelte';

	let providerId: string = $state('');
	let provider: Provider | undefined = $state();
	let providerState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	$effect(() => {
		if (!providerId) {
			provider = undefined;
			return;
		}
		Submit(async () => {
			provider = (await Glue.provider.get(create(GetProviderRequestSchema, { id: providerId }))).provider;
		}, providerState);
	});

	let providerSuggestions: Provider[] = $state([]);
	let providerSuggestionsState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	onMount(() => {
		Submit(async () => {
			providerSuggestions = (await Glue.provider.list(create(ListProviderRequestSchema, { limit: 100 }))).providers;
		}, providerSuggestionsState);
	});

	let definitionsState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	// bumping the version remounts the table and therefore reloads the definitions.
	let version = $state(0);

	let selected: Definition | undefined = $state();
	let openSelected = $state(false);
</script>

<svelte:head>
	<title>Design | CloudJam</title>
	<meta property="og:title" content="Design | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => goto('/host')}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">Design</h1>
	</div>

	<p class="text-sm text-primary/60">Create, update and manage challenge definitions</p>

	<div class="flex flex-row items-center gap-4">
		<OptionalSelect
			bind:value={providerId}
			placeholder="Select Provider"
			suggestions={providerSuggestions.map((provider) => ({ id: provider.id, title: provider.name }))}
			class="w-64"
		/>

		{#if provider}
			<a href={`/provider/${provider.id}`}>
				<StaticBadge variant="primary" text={provider.id} />
			</a>
		{/if}

		<Help class="ml-auto" />
	</div>

	{#if provider}
		<CreateDefinition {provider} refresh={() => version++} />

		<div class="flex w-full flex-col gap-4">
			<h2 class="text-xl opacity-80">Challenge Definitions on the Provider</h2>
			{#if definitionsState.forbidden}
				<Alert title="Permission Denied" message="You are not allowed to list the definitions of this provider" />
			{:else}
				{#key `${providerId}-${version}`}
					<DataTable
						class="max-h-[80vh]"
						limit={100}
						load={async (startAfter: Definition | undefined) => {
							const definitions: Definition[] = [];
							await Submit(async () => {
								const resp = await Glue.definition.list(
									create(ListRequestSchema, { providerId: providerId, limit: 100, startAfter: startAfter?.id })
								);
								definitions.push(...resp.definitions);
							}, definitionsState);
							return definitions;
						}}
						onRowClick={(definition) => {
							selected = definition;
							openSelected = true;
						}}
						loadState={definitionsState}
						columns={[
							{ title: 'Name', width: '25%', render: renderName },
							{ title: 'Version', width: '15%', render: renderVersion },
							{ title: 'Description', width: '30%', render: renderDescription },
							{ title: 'Digest', width: '30%', render: renderDigest }
						]}
					/>
				{/key}
				{#snippet renderName(definition: Definition)}
					{definition.name}
				{/snippet}
				{#snippet renderVersion(definition: Definition)}
					{definition.version}
				{/snippet}
				{#snippet renderDescription(definition: Definition)}
					<span class="text-primary/60">{definition.description}</span>
				{/snippet}
				{#snippet renderDigest(definition: Definition)}
					<span class="font-mono text-xs">{toDigest(definition.hash)}</span>
				{/snippet}
			{/if}

			{#if selected}
				<DefinitionPanel
					bind:open={openSelected}
					definition={selected}
					refresh={() => {
						openSelected = false;
						version++;
					}}
				/>
			{/if}
		</div>
	{:else}
		<div class="flex h-[60vh] w-full flex-col items-center justify-center gap-8">
			<WandSparklesIcon class="h-48 w-48 text-muted" />
			<h1 class="text-4xl font-bold text-muted">The Canvas is yours</h1>
		</div>
	{/if}
</div>
