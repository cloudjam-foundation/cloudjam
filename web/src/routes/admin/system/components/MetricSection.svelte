<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import type { ChartConfig } from '$lib/components/chart/chart.ts';
	import Card from '$lib/components/Card.svelte';
	import {
		AggregateHitsRequestSchema,
		AggregateLatencyRequestSchema,
		AggregateWindow,
		AggregateWindowSchema
	} from '$lib/sdk/v1/admin/system/system_pb.ts';
	import { create, enumToJson } from '@bufbuild/protobuf';
	import { timestampFromDate } from '@bufbuild/protobuf/wkt';
	import ResetIcon from '@lucide/svelte/icons/rotate-ccw';
	import ZoomOutIcon from '@lucide/svelte/icons/zoom-out';
	import ZoomChart from './ZoomChart.svelte';
	import { SvelteDate } from 'svelte/reactivity';

	// safe generates "layerchart" safe identifiers / keys.
	const safe = (k: string) => k.replace(/[^a-zA-Z0-9_]/g, '_');

	const initialFrom = new Date(Date.now() - 24 * 60 * 60 * 1000);
	const initialTo = new Date();

	let from = $state(initialFrom);
	let to = $state(clampEnd(AggregateWindow.Hour, initialTo));

	// zoomWindow picks the aggregation window (and its bucket size in ms)
	// based on the zoomed range, so zooming in yields higher resolution data.
	function zoomWindow(from: Date, to: Date): [AggregateWindow, number] {
		const span = to.getTime() - from.getTime();
		if (span <= 6 * 60 * 60 * 1000) return [AggregateWindow.Minute, 60 * 1000];
		if (span <= 7 * 24 * 60 * 60 * 1000) return [AggregateWindow.Hour, 60 * 60 * 1000];
		return [AggregateWindow.Day, 24 * 60 * 60 * 1000];
	}

	let [window, bucket] = $derived(zoomWindow(from, to));

	// TODO: Type this file properly

	function zoomTimeformat(from: Date, to: Date) {
		// if > 48h show a per day format
		if (to.getTime() - from.getTime() <= 48 * 60 * 60 * 1000) {
			return (v: any) => {
				return v.toLocaleTimeString('en-US', {
					hour: '2-digit',
					minute: '2-digit',
					hour12: false
				});
			};
		}
		return (v: any) => {
			return v.toLocaleDateString('en-US', {
				month: '2-digit',
				day: '2-digit'
			});
		};
	}

	// clampEnd clamps the to date smoothly to the end of the graph depending on the aggregation window.
	function clampEnd(window: AggregateWindow, to: Date): Date {
		let topClamp = new SvelteDate();
		switch (window) {
			case AggregateWindow.Month:
				topClamp.setUTCDate(1);
				topClamp.setUTCHours(0, 0, 0, 0);
				break;
			case AggregateWindow.Day:
				topClamp.setUTCHours(0, 0, 0, 0);
				break;
			case AggregateWindow.Hour:
				topClamp.setUTCMinutes(0, 0, 0);
				break;
			case AggregateWindow.Minute:
				topClamp.setUTCMinutes(0, 0, 0);
				break;
		}
		return new Date(Math.min(to.getTime(), topClamp.getTime()));
	}

	let hitLabels: ChartConfig = $state({});
	let hitData: any[] = $state([]);
	let totalHits: number = $state(0);
	let latencyLabels: ChartConfig = $state({});
	let latencyData: any[] = $state([]);
	let overallLatency: number = $state(0);

	async function loadHits(
		window: AggregateWindow,
		bucket: number,
		from: Date,
		to: Date
	): Promise<[number, ChartConfig, any[]]> {
		let overallHits = 0;
		const resp = await Glue.system.aggregateHits(
			create(AggregateHitsRequestSchema, {
				window: window,
				from: timestampFromDate(from),
				to: timestampFromDate(to)
			})
		);
		const endpoints = Object.keys(resp.endpoints);
		const records: Record<number, any> = {};
		const start = Math.floor(from.getTime() / bucket) * bucket;
		for (let t = start; t <= to.getTime(); t += bucket) {
			records[t] = { date: new Date(t) };
			for (const endpoint of endpoints) {
				records[t][safe(endpoint)] = 0;
			}
		}

		for (const [endpoint, series] of Object.entries(resp.endpoints)) {
			for (let i = 0; i < series.time.length; i++) {
				overallHits += Number(series.count[i]);
				const timestamp = Math.floor((Number(series.time[i].seconds) * 1000) / bucket) * bucket;
				if (records[timestamp]) {
					records[timestamp][safe(endpoint)] = Number(series.count[i]);
				}
			}
		}
		const labels: ChartConfig = {};
		for (let i = 0; i < endpoints.length; i++) {
			labels[safe(endpoints[i])] = {
				label: endpoints[i],
				color: `var(--chart-${(i % 5) + 1})`
			};
		}
		return [overallHits, labels, Object.values(records).sort((a, b) => a.date - b.date)];
	}

	async function loadLatency(
		window: AggregateWindow,
		bucket: number,
		from: Date,
		to: Date
	): Promise<[number, ChartConfig, any[]]> {
		let overallLatency = 0;
		let overallLatencyCount = 0;
		const resp = await Glue.system.aggregateLatency(
			create(AggregateLatencyRequestSchema, {
				window: window,
				from: timestampFromDate(from),
				to: timestampFromDate(to)
			})
		);
		const endpoints = Object.keys(resp.endpoints);
		const records: Record<number, any> = {};
		const start = Math.floor(from.getTime() / bucket) * bucket;
		for (let t = start; t <= to.getTime(); t += bucket) {
			records[t] = { date: new Date(t) };
			for (const endpoint of endpoints) {
				records[t][safe(endpoint)] = 0;
			}
		}

		for (const [endpoint, series] of Object.entries(resp.endpoints)) {
			for (let i = 0; i < series.time.length; i++) {
				overallLatency += Number(series.latency[i]) / 1_000_000;
				overallLatencyCount++;
				const timestamp = Math.floor((Number(series.time[i].seconds) * 1000) / bucket) * bucket;
				if (records[timestamp]) {
					records[timestamp][safe(endpoint)] = Math.floor(Number(series.latency[i]) / 1_000_000);
				}
			}
		}
		const labels: ChartConfig = {};
		for (let i = 0; i < endpoints.length; i++) {
			labels[safe(endpoints[i])] = {
				label: endpoints[i],
				color: `var(--chart-${(i % 5) + 1})`
			};
		}
		return [
			Math.floor(overallLatency / overallLatencyCount),
			labels,
			Object.values(records).sort((a, b) => a.date - b.date)
		];
	}

	let scanState: SubmitState = $state({ error: '', forbidden: false, loading: false });

	$effect(() => {
		Submit(async () => {
			[totalHits, hitLabels, hitData] = await loadHits(window, bucket, from, to);
			[overallLatency, latencyLabels, latencyData] = await loadLatency(window, bucket, from, to);
		}, scanState);
	});
</script>

{#if scanState.error}
	<Alert title="Failed to load chart data" message={scanState.error} />
{:else if scanState.forbidden}
	<Alert title="Permission Denied" message="You are not allowed to view this section" />
{:else}
	<Card class="w-full items-start gap-8">
		<div class="flex w-full flex-col items-start gap-4 border-b pb-5 sm:flex-row sm:items-center sm:gap-20">
			<div>
				<h1 class="text-4xl font-semibold text-primary/90">{totalHits}</h1>
				<p class="text-xl text-primary/60">requests served</p>
			</div>

			<div>
				<h1 class="text-4xl font-semibold text-primary/90">{overallLatency} ms</h1>
				<p class="text-xl text-primary/60">avg latency delivered</p>
			</div>

			<div class="flex items-center gap-2 sm:ml-auto">
				<span class="text-xs text-primary/60">
					{from.toLocaleDateString(undefined, { hour: '2-digit', minute: '2-digit' })}
					–
					{to.toLocaleDateString(undefined, { hour: '2-digit', minute: '2-digit' })}
				</span>
				<Button
					variant="primary"
					title="Zoom out"
					onclick={() => {
						const span = to.getTime() - from.getTime();
						from = new Date(from.getTime() - span / 2);
						to = clampEnd(window, new Date(to.getTime() + span / 2));
					}}
				>
					<ZoomOutIcon />
				</Button>
				<Button
					variant="primary"
					title="Reset"
					onclick={() => {
						from = initialFrom;
						to = clampEnd(window, new Date());
					}}
				>
					<ResetIcon />
				</Button>
			</div>
		</div>

		<div>
			<h2 class="text-xl font-semibold text-primary/80">Endpoint Requests</h2>
			<p class="text-sm text-primary/60">
				Hit count per endpoint in {enumToJson(AggregateWindowSchema, window)?.toString().toLowerCase()}s
			</p>
		</div>
		<ZoomChart
			timeFormat={zoomTimeformat(from, to)}
			labels={hitLabels}
			data={hitData}
			unit={`requests / ${enumToJson(AggregateWindowSchema, window)?.toString().toLowerCase()}`}
			bind:from
			bind:to
		/>

		<div>
			<h2 class="text-xl font-semibold text-primary/80">Endpoint Latency</h2>
			<p class="text-sm text-primary/60">Average endpoint latency in milliseconds</p>
		</div>
		<ZoomChart
			timeFormat={zoomTimeformat(from, to)}
			labels={latencyLabels}
			data={latencyData}
			unit="ms"
			bind:from
			bind:to
		/>
	</Card>
{/if}
