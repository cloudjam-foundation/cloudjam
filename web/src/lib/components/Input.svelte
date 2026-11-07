<script lang="ts">
	import { cn } from '$lib/utils.svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props extends HTMLInputAttributes {
		label?: string;
		validation?: string;
		class?: string;
	}

	let {
		value = $bindable(),
		files = $bindable(),
		type,
		label,
		validation,
		class: className,
		...restProps
	}: Props = $props();

	const base =
		'h-9 w-full min-w-0 rounded-md border border-input bg-transparent px-2.5 py-1 text-base shadow-xs transition-[color,box-shadow] outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30 file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground';
</script>

{#snippet input()}
	{#if type === 'file'}
		<input type="file" bind:files bind:value class={cn(base, className)} {...restProps} />
	{:else}
		<input {type} bind:value class={cn(base, className)} {...restProps} />
	{/if}
{/snippet}

{#if label}
	<div class="flex flex-col gap-1">
		<label class="text-sm">
			{label}
			{@render input()}
		</label>
		{#if validation && value}
			<p class="text-xs text-destructive">{validation}</p>
		{/if}
	</div>
{:else}
	{@render input()}
{/if}
