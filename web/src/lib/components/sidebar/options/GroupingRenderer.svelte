<script lang="ts">
	import OptionRenderer from '$lib/components/sidebar/options/OptionRenderer.svelte';
	import SidebarButton from '$lib/components/sidebar/SidebarButton.svelte';
	import type { sidebarGroup } from '$lib/components/sidebar/sidebarOptions';
	import { useSidebar } from '$lib/components/sidebar/state.svelte';
	import { ChevronDownIcon } from '@lucide/svelte';

	let { group }: { group: sidebarGroup } = $props();

	const sidebar = useSidebar();

	let open = $state(true);
</script>

{#snippet options()}
	<ul
		class="mx-3.5 flex min-w-0 translate-x-px flex-col gap-1 border-l border-border px-2.5 py-0.5 {sidebar.expanded
			? ''
			: 'hidden'}"
	>
		{#each group.contents as item (item.link)}
			<OptionRenderer option={item} />
		{/each}
	</ul>
{/snippet}

<ul class="flex w-full min-w-0 flex-col gap-1">
	<li class="relative">
		{#if group.collapsible}
			<SidebarButton onclick={() => (open = !open)}>
				{@const Icon = group.icon}
				<Icon class="mr-2 h-4 w-4" />
				{group.title}
				<ChevronDownIcon
					class="ml-auto h-4 w-4 {open ? 'rotate-0' : '-rotate-180'} transition-all duration-250 ease-in-out"
				/>
			</SidebarButton>
			{#if open}
				{@render options()}
			{/if}
		{:else}
			<SidebarButton>
				{@const Icon = group.icon}
				<Icon class="mr-2 h-4 w-4" />
				<span>{group.title}</span>
			</SidebarButton>
			{@render options()}
		{/if}
	</li>
</ul>
