<script lang="ts">
	import * as DropdownMenu from '$lib/components/shad/ui/dropdown-menu';
	import { Button } from '$lib/components/shad/ui/button';
	import { EllipsisIcon } from '@lucide/svelte';
	import { cn, type WithElementRef } from '$lib/utils';
	import type { HTMLButtonAttributes } from 'svelte/elements';
	import { setContext, type Snippet } from 'svelte';

	let {
		trigger,
		children,
		disabled,
		open = $bindable(false),
		width = 'w-48',
		class: className
	}: WithElementRef<HTMLButtonAttributes> & {
		trigger?: Snippet;
		open?: boolean;
		width?: string;
	} = $props();

	// Items keep the menu open while their action runs, then close it through this
	setContext('dataTableAction', { close: () => (open = false) });
</script>

<DropdownMenu.Root bind:open>
	<DropdownMenu.Trigger data-rybbit-event="rowActions-openMenu" {disabled}>
		{#if trigger}
			{@render trigger?.()}
		{:else}
			<Button
				data-rybbit-event="rowActions-openMenu"
				class={cn(
					'pointer-events-auto relative size-8 rounded-full border border-muted bg-card/90 p-0 shadow-sm backdrop-blur supports-backdrop-filter:bg-card/70',
					className
				)}
				size="icon"
				variant="ghost"
			>
				<span class="sr-only">Open menu</span>
				<EllipsisIcon />
			</Button>
		{/if}
	</DropdownMenu.Trigger>

	<!--
		No scroll lock: the items here open dialogues, and the menu's lock and the dialogue's
		lock overlap. When they do, closing the dialogue leaves the page locked with
		pointer-events: none on the body, so nothing can be clicked until a refresh.
	-->
	<DropdownMenu.Content class={width} preventScroll={false}>
		{#if children}
			{@render children?.()}
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
