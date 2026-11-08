<script lang="ts">
	import CostAccountPanel from './CostAccountPanel.svelte';
	import CostProviderPanel from './CostProviderPanel.svelte';

	let selectedProvider = $state('');
	let selectedAccount = $state('');
	let selectedTimewindow = $state('');
</script>

<CostProviderPanel
	select={(providerId, accountId, timewindow) => {
		selectedProvider = providerId;
		selectedAccount = accountId;
		selectedTimewindow = timewindow;
	}}
/>

<CostAccountPanel providerId={selectedProvider} accountId={selectedAccount} timewindow={selectedTimewindow} />
