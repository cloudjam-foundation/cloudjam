<!-- ported from shad (eventually we should make this simpler and better) -->
<script lang="ts">
	import Spinner from '$lib/components/Spinner.svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { cn } from 'tailwind-variants';

	type Variant = 'primary' | 'ghost' | 'jam' | 'destructive';
	type ButtonState = 'idle' | 'success' | 'failed';

	interface Props extends HTMLButtonAttributes {
		variant: Variant;
		hint?: string;
		processing?: boolean;
		class?: string;
		children: Snippet;
		onclick: () => void | Promise<void>;
	}

	const variants: Record<Variant, string> = {
		primary: 'glass text-primary/80 hover:text-primary',
		ghost: 'hover:bg-primary/5 text-primary/80 hover:text-primary',
		jam: 'jam-glass text-jam/60 hover:text-jam/80',
		destructive: 'destructive-glass text-destructive/60 hover:text-destructive/80'
	};

	let { processing, hint, variant, class: className, children, onclick, ...restProps }: Props = $props();

	const buttonColorByState: Record<ButtonState, string> = {
		idle: '',
		success: 'bg-green-600 hover:bg-green-700 text-white',
		failed: 'bg-red-600 hover:bg-red-700 text-white'
	};
	let buttonState: ButtonState = $state('idle');
</script>

<button
	disabled={processing}
	class={cn(
		'relative flex cursor-pointer items-center justify-center rounded-md px-3 py-1.5 transition-all active:translate-y-px ',
		variants[variant],
		buttonColorByState[buttonState],
		className
	)}
	{...restProps}
	onclick={async () => {
		processing = true;
		try {
			await onclick();
			buttonState = 'success';
		} catch (e) {
			console.error(e);
			buttonState = 'failed';
		} finally {
			processing = false;
			setTimeout(() => (buttonState = 'idle'), 500);
		}
	}}
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
