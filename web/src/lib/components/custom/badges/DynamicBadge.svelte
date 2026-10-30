<!-- Dynamic badge with a freeform body -->
<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

	type Variant = 'primary' | 'filled' | 'jam' | 'destructive';

	const variants: Record<Variant, string> = {
		primary: 'glass text-primary/80',
		filled: 'glass text-foreground bg-background',
		jam: 'jam-glass text-jam/60',
		destructive: 'destructive-glass text-destructive/60'
	};

	let { variant, class: classNames, children }: { variant: Variant; class?: string; children: Snippet } = $props();
</script>

<div
	class={cn(
		'flex w-max flex-row items-center gap-1 rounded-full px-3 py-[0.2rem] text-center text-xs',
		variants[variant],
		classNames
	)}
>
	{@render children()}
</div>
