<script generics="TData, TValue" lang="ts">
	import type { Column, Row } from '@tanstack/table-core';
	import {
		type ColumnDef,
		type ColumnFiltersState,
		type ExpandedState,
		getCoreRowModel,
		getExpandedRowModel,
		getFacetedRowModel,
		getFacetedUniqueValues,
		getFilteredRowModel,
		getGroupedRowModel,
		getPaginationRowModel,
		getSortedRowModel,
		type GroupingState,
		type PaginationState,
		type RowSelectionState,
		type SortingState,
		type VisibilityState
	} from '@tanstack/table-core';
	import { createSvelteTable, FlexRender } from '$lib/components/shad/ui/data-table';
	import { Button } from '$lib/components/shad/ui/button';
	import DataTableToolbar from '$lib/components/v2/dataTable/components/DataTableToolbar.svelte';
	import * as Table from '$lib/components/shad/ui/table';
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';
	import { cn } from '$lib/utils.js';
	import { Skeleton } from '$lib/components/shad/ui/skeleton';
	import { InboxIcon, LinkIcon, LoaderCircleIcon, SearchXIcon } from '@lucide/svelte';
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import { readTableState, writeTableState } from '$lib/tableUrlState';
	import * as Empty from '$lib/components/shad/ui/empty';

	function shallowEqual(a: unknown, b: unknown): boolean {
		if (a === b) return true;
		if (a === null || b === null || typeof a !== 'object' || typeof b !== 'object') return false;

		if (Array.isArray(a) || Array.isArray(b)) {
			if (!Array.isArray(a) || !Array.isArray(b)) return false;
			if (a.length !== b.length) return false;
			return a.every((val, i) => shallowEqual(val, b[i]));
		}

		const aObj = a as Record<string, unknown>;
		const bObj = b as Record<string, unknown>;
		const aKeys = Object.keys(aObj);
		const bKeys = Object.keys(bObj);
		if (aKeys.length !== bKeys.length) return false;
		return aKeys.every((key) => aObj[key] === bObj[key]);
	}

	type TableColumnMeta = {
		stickyRight?: boolean;
		headerClass?: string;
		cellClass?: string;
	};

	type GroupingOptions = {
		enabled?: boolean;
		initial?: string[];
		expandByDefault?: boolean;
	};

	const EMPTY_GROUPING: GroupingState = [];
	const EMPTY_EXPANDED: ExpandedState = {};

	const getColumnMeta = (meta: unknown): TableColumnMeta => {
		if (!meta || typeof meta !== 'object') return {};
		return meta as TableColumnMeta;
	};

	let {
		allowSearch = true,
		searchColumnId = 'name',
		searchPlaceholder = 'Search by name...',
		loading = $bindable(false),
		// eslint-disable-next-line no-useless-assignment
		selected = $bindable([]),
		initialSelectedIds,
		data,
		columns,
		rowClasses,
		facets,
		toolbar,
		bulkActions,
		groupHeader,
		groupHeaderSpan,
		groupCell,
		children,
		empty,
		useChildren = true,
		containerClass,
		pageSize = 10,
		grouping,
		getRowId,
		urlSync
	}: {
		allowSearch?: boolean;
		searchColumnId?: string;
		searchPlaceholder?: string;
		loading?: boolean;
		/** The `original` data of every currently-selected row. Bindable so callers can read
		 *  the selection from outside the table; stays in sync automatically, including
		 *  resetting to [] once nothing is selected. */
		selected?: TData[];
		/** Row ids (per `getRowId`) to check on first render once `data` is non-empty - e.g. a
		 *  user's current groups when opening a group picker. Requires `getRowId`. Applied once;
		 *  not kept in sync with `selected` afterwards (unlike `selected`, this is input-only). */
		initialSelectedIds?: string[];
		data: TData[];
		columns: ColumnDef<TData, TValue>[];
		rowClasses?: (row: TData) => string;
		facets?: Snippet<[{ getColumn: (columnId: string) => Column<TData, unknown> | undefined }]>;
		bulkActions?: Snippet<[{ selected: TData[] }]>;
		toolbar?: Snippet;
		groupHeader?: Snippet<
			[
				{
					row: Row<TData>;
					groupingColumnId: string | undefined;
					groupingValue: unknown;
					subRowCount: number;
				}
			]
		>;
		/** Number of leading columns the group header cell spans. When set together with
		 *  `groupCell`, the remaining columns of a group row are rendered as individual cells
		 *  so group-level values line up under their column headers. */
		groupHeaderSpan?: number;
		groupCell?: Snippet<[{ row: Row<TData>; columnId: string }]>;
		children?: Snippet<[{ rows: Row<TData>[] }]>;
		/** Shown when there's no data at all, e.g. "No assets yet" with a button to create one. When filters
		 *  hide every row, a built-in "Nothing matches" with a Clear filters button is shown instead */
		empty?: Snippet;
		/** Whether to render `children` instead of the built-in table body. Lets a single
		 *  DataTable instance be toggled between its default body and custom content
		 *  without unmounting (which would reset table state and any in-flight animations). */
		useChildren?: boolean;
		containerClass?: string;
		pageSize?: number;
		grouping?: GroupingOptions;
		getRowId?: (originalRow: TData, index: number, parent?: Row<TData>) => string;
		/** Keep the search, filters, sort and page in the URL so views can be bookmarked and shared, and
		 *  offer a Copy link button. Pass a prefix when more than one table on a page syncs */
		urlSync?: boolean | { prefix?: string };
	} = $props();

	let pagination = $state<PaginationState>({ pageIndex: 0, pageSize: pageSize });
	let sorting = $state<SortingState>([]);
	let columnFilters = $state<ColumnFiltersState>([]);
	let columnVisibility = $state<VisibilityState>({});
	let rowSelection = $state<RowSelectionState>({});
	let groupingState = $state<GroupingState>([]);
	let expanded = $state<ExpandedState>({});

	$effect(() => {
		if (!grouping?.enabled) return;
		if (!groupingState.length && grouping?.initial?.length) {
			groupingState = grouping.initial;
		}
		if (grouping?.expandByDefault && typeof expanded === 'object' && Object.keys(expanded).length === 0) {
			expanded = true;
		}
	});

	const table = createSvelteTable({
		get data() {
			return data;
		},
		get columns() {
			return columns;
		},
		getRowId,
		autoResetExpanded: false,
		autoResetPageIndex: false,
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		getSortedRowModel: getSortedRowModel(),
		getFilteredRowModel: getFilteredRowModel(),
		getFacetedRowModel: getFacetedRowModel(),
		getFacetedUniqueValues: getFacetedUniqueValues(),
		getGroupedRowModel: getGroupedRowModel(),
		getExpandedRowModel: getExpandedRowModel(),
		onPaginationChange: (updater) => {
			const next = typeof updater === 'function' ? updater(pagination) : updater;
			if (!shallowEqual(next, pagination)) {
				pagination = next;
			}
		},
		onSortingChange: (updater) => {
			const next = typeof updater === 'function' ? updater(sorting) : updater;
			if (!shallowEqual(next, sorting)) {
				sorting = next;
			}
		},
		onColumnFiltersChange: (updater) => {
			const next = typeof updater === 'function' ? updater(columnFilters) : updater;
			if (!shallowEqual(next, columnFilters)) {
				columnFilters = next;
			}
		},
		onColumnVisibilityChange: (updater) => {
			const next = typeof updater === 'function' ? updater(columnVisibility) : updater;
			if (!shallowEqual(next, columnVisibility)) {
				columnVisibility = next;
			}
		},
		onGroupingChange: (updater) => {
			if (!grouping?.enabled) return;
			const next = typeof updater === 'function' ? updater(groupingState) : updater;
			if (!shallowEqual(next, groupingState)) {
				groupingState = next;
			}
		},
		onExpandedChange: (updater) => {
			if (!grouping?.enabled) return;
			const next = typeof updater === 'function' ? updater(expanded) : updater;
			if (!shallowEqual(next, expanded)) {
				expanded = next;
			}
		},
		onRowSelectionChange: (updater) => {
			const next = typeof updater === 'function' ? updater(rowSelection) : updater;
			if (!shallowEqual(next, rowSelection)) {
				rowSelection = next;
			}
		},
		state: {
			get pagination() {
				return pagination;
			},
			get sorting() {
				return sorting;
			},
			get columnFilters() {
				return columnFilters;
			},
			get columnVisibility() {
				return columnVisibility;
			},
			get grouping() {
				return grouping?.enabled ? groupingState : EMPTY_GROUPING;
			},
			get expanded() {
				return grouping?.enabled ? expanded : EMPTY_EXPANDED;
			},
			get rowSelection() {
				return rowSelection;
			}
		}
	});

	// Seed row selection from `initialSelectedIds` once real data is available, so a caller can
	// open the table with some rows already checked (e.g. a user's current groups). Deliberately
	// depends only on `data`/`getRowId`/`initialSelectedIds` - never on `selected` itself, since
	// `selected` is an *output* written by the effect below and echoed back into this component
	// via the caller's bindable prop; depending on it here would race that echo (and, because
	// the guard below returns before reading it on the seeded run, wouldn't even be tracked
	// reliably) rather than seed cleanly.
	let initialSelectionSeeded = false;
	$effect(() => {
		if (initialSelectionSeeded || !getRowId || data.length === 0) return;
		initialSelectionSeeded = true;

		if (!initialSelectedIds || initialSelectedIds.length === 0) return;

		const idsToSelect = new Set(initialSelectedIds);
		const seeded: RowSelectionState = {};
		data.forEach((row, index) => {
			const id = getRowId(row, index);
			if (idsToSelect.has(id)) seeded[id] = true;
		});
		rowSelection = seeded;
	});

	$effect(() => {
		selected = table.getSelectedRowModel().rows.map((row) => row.original);
	});

	// URL parameters (see $lib/tableUrlState)
	const urlOptions = $derived({
		prefix: typeof urlSync === 'object' ? urlSync.prefix : undefined,
		searchColumnId
	});
	let urlReady = $state(false);
	let selectionFromUrl: string[] = [];

	onMount(() => {
		if (!urlSync) return;
		const state = readTableState(page.url.searchParams, {
			...urlOptions,
			columnIds: table.getAllLeafColumns().map((column) => column.id)
		});
		if (state.columnFilters.length > 0) columnFilters = state.columnFilters;
		if (state.sorting.length > 0) sorting = state.sorting;
		if (state.pageIndex > 0) pagination = { ...pagination, pageIndex: state.pageIndex };
		selectionFromUrl = state.selected;
		urlReady = true;
	});

	// A selection in a link is applied once the rows it names have loaded
	$effect(() => {
		if (!urlReady || selectionFromUrl.length === 0 || !getRowId || data.length === 0) return;
		const wanted = new Set(selectionFromUrl);
		selectionFromUrl = [];
		rowSelection = Object.fromEntries(
			data
				.map((row, index) => getRowId(row, index))
				.filter((id) => wanted.has(id))
				.map((id) => [id, true])
		);
	});

	function stateForUrl(withSelection: boolean) {
		return {
			columnFilters,
			sorting,
			pageIndex: pagination.pageIndex,
			selected: withSelection ? Object.keys(rowSelection).filter((id) => rowSelection[id]) : []
		};
	}

	// Keep the URL up to date, without a history entry per keystroke
	let urlTimer: ReturnType<typeof setTimeout> | undefined;
	$effect(() => {
		if (!urlReady) return;
		const state = stateForUrl(false);
		clearTimeout(urlTimer);
		urlTimer = setTimeout(() => {
			const url = new URL(page.url);
			writeTableState(url.searchParams, state, urlOptions);
			if (url.search !== page.url.search) {
				// Only the query string of the current page changes, so there's no route to resolve()
				// eslint-disable-next-line svelte/no-navigation-without-resolve
				replaceState(url, page.state);
			}
		}, 300);
		return () => clearTimeout(urlTimer);
	});

	async function copyLink() {
		const url = new URL(page.url);
		writeTableState(url.searchParams, stateForUrl(true), urlOptions);
		try {
			await navigator.clipboard.writeText(url.toString());
			toast.success('Link copied');
		} catch {
			toast.error("Couldn't copy the link");
		}
	}
</script>

<DataTableToolbar {allowSearch} {searchColumnId} {searchPlaceholder} {table}>
	{#snippet facetsInternal()}
		{@render facets?.({ getColumn: (columnId) => table.getColumn(columnId) })}
	{/snippet}

	{#snippet other()}
		{#if table.getSelectedRowModel().flatRows.length > 0}
			{@render bulkActions?.({
				selected: table.getSelectedRowModel().rows.map((row) => row.original)
			})}
		{/if}
		{@render toolbar?.()}
		{#if urlSync}
			<Button variant="outline" title="Copy a link to this view, including any selected rows" onclick={copyLink}>
				<LinkIcon />
				Copy link
			</Button>
		{/if}
	{/snippet}
</DataTableToolbar>

{#if children && useChildren}
	{@render children({ rows: table.getRowModel().rows })}
{:else}
	<div class={cn('w-full min-w-0 overflow-auto rounded-md border', containerClass)}>
		<Table.Root>
			<Table.Header>
				{#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
					<Table.Row>
						{#each headerGroup.headers as header (header.id)}
							{#if loading}
								<Table.Head>
									<Skeleton class="h-6 w-full" />
								</Table.Head>
							{:else}
								{@const meta = getColumnMeta(header.column.columnDef.meta)}
								{@const stickyRight = Boolean(meta.stickyRight)}
								<Table.Head
									colspan={header.colSpan}
									class={cn('sticky top-0 z-10', stickyRight && 'sticky right-0 z-20 bg-transparent', meta.headerClass)}
								>
									{#if !header.isPlaceholder}
										<FlexRender content={header.column.columnDef.header} context={header.getContext()} />
									{/if}
								</Table.Head>
							{/if}
						{/each}
					</Table.Row>
				{/each}
			</Table.Header>
			<Table.Body>
				{#if loading}
					<Table.Row>
						<Table.Cell colspan={columns.length} class="h-24 text-center">
							<span class="mx-auto inline-flex w-max items-center">
								<LoaderCircleIcon class="mr-2 animate-spin" />
								Loading...
							</span>
						</Table.Cell>
					</Table.Row>
				{:else}
					{#each table.getRowModel().rows as row (row.id)}
						{#if grouping?.enabled && row.getIsGrouped()}
							{@const splitGroupCells = Boolean(groupCell && groupHeaderSpan)}
							{@const visibleCells = row.getVisibleCells()}
							<Table.Row class="bg-muted/30">
								<Table.Cell colspan={splitGroupCells ? groupHeaderSpan : visibleCells.length} class="font-medium">
									{@const groupingColumnId = row.groupingColumnId}
									{@const groupingValue = row.getGroupingValue(groupingColumnId ?? '')}
									<div class="flex items-center gap-2" style={`padding-left: ${row.depth * 1.25}rem`}>
										<button
											type="button"
											onclick={row.getToggleExpandedHandler()}
											class="inline-flex size-5 items-center justify-center rounded hover:bg-muted"
											aria-label="Toggle group"
										>
											<span class="inline-block w-4 text-center">{row.getIsExpanded() ? '-' : '+'}</span>
										</button>
										{#if groupHeader}
											{@render groupHeader({
												row,
												groupingColumnId,
												groupingValue,
												subRowCount: row.subRows.length
											})}
										{:else}
											<span>
												{groupingColumnId}: {String(groupingValue)}
											</span>
											<span class="text-muted-foreground">({row.subRows.length})</span>
										{/if}
									</div>
								</Table.Cell>
								{#if splitGroupCells && groupCell}
									{#each visibleCells.slice(groupHeaderSpan) as cell (cell.id)}
										{@const meta = getColumnMeta(cell.column.columnDef.meta)}
										{@const stickyRight = Boolean(meta.stickyRight)}
										<Table.Cell class={cn(stickyRight && 'sticky right-0 z-20 bg-transparent', meta.cellClass)}>
											{@render groupCell({ row, columnId: cell.column.id })}
										</Table.Cell>
									{/each}
								{/if}
							</Table.Row>
						{:else}
							<Table.Row
								data-state={row.getIsSelected() && 'selected'}
								class={rowClasses ? rowClasses(row.original) : ''}
							>
								{#each row.getVisibleCells() as cell (cell.id)}
									{@const meta = getColumnMeta(cell.column.columnDef.meta)}
									{@const stickyRight = Boolean(meta.stickyRight)}
									<Table.Cell class={cn(stickyRight && 'sticky right-0 z-20 bg-transparent', meta.cellClass)}>
										{#if !cell.getIsPlaceholder() && cell.getIsAggregated()}
											<FlexRender
												content={cell.column.columnDef.aggregatedCell ?? cell.column.columnDef.cell}
												context={cell.getContext()}
											/>
										{:else if !cell.getIsPlaceholder()}
											<FlexRender content={cell.column.columnDef.cell} context={cell.getContext()} />
										{/if}
									</Table.Cell>
								{/each}
							</Table.Row>
						{/if}
					{:else}
						<Table.Row class="hover:bg-transparent">
							<Table.Cell colspan={columns.length} class="py-10">
								{#if data.length > 0}
									<!-- There is data, but the search or filters hide all of it -->
									<Empty.Root>
										<Empty.Header>
											<Empty.Media variant="icon"><SearchXIcon /></Empty.Media>
											<Empty.Title>Nothing matches</Empty.Title>
											<Empty.Description>Nothing matches your search or filters.</Empty.Description>
										</Empty.Header>
										<Empty.Content>
											<Button variant="outline" onclick={() => table.resetColumnFilters()}>Clear filters</Button>
										</Empty.Content>
									</Empty.Root>
								{:else if empty}
									{@render empty()}
								{:else}
									<Empty.Root>
										<Empty.Header>
											<Empty.Media variant="icon"><InboxIcon /></Empty.Media>
											<Empty.Title>Nothing here yet</Empty.Title>
										</Empty.Header>
									</Empty.Root>
								{/if}
							</Table.Cell>
						</Table.Row>
					{/each}
				{/if}
			</Table.Body>
		</Table.Root>
	</div>
{/if}

<div class="flex items-center justify-end space-x-2 py-4">
	<Button disabled={!table.getCanPreviousPage()} onclick={() => table.previousPage()} size="sm" variant="outline">
		Previous
	</Button>
	<Button disabled={!table.getCanNextPage()} onclick={() => table.nextPage()} size="sm" variant="outline">Next</Button>
</div>
