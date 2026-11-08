<script lang="ts">
	import GroupingRenderer from '$lib/components/sidebar/options/GroupingRenderer.svelte';
	import OptionRenderer from '$lib/components/sidebar/options/OptionRenderer.svelte';
	import type { sidebarSection } from '$lib/components/sidebar/sidebarOptions';
	import { useSidebar } from '$lib/components/sidebar/state.svelte';

	let { section }: { section: sidebarSection } = $props();

	const sidebar = useSidebar();
	const Icon = $derived(section.icon);
</script>

<div id={section.title} class="relative flex w-full min-w-0 flex-col p-2">
	<div
		class="flex h-8 shrink-0 items-center rounded-md px-2 text-xs font-medium text-muted-foreground transition-[margin,opacity] duration-200 ease-linear [&>svg]:size-4 [&>svg]:shrink-0 {sidebar.expanded
			? ''
			: '-mt-8 opacity-0'}"
	>
		<Icon class="mr-2 h-4 w-4" />
		{section.title}
	</div>
	<div class="w-full pl-4 text-sm">
		{#each section.items as item (item.title)}
			{#if 'contents' in item}
				<GroupingRenderer group={item} />
			{:else}
				<OptionRenderer option={item} />
			{/if}
		{/each}
	</div>
</div>
