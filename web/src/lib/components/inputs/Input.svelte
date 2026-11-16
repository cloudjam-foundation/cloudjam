<script lang="ts">
	import { cn } from '$lib/utils.svelte.js';
	import type { HTMLInputAttributes } from 'svelte/elements';
	import Checkbox from '$lib/components/inputs/Checkbox.svelte';

	interface Props extends HTMLInputAttributes {
		id?: string;
		label?: string;
		validation?: string;
		forceValidation?: boolean;
		class?: string;
	}

	let {
		id,
		value = $bindable(),
		files = $bindable(),
		checked = $bindable(),
		type,
		label,
		validation,
		forceValidation = false,
		class: className,
		...restProps
	}: Props = $props();

	const base =
		'h-9 w-full min-w-0 rounded-md border border-input bg-transparent px-2.5 py-1 text-base shadow-xs transition-[color,box-shadow] outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm dark:bg-input/30 file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground';
</script>

{#snippet input()}
	{#if type === 'file'}
		<input type="file" {id} bind:files bind:value class={cn(base, className)} {...restProps} />
	{:else if type === 'checkbox'}
		<Checkbox bind:checked {id} {...restProps} />
	{:else}
		<input {type} bind:value {id} class={cn(base, className)} {...restProps} />
	{/if}
{/snippet}

{#if label || forceValidation}
	<div class="flex flex-col gap-1">
		{#if label}
			<label class="text-sm" for={id}>
				{label}
			</label>
		{/if}
		{@render input()}
		{#if validation && (value || forceValidation)}
			<p id={id ? `${id}-error` : undefined} class="text-xs text-destructive">{validation}</p>
		{/if}
	</div>
{:else}
	{@render input()}
{/if}
