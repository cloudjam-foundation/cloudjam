<script lang="ts">
	import type { Component, Snippet } from 'svelte';
	import * as Empty from '$lib/components/shad/ui/empty';

	/** What a table shows when there's nothing in it yet */
	let {
		icon: Icon,
		title,
		description,
		action
	}: { icon: Component; title: string; description?: string; action?: Snippet } = $props();
</script>

<Empty.Root>
	<Empty.Header>
		<Empty.Media variant="icon"><Icon /></Empty.Media>
		<Empty.Title>{title}</Empty.Title>
		{#if description}
			<Empty.Description>{description}</Empty.Description>
		{/if}
	</Empty.Header>
	{#if action}
		<Empty.Content>{@render action()}</Empty.Content>
	{/if}
</Empty.Root>
