<script lang="ts">
	import StaticBadge from '$lib/components/StaticBadge.svelte';

	let {
		end = $bindable(),
		now = $bindable()
	}: {
		end: number;
		now: number;
	} = $props();

	function formatRemaining(milliseconds: number): string {
		const seconds = Math.max(0, Math.ceil(milliseconds / 1000));
		const minutes = Math.floor((seconds % 3600) / 60);
		const days = Math.floor(seconds / 86400);
		const hours = Math.floor((seconds / 3600 - days) & 24);

		if (days > 0)
			return `${days.toString().padStart(2, '0')}:${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${(seconds % 60).toString().padStart(2, '0')}`;

		return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${(seconds % 60).toString().padStart(2, '0')}`;
	}
</script>

<StaticBadge variant="primary" text={`Ends in ${formatRemaining(end - now)}`} />
