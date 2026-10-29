<!-- button used to automatically support some async action feedback -->
<script lang="ts">
	import Spinner from '$lib/components/shad/spinner/spinner.svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { cn } from 'tailwind-variants';

	type Variant = 'primary' | 'secondary' | 'destructive';

	interface Props extends HTMLButtonAttributes {
		processing?: boolean;
		hint?: string;
		variant: Variant;
		class?: string;
		children: Snippet;
	}

	const variants: Record<Variant, string> = {
		primary: '',
		secondary: '',
		destructive: ''
	};

	let { processing, hint, variant, class: className, children, ...restProps }: Props = $props();
</script>

<button
	disabled={processing}
	class={cn(
		'relative flex cursor-pointer items-center justify-center overflow-hidden rounded-md glass px-3 py-2 transition-all active:translate-y-px ',
		variants[variant],
		className
	)}
	{...restProps}
>
	<span
		class="absolute top-1/2 left-1/2 flex -translate-x-1/2 -translate-y-1/2 shimmer flex-row items-center justify-center gap-2 overflow-hidden transition-all shimmer-color-muted-foreground {processing
			? ''
			: 'opacity-0'}"
	>
		{#if hint}
			{hint}
		{:else}
			<Spinner />
		{/if}
	</span>
	<span class="flex flex-row items-center justify-center gap-1 {processing ? 'opacity-0' : ''}">
		{@render children()}
	</span>
</button>
