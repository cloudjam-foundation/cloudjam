<script lang="ts">
	import { ProviderType } from '$lib/sdk/v1/cloud/provider_pb';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import AccountSection from './AccountSection.svelte';
	import DefinitionSection from './DefinitionSection.svelte';
	import ProviderPanel from './ProviderPanel.svelte';
	import { goto } from '$app/navigation';
	import TabPage from '$lib/components/TabPage.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';

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
		<StaticBadge text={ProviderType[data.provider.type]} variant="primary" />
	{/if}
{/snippet}

{#snippet uuid()}
	<Button
		variant={'primary'}
		class="ml-auto font-mono text-xs"
		title="Copy game id"
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
	headers={[type, uuid]}
	back={() => goto('/provider')}
	dataState={data.providerState}
	tabs={['overview', 'accounts', 'definitions']}
	tabPages={[overview, accounts, definitions]}
/>
