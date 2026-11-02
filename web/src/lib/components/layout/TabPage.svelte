<script lang="ts">
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import type { Snippet } from 'svelte';
	import type { SubmitState } from '$lib';
	import Button from '$lib/components/custom/buttons/Button.svelte';
	import Alert from '$lib/components/custom/Alert.svelte';

	let {
		title,
		headers,
		back,
		dataState,
		tabs,
		tabPages
	}: {
		title: string;
		headers: Snippet[];
		back: () => {};
		dataState: SubmitState;
		tabs: string[];
		tabPages: Snippet[];
	} = $props();

	let selected: number = $state(0);
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => back()}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">{title}</h1>
		{#each headers as header}
			{@render header()}
		{/each}
	</div>

	<div class="flex flex-row gap-2 border-b">
		{#each tabs as name, i (name)}
			<button
				type="button"
				class="cursor-pointer border-b-2 px-4 py-2 capitalize transition-all duration-200 {selected === i
					? 'border-primary'
					: 'border-transparent opacity-60 hover:opacity-100'}"
				onclick={() => (selected = i)}
			>
				{name}
			</button>
		{/each}
	</div>

	{#if dataState.forbidden}
		<Alert title="Permission Denied" message="You are not allowed to open this section" />
	{:else}
		{@render tabPages[selected]()}
	{/if}

	{#if dataState.error}
		<Alert title="Something went wrong" message={dataState.error} />
	{/if}
</div>
