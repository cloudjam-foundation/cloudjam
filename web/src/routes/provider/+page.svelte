<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, Submit, type SubmitState } from '$lib';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import LinkButton from '$lib/components/LinkButton.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { ListRequestSchema } from '$lib/sdk/v1/cloud/provider/provider_pb';
	import { ProviderType, type Provider } from '$lib/sdk/v1/cloud/provider_pb';
	import { create } from '@bufbuild/protobuf';
	import PlusIcon from '@lucide/svelte/icons/plus';

	let id = $state('');

	let providersState: SubmitState = $state({ error: '', loading: true, forbidden: false });
</script>

<svelte:head>
	<title>Provider | CloudJam</title>
	<meta property="og:title" content="Provider | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

<div class="flex h-11/12 w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<h1 class="text-3xl opacity-80">Providers</h1>
		<LinkButton class="ml-auto" href="/provider/new/">
			<PlusIcon /> Add Provider
		</LinkButton>
	</div>

	{#if providersState.forbidden}
		<div class="flex h-full w-full items-center justify-center">
			<Card>
				<div class="flex flex-col items-start gap-1">
					<h1 class="text-2xl font-bold text-primary/80">Find Provider</h1>
					<p class="text-primary/60">You are not allowed to list providers, find the one you need by its id</p>
				</div>
				<form class="flex w-full flex-row items-center gap-2" onsubmit={() => goto(`/provider/${id.trim()}`)}>
					<Input class="w-full" bind:value={id} placeholder="Provider ID" />
					<Button type="submit" variant="primary" disabled={!id.trim()}>Open</Button>
				</form>
			</Card>
		</div>
	{:else}
		<div class="h-full rounded-sm p-2">
			<DataTable
				class="max-h-full"
				load={async (startAfter: Provider | undefined) => {
					const providers: Provider[] = [];
					await Submit(async () => {
						const resp = await Glue.provider.list(
							create(ListRequestSchema, { limit: 100, startAfter: startAfter?.id })
						);
						providers.push(...resp.providers);
					}, providersState);
					return providers;
				}}
				onRowClick={(provider) => {
					goto(`/provider/${provider.id}`);
				}}
				loadState={providersState}
				columns={[
					{ title: 'Type', width: '90px', render: renderType },
					{ title: 'Name', width: '20%', render: renderName },
					{ title: 'Description', width: '35%', render: renderDescription },
					{ title: 'Scope', width: '15%', render: renderScope },
					{ title: 'Regions', width: '30%', render: renderRegions }
				]}
			/>
			{#snippet renderType(provider: Provider)}
				<StaticBadge variant="primary" text={ProviderType[provider.type]} />
			{/snippet}
			{#snippet renderName(provider: Provider)}
				{provider.name}
			{/snippet}
			{#snippet renderDescription(provider: Provider)}
				<span class="text-primary/60">{provider.description}</span>
			{/snippet}
			{#snippet renderScope(provider: Provider)}
				{provider.scope}
			{/snippet}
			{#snippet renderRegions(provider: Provider)}
				<div class="flex flex-row items-center gap-1">
					{#each provider.regions as region (region)}
						<StaticBadge variant="primary" text={region} />
					{/each}
				</div>
			{/snippet}
		</div>
	{/if}
</div>
