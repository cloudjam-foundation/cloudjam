<script lang="ts">
	import * as Alert from '$lib/components/shad/alert';
	import { Button } from '$lib/components/shad/button';
	import AlertCircleIcon from '@lucide/svelte/icons/alert-circle';
	import ChevronLeftIcon from '@lucide/svelte/icons/chevron-left';
	import type { Snippet } from 'svelte';
	import type { SubmitState } from '$lib';

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
		<Button variant="ghost" size="icon" class="cursor-pointer" onclick={() => back()}>
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
		<Alert.Root>
			<AlertCircleIcon />
			<Alert.Title>Permission Denied</Alert.Title>
			<Alert.Description>You are not allowed to open this provider</Alert.Description>
		</Alert.Root>
	{:else}
		{@render tabPages[selected]()}
	{/if}

	{#if dataState.error}
		<Alert.Root variant="destructive">
			<AlertCircleIcon />
			<Alert.Title>Something went wrong</Alert.Title>
			<Alert.Description>{dataState.error}</Alert.Description>
		</Alert.Root>
	{/if}
</div>
