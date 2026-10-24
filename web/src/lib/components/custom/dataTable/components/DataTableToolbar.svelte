<script generics="TData" lang="ts">
	import * as DropdownMenu from '$lib/components/shad/ui/dropdown-menu';
	import { Button } from '$lib/components/shad/ui/button';
	import { Input } from '$lib/components/shad/ui/input';
	import type { Table } from '@tanstack/table-core';
	import { SearchIcon, EyeIcon } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		allowSearch,
		searchColumnId = 'name',
		searchPlaceholder = 'Search by name...',
		table,
		facetsInternal,
		other
	}: {
		allowSearch?: boolean;
		searchColumnId?: string;
		searchPlaceholder?: string;
		table: Table<TData>;
		facetsInternal?: Snippet;
		other?: Snippet;
	} = $props();

	const searchColumnExists = $derived(table.getAllColumns().some((column) => column.id === searchColumnId));
	const searchColumn = $derived(searchColumnExists ? table.getColumn(searchColumnId) : undefined);
</script>

<div class="@container flex w-full min-w-0 flex-col gap-4 py-4">
	<div class="flex w-full min-w-0 flex-col items-stretch gap-4 @3xl:flex-row @3xl:flex-wrap @3xl:items-center">
		{#if allowSearch}
			<div class="relative min-w-0 flex-1 @3xl:w-64 @3xl:flex-none">
				<SearchIcon
					class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
				/>
				<Input
					class="w-full pl-8 @3xl:max-w-sm"
					onchange={(e) => {
						searchColumn?.setFilterValue(e.currentTarget.value);
					}}
					oninput={(e) => {
						searchColumn?.setFilterValue(e.currentTarget.value);
					}}
					placeholder={searchPlaceholder}
					value={searchColumn?.getFilterValue() ?? ''}
				/>
			</div>
		{/if}

		<div class="mr-auto hidden w-full min-w-0 flex-1 items-center gap-2 @7xl:flex">
			{@render facetsInternal?.()}
		</div>

		<div
			class="flex w-full min-w-0 flex-wrap items-center justify-center gap-4 @3xl:ml-auto @3xl:w-auto @3xl:justify-end"
		>
			{@render other?.()}

			<DropdownMenu.Root>
				<DropdownMenu.Trigger data-rybbit-event="table-openColumns">
					{#snippet child({ props })}
						<Button data-rybbit-event="table-openColumns" {...props} variant="outline" class="ms-auto">
							<EyeIcon />
							Columns
						</Button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content align="end" class="min-w-48">
					{#each table.getAllColumns().filter((col) => col.getCanHide()) as column (column.id)}
						<DropdownMenu.CheckboxItem
							class="capitalize"
							bind:checked={() => column.getIsVisible(), (v) => column.toggleVisibility(!!v)}
						>
							{column.id}
						</DropdownMenu.CheckboxItem>
					{/each}
				</DropdownMenu.Content>
			</DropdownMenu.Root>
		</div>
	</div>

	<div class="grid w-full grid-cols-2 items-center gap-2 @7xl:hidden">
		{@render facetsInternal?.()}
	</div>
</div>
