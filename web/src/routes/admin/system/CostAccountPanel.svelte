<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import ProcessingButton from '$lib/components/ProcessingButton.svelte';
	import type { ChartConfig } from '$lib/components/chart/chart';
	import OptionalSelect, { type Suggestion } from '$lib/components/OptionalSelect.svelte';
	import Separator from '$lib/components/Separator.svelte';
	import Card from '$lib/components/Card.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import {
		AggregateCostRequestSchema,
		ListRequestSchema as ListAccountRequestSchema
	} from '$lib/sdk/v1/cloud/account/account_pb';
	import type { Account } from '$lib/sdk/v1/cloud/account_pb';
	import { ListRequestSchema as ListProviderRequestSchema } from '$lib/sdk/v1/cloud/provider/provider_pb';
	import type { Provider } from '$lib/sdk/v1/cloud/provider_pb';
	import { create } from '@bufbuild/protobuf';
	import { durationFromMs } from '@bufbuild/protobuf/wkt';
	import { ChartNoAxesCombinedIcon, ZoomOutIcon } from '@lucide/svelte';
	import ResetIcon from '@lucide/svelte/icons/rotate-ccw';
	import { onMount } from 'svelte';
	import ZoomChart from './ZoomChart.svelte';

	let { providerId, accountId, timewindow }: { providerId: string; accountId: string; timewindow: string } = $props();

	let providerSuggestions: Provider[] = $state([]);
	let providerSuggestionsState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	onMount(() => {
		Submit(async () => {
			providerSuggestions = (await Glue.provider.list(create(ListProviderRequestSchema, { limit: 100 }))).providers;
		}, providerSuggestionsState);
	});

	let accountSuggestions: Account[] = $state([]);
	let accountSuggestionsState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	$effect(() => {
		if (!providerId) return;
		Submit(async () => {
			accountSuggestions = (
				await Glue.account.list(create(ListAccountRequestSchema, { providerId: providerId, limit: 100 }))
			).accounts;
		}, accountSuggestionsState);
	});

	// safe generates "layerchart" safe identifiers / keys.
	const safe = (k: string) => k.replace(/[^a-zA-Z0-9_]/g, '_');
	const now = new Date(Date.now());

	let timewindowSuggestions: Suggestion[] = [
		{ id: String(7 * 24 * 60 * 60 * 1000), title: 'Last 7 days' },
		{ id: String(30 * 24 * 60 * 60 * 1000), title: 'Last 30 days' },
		{ id: String(365 * 24 * 60 * 60 * 1000), title: 'Last Year' },
		{ id: String(now.getTime() - new Date(now.getFullYear(), now.getMonth(), 1).getTime()), title: 'This Month' },
		{ id: String(now.getTime() - new Date(now.getFullYear(), 1, 1).getTime()), title: 'This Year' }
	];

	let costLabels: ChartConfig = $state({});
	// TODO: Find out a way to type this properly
	let costData: any[] = $state([]);
	let totalCost: number = $state(0);
	let serviceCost: Record<string, number> = $state({});
	// bumping the version remounts the service table with the freshly aggregated data.
	let version = $state(0);

	type ServiceCost = { service: string; cost: number };
	const serviceRows = $derived<ServiceCost[]>(
		Object.entries(serviceCost)
			.toSorted((a, b) => b[1] - a[1])
			.map(([service, cost]) => ({ service, cost: Math.round(cost * 100) / 100 }))
			.filter((row) => row.cost > 0)
	);

	async function loadCost(
		providerId: string,
		accountId: string,
		window: number
	): Promise<[number, Record<string, number>, ChartConfig, any[]]> {
		let bucket = 24 * 60 * 60 * 1000; // bucket is always daily
		let totalCost = 0;
		let serviceCost: Record<string, number> = {};
		const resp = await Glue.account.aggregateCost(
			create(AggregateCostRequestSchema, {
				providerId: providerId,
				id: accountId,
				window: durationFromMs(window)
			})
		);
		const services = Object.keys(resp.services);
		const records: Record<number, any> = {};
		const start = Math.floor(new Date(now.getTime() - window).getTime() / bucket) * bucket;
		for (let t = start; t <= now.getTime(); t += bucket) {
			records[t] = { date: new Date(t) };
			for (const service of services) {
				records[t][safe(service)] = 0;
			}
		}

		for (const [service, series] of Object.entries(resp.services)) {
			for (let i = 0; i < series.time.length; i++) {
				totalCost += Number(series.cost[i]);
				if (!serviceCost[service]) serviceCost[service] = 0;
				serviceCost[service] += Number.isNaN(series.cost[i]) ? 0 : series.cost[i];
				const timestamp = Math.floor((Number(series.time[i].seconds) * 1000) / bucket) * bucket;
				if (records[timestamp]) {
					records[timestamp][safe(service)] = Number(series.cost[i]);
				}
			}
		}
		const labels: ChartConfig = {};
		for (let i = 0; i < services.length; i++) {
			labels[safe(services[i])] = {
				label: services[i],
				color: `var(--chart-${(i % 5) + 1})`
			};
		}
		return [totalCost, serviceCost, labels, Object.values(records).sort((a, b) => a.date - b.date)];
	}

	let aggregateState: SubmitState = $state({ error: '', forbidden: false, loading: false });

	let chartTo = $state(new Date());
	let chartFrom = $state(new Date());
</script>

<Card class="w-full items-start gap-8">
	<div class="flex w-full flex-col items-start gap-4 border-b pb-5 sm:flex-row sm:items-center sm:gap-20">
		<div>
			<h1 class="text-4xl font-semibold text-primary/90">{Math.round(totalCost * 100) / 100} $</h1>
			<p class="text-xl text-primary/60">total account cost</p>
		</div>

		<div class="flex flex-wrap items-center gap-2 sm:ml-auto">
			<OptionalSelect
				bind:value={timewindow}
				placeholder="Select Timewindow"
				suggestions={timewindowSuggestions}
				class="w-64"
			/>
			<OptionalSelect
				bind:value={providerId}
				placeholder="Select Provider"
				suggestions={providerSuggestions.map((provider) => ({ id: provider.id, title: provider.name }))}
				class="w-64"
			/>
			<OptionalSelect
				bind:value={accountId}
				placeholder="Select Account"
				suggestions={accountSuggestions.map((account) => ({ id: account.id, title: account.name }))}
				class="w-64"
			/>
			<ProcessingButton
				variant="primary"
				processing={aggregateState.loading}
				hint="Aggregating..."
				disabled={!timewindow || !providerId || !accountId}
				onclick={() => {
					Submit(async () => {
						[totalCost, serviceCost, costLabels, costData] = await loadCost(providerId, accountId, Number(timewindow));
						chartFrom = new Date(now.getTime() - Number(timewindow));
						chartTo = now;
						version++;
					}, aggregateState);
				}}
			>
				<ChartNoAxesCombinedIcon />
				Aggregate Cost
			</ProcessingButton>
		</div>
	</div>

	{#if aggregateState.error}
		<Alert title="Failed to load chart data" message={aggregateState.error} />
	{:else if aggregateState.forbidden}
		<Alert title="Permission Denied" message="You are not allowed to view this section" />
	{:else if costData.length > 0}
		<ZoomChart
			timeFormat={(v: any) => {
				return v.toLocaleDateString('en-US', {
					month: '2-digit',
					day: '2-digit'
				});
			}}
			labels={costLabels}
			data={costData}
			unit="$ / day"
			bind:from={chartFrom}
			bind:to={chartTo}
		/>

		<div class="flex flex-row items-center gap-2">
			<Button
				variant="primary"
				title="Zoom out"
				onclick={() => {
					const span = chartTo.getTime() - chartFrom.getTime();
					chartFrom = new Date(Math.max(chartFrom.getTime() - span / 2, now.getTime() - Number(timewindow)));
					chartTo = new Date(Math.min(chartTo.getTime() + span / 2, now.getTime()));
				}}
			>
				<ZoomOutIcon />
			</Button>
			<Button
				variant="primary"
				title="Reset"
				onclick={() => {
					chartFrom = new Date(now.getTime() - Number(timewindow));
					chartTo = now;
				}}
			>
				<ResetIcon />
			</Button>
		</div>

		<Separator />

		{#key version}
			<DataTable
				class="max-h-[60vh] w-full"
				limit={Infinity}
				load={async () => serviceRows}
				loadState={{ error: '', loading: false, forbidden: false }}
				columns={[
					{ title: 'Service', width: '70%', render: renderService },
					{ title: 'Cost', width: '30%', render: renderCost }
				]}
			/>
		{/key}
		{#snippet renderService(row: ServiceCost)}
			{row.service}
		{/snippet}
		{#snippet renderCost(row: ServiceCost)}
			{row.cost.toFixed(2)} $
		{/snippet}
	{:else}
		<div class="flex h-52 w-full items-center justify-center">
			<span class="text-4xl text-muted">No Cost Report</span>
		</div>
	{/if}
</Card>
