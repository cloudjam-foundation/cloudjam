<script lang="ts">
	import { goto } from '$app/navigation';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import TabPage from '$lib/components/TabPage.svelte';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import UserPanel from './UserPanel.svelte';

	let { data } = $props();
</script>

<svelte:head>
	<title>{data.user?.username ?? 'User'} | CloudJam</title>
	<meta property="og:title" content="User | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

{#snippet privileged()}
	{#if data.user?.privileged}
		<StaticBadge text="privileged" variant="jam" />
	{/if}
{/snippet}

{#snippet uuid()}
	<Button
		variant="primary"
		class="ml-auto font-mono text-xs"
		title="Copy user id"
		onclick={() => navigator.clipboard.writeText(data.user?.id ?? '')}
	>
		<CopyIcon />
		{data.user?.id}
	</Button>
{/snippet}

{#snippet overview()}
	{#if data.user}
		<UserPanel user={data.user} refresh={() => data.refresh()} deleted={() => goto('/admin/users')} />
	{/if}
{/snippet}

<TabPage
	title={data.user?.username ?? 'User'}
	headers={[privileged, uuid]}
	back={() => goto('/admin/users')}
	dataState={data.userState}
	tabs={['overview']}
	tabPages={[overview]}
/>
