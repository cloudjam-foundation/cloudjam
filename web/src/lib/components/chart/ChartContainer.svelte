<!-- Wrapper applying the series colors and layerchart style overrides (replaces the shad chart container) -->
<script lang="ts">
	import { cn } from '$lib/utils.svelte';
	import type { Snippet } from 'svelte';
	import type { ChartConfig } from './chart';

	let { config, class: className, children }: { config: ChartConfig; class?: string; children: Snippet } = $props();

	const colors = $derived(
		Object.entries(config)
			.filter(([, entry]) => entry.color)
			.map(([key, entry]) => `--color-${key}: ${entry.color};`)
			.join(' ')
	);
</script>

<div
	style={colors}
	class={cn(
		'flex aspect-video justify-center overflow-visible text-xs',
		'[&_.lc-highlight-line]:stroke-0 [&_.lc-highlight-point]:stroke-transparent [&_.lc-line]:stroke-border/50',
		'[&_.lc-area-path]:opacity-100 [&_.lc-highlight-line]:opacity-100 [&_.lc-highlight-point]:opacity-100 [&_.lc-spline-path]:opacity-100 [&_.lc-text]:text-xs [&_.lc-text-svg]:overflow-visible',
		'[&_.lc-axis-tick]:stroke-0 [&_.lc-rule-x-line:not(.lc-grid-x-rule)]:stroke-0 [&_.lc-rule-y-line:not(.lc-grid-y-rule)]:stroke-0',
		'[&_.lc-legend-swatch]:size-2.5 [&_.lc-legend-swatch]:rounded-[2px] [&_.lc-legend-swatch-button]:items-center [&_.lc-legend-swatch-button]:gap-1.5 [&_.lc-legend-swatch-group]:items-center [&_.lc-legend-swatch-group]:gap-4',
		'[&_.lc-labels-text:not([fill])]:fill-foreground [&_text]:stroke-transparent',
		'[&_.lc-axis-tick-label]:fill-muted-foreground [&_.lc-axis-tick-label]:font-normal',
		'[&_.lc-layout-svg-g]:fill-transparent [&_.lc-root-container]:w-full [&_.lc-tooltip-rects-g]:fill-transparent',
		className
	)}
>
	{@render children()}
</div>
