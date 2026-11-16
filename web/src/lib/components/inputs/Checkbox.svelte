<script lang="ts">
	import { CheckIcon, MinusIcon } from '@lucide/svelte';
	import { cn } from '$lib/utils.svelte.js';
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface CheckboxProps extends Omit<HTMLInputAttributes, 'type'> {
		ref?: HTMLInputElement | null;
		checked?: boolean | null | undefined;
		indeterminate?: boolean | null;
		class?: string;
	}

	let {
		ref = $bindable(null),
		checked = $bindable(),
		indeterminate = $bindable(),
		class: className,
		...restProps
	}: CheckboxProps = $props();
</script>

<span class={cn('relative inline-grid size-4 shrink-0 place-items-center', className)}>
	<input
		type="checkbox"
		data-slot="checkbox"
		bind:this={ref}
		bind:checked
		bind:indeterminate
		class="col-start-1 row-start-1 size-4 cursor-pointer appearance-none rounded-xs glass outline-none checked:glass indeterminate:glass hover:bg-foreground/5 focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
		{...restProps}
	/>
	<span
		data-slot="checkbox-indicator"
		class="pointer-events-none relative col-start-1 row-start-1 text-primary disabled:opacity-50 [&>svg]:size-3 [&>svg]:zoom-in-50"
	>
		{#if indeterminate}
			<MinusIcon strokeWidth={3} />
		{:else if checked}
			<CheckIcon strokeWidth={3} />
		{/if}
	</span>
</span>
