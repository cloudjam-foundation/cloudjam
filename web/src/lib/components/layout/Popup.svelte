<script lang="ts">
	import * as Alert from '$lib/components/shad/alert';
	import * as Dialog from '$lib/components/shad/dialog';
	import AlertCircleIcon from '@lucide/svelte/icons/alert-circle';
	import type { Snippet } from 'svelte';

	let {
		open = $bindable(),
		title,
		description,
		badges,
		body,
		error
	}: {
		open: boolean;
		title: string;
		description?: string;
		badges: Snippet;
		body: Snippet;
		error: string;
	} = $props();
</script>

<Dialog.Root bind:open>
	<Dialog.Content class="flex w-full flex-col gap-6 sm:max-w-[90vw]">
		{#if error}
			<Alert.Root variant="destructive">
				<AlertCircleIcon />
				<Alert.Title>Error occured</Alert.Title>
				<Alert.Description class="font-mono text-xs break-all">{error}</Alert.Description>
			</Alert.Root>
		{/if}
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
