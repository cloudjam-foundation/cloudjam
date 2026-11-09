<script lang="ts">
	import { ChevronLeftIcon } from '@lucide/svelte';
	import { onMount, type Snippet } from 'svelte';
	import type { SubmitState } from '$lib';
	import Button from '$lib/components/Button.svelte';
	import Alert from '$lib/components/Alert.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';

	let {
		title,
		headers,
		back,
		dataState,
		tabs,
		tabPages
	}: {
		title: string;
		headers?: Snippet[];
		back: () => void | Promise<void>;
		dataState?: SubmitState;
		tabs: string[];
		tabPages: Snippet[];
	} = $props();

	type Tab = (typeof tabs)[number];

	function isTab(value: string): value is Tab {
		return tabs.includes(value as Tab);
	}

	onMount(() => {
		const hash = page.url.hash.slice(1);
		if (isTab(hash)) selected = tabs.indexOf(hash);
	});

	function selectTab(name: Tab) {
		selected = tabs.indexOf(name);
		goto(`#${name}`, { state: page.state, shallow: true, replace: true });
	}

	let selected: number = $state(0);
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => back()}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">{title}</h1>
		{#each headers as header (header)}
			{@render header()}
		{/each}
	</div>

	<div class="flex flex-row gap-2 border-b">
		{#each tabs as name, i (i)}
			<button
				type="button"
				class="cursor-pointer border-b-2 px-4 py-2 capitalize transition-all duration-200 {selected === i
					? 'border-primary'
					: 'border-transparent opacity-60 hover:opacity-100'}"
				onclick={() => selectTab(name)}
			>
				{name}
			</button>
		{/each}
	</div>

	{#if dataState?.forbidden}
		<Alert title="Permission Denied" message="You are not allowed to open this section" />
	{:else}
		{@render tabPages[selected]()}
	{/if}

	{#if dataState?.error}
		<Alert title="Something went wrong" message={dataState?.error ?? ''} />
	{/if}
</div>
