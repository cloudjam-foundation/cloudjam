<script lang="ts">
	import * as Chart from '$lib/components/shad/chart/index.js';
	import { scaleUtc } from 'd3-scale';
	import { Area, AreaChart, ChartClipPath } from 'layerchart';
	import { curveNatural } from 'd3-shape';
	import { cubicInOut } from 'svelte/easing';
	import ChartContainer from '$lib/components/shad/chart/chart-container.svelte';

	let {
		timeFormat,
		labels,
		data,
		brushDisabled: disabled,
		unit = '',
		from = $bindable(),
		to = $bindable()
	}: {
		timeFormat: any;
		labels: Chart.ChartConfig;
		data: any[];
		brushDisabled?: boolean;
		unit?: string;
		from?: Date;
		to?: Date;
	} = $props();

	let locked = $state(false);
</script>

<ChartContainer config={labels} class="-ml-3 aspect-auto h-62.5 w-full">
	<AreaChart
		{data}
		x="date"
		xScale={scaleUtc()}
		xDomain={from && to ? [from, to] : undefined}
		series={Object.entries(labels).map(([key, value]) => ({
			key: key,
			label: value.label,
			color: value.color
		}))}
		seriesLayout="stack"
		tooltipContext={{
			locked,
			onclick: () => (locked = !locked)
		}}
		brush={{
			disabled: disabled,
			axis: 'x',
			zoomOnBrush: false,
			onBrushEnd: ({ brush }) => {
				const [start, end] = brush.x ?? [];
				if (start != null && end != null && +end > +start) {
					from = new Date(+start);
					to = new Date(+end);
				}
				brush.reset();
			}
		}}
		props={{
			xAxis: {
				format: timeFormat
			},
			yAxis: { format: () => '' }
		}}
	>
		{#snippet marks({ context })}
			<ChartClipPath
				initialWidth={0}
				motion={{
					width: { type: 'tween', duration: 1000, easing: cubicInOut }
				}}
			>
				{#each context.series.visibleSeries as s (s.key)}
					<Area
						seriesKey={s.key}
						curve={curveNatural}
						fillOpacity={0.4}
						line={{ class: 'stroke-1' }}
						motion="tween"
						{...s.props}
					/>
				{/each}
			</ChartClipPath>
		{/snippet}
		{#snippet tooltip({ context })}
			{#if Number(context) !== 0}
				<Chart.Tooltip
					class="overflow-scroll-hidden pointer-events-auto! z-50 max-h-72 w-max"
					labelFormatter={(v: Date) => {
						return v.toLocaleDateString('en-US', {
							month: '2-digit',
							day: '2-digit',
							hour: '2-digit',
							minute: '2-digit',
							hour12: false
						});
					}}
					indicator="line"
				>
					{#snippet formatter({ value, name, item })}
						<div
							style="--color-bg: {item.config?.color ?? item.color}"
							class="w-1 shrink-0 self-stretch rounded-xs bg-(--color-bg)"
						></div>
						<div class="flex flex-1 items-center justify-between gap-6 leading-none">
							<span class="whitespace-nowrap text-muted-foreground">{name}</span>
							<span class="font-mono font-medium whitespace-nowrap text-foreground tabular-nums">
								{typeof value === 'number' ? value.toLocaleString() : value}{unit ? ` ${unit}` : ''}
							</span>
						</div>
					{/snippet}
				</Chart.Tooltip>
			{/if}
		{/snippet}
	</AreaChart>
</ChartContainer>
