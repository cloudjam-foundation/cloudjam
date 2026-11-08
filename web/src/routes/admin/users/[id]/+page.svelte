<script lang="ts">
	import { goto } from '$app/navigation';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import TabPage from '$lib/components/TabPage.svelte';
	import UserPanel from './components/UserPanel.svelte';
	import { resolve } from '$app/paths';
	import CopyableValue from '$lib/components/CopyableValue.svelte';

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
	<CopyableValue value={data.user?.id} descriptor="User ID" />
{/snippet}

{#snippet overview()}
	{#if data.user}
		<UserPanel user={data.user} refresh={() => data.refresh()} deleted={() => goto(resolve('/admin/users'))} />
	{/if}
{/snippet}

<TabPage
	title={data.user?.username ?? 'User'}
	headers={[privileged, uuid]}
	back={() => goto(resolve('/admin/users'))}
	dataState={data.userState}
	tabs={['overview']}
	tabPages={[overview]}
/>
