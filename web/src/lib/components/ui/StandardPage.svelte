<script lang="ts">
	import type { PageProps } from '$lib/components/ui/pageProps';
	import { ChevronLeftIcon } from '@lucide/svelte';
	import Button from '$lib/components/buttons/Button.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';
	import type { Snippet } from 'svelte';

	let {
		title,
		headers,
		back,
		dataState = $bindable(),
		children = $bindable()
	}: PageProps & { children: Snippet } = $props();
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		{#if back}
			<Button variant="ghost" onclick={async () => await back()}>
				<ChevronLeftIcon />
			</Button>
		{/if}
		<h1 class="text-3xl opacity-80">{title}</h1>
		{#each headers as header (header)}
			{@render header()}
		{/each}
	</div>

	{#if dataState?.forbidden}
		<Alert title="Permission Denied" message="You are not allowed to view this page." />
	{:else if !dataState || (dataState && !dataState?.loading)}
		{@render children()}
	{/if}

	{#if dataState?.error}
		<Alert title="Something went wrong" message={dataState?.error ?? ''} />
	{/if}
</div>
