<script lang="ts">
	import Dialog from '$lib/components/Dialog.svelte';
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(),
		title,
		description,
		badges,
		body
	}: {
		open: boolean;
		title: string;
		description?: string;
		badges: Snippet;
		body: Snippet;
	} = $props();
</script>

<Dialog bind:open {title} {description} class="sm:max-w-[90vw]">
	<div class="flex flex-row flex-wrap gap-2">
		{@render badges()}
	</div>

	{@render body()}
</Dialog>
