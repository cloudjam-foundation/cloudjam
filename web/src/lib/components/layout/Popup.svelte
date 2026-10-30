<script lang="ts">
	import * as Dialog from '$lib/components/shad/dialog';
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

<Dialog.Root bind:open>
	<Dialog.Content class="flex w-full flex-col gap-6 sm:max-w-[90vw]">
		<Dialog.Header>
			<Dialog.Title class="text-2xl">{title}</Dialog.Title>
			{#if description}
				<Dialog.Description>{description}</Dialog.Description>
			{/if}
			<div class="flex flex-row flex-wrap gap-2">
				{@render badges()}
			</div>
		</Dialog.Header>

		{@render body()}
	</Dialog.Content>
</Dialog.Root>
