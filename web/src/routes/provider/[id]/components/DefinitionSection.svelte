<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import DataTable from '$lib/components/DataTable.svelte';
	import { toDigest } from '$lib/utils.svelte.js';
	import { ListRequestSchema } from '$lib/sdk/v1/cloud/definition/definition_pb.ts';
	import type { Definition } from '$lib/sdk/v1/cloud/definition_pb.ts';
	import { create } from '@bufbuild/protobuf';

	let { providerId }: { providerId: string } = $props();

	let definitionsState: SubmitState = $state({ error: '', forbidden: false, loading: false });
</script>

{#if definitionsState.forbidden}
	<Alert title="Permission Denied" message="You are not allowed to list the definitions of this provider" />
{:else}
	<DataTable
		class="max-h-[80vh]"
		limit={100}
		load={async (startAfter: Definition | undefined) => {
			const definitions: Definition[] = [];
			await Submit(async () => {
				const resp = await Glue.definition.list(
					create(ListRequestSchema, { providerId: providerId, limit: 100, startAfter: startAfter?.id })
				);
				definitions.push(...resp.definitions);
			}, definitionsState);
			return definitions;
		}}
		loadState={definitionsState}
		columns={[
			{ title: 'Name', width: '20%', render: renderName },
			{ title: 'Version', width: '10%', render: renderVersion },
			{ title: 'Description', width: '40%', render: renderDescription },
			{ title: 'Digest', width: '30%', render: renderDigest }
		]}
	/>
	{#snippet renderName(definition: Definition)}
		{definition.name}
	{/snippet}
	{#snippet renderVersion(definition: Definition)}
		{definition.version}
	{/snippet}
	{#snippet renderDescription(definition: Definition)}
		<span class="text-primary/60">{definition.description}</span>
	{/snippet}
	{#snippet renderDigest(definition: Definition)}
		<span class="font-mono text-xs">{toDigest(definition.hash)}</span>
	{/snippet}
{/if}
