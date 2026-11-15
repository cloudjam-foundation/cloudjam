<script lang="ts">
	import { cn } from 'tailwind-variants';
	import type { HTMLAttributes } from 'svelte/elements';
	let { class: className, ...restProps }: HTMLAttributes<HTMLDivElement> = $props();
</script>

<div data-slot="skeleton" class={cn('animate-pulse rounded-md bg-muted', className)} {...restProps}></div>
