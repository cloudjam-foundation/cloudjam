<script lang="ts">
	import type { Component } from 'svelte';
	import { Button } from '$lib/components/shad/ui/button';
	import type { WithElementRef } from '$lib/utils';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		icon,
		text,
		action,
		quantity = $bindable(),
		disabled,
		class: className
	}: WithElementRef<HTMLButtonAttributes> & {
		icon?: Component;
		text: string;
		action: (() => Promise<void>) | (() => void);
		quantity: number;
	} = $props();
</script>

<Button
	data-rybbit-event={`bulk-${text.replace(/\s+(\w)/g, (_, c) => c.toUpperCase()).replace(/^\w/, (c) => c.toLowerCase())}`}
	class={className}
	{disabled}
	onclick={async () => {
		if (action) await action();
	}}
>
	{#if icon}
		{@const Icon = icon}
		<Icon class="mr-2" />
	{/if}
	{text}
	{#if quantity > 0}
		({quantity})
	{/if}
</Button>
