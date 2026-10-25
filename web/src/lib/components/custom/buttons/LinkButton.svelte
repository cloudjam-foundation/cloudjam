<!-- button that acts as a link -->
<script lang="ts">
	import { SquareArrowOutUpRight } from '@lucide/svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import { cn } from 'tailwind-variants';

	interface Props extends HTMLAnchorAttributes {
		href: string;
		text?: string;
		class?: string;
	}

	let { href, text, class: className, ...restProps }: Props = $props();
</script>

<a
	{href}
	class={cn(
		'flex cursor-pointer flex-row items-center justify-center gap-2 rounded-md bg-primary/5 transition-all active:translate-y-px active:scale-[0.99] ',
		text ? 'px-3 py-1.5 underline' : 'w-10 p-2',
		className
	)}
	{...restProps}
>
	{#if text}
		{text}
	{/if}
	<SquareArrowOutUpRight />
</a>
