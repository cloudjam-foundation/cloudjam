<script lang="ts">
	import { goto } from '$app/navigation';
	import { Glue, Submit, type SubmitState } from '$lib';
	import Alert from '$lib/components/Alert.svelte';
	import Button from '$lib/components/Button.svelte';
	import Input from '$lib/components/Input.svelte';
	import OptionalSelect from '$lib/components/OptionalSelect.svelte';
	import Card from '$lib/components/Card.svelte';
	import Select from '$lib/components/Select.svelte';
	import { scopes } from '$lib/scopes.svelte';
	import { CreateRequestSchema } from '$lib/sdk/v1/cloud/provider/provider_pb';
	import { ProviderSchema, ProviderType } from '$lib/sdk/v1/cloud/provider_pb';
	import { create } from '@bufbuild/protobuf';
	import AwsCredentials from '../AwsCredentials.svelte';

	const types = [{ value: String(ProviderType.AWS), label: 'AWS' }];

	let init = $state(create(ProviderSchema, { id: crypto.randomUUID() }));
	let type = $state(String(ProviderType.AWS));
	let regions = $state('');

	let createState: SubmitState = $state({ error: '', loading: false, forbidden: false });

	let createRequest = $derived(
		create(CreateRequestSchema, {
			init: {
				...init,
				type: Number(type) as ProviderType,
				regions: regions
					.split(',')
					.map((region) => region.trim())
					.filter((region) => region)
			}
		})
	);
</script>

<Card class="w-full items-start">
	<div>
		<h1 class="text-xl font-semibold text-primary/80">Add Provider</h1>
		<p class="text-primary/60">
			Registers a CloudJam cloud provider. Depending on provider this may already provision some bootstrap
			infrastructure.
		</p>
	</div>
	<form
		class="flex w-full flex-col gap-4"
		onsubmit={() =>
			Submit(async () => {
				const resp = await Glue.provider.create(createRequest);
				goto(`/provider/${resp.id}`);
			}, createState)}
	>
		<div class="grid gap-4 md:grid-cols-2">
			<Input
				bind:value={init.name}
				label="Name"
				placeholder="Name of the provider"
				validation={Glue.Validate(ProviderSchema, init).violation.name}
			/>
			<div class="flex flex-col gap-1">
				<label class="text-sm">
					Type
					<Select bind:value={type} class="w-full" items={types} />
				</label>
			</div>
			<Input
				bind:value={init.description}
				label="Description"
				placeholder="Description of the provider"
				validation={Glue.Validate(ProviderSchema, init).violation.description}
			/>
			<Input
				bind:value={init.email}
				label="Email"
				placeholder="Root email of the provider (and prefix for accounts)"
				validation={Glue.Validate(ProviderSchema, init).violation.email}
			/>
			<Input
				bind:value={regions}
				label="Regions"
				placeholder="Comma separated list of supported regions (e.g. us-east-1, eu-central-1, ...)"
			/>
			<div class="flex flex-col gap-1">
				<label class="text-sm">
					Scope
					<OptionalSelect
						bind:value={init.scope}
						placeholder="Scope the provider is placed in"
						suggestions={scopes.map((scope) => ({ id: scope, title: scope }))}
					/>
				</label>
				<p class="text-xs text-primary/60">You can only attach a scope you possess yourself.</p>
			</div>
		</div>

		<div class="flex flex-col gap-2">
			<h2 class="text-sm font-medium">Credentials</h2>
			{#if Number(type) === ProviderType.AWS}
				<AwsCredentials bind:value={init.credentials} />
			{:else}
				<Input bind:value={init.credentials} placeholder="Provider specific credentials" />
			{/if}
			<p class="text-xs text-destructive">{Glue.Validate(ProviderSchema, init).violation.credentials ?? ''}</p>
		</div>
		<Button
			type="submit"
			variant="primary"
			class="self-start"
			processing={createState.loading}
			disabled={createState.loading || Boolean(Glue.Validate(CreateRequestSchema, createRequest).error)}
		>
			Add
		</Button>
		{#if createState.error}
			<Alert
				title={createState.forbidden ? 'Permission denied' : 'Failed to add provider'}
				message={createState.error}
				animation={true}
			/>
		{/if}
	</form>
</Card>
