<script lang="ts">
	import { featureMap } from './featureMap';
	import { Feature } from '$lib/sdk/v1/admin/role_pb';
	import Input from '$lib/components/inputs/Input.svelte';

	let { features = $bindable() }: { features: Record<Feature, boolean> } = $props();
</script>

<div class="flex flex-col flex-wrap items-start justify-start gap-2">
	{#each featureMap as feature (feature.id)}
		<button
			class="flex flex-row items-center justify-center hover:cursor-pointer"
			id={feature.title}
			type="button"
			onclick={() => (features[feature.id] = !features[feature.id])}
		>
			<Input type="checkbox" bind:checked={features[feature.id]} />
			<span class="flex flex-col items-start justify-start pl-2 text-left">
				<span class="font-medium">{feature.title}</span>
				<span class="text-muted-foreground">{feature.description}</span>
			</span>
		</button>
	{/each}
</div>
