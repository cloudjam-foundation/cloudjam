<!-- button that acts as a link -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import { cn } from 'tailwind-variants';

	interface Props extends HTMLAnchorAttributes {
		href: string;
		class?: string;
		ghost?: boolean;
		children: Snippet;
	}

	let { href, class: className, ghost, children, ...restProps }: Props = $props();
</script>

<a
	{href}
	class={cn(
		'flex cursor-pointer flex-row items-center justify-center gap-1 overflow-hidden rounded-md px-3 py-1.5 text-primary/80 transition-all hover:text-primary active:translate-y-px',
		ghost ? 'hover:glass' : 'glass',
		className
	)}
	{...restProps}
>
	{@render children()}
</a>
