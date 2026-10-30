<script lang="ts" generics="T">
	import type { SubmitState } from '$lib';
	import ProcessingButton from '$lib/components/custom/buttons/ProcessingButton.svelte';
	import DelaySpinner from '$lib/components/custom/DelaySpinner.svelte';
	import { cn } from '$lib/utils';
	import { onMount, type Snippet } from 'svelte';
	import Alert from '$lib/components/custom/Alert.svelte';

	let {
		load,
		loadState,
		columns,
		onRowClick,
		class: className
	}: {
		load: (startAfter: T | undefined) => Promise<T[]>;
		loadState: SubmitState;
		columns: { title: string; width: string; render: Snippet<[data: T]> }[];
		onRowClick?: (row: T) => void;
		class?: string;
	} = $props();

	let exhausted = $state(false);
	let pageSize: number | null = $state(null);
	let data: T[] = $state([]);

	async function appendData(startAfter?: T) {
		const newData = await load(startAfter);
		if (pageSize === null) pageSize = newData.length;
		if (newData.length < pageSize || newData.length === 1) exhausted = true;
		data.push(...newData);
	}

	let table: HTMLTableElement | undefined = $state();
	// here we just manually map the tables space calculations to the header grid.
	// It seems most stable to do this in javascript with raw pixels and a observer...
	let gridTemplateColumns = $state('');
	$effect(() => {
		if (!table) return;
		const headers = Array.from(table.querySelectorAll('thead th'));
		const updateColumns = () => {
			gridTemplateColumns = headers.map((th) => `${th.getBoundingClientRect().width}px`).join(' ');
		};
		const observer = new ResizeObserver(updateColumns);
		headers.forEach((th) => observer.observe(th));
		updateColumns();
		return () => observer.disconnect();
	});

	onMount(() => {
		data = [];
		appendData();
	});
</script>

{#if data.length < 1 && loadState.loading}
	<DelaySpinner class="mt-10 flex justify-center" delay={200} />
{:else if loadState.error}
	<Alert title="Failed to load data" message={loadState.error} />
{:else if !loadState.forbidden}
	<div
		class={cn('overflow-scroll-hidden justify-center rounded-md p-2', className)}
		style="container-type: scroll-state;"
	>
		<div
			class="sticky top-0 z-10 grid w-full rounded-md overlay-glass p-2 transition-all"
			style="grid-template-columns: {gridTemplateColumns};"
		>
			{#each columns as column}
				<div class="min-w-0 text-start font-medium text-primary/90" style="width: {column.width};">
					{column.title}
				</div>
			{/each}
		</div>

		<div class="p-2">
			<table bind:this={table} class="w-full table-fixed">
				<colgroup>
					{#each columns as column}
						<col style="width: {column.width};" />
					{/each}
				</colgroup>
				<thead>
					<tr>
						{#each columns as _}
							<th class="invisible text-start"> </th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each data as row}
						<tr onclick={() => onRowClick?.(row)} class={onRowClick ? 'cursor-pointer hover:bg-foreground/3' : ''}>
							{#each columns as column}
								<td class="overflow-scroll-hidden border-t border-primary/20 p-1 text-nowrap">
									{@render column.render(row)}
								</td>
							{/each}
						</tr>
					{/each}
					{#if !exhausted}
						<tr>
							<td colspan={columns.length} class="w-full">
								<div class="flex flex-row items-center justify-center">
									<ProcessingButton
										variant="secondary"
										processing={loadState.loading}
										class="cursor-pointer self-center"
										onclick={() => appendData(data.at(-1))}
									>
										Load More
									</ProcessingButton>
								</div>
							</td>
						</tr>
					{/if}
				</tbody>
			</table>
		</div>
	</div>
{/if}
