import type { Tooltip } from 'layerchart';

export type ChartConfig = Record<string, { label?: string; color?: string }>;

export type TooltipPayload = Tooltip.TooltipSeries;
