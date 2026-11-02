<script lang="ts">
	import { goto } from '$app/navigation';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import TabPage from '$lib/components/TabPage.svelte';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import RolePanel from './RolePanel.svelte';

	let { data } = $props();
</script>

<svelte:head>
	<title>{data.role?.name ?? 'Role'} | CloudJam</title>
	<meta property="og:title" content="Role | CloudJam" />
	<meta property="og:type" content="website" />
	<meta property="og:image" content="/favicon.png" />
</svelte:head>

{#snippet builtin()}
	{#if data.role?.builtin}
		<StaticBadge text="builtin" variant="jam" />
	{/if}
{/snippet}

{#snippet uuid()}
	<Button
		variant="primary"
		class="ml-auto font-mono text-xs"
		title="Copy role id"
		onclick={() => navigator.clipboard.writeText(data.role?.id ?? '')}
	>
		<CopyIcon />
		{data.role?.id}
	</Button>
{/snippet}

{#snippet overview()}
	{#if data.role}
		{#key data.role.id}
			<RolePanel role={data.role} refresh={() => data.refresh()} deleted={() => goto('/admin/roles')} />
		{/key}
	{/if}
{/snippet}

<TabPage
	title={data.role?.name ?? 'Role'}
	headers={[builtin, uuid]}
	back={() => goto('/admin/roles')}
	dataState={data.roleState}
	tabs={['overview']}
	tabPages={[overview]}
/>
