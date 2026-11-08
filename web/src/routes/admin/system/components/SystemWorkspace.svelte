<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import { ChevronLeftIcon } from '@lucide/svelte';
	import CostSection from './CostSection.svelte';
	import LogSection from './LogSection.svelte';
	import MetricSection from './MetricSection.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	let { close }: { close: () => void } = $props();

	const tabs = ['logs', 'metrics', 'cost'] as const;
	type Tab = (typeof tabs)[number];

	function isTab(value: string): value is Tab {
		return tabs.includes(value as Tab);
	}

	let tab: Tab = $state('logs');

	onMount(() => {
		const hash = page.url.hash.slice(1);
		if (isTab(hash)) tab = hash;
	});

	function selectTab(name: Tab) {
		tab = name;
		goto(`#${name}`, { state: page.state, shallow: true, replace: true });
	}
</script>

<div class="flex w-full flex-col gap-4">
	<div class="flex flex-row items-center gap-2">
		<Button variant="ghost" onclick={() => close()}>
			<ChevronLeftIcon />
		</Button>
		<h1 class="text-3xl opacity-80">System</h1>
	</div>

	<div class="flex flex-row gap-2 border-b">
		{#each tabs as name (name)}
			<button
				type="button"
				class="cursor-pointer border-b-2 px-4 py-2 capitalize transition-all duration-200 {tab === name
					? 'border-primary'
					: 'border-transparent opacity-60 hover:opacity-100'}"
				onclick={() => selectTab(name)}
			>
				{name}
			</button>
		{/each}
	</div>

	{#if tab === 'logs'}
		<LogSection />
	{:else if tab === 'metrics'}
		<MetricSection />
	{:else if tab === 'cost'}
		<CostSection />
	{/if}
</div>
