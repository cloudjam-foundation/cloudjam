<script lang="ts">
	import type { Snippet } from 'svelte';

	let { caption, children }: { caption: string; children: Snippet } = $props();
</script>

<span class="group/tooltip relative inline-flex">
	{@render children()}
	<span
		role="tooltip"
		class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1.5 w-max max-w-50 -translate-x-1/2 rounded-md bg-foreground px-3 py-1.5 text-center text-xs text-background opacity-0 transition-opacity delay-75 duration-100 group-focus-within/tooltip:opacity-100 group-hover/tooltip:opacity-100"
	>
		{caption}
	</span>
</span>
