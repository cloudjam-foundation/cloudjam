<script lang="ts">
	import { cn } from '$lib/utils';
	import { OctagonAlert } from '@lucide/svelte';

	let {
		title,
		message,
		animation,
		class: className
	}: {
		title?: string;
		message: string;
		animation?: boolean;
		class?: string;
	} = $props();
</script>

<div class={cn('flex flex-col items-start rounded-sm destructive-glass p-3', animation ? 'popup' : '', className)}>
	{#if title}
		<div class="flex flex-row items-center gap-2">
			<OctagonAlert size={20} class="text-destructive/60" />
			<h1 class="text-md font-semibold text-destructive/60">{title}</h1>
		</div>
	{/if}
	<p class="text-sm whitespace-pre-line text-destructive/40">{message}</p>
</div>

<style>
	.popup {
		animation: popup 0.2s ease forwards;
	}

	@keyframes popup {
		0% {
			opacity: 0;
			scale: 0.95;
		}
		80% {
			opacity: 1;
			scale: 1.03;
		}
		100% {
			scale: 1;
		}
	}
</style>
