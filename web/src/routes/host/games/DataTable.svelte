<script lang="ts" generics="T">
	import type { SubmitState } from '$lib';
	import ProcessingButton from '$lib/components/custom/buttons/ProcessingButton.svelte';
	import DelaySpinner from '$lib/components/custom/DelaySpinner.svelte';
	import * as Alert from '$lib/components/shad/alert';
	import { OctagonAlertIcon } from '@lucide/svelte';
	import { onMount, type Snippet } from 'svelte';

	let {
		load,
		loadState,
		columns
	}: {
		load: (startAfter: T | undefined) => Promise<T[]>;
		loadState: SubmitState;
		columns: { title: string; width: string; render: Snippet<[data: T]> }[];
	} = $props();

	let exhausted = $state(false);
	let pageSize: number | null = $state(null);
	let data: T[] = $state([]);

	async function appendData(startAfter?: T) {
		const newData = await load(startAfter);
		if (pageSize === null) pageSize = newData.length;
		else if (newData.length < pageSize) exhausted = true;
		data.push(...newData);
	}

	onMount(() => {
		data = [];
		appendData();
	});
</script>

{#if data.length < 1 && loadState.loading}
	<DelaySpinner class="mt-10 flex justify-center" delay={200} />
{:else if !loadState.forbidden}
	<div class="overflow-scroll-hidden max-h-96 justify-center px-1 py-0.5" style="container-type: scroll-state;">
		<div
			class="table-apple-glass sticky top-0 z-10 grid w-full rounded-md p-2 transition-all"
			style="grid-template-columns: {columns.map((c) => c.width.replace('%', 'fr')).join(' ')};"
		>
			{#each columns as column}
				<div class="min-w-0 text-start font-medium text-primary/90" style="width: {column.width};">
					{column.title}
				</div>
			{/each}
		</div>

		<div class="p-2">
			<table class="w-full table-fixed shadow-inner">
				<thead>
					<tr>
						{#each columns as column}
							<th class="invisible text-start" style="width: {column.width};"> </th>
						{/each}
					</tr>
				</thead>
				<tbody>
					{#each data as row}
						<tr>
							{#each columns as column}
								<td class="overflow-scroll-hidden border-t border-primary/20 p-1">{@render column.render(row)}</td>
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
	{#if loadState.error}
		<Alert.Root variant="destructive">
			<OctagonAlertIcon />
			<Alert.Title>Failed to load data</Alert.Title>
			<Alert.Description>{loadState.error}</Alert.Description>
		</Alert.Root>
	{/if}
{/if}

<style>
	.overflow-scroll-hidden::-webkit-scrollbar {
		display: none;
		/* Chrome, Safari and Opera */
	}

	.overflow-scroll-hidden {
		overflow: scroll;
		-ms-overflow-style: none;
		/* IE and Edge */
		scrollbar-width: none;
		/* Firefox */
	}

	@container scroll-state(scrollable: top) {
		.table-apple-glass {
			box-shadow:
				rgba(255, 255, 255, 0.08) 0px 4px 6px 0px,
				rgba(255, 255, 255, 0.2) 0px 0px 0px 1px;
			background-color: rgba(255, 255, 255, 0.04);
			backdrop-filter: blur(16px);

			animation: glass-appear 400ms ease-out both;
		}
	}
	@keyframes glass-appear {
		from {
			transform: scale(0.99);
			backdrop-filter: blur(0px);
		}
		to {
			transform: scale(1);
			backdrop-filter: blur(8px);
		}
	}
</style>
