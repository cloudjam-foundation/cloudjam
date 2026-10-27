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
		'flex cursor-pointer flex-row items-center justify-center gap-2 rounded-md px-3 py-1.5 transition-all active:translate-y-px',
		ghost ? 'hover:glass' : 'glass',
		className
	)}
	{...restProps}
>
	{@render children()}
</a>
