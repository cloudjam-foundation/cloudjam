<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	let {
		class: className,
		children
	}: {
		class?: string;
		children: Snippet;
	} = $props();
</script>

<div class={cn('flex flex-col items-center gap-4 rounded-2xl glass p-7', className)}>
	{@render children()}
</div>
