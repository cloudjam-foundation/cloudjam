<script lang="ts">
	import CostSection from './CostSection.svelte';
	import LogSection from './LogSection.svelte';
	import MetricSection from './MetricSection.svelte';
	import TabPage from '$lib/components/TabPage.svelte';

	let { close }: { close: () => void } = $props();
</script>

{#snippet logs()}
	<LogSection />
{/snippet}

{#snippet metrics()}
	<MetricSection />
{/snippet}

{#snippet cost()}
	<CostSection />
{/snippet}

<TabPage title="System" back={close} tabs={['logs', 'metrics', 'cost']} tabPages={[logs, metrics, cost]} />
