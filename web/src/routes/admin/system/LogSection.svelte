<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import type { ChartConfig } from '$lib/components/chart/chart';
	import Input from '$lib/components/Input.svelte';
	import Select from '$lib/components/Select.svelte';
	import Card from '$lib/components/Card.svelte';
	import {
		AggregateLogsRequestSchema,
		AggregateWindow,
		AggregateWindowSchema
	} from '$lib/sdk/v1/admin/system/system_pb';
	import { create, enumToJson } from '@bufbuild/protobuf';
	import { timestampFromDate } from '@bufbuild/protobuf/wkt';
	import ResetIcon from '@lucide/svelte/icons/rotate-ccw';
	import ZoomOutIcon from '@lucide/svelte/icons/zoom-out';
	import LogPanel from './LogPanel.svelte';
	import ZoomChart from './ZoomChart.svelte';

	const levels = [
		{ value: '', label: 'All levels' },
		{ value: 'DEBUG', label: 'Debug' },
		{ value: 'INFO', label: 'Info' },
		{ value: 'WARN', label: 'Warning' },
		{ value: 'ERROR', label: 'Error' }
	];

	const limits = [
		{ value: '20', label: '20 lines' },
		{ value: '50', label: '50 lines' },
		{ value: '100', label: '100 lines' },
		{ value: '200', label: '200 lines' }
	];

	const levelColors: Record<string, string> = {
		DEBUG: 'var(--color-sky-500)',
		INFO: 'var(--color-emerald-500)',
		WARN: 'var(--color-amber-500)',
		ERROR: 'var(--color-red-500)'
	};

	const initialFrom = new Date(Date.now() - 24 * 60 * 60 * 1000);

	let from = $state(initialFrom);
	let to = $state(new Date());

	let level = $state('');
	let system = $state('');
	let procedure = $state('');
	let challenge = $state('');
	let limit = $state('50');

	let labels: ChartConfig = $state({});
	let data: any[] = $state([]);
	let counts: Record<string, number> = $state({});

	function zoomWindow(from: Date, to: Date): [AggregateWindow, number] {
		const span = to.getTime() - from.getTime();
		if (span <= 6 * 60 * 60 * 1000) return [AggregateWindow.Minute, 60 * 1000];
		if (span <= 7 * 24 * 60 * 60 * 1000) return [AggregateWindow.Hour, 60 * 60 * 1000];
		return [AggregateWindow.Day, 24 * 60 * 60 * 1000];
	}

	let [window, bucket] = $derived(zoomWindow(from, to));

	function zoomTimeformat(from: Date, to: Date) {
		if (to.getTime() - from.getTime() <= 48 * 60 * 60 * 1000) {
			return (v: any) => v.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
		}
		return (v: any) => v.toLocaleDateString('en-US', { month: '2-digit', day: '2-digit' });
	}

	async function loadLevels(window: AggregateWindow, bucket: number, from: Date, to: Date) {
		const resp = await Glue.system.aggregateLogs(
			create(AggregateLogsRequestSchema, {
				window: window,
				from: timestampFromDate(from),
				to: timestampFromDate(to)
			})
		);

		const found = Object.keys(resp.levels);
		const records: Record<number, any> = {};
		const start = Math.floor(from.getTime() / bucket) * bucket;
		for (let t = start; t <= to.getTime(); t += bucket) {
			records[t] = { date: new Date(t) };
			for (const level of found) {
				records[t][level] = 0;
			}
		}

		const totals: Record<string, number> = {};
		for (const [level, series] of Object.entries(resp.levels)) {
			totals[level] = 0;
			for (let i = 0; i < series.time.length; i++) {
				totals[level] += Number(series.count[i]);
				const timestamp = Math.floor((Number(series.time[i].seconds) * 1000) / bucket) * bucket;
				if (records[timestamp]) {
					records[timestamp][level] = Number(series.count[i]);
				}
			}
		}

		const config: ChartConfig = {};
		for (let i = 0; i < found.length; i++) {
			config[found[i]] = {
				label: found[i],
				color: levelColors[found[i]] ?? `var(--chart-${(i % 5) + 1})`
			};
		}
		return [totals, config, Object.values(records).sort((a, b) => a.date - b.date)] as const;
	}

	let levelState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	$effect(() => {
		Submit(async () => {
			[counts, labels, data] = await loadLevels(window, bucket, from, to);
		}, levelState);
	});
</script>

{#if levelState.forbidden}
	<Alert title="Permission Denied" message="You are not allowed to view this section" />
{:else}
	<Card class="w-full items-start gap-6">
		<div class="flex w-full flex-col items-start gap-4 border-b pb-5 sm:flex-row sm:items-center sm:gap-12">
			{#each ['ERROR', 'WARN', 'INFO', 'DEBUG'] as name (name)}
				<div>
					<h1 class="text-3xl font-semibold" style="color: {levelColors[name]}">{counts[name] ?? 0}</h1>
					<p class="text-primary/60">{name.toLowerCase()}</p>
				</div>
			{/each}

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
						to = new Date(Math.min(Date.now(), to.getTime() + span / 2));
					}}
				>
					<ZoomOutIcon />
				</Button>
				<Button
					variant="primary"
					title="Reset"
					onclick={() => ((from = new Date(Date.now() - 24 * 60 * 60 * 1000)), (to = new Date()))}
				>
					<ResetIcon />
				</Button>
			</div>
		</div>

		<div>
			<h2 class="text-xl font-semibold text-primary/80">Log Volume</h2>
			<p class="text-sm text-primary/60">
				Log lines per severity in {enumToJson(AggregateWindowSchema, window)?.toString().toLowerCase()}s. Drag on the
				chart to zoom into a range.
			</p>
		</div>
		<ZoomChart
			timeFormat={zoomTimeformat(from, to)}
			{labels}
			{data}
			unit={`lines / ${enumToJson(AggregateWindowSchema, window)?.toString().toLowerCase()}`}
			bind:from
			bind:to
		/>

		<div class="flex flex-row flex-wrap items-end gap-2">
			<div class="flex flex-col gap-1 text-sm">
				Level
				<Select bind:value={level} class="w-40" items={levels} />
			</div>
			<Input class="w-56" bind:value={system} label="System" placeholder="e.g. server" />
			<Input class="w-72" bind:value={procedure} label="Procedure" placeholder="e.g. /v1.admin.user.UserService/List" />
			<Input class="w-72" bind:value={challenge} label="Challenge" placeholder="00000000-00000000-00000000-00000000" />
			<div class="flex flex-col gap-1 text-sm">
				Lines
				<Select bind:value={limit} class="w-32" items={limits} />
			</div>
			<Button
				variant="primary"
				disabled={!level && !system && !procedure && !challenge}
				onclick={() => ((level = ''), (system = ''), (procedure = ''), (challenge = ''))}
			>
				Clear
			</Button>
		</div>
	</Card>

	<LogPanel {from} {to} {level} {challenge} bind:system bind:procedure limit={Number(limit)} />

	{#if levelState.error}
		<Alert title="Failed to load log data" message={levelState.error} animation={true} />
	{/if}
{/if}
