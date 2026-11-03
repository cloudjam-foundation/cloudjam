<script lang="ts">
	import { isMacOS } from './';
	import SidebarTrigger from './SidebarTrigger.svelte';
	import { useSidebar } from './state.svelte';

	const sidebar = useSidebar();

	const kbd =
		'pointer-events-none inline-flex h-5 w-fit min-w-5 items-center justify-center rounded-sm bg-muted px-1 font-sans text-xs font-medium text-muted-foreground select-none';
</script>

{#if !sidebar.isMobile}
	<span class="ml-auto inline-flex items-center gap-1">
		{#if sidebar.open}
			<kbd class="inline-flex items-center gap-1">
				<kbd class={kbd}>{isMacOS() ? '⌘' : 'Ctrl'}</kbd>
				<span>+</span>
				<kbd class={kbd}>B</kbd>
			</kbd>
		{/if}
		<SidebarTrigger class="w-8" />
	</span>
{/if}
