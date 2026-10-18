<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import * as Alert from '$lib/components/shad/alert';
	import * as Card from '$lib/components/shad/card';
	import * as Table from '$lib/components/shad/table';
	import AlertCircleIcon from '@lucide/svelte/icons/alert-circle';
	import { create } from '@bufbuild/protobuf';
	import { durationFromMs } from '@bufbuild/protobuf/wkt';
	import * as Chart from '$lib/components/shad/chart/index.js';
	import ZoomChart from './ZoomChart.svelte';
	import { AggregateCostRequestSchema } from '$lib/sdk/v1/cloud/account/account_pb';
	import OptionalSelect, { type Suggestion } from '$lib/components/custom/OptionalSelect.svelte';
	import type { Provider } from '$lib/sdk/v1/cloud/provider_pb';
	import { onMount } from 'svelte';
	import type { Account } from '$lib/sdk/v1/cloud/account_pb';
	import { ListRequestSchema as ListProviderRequestSchema } from '$lib/sdk/v1/cloud/provider/provider_pb';
	import { ListRequestSchema as ListAccountRequestSchema } from '$lib/sdk/v1/cloud/account/account_pb';
	import ProcessingButton from '$lib/components/custom/buttons/ProcessingButton.svelte';
	import { ChartNoAxesCombinedIcon, ZoomOutIcon } from '@lucide/svelte';
	import Separator from '$lib/components/shad/separator/separator.svelte';
	import Button from '$lib/components/shad/button/button.svelte';
	import ResetIcon from '@lucide/svelte/icons/rotate-ccw';

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

	let costLabels: Chart.ChartConfig = $state({});
	let costData: any[] = $state([]);
	let totalCost: number = $state(0);
	let serviceCost: Record<string, number> = $state({});

	async function loadCost(
		providerId: string,
		accountId: string,
		window: number
	): Promise<[number, Record<string, number>, Chart.ChartConfig, any[]]> {
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
		const labels: Chart.ChartConfig = {};
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

<Card.Root class="w-full">
	<Card.Header class="flex items-center justify-start gap-20 space-y-0 border-b py-5 sm:flex-row">
		<div>
			<Card.Title class="text-4xl">
				{Math.round(totalCost * 100) / 100} $
			</Card.Title>
			<Card.Description class="text-xl">total account cost</Card.Description>
		</div>

		<div class="ml-auto flex items-center gap-2">
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
					}, aggregateState);
				}}
			>
				<ChartNoAxesCombinedIcon />
				Aggregate Cost
			</ProcessingButton>
		</div>
	</Card.Header>
	<Card.Content class="flex flex-col gap-8">
		{#if aggregateState.error}
			<Alert.Root variant="destructive">
				<AlertCircleIcon />
				<Alert.Title>Failed to load chart data</Alert.Title>
				<Alert.Description>{aggregateState.error}</Alert.Description>
			</Alert.Root>
		{:else if aggregateState.forbidden}
			<Alert.Root variant="destructive">
				<AlertCircleIcon />
				<Alert.Title>Permission Denied</Alert.Title>
				<Alert.Description>You are not allowed to view this section</Alert.Description>
			</Alert.Root>
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
				unit={`$ / day`}
				bind:from={chartFrom}
				bind:to={chartTo}
			/>

			<div class="flex flex-row items-center gap-2">
				<Button
					variant="outline"
					size="icon"
					title="Zoom out"
					class="cursor-pointer"
					onclick={() => {
						const span = chartTo.getTime() - chartFrom.getTime();
						chartFrom = new Date(Math.max(chartFrom.getTime() - span / 2, now.getTime() - Number(timewindow)));
						chartTo = new Date(Math.min(chartTo.getTime() + span / 2, now.getTime()));
					}}
				>
					<ZoomOutIcon />
				</Button>
				<Button
					variant="outline"
					size="icon"
					title="Reset"
					class="cursor-pointer"
					onclick={() => {
						chartFrom = new Date(now.getTime() - Number(timewindow));
						chartTo = now;
					}}
				>
					<ResetIcon />
				</Button>
			</div>

			<Separator />

			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head class="font-bold">Service</Table.Head>
						<Table.Head class="w-min font-bold">Cost</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each Object.entries(serviceCost).toSorted((a, b) => b[1] - a[1]) as [service, cost]}
						{@const roundedCost = Math.round(cost * 100) / 100}
						{#if roundedCost > 0}
							<Table.Row class="odd:bg-muted/20">
								<Table.Cell>{service}</Table.Cell>
								<Table.Cell>{roundedCost.toFixed(2)} $</Table.Cell>
							</Table.Row>
						{/if}
					{/each}
				</Table.Body>
			</Table.Root>
		{:else}
			<div class="flex h-52 w-full items-center justify-center">
				<span class="text-4xl text-muted">No Cost Report</span>
			</div>
		{/if}
	</Card.Content>
</Card.Root>
