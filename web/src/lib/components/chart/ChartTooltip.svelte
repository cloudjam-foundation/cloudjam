<!-- Styled layerchart tooltip listing the hovered series (replaces the shad chart tooltip) -->
<script lang="ts">
	import { cn } from '$lib/utils.svelte';
	import { getChartContext, Tooltip as TooltipPrimitive } from 'layerchart';
	import type { Snippet } from 'svelte';
	import type { TooltipPayload } from './chart';

	let {
		class: className,
		labelFormatter,
		formatter
	}: {
		class?: string;
		labelFormatter?: (value: any) => string;
		formatter?: Snippet<[{ value: unknown; name: string; item: TooltipPayload }]>;
	} = $props();

	const chart = getChartContext();

	// item based charts only carry a value for the hovered series
	const series = $derived(chart.tooltip.series.filter((entry: TooltipPayload) => entry.value !== undefined));
	const label = $derived.by(() => {
		if (chart.tooltip.data == null) return undefined;
		const value = chart.x(chart.tooltip.data);
		return labelFormatter ? labelFormatter(value) : `${value}`;
	});
</script>

<TooltipPrimitive.Root variant="none">
	<div
		class={cn(
			'grid min-w-[9rem] items-start gap-1.5 rounded-lg border border-border/50 bg-background px-2.5 py-1.5 text-xs shadow-xl',
			className
		)}
	>
		{#if label !== undefined}
			<div class="font-medium">{label}</div>
		{/if}
		<div class="grid gap-1.5">
			{#each series as item, i (item.key + i)}
				<div class="flex w-full flex-wrap items-stretch gap-2">
					{#if formatter}
						{@render formatter({ value: item.value, name: item.label, item })}
					{:else}
						<div style="background: {item.config?.color ?? item.color}" class="h-full w-1 shrink-0 rounded-[2px]"></div>
						<div class="flex flex-1 shrink-0 items-center justify-between leading-none">
							<span class="text-muted-foreground">{item.label}</span>
							<span class="font-mono font-medium text-foreground tabular-nums">{item.value?.toLocaleString()}</span>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	</div>
</TooltipPrimitive.Root>
