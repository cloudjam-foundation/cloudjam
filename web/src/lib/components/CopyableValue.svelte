<script lang="ts">
	import { CopyIcon, CheckIcon } from '@lucide/svelte';
	import Button from '$lib/components/Button.svelte';

	let {
		value = $bindable(),
		descriptor = $bindable()
	}: {
		value: string | number | undefined;
		descriptor: string;
	} = $props();

	let copied = $state(false);
	let timeout: ReturnType<typeof setTimeout> | undefined;

	async function copy() {
		try {
			// Why is this async?
			await navigator.clipboard.writeText(value ? value.toString() : '');
			copied = true;
			clearTimeout(timeout);
			timeout = setTimeout(() => (copied = false), 750);
		} catch (error) {
			console.error(`Unable to copy ${descriptor}`, error);
		}
	}
</script>

<Button variant="primary" class="ml-auto font-mono text-xs" title={`Copy ${descriptor}`} onclick={copy}>
	{descriptor}:
	{value}
	{#if copied}
		<CheckIcon />
	{:else}
		<CopyIcon />
	{/if}
</Button>
