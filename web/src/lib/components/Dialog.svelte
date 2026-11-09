<script lang="ts">
	import { cn } from '$lib/utils.svelte';
	import { XIcon } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import Button from './Button.svelte';

	let {
		open = $bindable(false),
		title,
		description,
		class: className,
		children
	}: {
		open: boolean;
		title: string;
		description?: string;
		class?: string;
		children: Snippet;
	} = $props();

	let dialog: HTMLDialogElement | undefined = $state();

	$effect(() => {
		if (!dialog) return;
		if (open && !dialog.open) dialog.showModal();
		if (!open && dialog.open) dialog.close();
	});
</script>

<dialog
	bind:this={dialog}
	onclose={() => (open = false)}
	onclick={(e) => {
		if (e.target === dialog) open = false;
	}}
	class={cn(
		'overflow-scroll-hidden m-auto w-full max-w-[calc(100%-2rem)] animate-in rounded-xl bg-background p-0 text-sm text-foreground duration-100 fade-in-0 zoom-in-95 backdrop:bg-black/10 backdrop:backdrop-blur-xs sm:max-w-md',
		className
	)}
>
	<div class="relative flex flex-col gap-6 rounded-xl glass p-6">
		<div class="flex flex-col gap-2 pr-8">
			<h1 class="text-2xl text-primary/90">{title}</h1>
			{#if description}
				<p class="text-sm text-primary/60">{description}</p>
			{/if}
		</div>

		{@render children()}

		<Button variant="ghost" class="absolute top-4 right-4" aria-label="Close" onclick={() => (open = false)}>
			<XIcon />
		</Button>
	</div>
</dialog>
