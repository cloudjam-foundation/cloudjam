<script generics="TData" lang="ts">
	import * as Popover from '$lib/components/shad/ui/popover';
	import * as Command from '$lib/components/shad/ui/command';
	import type { Column } from '@tanstack/table-core';
	import { SvelteSet } from 'svelte/reactivity';
	import { Button } from '$lib/components/shad/ui/button';
	import { Separator } from '$lib/components/shad/ui/separator';
	import { Badge } from '$lib/components/shad/ui/badge';
	import { cn } from '$lib/utils';
	import { ListFilterIcon, CheckIcon } from '@lucide/svelte';
	import AvatarRenderer from '../../../custom/dataTable/renderers/AvatarRenderer.svelte';
	import type { FacetOption } from '../../../../../ambient';

	let {
		column,
		title,
		options
	}: {
		column: Column<TData, unknown> | undefined;
		title: string;
		options: FacetOption[];
	} = $props();

	const facets = $derived(column?.getFacetedUniqueValues());

	// Never heard of this object before. Inspired by Shadcn homepage!
	const selectedValues = $derived(new SvelteSet(column?.getFilterValue() as string[]));
</script>

<Popover.Root>
	<Popover.Trigger data-rybbit-event="dataTable-FacetOn-{title}">
		{#snippet child({ props })}
			<Button data-rybbit-event="facet-open" {...props} variant="outline" size="sm" class="border-dashed">
				<ListFilterIcon />
				{title}
				{#if selectedValues.size > 0}
					<Separator orientation="vertical" class="mx-2 h-4" />
					<Badge variant="secondary" class="rounded-sm px-1 font-normal lg:hidden">
						{selectedValues.size}
					</Badge>
					<div class="hidden space-x-1 lg:flex">
						{#if selectedValues.size > 2}
							<Badge variant="secondary" class="rounded-sm px-1 font-normal">
								{selectedValues.size} selected
							</Badge>
						{:else}
							{#each options.filter((opt) => selectedValues.has(opt.value)) as option (option)}
								<Badge variant="secondary" class="rounded-sm px-1 font-normal">
									{#if option.image}
										<AvatarRenderer name={option.label} image={option.image ?? null} height="h-6" width="w-6" />
									{:else if option.icon}
										{@const Icon = option.icon}
										<Icon class="text-muted-foreground" />
									{:else}
										{option.label.substring(0, 3) + '...'}
									{/if}
								</Badge>
							{/each}
						{/if}
					</div>
				{/if}
			</Button>
		{/snippet}
	</Popover.Trigger>
	<Popover.Content align="start" class="w-56 p-0">
		<Command.Root>
			<Command.Input placeholder={title} />
			<Command.List>
				<Command.Empty>No results found.</Command.Empty>
				<Command.Group>
					{#each options as option (option)}
						{@const isSelected = selectedValues.has(option.value)}
						<Command.Item
							data-rybbit-event="facet-toggleOption"
							class="w-full"
							onSelect={() => {
								if (isSelected) {
									selectedValues.delete(option.value);
								} else {
									selectedValues.add(option.value);
								}
								const filterValues = Array.from(selectedValues);
								column?.setFilterValue(filterValues.length ? filterValues : undefined);
							}}
						>
							<div
								class={cn(
									'me-2 flex size-4 items-center justify-center rounded-sm border border-primary',
									isSelected ? 'bg-primary text-primary-foreground' : 'opacity-50 [&_svg]:invisible'
								)}
							>
								<CheckIcon class="size-4" />
							</div>
							{#if option.image}
								<AvatarRenderer name={option.label} image={option.image ?? null} height="h-6" width="w-6" />
							{:else if option.icon}
								{@const Icon = option.icon}
								<Icon class="text-muted-foreground" />
							{/if}

							<span class="mr-auto w-full">{option.label}</span>
							{#if facets?.get(option.value)}
								<span class="ms-auto flex size-4 items-center justify-center font-mono text-xs">
									{facets.get(option.value)}
								</span>
							{/if}
						</Command.Item>
					{/each}
				</Command.Group>
				{#if selectedValues.size > 0}
					<Command.Separator />
					<Command.Group>
						<Command.Item
							data-rybbit-event="facet-clear"
							onSelect={() => column?.setFilterValue(undefined)}
							class="justify-center text-center"
						>
							Clear filters
						</Command.Item>
					</Command.Group>
				{/if}
			</Command.List>
		</Command.Root>
	</Popover.Content>
</Popover.Root>
