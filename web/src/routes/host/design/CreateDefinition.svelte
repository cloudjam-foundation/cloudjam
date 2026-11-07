<script lang="ts">
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import LoadingBadge from '$lib/components/LoadingBadge.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import Card from '$lib/components/Card.svelte';
	import { compress } from './compress';
	import { CreateRequestSchema } from '$lib/sdk/v1/cloud/definition/definition_pb';
	import { CompressionMode, DefinitionSchema } from '$lib/sdk/v1/cloud/definition_pb';
	import type { Provider } from '$lib/sdk/v1/cloud/provider_pb';
	import { create } from '@bufbuild/protobuf';

	let { provider, refresh }: { provider: Provider; refresh: () => void } = $props();

	let init = $derived(create(DefinitionSchema, { id: crypto.randomUUID(), scope: provider.scope }));
	let files: FileList | undefined = $state();

	let createState: SubmitState = $state({ error: '', loading: false, forbidden: false });
	let createRequest = $derived(
		create(CreateRequestSchema, { init: { ...init, providerId: provider.id }, compression: CompressionMode.Zstd })
	);
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Upload Definition</h1>
		<p class="text-primary/60">
			Uploads a compiled challenge plugin to the provider you pick. The wasm binary is zstd compressed in your browser
			before it is sent.
		</p>
	</div>
	<form
		class="flex w-full flex-col gap-4"
		onsubmit={() =>
			Submit(async () => {
				const binary = files?.[0] ? await compress(new Uint8Array(await files[0].arrayBuffer())) : new Uint8Array();
				await Glue.definition.create({ ...createRequest, binary: binary });
				init = create(DefinitionSchema, { id: crypto.randomUUID(), scope: provider.scope });
				files = undefined;
				refresh();
			}, createState)}
	>
		<div class="grid gap-4 md:grid-cols-2">
			<Input
				bind:value={init.name}
				label="Name"
				placeholder="Name of the challenge definition"
				validation={Glue.Validate(DefinitionSchema, init).violation.name}
			/>
			<Input
				bind:value={init.description}
				label="Description"
				placeholder="Description of the challenge definition"
				validation={Glue.Validate(DefinitionSchema, init).violation.description}
			/>
			<Input
				bind:value={init.version}
				label="Version"
				placeholder="Plugin Version (currently irrelevant)"
				validation={Glue.Validate(DefinitionSchema, init).violation.version}
			/>
			<div class="flex flex-col gap-1 md:col-span-2">
				<Input type="file" accept=".wasm" bind:files label="Plugin" />
				<p class="text-xs text-primary/60">
					The compiled challenge.wasm (must be smaller than 50 MB after zstd compression)
				</p>
			</div>
		</div>
		<div class="flex flex-row items-center justify-start gap-2">
			<Button
				type="submit"
				variant="primary"
				class="self-start"
				disabled={createState.loading ||
					!files?.length ||
					Boolean(Glue.Validate(CreateRequestSchema, createRequest).error)}
			>
				Upload
			</Button>
			{#if createState.loading}
				<LoadingBadge text="Compressing and Uploading" />
			{/if}
		</div>

		{#if createState.error}
			<Alert
				title={createState.forbidden ? 'Permission denied' : 'Failed to upload definition'}
				message={createState.error}
				animation={true}
			/>
		{/if}
	</form>
</Card>
