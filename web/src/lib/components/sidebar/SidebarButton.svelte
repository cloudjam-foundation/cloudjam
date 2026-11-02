<!-- Menu entry of the sidebar, rendered as a link or a button -->
<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { useSidebar } from './state.svelte';

	interface Props extends HTMLButtonAttributes {
		href?: string;
		size?: 'default' | 'lg';
		class?: string;
		children: Snippet;
	}

	let { href, size = 'default', class: className, children, ...restProps }: Props = $props();

	const sidebar = useSidebar();

	const classes = $derived(
		cn(
			'flex w-full cursor-pointer items-center gap-2 overflow-hidden rounded-md p-2 text-left text-sm text-primary/80 transition-[width,height,padding] outline-hidden hover:bg-foreground/5 hover:text-primary focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0 [&>span:last-child]:truncate',
			size === 'lg' ? 'h-12' : 'h-8',
			sidebar.expanded ? '' : 'size-8! p-2!',
			className
		)
	);
</script>

{#if href}
	<a {href} class={classes}>
		{@render children()}
	</a>
{:else}
	<button type="button" class={classes} {...restProps}>
		{@render children()}
	</button>
{/if}
