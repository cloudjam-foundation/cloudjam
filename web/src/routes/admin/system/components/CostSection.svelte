<script lang="ts">
	import CostAccountPanel from './CostAccountPanel.svelte';
	import CostProviderPanel from './CostProviderPanel.svelte';

	let selectedProvider = $state('');
	let selectedAccount = $state('');
	let selectedTimeWindow = $state('');
</script>

<CostProviderPanel
	select={(providerId, accountId, timeWindow) => {
		selectedProvider = providerId;
		selectedAccount = accountId;
		selectedTimeWindow = timeWindow;
	}}
/>

<CostAccountPanel providerId={selectedProvider} accountId={selectedAccount} timeWindow={selectedTimeWindow} />
