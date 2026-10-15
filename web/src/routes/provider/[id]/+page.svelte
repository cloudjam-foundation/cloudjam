<script lang="ts">
	import { page } from '$app/state';

	import { Glue, Submit, type SubmitState } from '$lib';
	import { Badge } from '$lib/components/shad/badge';
	import { Button } from '$lib/components/shad/button';
	import { ProviderType } from '$lib/sdk/v1/cloud/provider_pb';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import AccountSection from './AccountSection.svelte';
	import DefinitionSection from './DefinitionSection.svelte';
	import ProviderPanel from './ProviderPanel.svelte';
	import { goto } from '$app/navigation';
	import TabPage from '$lib/components/layout/TabPage.svelte';

	let { data } = $props();
</script>

<svelte:head>
	<title>{data.provider?.name.toUpperCase() ?? 'Provider'} | CloudJam</title>
	<meta property="og:title" content="Provider | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

{#snippet type()}
	{#if data.provider}
		<Badge variant="secondary">{ProviderType[data.provider.type]}</Badge>
	{/if}
{/snippet}

{#snippet scope()}
	{#if data.provider}
		<Badge variant="outline">scope: {data.provider.scope}</Badge>
	{/if}
{/snippet}

{#snippet uuid()}
	<Button
		variant="outline"
		class="ml-auto cursor-pointer font-mono text-xs"
		title="Copy provider id"
		onclick={() => navigator.clipboard.writeText(data.provider?.id ?? '')}
	>
		<CopyIcon />
		{data.provider?.id}
	</Button>
{/snippet}

{#snippet overview()}
	{#if data.provider}
		<ProviderPanel provider={data.provider} refresh={() => data.refresh()} deleted={() => goto('/provider')} />
	{/if}
{/snippet}

{#snippet accounts()}
	<AccountSection providerId={data.provider?.id ?? ''} />
{/snippet}

{#snippet definitions()}
	<DefinitionSection providerId={data.provider?.id ?? ''} />
{/snippet}

<TabPage
	title={data.provider?.name.toUpperCase() ?? 'Provider'}
	headers={[type, scope, uuid]}
	back={() => goto('/provider')}
	dataState={data.providerState}
	tabs={['overview', 'accounts', 'definitions']}
	tabPages={[overview, accounts, definitions]}
/>
