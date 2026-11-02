<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import SidebarTrigger from './SidebarTrigger.svelte';
	import { setSidebar } from './state.svelte';

	let {
		header,
		footer,
		children,
		class: className
	}: { header?: Snippet; footer?: Snippet; children: Snippet; class?: string } = $props();

	const sidebar = setSidebar();
</script>

<svelte:window onkeydown={sidebar.handleShortcut} />

{#if sidebar.isMobile}
	<span class="absolute top-2 left-2 z-30">
		<SidebarTrigger />
	</span>
	{#if sidebar.openMobile}
		<!-- svelte-ignore a11y_no_static_element_interactions, a11y_click_events_have_key_events -->
		<div class="fixed inset-0 z-40 bg-black/10 backdrop-blur-xs" onclick={() => sidebar.setOpen(false)}></div>
	{/if}
	<aside
		class={cn(
			'fixed inset-y-0 left-0 z-50 flex w-72 flex-col transition-transform duration-200 ease-in-out',
			sidebar.openMobile ? 'translate-x-0' : '-translate-x-full',
			className
		)}
	>
		{@render inner()}
	</aside>
{:else}
	<aside
		class={cn(
			'sticky top-0 flex h-dvh shrink-0 flex-col transition-[width] duration-200 ease-linear',
			sidebar.open ? 'w-64' : 'w-12',
			className
		)}
	>
		{@render inner()}
	</aside>
{/if}

{#snippet inner()}
	{#if header}
		<div class="flex flex-col gap-2 p-2">
			{@render header()}
		</div>
	{/if}
	<div class={cn('flex min-h-0 flex-1 flex-col gap-2', sidebar.expanded ? 'overflow-y-auto' : 'overflow-hidden')}>
		{@render children()}
	</div>
	{#if footer}
		<div class="flex flex-col gap-2 p-2">
			{@render footer()}
		</div>
	{/if}
{/snippet}
