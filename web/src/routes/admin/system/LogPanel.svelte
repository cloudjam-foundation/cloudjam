<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Dialog from '$lib/components/Dialog.svelte';
	import Card from '$lib/components/Card.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { ScanLogsRequestSchema, type Log } from '$lib/sdk/v1/admin/system/system_pb';
	import { create } from '@bufbuild/protobuf';
	import { timestampDate, timestampFromDate } from '@bufbuild/protobuf/wkt';
	import CopyIcon from '@lucide/svelte/icons/copy';
	import RefreshIcon from '@lucide/svelte/icons/refresh-cw';
	import XIcon from '@lucide/svelte/icons/x';

	let {
		from,
		to,
		level,
		challenge,
		system = $bindable(),
		procedure = $bindable(),
		limit
	}: {
		from: Date;
		to: Date;
		level: string;
		challenge: string;
		system: string;
		procedure: string;
		limit: number;
	} = $props();

	const severities: Record<string, { short: string; level: string }> = {
		DEBUG: { short: 'DBG', level: 'text-sky-500' },
		INFO: { short: 'INF', level: 'text-emerald-500' },
		WARN: { short: 'WRN', level: 'text-amber-500' },
		ERROR: { short: 'ERR', level: 'text-red-500' }
	};

	let scanState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let count = $state(0);
	let nonce = $state(0);

	// the table is remounted (and therefore rescanned) whenever a filter or the nonce changes.
	const scanKey = $derived([from.getTime(), to.getTime(), level, system, procedure, challenge, limit, nonce].join('-'));

	let selected: Log | undefined = $state();
	let selectedOpen = $state(false);
</script>

<Card class="w-full items-start">
	<div class="flex w-full flex-row flex-wrap items-center gap-2 border-b pb-5">
		<h1 class="text-2xl font-semibold text-primary/90">
			{#if challenge}
				Challenge Logs ({challenge})
			{:else}
				System Logs
			{/if}
		</h1>
		<div class="ml-auto flex flex-row flex-wrap items-center gap-1">
			{#if level}
				<StaticBadge variant="primary" text={level} />
			{/if}
			{#if system}
				<Button variant="primary" class="px-2 py-0.5 text-xs" onclick={() => (system = '')}>
					system: {system}
					<XIcon class="size-3" />
				</Button>
			{/if}
			{#if procedure}
				<Button variant="primary" class="px-2 py-0.5 text-xs" onclick={() => (procedure = '')}>
					procedure: {procedure}
					<XIcon class="size-3" />
				</Button>
			{/if}
			<Button variant="primary" title="Refresh" disabled={scanState.loading} onclick={() => nonce++}>
				<RefreshIcon />
			</Button>
		</div>
	</div>

	{#if scanState.forbidden}
		<p class="text-sm text-primary/60 italic">You are not allowed to read the system logs.</p>
	{:else}
		{#key scanKey}
			<DataTable
				class="max-h-[70vh] w-full"
				limit={Infinity}
				load={async () => {
					let logs: Log[] = [];
					await Submit(async () => {
						const resp = await Glue.system.scanLogs(
							create(ScanLogsRequestSchema, {
								from: timestampFromDate(from),
								to: timestampFromDate(to),
								system: system,
								procedure: procedure,
								level: level,
								limit: limit,
								challenge: challenge
							})
						);
						logs = resp.logs;
					}, scanState);
					count = logs.length;
					return logs;
				}}
				loadState={scanState}
				onRowClick={(log) => {
					selected = log;
					selectedOpen = true;
				}}
				columns={[
					{ title: 'Time', width: '150px', render: renderTime },
					{ title: 'Level', width: '60px', render: renderLevel },
					{ title: 'System', width: '130px', render: renderSystem },
					{ title: 'Procedure', width: '240px', render: renderProcedure },
					{ title: 'Message', width: '100%', render: renderMessage }
				]}
			/>
		{/key}
		{#if count === 0 && !scanState.loading}
			<p class="text-sm text-primary/60 italic">No logs matched the selected range and filters.</p>
		{/if}
		{#snippet renderTime(log: Log)}
			<span class="font-mono text-xs opacity-70">
				{log.timestamp ? timestampDate(log.timestamp).toLocaleString('en-GB') : ''}
			</span>
		{/snippet}
		{#snippet renderLevel(log: Log)}
			<span class="font-mono text-xs font-bold {severities[log.level]?.level ?? ''}">
				{severities[log.level]?.short ?? log.level}
			</span>
		{/snippet}
		{#snippet renderSystem(log: Log)}
			<button
				type="button"
				class="max-w-full cursor-pointer truncate text-left hover:underline"
				title="Filter by {log.system}"
				onclick={(e) => (e.stopPropagation(), (system = system === log.system ? '' : log.system))}
			>
				{log.system || '-'}
			</button>
		{/snippet}
		{#snippet renderProcedure(log: Log)}
			<button
				type="button"
				class="max-w-full cursor-pointer truncate text-left font-mono text-xs hover:underline"
				title="Filter by {log.procedure}"
				onclick={(e) => (e.stopPropagation(), (procedure = procedure === log.procedure ? '' : log.procedure))}
			>
				{log.procedure || '-'}
			</button>
		{/snippet}
		{#snippet renderMessage(log: Log)}
			<span class="font-mono text-xs">{log.message}</span>
		{/snippet}
	{/if}
</Card>

{#if scanState.error}
	<Alert title="Failed to load log data" message={scanState.error} animation={true} />
{/if}

{#if selected}
	<Dialog
		bind:open={selectedOpen}
		title="Log line"
		description={selected.timestamp ? timestampDate(selected.timestamp).toLocaleString('en-GB') : ''}
		class="sm:max-w-[90vw]"
	>
		<div class="flex flex-row flex-wrap items-center gap-2">
			<span class="font-mono text-xs font-bold {severities[selected.level]?.level ?? ''}">
				{severities[selected.level]?.short ?? selected.level}
			</span>
			{#if selected.system}
				<StaticBadge variant="primary" text={`system: ${selected.system}`} />
			{/if}
			{#if selected.procedure}
				<StaticBadge variant="primary" text={`procedure: ${selected.procedure}`} class="font-mono" />
			{/if}
			<Button
				variant="primary"
				title="Copy log line"
				class="ml-auto"
				onclick={() => navigator.clipboard.writeText(JSON.stringify(selected, null, 2))}
			>
				<CopyIcon />
			</Button>
		</div>
		<pre class="font-mono text-xs whitespace-pre-wrap">{selected.message}</pre>
		<span class="text-sm font-medium">Trace</span>
		<pre class="overflow-x-auto font-mono text-xs whitespace-pre-wrap text-primary/60">{selected.trace ||
				'no trace attached'}</pre>
	</Dialog>
{/if}
