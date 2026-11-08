<script lang="ts">
	import Button from '$lib/components/Button.svelte';
	import { Feature } from '$lib/sdk/v1/admin/role_pb.ts';

	let { features = $bindable() }: { features: Record<Feature, boolean> } = $props();
</script>

<div class="flex flex-wrap items-center justify-start gap-2">
	{#each Object.keys(Feature).filter((key) => isNaN(Number(key))) as featureName (featureName)}
		{@const feature = Feature[featureName as keyof typeof Feature]}
		{@const active = features[feature]}
		<Button
			type="button"
			variant={active ? 'primary' : 'ghost'}
			class="text-sm"
			onclick={() => (features[feature] = !active)}
		>
			{featureName}
		</Button>
	{/each}
</div>
