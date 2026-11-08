<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import LoadingBadge from '$lib/components/LoadingBadge.svelte';
	import StaticBadge from '$lib/components/StaticBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Popup from '$lib/components/Popup.svelte';
	import Separator from '$lib/components/Separator.svelte';
	import { compress } from '../compress.ts';
	import { toDigest } from '$lib/utils.svelte.js';
	import { DeleteRequestSchema, UpdateRequestSchema } from '$lib/sdk/v1/cloud/definition/definition_pb.ts';
	import { CompressionMode, DefinitionSchema, type Definition } from '$lib/sdk/v1/cloud/definition_pb.ts';
	import { create } from '@bufbuild/protobuf';

	let {
		open = $bindable(),
		definition,
		refresh
	}: { open: boolean; definition: Definition; refresh: () => void } = $props();

	let mod = $derived({ ...definition });
	let files: FileList | undefined = $state();
	let confirmDelete = $state(false);

	let updateState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let removeState: SubmitState = $state({ error: '', loading: false, forbidden: false });
</script>

<Popup bind:open title={definition.name} description={definition.description} badges={renderBadges} body={renderBody} />

{#snippet renderBadges()}
	<StaticBadge variant="jam" text={`version: ${definition.version || 'unversioned'}`} />
	<StaticBadge variant="primary" text={`scope: ${definition.scope}`} />
	<StaticBadge variant="primary" text={definition.id} class="font-mono" />
	{#if definition.hash}
		<StaticBadge variant="primary" text={toDigest(definition.hash)} class="font-mono" />
	{/if}
{/snippet}

{#snippet renderBody()}
	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Definition</h1>
		{#if updateState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to update this definition.</p>
		{:else}
			<form
				class="flex flex-col gap-4"
				onsubmit={() =>
					Submit(async () => {
						const binary = files?.[0] ? await compress(new Uint8Array(await files[0].arrayBuffer())) : new Uint8Array();
						await Glue.definition.update({
							...create(UpdateRequestSchema, { mod: mod, compression: CompressionMode.Zstd }),
							binary: binary
						});
						files = undefined;
						refresh();
					}, updateState)}
			>
				<div class="grid gap-4 md:grid-cols-2">
					<Input
						bind:value={mod.name}
						label="Name"
						placeholder="Name of the challenge definition"
						validation={Glue.Validate(DefinitionSchema, mod).violation.name}
					/>
					<Input
						bind:value={mod.description}
						label="Description"
						placeholder="Description of the challenge definition"
						validation={Glue.Validate(DefinitionSchema, mod).violation.description}
					/>
					<Input
						bind:value={mod.version}
						label="Version"
						placeholder="Plugin Version (currently irrelevant)"
						validation={Glue.Validate(DefinitionSchema, mod).violation.version}
					/>
					<div class="flex flex-col gap-1">
						<Input type="file" accept=".wasm" bind:files label="Plugin" />
						<p class="text-xs text-primary/60">Optional; the stored plugin is kept if no file is selected.</p>
					</div>
				</div>

				<div class="flex flex-row items-center justify-start gap-2">
					<Button type="submit" variant="primary" class="self-start" disabled={updateState.loading}>
						Save and Upload
					</Button>
					{#if updateState.loading && files}
						<LoadingBadge text="Compressing and Uploading" />
					{/if}
				</div>
			</form>
			{#if updateState.error}
				<Alert class="p-2" message={updateState.error} animation={true} />
			{/if}
		{/if}
	</div>

	<Separator />

	<div class="flex flex-col gap-2">
		<h1 class="text-xl font-semibold text-primary/80">Danger Zone</h1>
		{#if removeState.forbidden}
			<p class="text-sm text-primary/60 italic">You are not allowed to delete this definition.</p>
		{:else}
			<div class="flex flex-row items-center gap-2">
				{#if confirmDelete}
					<Button
						variant="destructive"
						disabled={removeState.loading}
						onclick={() =>
							Submit(async () => {
								await Glue.definition.delete(
									create(DeleteRequestSchema, { providerId: definition.providerId, id: definition.id })
								);
								refresh();
							}, removeState)}
					>
						Yes, delete {definition.name}
					</Button>
					<Button variant="primary" onclick={() => (confirmDelete = false)}>Cancel</Button>
				{:else}
					<Button variant="destructive" onclick={() => (confirmDelete = true)}>Delete Definition</Button>
				{/if}
			</div>
			{#if removeState.error}
				<Alert class="p-2" message={removeState.error} animation={true} />
			{/if}
		{/if}
	</div>
{/snippet}
