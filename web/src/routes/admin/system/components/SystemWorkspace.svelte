<script lang="ts">
	import CostSection from './CostSection.svelte';
	import LogSection from './LogSection.svelte';
	import MetricSection from './MetricSection.svelte';
	import TabPage from '$lib/components/ui/TabPage.svelte';
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

<TabPage title="System" tabs={['logs', 'metrics', 'cost']} tabPages={[logs, metrics, cost]} />
